package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"../api"
	"../config"
	"../database"
	"../grpc"
	"../inference"
	"../monitoring"
	"../training"
)

// Server represents the ML Pipeline server
type Server struct {
	config        *config.Config
	logger        *zap.Logger
	httpServer    *http.Server
	grpcServer    *grpc.Server
	repos         *database.Repositories
	monitor       *monitoring.ModelMonitor
	trainer       *training.TrainingEngine
	inferencer    *inference.InferenceEngine
	shutdownChan  chan os.Signal
}

// NewServer creates a new ML Pipeline server
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Initialize database connection
	db, err := database.NewConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := database.RunMigrations(db, cfg.Database.MigrationsPath); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize repositories
	repos := database.NewRepositories(db)

	// Initialize training engine
	trainer := training.NewTrainingEngine(cfg, repos, logger)

	// Initialize inference engine
	inferencer := inference.NewInferenceEngine(cfg, repos, logger)

	// Initialize model monitor
	monitor := monitoring.NewModelMonitor(cfg, repos, logger)

	server := &Server{
		config:       cfg,
		logger:       logger,
		repos:        repos,
		monitor:      monitor,
		trainer:      trainer,
		inferencer:   inferencer,
		shutdownChan: make(chan os.Signal, 1),
	}

	// Setup HTTP server
	if err := server.setupHTTPServer(); err != nil {
		return nil, fmt.Errorf("failed to setup HTTP server: %w", err)
	}

	// Setup gRPC server
	if err := server.setupGRPCServer(); err != nil {
		return nil, fmt.Errorf("failed to setup gRPC server: %w", err)
	}

	return server, nil
}

// setupHTTPServer initializes the HTTP/REST API server
func (s *Server) setupHTTPServer() error {
	router := api.SetupRouter(s.config, s.logger, s.repos, s.monitor, s.trainer, s.inferencer)

	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.config.Server.HTTP.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(s.config.Server.HTTP.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(s.config.Server.HTTP.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(s.config.Server.HTTP.IdleTimeout) * time.Second,
		MaxHeaderBytes: s.config.Server.HTTP.MaxHeaderBytes,
	}

	return nil
}

// setupGRPCServer initializes the gRPC server
func (s *Server) setupGRPCServer() error {
	grpcHandler := grpc.NewServer(s.config, s.logger, s.repos, s.monitor, s.trainer, s.inferencer)

	s.grpcServer = grpc.NewServer()
	// Register the ML Pipeline service
	// pb.RegisterMLPipelineServiceServer(s.grpcServer, grpcHandler)

	return nil
}

// Start starts the ML Pipeline server
func (s *Server) Start() error {
	s.logger.Info("Starting ML Pipeline server",
		zap.Int("http_port", s.config.Server.HTTP.Port),
		zap.Int("grpc_port", s.config.Server.GRPC.Port))

	// Start background services
	if err := s.startBackgroundServices(); err != nil {
		return fmt.Errorf("failed to start background services: %w", err)
	}

	// Start HTTP server
	go func() {
		s.logger.Info("Starting HTTP server", zap.Int("port", s.config.Server.HTTP.Port))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server failed", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		s.logger.Info("Starting gRPC server", zap.Int("port", s.config.Server.GRPC.Port))
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Server.GRPC.Port))
		if err != nil {
			s.logger.Error("Failed to listen for gRPC", zap.Error(err))
			return
		}

		if err := s.grpcServer.Serve(listener); err != nil {
			s.logger.Error("gRPC server failed", zap.Error(err))
		}
	}()

	// Setup signal handling for graceful shutdown
	signal.Notify(s.shutdownChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-s.shutdownChan
	s.logger.Info("Shutdown signal received")

	return s.Shutdown()
}

// startBackgroundServices starts background processing services
func (s *Server) startBackgroundServices() error {
	ctx := context.Background()

	// Start training engine
	if err := s.trainer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start training engine: %w", err)
	}

	// Start inference engine
	if err := s.inferencer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start inference engine: %w", err)
	}

	// Start monitoring
	if err := s.monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	s.logger.Info("Background services started successfully")
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.logger.Info("Starting graceful shutdown")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown failed", zap.Error(err))
	} else {
		s.logger.Info("HTTP server shutdown completed")
	}

	// Shutdown gRPC server
	s.grpcServer.GracefulStop()
	s.logger.Info("gRPC server shutdown completed")

	// Shutdown background services
	if err := s.shutdownBackgroundServices(ctx); err != nil {
		s.logger.Error("Background services shutdown failed", zap.Error(err))
	} else {
		s.logger.Info("Background services shutdown completed")
	}

	// Close database connections
	if err := s.repos.Close(); err != nil {
		s.logger.Error("Database shutdown failed", zap.Error(err))
	} else {
		s.logger.Info("Database connections closed")
	}

	s.logger.Info("Graceful shutdown completed")
	return nil
}

// shutdownBackgroundServices stops background processing services
func (s *Server) shutdownBackgroundServices(ctx context.Context) error {
	// Stop monitoring
	if err := s.monitor.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop monitoring", zap.Error(err))
	}

	// Stop inference engine
	if err := s.inferencer.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop inference engine", zap.Error(err))
	}

	// Stop training engine
	if err := s.trainer.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop training engine", zap.Error(err))
	}

	return nil
}

// Health returns the health status of the server
func (s *Server) Health() map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"services": map[string]interface{}{
			"http":      s.checkHTTPHealth(),
			"grpc":      s.checkGRPCHealth(),
			"database":  s.checkDatabaseHealth(),
			"trainer":   s.checkTrainerHealth(),
			"inference": s.checkInferenceHealth(),
			"monitor":   s.checkMonitorHealth(),
		},
	}

	// Determine overall status
	allHealthy := true
	for _, service := range health["services"].(map[string]interface{}) {
		if service.(map[string]interface{})["status"] != "healthy" {
			allHealthy = false
			break
		}
	}

	if !allHealthy {
		health["status"] = "degraded"
	}

	return health
}

// Health check methods

func (s *Server) checkHTTPHealth() map[string]interface{} {
	if s.httpServer == nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "HTTP server not initialized",
		}
	}
	return map[string]interface{}{
		"status": "healthy",
		"port":   s.config.Server.HTTP.Port,
	}
}

func (s *Server) checkGRPCHealth() map[string]interface{} {
	if s.grpcServer == nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "gRPC server not initialized",
		}
	}
	return map[string]interface{}{
		"status": "healthy",
		"port":   s.config.Server.GRPC.Port,
	}
}

func (s *Server) checkDatabaseHealth() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.repos.ModelRepo.GetAll(ctx)
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	}

	return map[string]interface{}{
		"status": "healthy",
	}
}

func (s *Server) checkTrainerHealth() map[string]interface{} {
	healthy := s.trainer.IsHealthy()
	if !healthy {
		return map[string]interface{}{
			"status": "unhealthy",
		}
	}
	return map[string]interface{}{
		"status": "healthy",
	}
}

func (s *Server) checkInferenceHealth() map[string]interface{} {
	healthy := s.inferencer.IsHealthy()
	if !healthy {
		return map[string]interface{}{
			"status": "unhealthy",
		}
	}
	return map[string]interface{}{
		"status": "healthy",
	}
}

func (s *Server) checkMonitorHealth() map[string]interface{} {
	healthy := s.monitor.IsHealthy()
	if !healthy {
		return map[string]interface{}{
			"status": "unhealthy",
		}
	}
	return map[string]interface{}{
		"status": "healthy",
	}
}

// GetMetrics returns server metrics
func (s *Server) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"uptime":    time.Since(time.Now()).Seconds(), // This would be tracked properly
		"requests":  0,                                // This would be tracked properly
		"errors":    0,                                // This would be tracked properly
		"cpu_usage": 0.0,                              // This would be collected from system
		"memory_usage": map[string]interface{}{
			"allocated": 0,
			"total":     0,
		},
	}
}