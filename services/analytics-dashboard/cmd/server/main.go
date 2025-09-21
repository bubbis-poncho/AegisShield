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

// Version information
var (
	Version   = "1.0.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
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
		logger.Info("AegisShield Analytics Dashboard Service",
			zap.String("version", Version),
			zap.String("git_commit", GitCommit),
			zap.String("build_time", BuildTime))
		return
	}

	logger.Info("Starting AegisShield Analytics Dashboard Service",
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

	// Start server
	logger.Info("Starting Analytics Dashboard server",
		zap.Int("http_port", cfg.Server.HTTP.Port),
		zap.Int("websocket_port", cfg.Server.WebSocket.Port))

	if err := srv.Start(); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

// initLogger initializes the application logger
func initLogger() *zap.Logger {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	var logger *zap.Logger
	var err error

	if env == "production" {
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logger, err = config.Build()
	} else {
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		logger, err = config.Build()
	}

	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

// validateEnvironment validates the runtime environment
func validateEnvironment(cfg *config.Config, logger *zap.Logger) error {
	logger.Info("Validating environment configuration")
	
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host not configured")
	}
	
	if cfg.Server.HTTP.Port == 0 {
		return fmt.Errorf("HTTP port not configured")
	}
	
	if cfg.Server.WebSocket.Port == 0 {
		return fmt.Errorf("WebSocket port not configured")
	}
	
	logger.Info("Environment validation completed")
	return nil
}