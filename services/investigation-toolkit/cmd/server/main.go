package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"investigation-toolkit/internal/config"
	"investigation-toolkit/internal/database"
	"investigation-toolkit/internal/server"
)

func main() {
	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	logger.Info("Starting Investigation Toolkit Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("Configuration loaded", 
		zap.String("environment", cfg.Environment),
		zap.Bool("debug", cfg.Debug))

	// Initialize database
	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Run database migrations
	if err := db.RunMigrations(); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	// Initialize server
	srv := server.New(cfg, logger, db)
	if err := srv.Initialize(); err != nil {
		logger.Fatal("Failed to initialize server", zap.Error(err))
	}

	// Start server
	if err := srv.Start(); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("Shutdown signal received")

	// Graceful shutdown
	if err := srv.Stop(); err != nil {
		logger.Error("Failed to stop server gracefully", zap.Error(err))
	}

	logger.Info("Investigation Toolkit Service stopped")
}

// initLogger initializes the zap logger
func initLogger() *zap.Logger {
	// Configure logger based on environment
	var config zap.Config

	env := os.Getenv("ENVIRONMENT")
	if env == "production" {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	} else {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Add caller information
	config.DisableCaller = false
	config.DisableStacktrace = false

	// Create logger
	logger, err := config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	return logger
}