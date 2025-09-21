package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"../internal/config"
	"../internal/server"
	"../internal/monitoring"
)

// HealthChecker provides health check capabilities
type HealthChecker struct {
	server *server.Server
	logger *zap.Logger
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(srv *server.Server, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		server: srv,
		logger: logger,
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hc.logger.Info("Health checker stopping")
			return
		case <-ticker.C:
			hc.performHealthCheck()
		}
	}
}

// performHealthCheck performs a comprehensive health check
func (hc *HealthChecker) performHealthCheck() {
	health := hc.server.Health()
	
	if health["status"] == "degraded" {
		hc.logger.Warn("Service health degraded", zap.Any("health", health))
	} else {
		hc.logger.Debug("Service health check passed", zap.Any("health", health))
	}
}

// MetricsCollector provides metrics collection capabilities
type MetricsCollector struct {
	logger *zap.Logger
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *zap.Logger) *MetricsCollector {
	return &MetricsCollector{
		logger: logger,
	}
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mc.logger.Info("Metrics collector stopping")
			return
		case <-ticker.C:
			mc.collectMetrics()
		}
	}
}

// collectMetrics collects system metrics
func (mc *MetricsCollector) collectMetrics() {
	// Collect and report system metrics
	mc.logger.Debug("Collecting system metrics")
	
	// In a production system, this would collect:
	// - CPU usage
	// - Memory usage
	// - Request rates
	// - Error rates
	// - Response times
	// - Database connection pools
	// - Cache hit rates
	// etc.
}

// GracefulShutdown handles graceful shutdown of all services
func GracefulShutdown(srv *server.Server, logger *zap.Logger) {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	
	// Register the channel to receive specific signals
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Wait for a signal
	sig := <-sigChan
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	
	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Shutdown the server gracefully
	if err := srv.Shutdown(); err != nil {
		logger.Error("Error during graceful shutdown", zap.Error(err))
		os.Exit(1)
	}
	
	logger.Info("Graceful shutdown completed")
}

// Version information
var (
	Version   = "1.0.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// PrintVersion prints version information
func PrintVersion(logger *zap.Logger) {
	logger.Info("AegisShield ML Pipeline Service",
		zap.String("version", Version),
		zap.String("git_commit", GitCommit),
		zap.String("build_time", BuildTime))
}

// validateEnvironment validates the runtime environment
func validateEnvironment(cfg *config.Config, logger *zap.Logger) error {
	logger.Info("Validating environment configuration")
	
	// Check required environment variables
	requiredVars := []string{
		"DATABASE_HOST",
		"DATABASE_NAME", 
		"DATABASE_USERNAME",
		"DATABASE_PASSWORD",
	}
	
	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			logger.Warn("Environment variable not set", zap.String("var", varName))
		}
	}
	
	// Validate configuration
	if cfg.Database.Host == "" {
		logger.Error("Database host not configured")
		return fmt.Errorf("database host not configured")
	}
	
	if cfg.Server.HTTP.Port == 0 {
		logger.Error("HTTP port not configured")
		return fmt.Errorf("HTTP port not configured")
	}
	
	logger.Info("Environment validation completed")
	return nil
}

// setupBackgroundServices sets up background monitoring and maintenance services
func setupBackgroundServices(srv *server.Server, logger *zap.Logger) {
	ctx := context.Background()
	
	// Start health checker
	healthChecker := NewHealthChecker(srv, logger)
	go healthChecker.Start(ctx)
	
	// Start metrics collector
	metricsCollector := NewMetricsCollector(logger)
	go metricsCollector.Start(ctx)
	
	logger.Info("Background services started")
}