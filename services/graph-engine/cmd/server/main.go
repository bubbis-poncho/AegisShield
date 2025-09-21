package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aegisshield/graph-engine/internal/analytics"
	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/database"
	"github.com/aegisshield/graph-engine/internal/engine"
	"github.com/aegisshield/graph-engine/internal/handlers"
	"github.com/aegisshield/graph-engine/internal/interceptors"
	"github.com/aegisshield/graph-engine/internal/kafka"
	"github.com/aegisshield/graph-engine/internal/metrics"
	"github.com/aegisshield/graph-engine/internal/neo4j"
	"github.com/aegisshield/graph-engine/internal/patterns"
	"github.com/aegisshield/graph-engine/internal/resolution"
	"github.com/aegisshield/graph-engine/internal/server"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	pb "github.com/aegisshield/shared/proto"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting Graph Engine Service",
		"version", "1.0.0",
		"environment", cfg.Environment)

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// Initialize database connection
	db, err := database.NewConnection(cfg.Database, logger)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(cfg.Database.URL); err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Initialize repository
	repo := database.NewRepository(db, logger)

	// Initialize Neo4j client
	neo4jClient, err := neo4j.NewClient(cfg.Neo4j, logger)
	if err != nil {
		logger.Error("Failed to connect to Neo4j", "error", err)
		os.Exit(1)
	}
	defer neo4jClient.Close()

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Error("Failed to create Kafka producer", "error", err)
		os.Exit(1)
	}
	defer kafkaProducer.Close()

	// Initialize graph engine
	graphEngine := engine.NewGraphEngine(
		repo,
		neo4jClient,
		kafkaProducer,
		cfg,
		metricsCollector,
		logger,
	)

	// Initialize gRPC server
	grpcServer := server.NewGRPCServer(graphEngine, cfg, logger)

	// Setup gRPC interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		interceptors.LoggingInterceptor(logger),
		interceptors.MetricsInterceptor(metricsCollector),
		interceptors.RecoveryInterceptor(logger),
		interceptors.ValidationInterceptor(logger),
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		interceptors.StreamLoggingInterceptor(logger),
		interceptors.StreamRecoveryInterceptor(logger),
	}

	// Create gRPC server with interceptors
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.ChainUnaryInterceptors(unaryInterceptors...)),
		grpc.StreamInterceptor(interceptors.ChainStreamInterceptors(streamInterceptors...)),
	)

	// Register gRPC service
	pb.RegisterGraphEngineServer(grpcSrv, grpcServer)

	// Initialize pattern detector
	patternDetector := patterns.NewPatternDetector(neo4jClient, logger)

	// Initialize graph analytics
	graphAnalytics := analytics.NewGraphAnalytics(neo4jClient, logger)

	// Initialize entity resolver
	entityResolver := resolution.NewEntityResolver(neo4jClient, logger)

	// Initialize HTTP handlers
	httpHandlers := handlers.NewHTTPHandlers(graphEngine, cfg, logger)
	enhancedHandlers := handlers.NewEnhancedHTTPHandlers(
		graphEngine,
		patternDetector,
		graphAnalytics,
		entityResolver,
		cfg,
		logger,
	)

	// Setup HTTP router
	router := mux.NewRouter()
	
	// Register routes
	httpHandlers.RegisterRoutes(router)
	enhancedHandlers.RegisterEnhancedRoutes(router)
	
	// Add Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Apply HTTP middleware
	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, graphEngine, logger)
	if err != nil {
		logger.Error("Failed to create Kafka consumer", "error", err)
		os.Exit(1)
	}
	defer kafkaConsumer.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		logger.Error("Failed to create gRPC listener", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("Starting gRPC server", "port", cfg.Server.GRPCPort)
		if err := grpcSrv.Serve(grpcListener); err != nil {
			logger.Error("gRPC server failed", "error", err)
			cancel()
		}
	}()

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", "port", cfg.Server.HTTPPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			cancel()
		}
	}()

	// Start Kafka consumer
	go func() {
		logger.Info("Starting Kafka consumer")
		if err := kafkaConsumer.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("Kafka consumer failed", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	logger.Info("Starting graceful shutdown")

	// Stop gRPC server
	grpcSrv.GracefulStop()

	// Stop HTTP server
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer httpCancel()
	if err := httpSrv.Shutdown(httpCtx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
	}

	// Cancel context to stop Kafka consumer
	cancel()

	logger.Info("Graph Engine Service shutdown completed")
}