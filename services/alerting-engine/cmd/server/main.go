package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
	"github.com/aegis-shield/services/alerting-engine/internal/handlers"
	"github.com/aegis-shield/services/alerting-engine/internal/interceptors"
	"github.com/aegis-shield/services/alerting-engine/internal/kafka"
	"github.com/aegis-shield/services/alerting-engine/internal/metrics"
	"github.com/aegis-shield/services/alerting-engine/internal/notification"
	"github.com/aegis-shield/services/alerting-engine/internal/scheduler"
	"github.com/aegis-shield/services/alerting-engine/internal/server"
	alertingpb "github.com/aegis-shield/shared/proto"
)

const (
	serviceName = "alerting-engine"
	version     = "1.0.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logger := setupLogging(cfg)
	logger.Info("Starting Alerting Engine Service",
		"service", serviceName,
		"version", version,
		"environment", cfg.Environment)

	// Setup database connection
	db, err := database.Connect(cfg.Database.ConnectionString)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", "error", err)
		}
	}()

	// Run database migrations
	if err := database.RunMigrations(db, logger); err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Setup repositories
	alertRepo := database.NewAlertRepository(db, logger)
	ruleRepo := database.NewRuleRepository(db, logger)
	notificationRepo := database.NewNotificationRepository(db, logger)
	escalationRepo := database.NewEscalationRepository(db, logger)

	// Setup notification manager
	notificationManager := notification.NewManager(cfg, logger)

	// Setup rule engine
	ruleEngine := engine.NewRuleEngine(cfg, logger, ruleRepo)

	// Setup scheduler for periodic tasks
	taskScheduler := scheduler.NewScheduler(cfg, logger)

	// Setup Kafka event processor
	eventProcessor := kafka.NewEventProcessor(cfg, logger, ruleEngine, alertRepo, notificationRepo)

	// Setup metrics collector
	metricsCollector := metrics.NewCollector(
		cfg,
		logger,
		alertRepo,
		ruleRepo,
		notificationRepo,
		escalationRepo,
		ruleEngine,
		notificationManager,
		eventProcessor,
		taskScheduler,
	)

	// Setup gRPC interceptors
	grpcInterceptors := interceptors.NewInterceptors(cfg, logger, metricsCollector)

	// Setup gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcInterceptors.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpcInterceptors.StreamServerInterceptor()),
	)

	// Register gRPC service
	alertingGRPCServer := server.NewGRPCServer(
		cfg,
		logger,
		alertRepo,
		ruleRepo,
		notificationRepo,
		escalationRepo,
		ruleEngine,
		notificationManager,
		eventProcessor,
		taskScheduler,
	)
	alertingpb.RegisterAlertingEngineServer(grpcServer, alertingGRPCServer)

	// Enable gRPC reflection for development
	if cfg.Debug {
		reflection.Register(grpcServer)
	}

	// Setup HTTP handlers
	httpHandlers := handlers.NewHTTPHandler(
		cfg,
		logger,
		alertRepo,
		ruleRepo,
		notificationRepo,
		escalationRepo,
		ruleEngine,
		notificationManager,
		eventProcessor,
		taskScheduler,
	)

	// Setup HTTP router
	httpRouter := mux.NewRouter()
	httpHandlers.RegisterRoutes(httpRouter)

	// Add Prometheus metrics endpoint
	httpRouter.Handle("/metrics", promhttp.Handler())

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      httpRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start metrics collector
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := metricsCollector.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("Metrics collector failed", "error", err)
			cancel()
		}
	}()

	// Start event processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := eventProcessor.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("Event processor failed", "error", err)
			cancel()
		}
	}()

	// Start scheduler
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := taskScheduler.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("Scheduler failed", "error", err)
			cancel()
		}
	}()

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
		if err != nil {
			logger.Error("Failed to listen on gRPC port", "port", cfg.Server.GRPCPort, "error", err)
			cancel()
			return
		}

		logger.Info("Starting gRPC server", "port", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			logger.Error("gRPC server failed", "error", err)
			cancel()
		}
	}()

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting HTTP server", "port", cfg.Server.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			cancel()
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down")
	}

	// Graceful shutdown
	logger.Info("Shutting down services...")

	// Cancel context to stop all services
	cancel()

	// Stop HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown HTTP server gracefully", "error", err)
	}

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Wait for all goroutines to finish
	wg.Wait()

	logger.Info("Service shutdown complete")
}

// setupLogging configures structured logging
func setupLogging(cfg *config.Config) *slog.Logger {
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}

	handlerOptions := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: cfg.Debug,
	}

	var handler slog.Handler
	if cfg.Environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	}

	logger := slog.New(handler)
	logger = logger.With(
		"service", serviceName,
		"version", version,
		"environment", cfg.Environment,
	)

	slog.SetDefault(logger)
	return logger
}