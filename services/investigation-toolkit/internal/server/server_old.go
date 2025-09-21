package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"investigation-toolkit/internal/config"
	"investigation-toolkit/internal/database"
	"investigation-toolkit/internal/handlers"
	"investigation-toolkit/internal/repository"
)

// Server represents the investigation toolkit server
type Server struct {
	config   *config.Config
	logger   *zap.Logger
	db       *database.Database
	
	// Repositories
	investigationRepo *repository.InvestigationRepository
	evidenceRepo      *repository.EvidenceRepository
	timelineRepo      *repository.TimelineRepository
	
	// Servers
	httpServer *http.Server
	grpcServer *grpc.Server
	
	// Health
	healthServer *health.Server
}

// New creates a new server instance
func New(cfg *config.Config, logger *zap.Logger, db *database.Database) *Server {
	return &Server{
		config: cfg,
		logger: logger.Named("server"),
		db:     db,
	}
}

// Initialize sets up the server components
func (s *Server) Initialize() error {
	s.logger.Info("Initializing investigation toolkit server")

	// Initialize repositories
	s.investigationRepo = repository.NewInvestigationRepository(s.db, s.logger)
	s.evidenceRepo = repository.NewEvidenceRepository(s.db, s.logger)
	s.timelineRepo = repository.NewTimelineRepository(s.db, s.logger)

	// Initialize health server
	s.healthServer = health.NewServer()

	// Initialize HTTP server
	if err := s.initHTTPServer(); err != nil {
		return errors.Wrap(err, "failed to initialize HTTP server")
	}

	// Initialize gRPC server
	if err := s.initGRPCServer(); err != nil {
		return errors.Wrap(err, "failed to initialize gRPC server")
	}

	s.logger.Info("Server initialized successfully")
	return nil
}

// initHTTPServer initializes the HTTP server with Gin
func (s *Server) initHTTPServer() error {
	// Set Gin mode based on environment
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	if s.config.Debug {
		router.Use(gin.Logger())
	}

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})

	// Initialize handlers
	investigationHandler := handlers.NewInvestigationHandler(s.investigationRepo, s.logger)
	evidenceHandler := handlers.NewEvidenceHandler(s.evidenceRepo, s.logger)
	timelineHandler := handlers.NewTimelineHandler(s.timelineRepo, s.logger)
	healthHandler := handlers.NewHealthHandler(s.db, s.logger)

	// Health endpoints
	router.GET("/health", healthHandler.Health)
	router.GET("/health/ready", healthHandler.Ready)
	router.GET("/health/live", healthHandler.Live)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Investigation routes
		investigations := v1.Group("/investigations")
		{
			investigations.POST("", investigationHandler.CreateInvestigation)
			investigations.GET("", investigationHandler.ListInvestigations)
			investigations.GET("/:id", investigationHandler.GetInvestigation)
			investigations.PUT("/:id", investigationHandler.UpdateInvestigation)
			investigations.DELETE("/:id", investigationHandler.DeleteInvestigation)
			investigations.GET("/:id/stats", investigationHandler.GetInvestigationStats)
			investigations.PUT("/:id/assignment", investigationHandler.UpdateAssignment)
			investigations.PUT("/:id/status", investigationHandler.UpdateStatus)
			
			// Evidence routes within investigation
			investigations.POST("/:id/evidence", evidenceHandler.CreateEvidence)
			investigations.GET("/:id/evidence", evidenceHandler.ListEvidence)
			
			// Timeline routes within investigation
			investigations.POST("/:id/timeline", timelineHandler.CreateTimelineEvent)
			investigations.GET("/:id/timeline", timelineHandler.ListTimelineEvents)
		}

		// Evidence routes
		evidence := v1.Group("/evidence")
		{
			evidence.GET("/:id", evidenceHandler.GetEvidence)
			evidence.PUT("/:id/file", evidenceHandler.UpdateEvidenceFile)
			evidence.PUT("/:id/authenticate", evidenceHandler.AuthenticateEvidence)
			evidence.PUT("/:id/status", evidenceHandler.UpdateEvidenceStatus)
			evidence.DELETE("/:id", evidenceHandler.DeleteEvidence)
			evidence.GET("/:id/chain-of-custody", evidenceHandler.GetChainOfCustody)
			evidence.POST("/:id/chain-of-custody", evidenceHandler.AddChainOfCustodyEntry)
		}

		// Timeline routes
		timeline := v1.Group("/timeline")
		{
			timeline.GET("/:id", timelineHandler.GetTimelineEvent)
			timeline.PUT("/:id", timelineHandler.UpdateTimelineEvent)
			timeline.DELETE("/:id", timelineHandler.DeleteTimelineEvent)
		}

		// Search and analytics
		search := v1.Group("/search")
		{
			search.GET("/investigations", investigationHandler.SearchInvestigations)
			search.GET("/evidence", evidenceHandler.SearchEvidence)
			search.GET("/timeline", timelineHandler.SearchTimelineEvents)
		}

		// User-specific routes
		user := v1.Group("/user")
		{
			user.GET("/investigations", investigationHandler.GetAssignedInvestigations)
			user.GET("/dashboard", investigationHandler.GetUserDashboard)
		}
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.config.Server.HTTPPort),
		Handler:        router,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		IdleTimeout:    s.config.Server.IdleTimeout,
		MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
	}

	return nil
}

// initGRPCServer initializes the gRPC server
func (s *Server) initGRPCServer() error {
	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 4), // 4MB
		grpc.MaxSendMsgSize(1024 * 1024 * 4), // 4MB
	}

	s.grpcServer = grpc.NewServer(opts...)

	// Register health service
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// Register gRPC services (will be implemented later)
	// pb.RegisterInvestigationServiceServer(s.grpcServer, grpcHandlers.NewInvestigationService(s.investigationRepo, s.logger))
	// pb.RegisterEvidenceServiceServer(s.grpcServer, grpcHandlers.NewEvidenceService(s.evidenceRepo, s.logger))
	// pb.RegisterTimelineServiceServer(s.grpcServer, grpcHandlers.NewTimelineService(s.timelineRepo, s.logger))

	// Enable reflection for development
	if s.config.Server.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	return nil
}

// Start starts both HTTP and gRPC servers
func (s *Server) Start() error {
	s.logger.Info("Starting investigation toolkit server",
		zap.Int("http_port", s.config.Server.HTTPPort),
		zap.Int("grpc_port", s.config.Server.GRPCPort))

	// Start HTTP server
	go func() {
		s.logger.Info("Starting HTTP server", zap.Int("port", s.config.Server.HTTPPort))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Server.GRPCPort))
		if err != nil {
			s.logger.Fatal("Failed to listen for gRPC", zap.Error(err))
			return
		}

		s.logger.Info("Starting gRPC server", zap.Int("port", s.config.Server.GRPCPort))
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	// Set health status to serving
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	s.logger.Info("Investigation toolkit server started successfully")
	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.logger.Info("Stopping investigation toolkit server")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Server.ShutdownTimeout)
	defer cancel()

	// Set health status to not serving
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Stop HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to gracefully shutdown HTTP server", zap.Error(err))
		} else {
			s.logger.Info("HTTP server stopped")
		}
	}

	// Stop gRPC server
	if s.grpcServer != nil {
		// Try graceful stop first
		done := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			s.logger.Info("gRPC server stopped gracefully")
		case <-ctx.Done():
			s.logger.Warn("Graceful stop timeout, forcing stop")
			s.grpcServer.Stop()
		}
	}

	// Close database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Failed to close database connection", zap.Error(err))
		} else {
			s.logger.Info("Database connection closed")
		}
	}

	s.logger.Info("Investigation toolkit server stopped")
	return nil
}

// GetHTTPServer returns the HTTP server instance
func (s *Server) GetHTTPServer() *http.Server {
	return s.httpServer
}

// GetGRPCServer returns the gRPC server instance
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}

// GetDatabase returns the database instance
func (s *Server) GetDatabase() *database.Database {
	return s.db
}

// SetHealthStatus sets the health status for the service
func (s *Server) SetHealthStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthServer != nil {
		s.healthServer.SetServingStatus(service, status)
	}
}

// RegisterHTTPRoutes allows external packages to register additional HTTP routes
func (s *Server) RegisterHTTPRoutes(routeGroup string, registerFunc func(*gin.RouterGroup)) {
	if s.httpServer != nil {
		if handler, ok := s.httpServer.Handler.(*gin.Engine); ok {
			group := handler.Group(routeGroup)
			registerFunc(group)
		}
	}
}