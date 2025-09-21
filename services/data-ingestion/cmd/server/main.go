// Data Ingestion Service - T028
// Constitutional Principle: Data Integrity & Real-time Processing

package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"aegisshield/services/data-ingestion/internal/config"
	"aegisshield/services/data-ingestion/internal/database"
	"aegisshield/services/data-ingestion/internal/handlers"
	"aegisshield/services/data-ingestion/internal/kafka"
	"aegisshield/services/data-ingestion/internal/metrics"
	"aegisshield/services/data-ingestion/internal/server"
	"aegisshield/services/data-ingestion/internal/storage"
	pb "aegisshield/shared/proto/data-ingestion"
)

var (
	logger = logrus.New()
	version = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	
	if os.Getenv("LOG_LEVEL") == "debug" {
		logger.SetLevel(logrus.DebugLevel)
	}
}

func main() {
	logger.WithFields(logrus.Fields{
		"version":   version,
		"buildTime": buildTime,
		"gitCommit": gitCommit,
	}).Info("Starting Data Ingestion Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize metrics
	metricsCollector := metrics.NewCollector()
	metricsCollector.Register()

	// Initialize database connection
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(cfg.Database.URL); err != nil {
		logger.WithError(err).Fatal("Failed to run database migrations")
	}

	// Initialize storage service
	storageService, err := storage.NewService(cfg.Storage)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize storage service")
	}

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Kafka producer")
	}
	defer kafkaProducer.Close()

	// Initialize repositories
	repos := &server.Repositories{
		FileUpload:   database.NewFileUploadRepository(db),
		DataJob:      database.NewDataJobRepository(db),
		Transaction:  database.NewTransactionRepository(db),
		Validation:   database.NewValidationRepository(db),
	}

	// Initialize services
	services := &server.Services{
		Storage:     storageService,
		Kafka:       kafkaProducer,
		Metrics:     metricsCollector,
		Logger:      logger,
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(server.LoggingInterceptor(logger)),
		grpc.StreamInterceptor(server.StreamLoggingInterceptor(logger)),
	)

	// Register service implementation
	dataIngestionServer := server.NewDataIngestionServer(repos, services, cfg)
	pb.RegisterDataIngestionServiceServer(grpcServer, dataIngestionServer)

	// Enable reflection for development
	if cfg.Environment == "development" {
		reflection.Register(grpcServer)
	}

	// Start gRPC server
	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
		if err != nil {
			logger.WithError(err).Fatal("Failed to listen on gRPC port")
		}

		logger.WithField("port", cfg.Server.GRPCPort).Info("Starting gRPC server")
		if err := grpcServer.Serve(listener); err != nil {
			logger.WithError(err).Fatal("Failed to serve gRPC")
		}
	}()

	// Start HTTP server for health checks and metrics
	go func() {
		httpRouter := mux.NewRouter()
		
		// Health check endpoint
		httpRouter.HandleFunc("/health", handlers.HealthCheckHandler(db, kafkaProducer)).Methods("GET")
		httpRouter.HandleFunc("/health/live", handlers.LivenessHandler).Methods("GET")
		httpRouter.HandleFunc("/health/ready", handlers.ReadinessHandler(db, kafkaProducer)).Methods("GET")
		
		// Metrics endpoint
		httpRouter.Handle("/metrics", promhttp.Handler()).Methods("GET")
		
		// File upload endpoints (REST API)
		api := httpRouter.PathPrefix("/api/v1").Subrouter()
		fileHandler := handlers.NewFileHandler(storageService, repos.FileUpload, kafkaProducer, logger)
		api.HandleFunc("/files/upload", fileHandler.Upload).Methods("POST")
		api.HandleFunc("/files/{id}/status", fileHandler.GetStatus).Methods("GET")
		
		httpServer := &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
			Handler:      httpRouter,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		logger.WithField("port", cfg.Server.HTTPPort).Info("Starting HTTP server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to serve HTTP")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Data Ingestion Service...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop gRPC server
	grpcServer.GracefulStop()

	logger.Info("Data Ingestion Service stopped")
}