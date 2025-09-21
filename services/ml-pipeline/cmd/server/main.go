package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go.uber.org/zap"

	"../internal/config"
	"../internal/server"
)

func main() {
	var configPath string
	var showVersion bool
	
	flag.StringVar(&configPath, "config", "config/config.yaml", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	// Show version if requested
	if showVersion {
		PrintVersion(logger)
		return
	}

	logger.Info("Starting AegisShield ML Pipeline Service",
		zap.String("config_path", configPath),
		zap.String("version", Version))

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("Configuration loaded successfully",
		zap.String("environment", cfg.Environment),
		zap.String("log_level", cfg.Logging.Level))

	// Validate environment
	if err := validateEnvironment(cfg, logger); err != nil {
		logger.Fatal("Environment validation failed", zap.Error(err))
	}

	// Create and start server
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Setup background services
	setupBackgroundServices(srv, logger)

	// Setup graceful shutdown
	go GracefulShutdown(srv, logger)

	// Start server
	logger.Info("Starting ML Pipeline server",
		zap.Int("http_port", cfg.Server.HTTP.Port),
		zap.Int("grpc_port", cfg.Server.GRPC.Port))

	if err := srv.Start(); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

// initLogger initializes the application logger
func initLogger() *zap.Logger {
	// Check if we're in development mode
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	var logger *zap.Logger
	var err error

	if env == "production" {
		// Production logger configuration
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logger, err = config.Build()
	} else {
		// Development logger configuration
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		logger, err = config.Build()
	}

	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}