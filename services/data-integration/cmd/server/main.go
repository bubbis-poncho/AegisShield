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

	"github.com/aegisshield/data-integration/internal/config"
	"github.com/aegisshield/data-integration/internal/etl"
	"github.com/aegisshield/data-integration/internal/handlers"
	"github.com/aegisshield/data-integration/internal/kafka"
	"github.com/aegisshield/data-integration/internal/lineage"
	"github.com/aegisshield/data-integration/internal/quality"
	"github.com/aegisshield/data-integration/internal/server"
	"github.com/aegisshield/data-integration/internal/storage"
	"github.com/aegisshield/data-integration/internal/validation"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
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
	var logger *zap.Logger
	if cfg.Environment == "production" {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	logger.Info("Starting Data Integration Service",
		zap.String("version", "1.0.0"),
		zap.String("environment", cfg.Environment))

	// Initialize storage manager
	storageManager, err := storage.NewManager(cfg.Storage, logger)
	if err != nil {
		logger.Fatal("Failed to initialize storage manager", zap.Error(err))
	}

	// Initialize lineage tracker
	lineageStore := lineage.NewInMemoryLineageStore(logger)
	lineageTracker := lineage.NewTracker(lineageStore, logger)

	// Initialize data validator
	dataValidator := validation.NewValidator(cfg.ETL.ValidationRules, logger)

	// Initialize quality checker
	qualityChecker := quality.NewChecker(cfg.ETL.DataQuality, logger)

	// Initialize ETL pipeline
	etlPipeline := etl.NewPipeline(
		cfg,
		dataValidator,
		qualityChecker,
		lineageTracker,
		storageManager,
		logger,
	)

	// Initialize Kafka components
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Fatal("Failed to create Kafka producer", zap.Error(err))
	}
	defer kafkaProducer.Close()

	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, etlPipeline, logger)
	if err != nil {
		logger.Fatal("Failed to create Kafka consumer", zap.Error(err))
	}
	defer kafkaConsumer.Close()

	// Initialize gRPC server
	grpcServer := server.NewGRPCServer(
		etlPipeline,
		storageManager,
		lineageTracker,
		cfg,
		logger,
	)

	// Create gRPC server
	grpcSrv := grpc.NewServer()
	pb.RegisterDataIntegrationServer(grpcSrv, grpcServer)

	// Initialize HTTP handlers
	httpHandlers := handlers.NewHTTPHandlers(
		etlPipeline,
		storageManager,
		lineageTracker,
		dataValidator,
		qualityChecker,
		cfg,
		logger,
	)

	// Setup HTTP router
	router := mux.NewRouter()
	httpHandlers.RegisterRoutes(router)

	// Add Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Create HTTP server
	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start ETL pipeline
	if err := etlPipeline.Start(ctx); err != nil {
		logger.Fatal("Failed to start ETL pipeline", zap.Error(err))
	}

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		logger.Fatal("Failed to create gRPC listener", zap.Error(err))
	}

	go func() {
		logger.Info("Starting gRPC server", zap.Int("port", cfg.Server.GRPCPort))
		if err := grpcSrv.Serve(grpcListener); err != nil {
			logger.Error("gRPC server failed", zap.Error(err))
			cancel()
		}
	}()

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", cfg.Server.HTTPPort))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", zap.Error(err))
			cancel()
		}
	}()

	// Start Kafka consumer
	go func() {
		logger.Info("Starting Kafka consumer")
		if err := kafkaConsumer.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("Kafka consumer failed", zap.Error(err))
			cancel()
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	logger.Info("Starting graceful shutdown")

	// Stop ETL pipeline
	if err := etlPipeline.Stop(); err != nil {
		logger.Error("ETL pipeline shutdown failed", zap.Error(err))
	}

	// Stop gRPC server
	grpcSrv.GracefulStop()

	// Stop HTTP server
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer httpCancel()
	if err := httpSrv.Shutdown(httpCtx); err != nil {
		logger.Error("HTTP server shutdown failed", zap.Error(err))
	}

	// Cancel context to stop Kafka consumer
	cancel()

	logger.Info("Data Integration Service shutdown completed")
}