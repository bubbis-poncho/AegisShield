package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"

	"aegisshield/services/api-gateway/internal/auth"
	"aegisshield/services/api-gateway/internal/config"
	"aegisshield/services/api-gateway/internal/graph"
	"aegisshield/services/api-gateway/internal/graph/generated"
	"aegisshield/services/api-gateway/internal/middleware"
	"aegisshield/services/api-gateway/internal/services"
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
	}).Info("Starting AegisShield API Gateway")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize services
	serviceClients, err := services.NewServiceClients(cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize service clients")
	}
	defer serviceClients.Close()

	// Initialize authentication
	authService := auth.NewService(cfg.Auth)

	// Create GraphQL server
	resolver := &graph.Resolver{
		Services: serviceClients,
		Auth:     authService,
		Logger:   logger,
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Create HTTP router
	router := mux.NewRouter()

	// Add middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.AuthMiddleware(authService))

	// GraphQL endpoints
	router.Handle("/query", srv).Methods("POST")
	router.Handle("/", playground.Handler("GraphQL playground", "/query")).Methods("GET")

	// Health and metrics endpoints
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/ready", readinessHandler(serviceClients)).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", cfg.Port).Info("Starting HTTP server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Failed to shutdown HTTP server gracefully")
	}

	logger.Info("Server shutdown complete")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"api-gateway"}`))
}

func readinessHandler(services *services.ServiceClients) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		
		// Check service connections
		if err := services.HealthCheck(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(fmt.Sprintf(`{"status":"not ready","error":"%s"}`, err.Error())))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","service":"api-gateway"}`))
	}
}