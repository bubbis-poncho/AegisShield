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

	"github.com/aegisshield/entity-resolution/internal/config"
	"github.com/aegisshield/entity-resolution/internal/database"
	"github.com/aegisshield/entity-resolution/internal/handlers"
	"github.com/aegisshield/entity-resolution/internal/interceptors"
	"github.com/aegisshield/entity-resolution/internal/kafka"
	"github.com/aegisshield/entity-resolution/internal/matching"
	"github.com/aegisshield/entity-resolution/internal/metrics"
	"github.com/aegisshield/entity-resolution/internal/neo4j"
	"github.com/aegisshield/entity-resolution/internal/resolver"
	"github.com/aegisshield/entity-resolution/internal/server"
	"github.com/aegisshield/entity-resolution/internal/standardization"
	pb "github.com/aegisshield/shared/proto"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting Entity Resolution Service",
		"version", "1.0.0",
		"grpc_port", cfg.Server.GRPCPort,
		"http_port", cfg.Server.HTTPPort,
		"database_host", cfg.Database.Host,
		"kafka_brokers", cfg.Kafka.Brokers,
		"neo4j_uri", cfg.Neo4j.URI)

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()
	metricsCollector.Register()

	// Initialize database repository
	repository, err := database.NewRepository(cfg.Database, logger)
	if err != nil {
		logger.Error("Failed to initialize database repository", "error", err)
		os.Exit(1)
	}
	defer repository.Close()

	// Run database migrations
	if err := repository.Migrate(); err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Neo4j client
	neo4jClient, err := neo4j.NewClient(cfg.Neo4j, logger)
	if err != nil {
		logger.Error("Failed to initialize Neo4j client", "error", err)
		os.Exit(1)
	}
	defer neo4jClient.Close()

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Error("Failed to initialize Kafka producer", "error", err)
		os.Exit(1)
	}
	defer kafkaProducer.Close()

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, logger)
	if err != nil {
		logger.Error("Failed to initialize Kafka consumer", "error", err)
		os.Exit(1)
	}
	defer kafkaConsumer.Close()

	// Initialize standardization engine
	standardizer := standardization.NewEngine(logger)

	// Initialize matching engine
	matcher := matching.NewEngine(cfg.Matching, standardizer, logger)

	// Initialize entity resolver
	entityResolver := resolver.NewEntityResolver(
		repository,
		neo4jClient,
		kafkaProducer,
		standardizer,
		matcher,
		metricsCollector,
		logger,
	)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(logger),
			interceptors.LoggingInterceptor(logger),
			interceptors.MetricsInterceptor(metricsCollector),
			interceptors.ValidationInterceptor(logger),
			interceptors.ErrorHandlingInterceptor(logger),
		)),
		grpc.StreamInterceptor(grpc.ChainStreamInterceptor(
			interceptors.StreamRecoveryInterceptor(logger),
			interceptors.StreamLoggingInterceptor(logger),
			interceptors.StreamMetricsInterceptor(metricsCollector),
			interceptors.StreamErrorHandlingInterceptor(logger),
		)),
	)

	// Initialize gRPC service
	grpcService := server.NewGRPCServer(
		entityResolver,
		metricsCollector,
		logger,
	)

	// Register services
	pb.RegisterEntityResolutionServiceServer(grpcServer, grpcService)

	// Register health service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		logger.Error("Failed to listen on gRPC port", "port", cfg.Server.GRPCPort, "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("gRPC server starting", "address", grpcListener.Addr())
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("gRPC server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Initialize HTTP handlers
	httpHandlers := handlers.NewHTTPHandlers(
		repository,
		entityResolver,
		metricsCollector,
		logger,
	)

	// Setup HTTP router
	router := mux.NewRouter()
	httpHandlers.RegisterRoutes(router)

	// Add metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP server starting", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Start Kafka consumer
	go func() {
		ctx := context.Background()
		logger.Info("Starting Kafka consumer")
		
		// Process transaction events for entity resolution
		if err := kafkaConsumer.ConsumeTransactionProcessedEvents(ctx, func(ctx context.Context, event *pb.TransactionProcessedEvent) error {
			return entityResolver.ProcessTransactionEvent(ctx, event)
		}); err != nil {
			logger.Error("Kafka consumer failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logger.Info("Received shutdown signal", "signal", sig)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop health checks
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
	} else {
		logger.Info("HTTP server stopped")
	}

	// Shutdown gRPC server
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("gRPC server stopped")
	case <-ctx.Done():
		logger.Warn("gRPC server shutdown timeout, forcing stop")
		grpcServer.Stop()
	}

	logger.Info("Entity Resolution Service stopped")
}