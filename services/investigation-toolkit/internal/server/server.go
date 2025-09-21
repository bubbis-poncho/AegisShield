package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
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
	config *config.Config
	logger *zap.Logger
	db     *database.Database
	
	// Repositories
	investigationRepo repository.InvestigationRepository
	evidenceRepo     repository.EvidenceRepository
	timelineRepo     repository.TimelineRepository
	workflowRepo     repository.WorkflowRepository
	collaborationRepo repository.CollaborationRepository
	auditRepo        repository.AuditRepository
	
	// Handlers
	investigationHandler *handlers.InvestigationHandler
	evidenceHandler     *handlers.EvidenceHandler
	timelineHandler     *handlers.TimelineHandler
	workflowHandler     *handlers.WorkflowHandler
	collaborationHandler *handlers.CollaborationHandler
	auditHandler        *handlers.AuditHandler
	healthHandler       *handlers.HealthHandler
	
	// HTTP and gRPC servers
	router     *gin.Engine
	httpServer *http.Server
	grpcServer *grpc.Server
	
	// Health server
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
	if err := s.initRepositories(); err != nil {
		return errors.Wrap(err, "failed to initialize repositories")
	}

	// Initialize handlers
	if err := s.initHandlers(); err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

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

// initRepositories initializes all repository instances
func (s *Server) initRepositories() error {
	s.logger.Info("Initializing repositories")
	
	s.investigationRepo = repository.NewInvestigationRepository(s.db.DB)
	s.evidenceRepo = repository.NewEvidenceRepository(s.db.DB)
	s.timelineRepo = repository.NewTimelineRepository(s.db.DB)
	s.workflowRepo = repository.NewWorkflowRepository(s.db.DB)
	s.collaborationRepo = repository.NewCollaborationRepository(s.db.DB)
	s.auditRepo = repository.NewAuditRepository(s.db.DB)
	
	s.logger.Info("Repositories initialized successfully")
	return nil
}

// initHandlers initializes all handler instances
func (s *Server) initHandlers() error {
	s.logger.Info("Initializing handlers")
	
	s.investigationHandler = handlers.NewInvestigationHandler(s.investigationRepo, s.auditRepo)
	s.evidenceHandler = handlers.NewEvidenceHandler(s.evidenceRepo, s.auditRepo)
	s.timelineHandler = handlers.NewTimelineHandler(s.timelineRepo, s.auditRepo)
	s.workflowHandler = handlers.NewWorkflowHandler(s.workflowRepo, s.auditRepo)
	s.collaborationHandler = handlers.NewCollaborationHandler(s.collaborationRepo, s.auditRepo)
	s.auditHandler = handlers.NewAuditHandler(s.auditRepo)
	s.healthHandler = handlers.NewHealthHandler(s.db)
	
	s.logger.Info("Handlers initialized successfully")
	return nil
}

// initHTTPServer initializes the HTTP server with Gin
func (s *Server) initHTTPServer() error {
	s.logger.Info("Initializing HTTP server")

	// Set Gin mode based on environment
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	s.router = gin.New()

	// Add middleware
	s.router.Use(gin.Recovery())
	if s.config.Debug {
		s.router.Use(gin.Logger())
	}

	// Add CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.config.Server.HTTPPort),
		Handler:        s.router,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		IdleTimeout:    s.config.Server.IdleTimeout,
		MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
	}

	s.logger.Info("HTTP server initialized", zap.Int("port", s.config.Server.HTTPPort))
	return nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health endpoints
	s.router.GET("/health", s.healthHandler.Health)
	s.router.GET("/health/ready", s.healthHandler.Ready)
	s.router.GET("/health/live", s.healthHandler.Live)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Investigation routes
		investigations := v1.Group("/investigations")
		{
			investigations.POST("", s.investigationHandler.CreateInvestigation)
			investigations.GET("/:id", s.investigationHandler.GetInvestigation)
			investigations.PUT("/:id", s.investigationHandler.UpdateInvestigation)
			investigations.DELETE("/:id", s.investigationHandler.DeleteInvestigation)
			investigations.GET("", s.investigationHandler.ListInvestigations)
			investigations.PUT("/:id/status", s.investigationHandler.UpdateStatus)
			investigations.PUT("/:id/assign", s.investigationHandler.AssignInvestigation)
			investigations.GET("/:id/stats", s.investigationHandler.GetInvestigationStats)
			investigations.GET("/user/:user_id", s.investigationHandler.GetUserInvestigations)
		}

		// Evidence routes
		evidence := v1.Group("/evidence")
		{
			evidence.POST("", s.evidenceHandler.CreateEvidence)
			evidence.GET("/:id", s.evidenceHandler.GetEvidence)
			evidence.PUT("/:id", s.evidenceHandler.UpdateEvidence)
			evidence.DELETE("/:id", s.evidenceHandler.DeleteEvidence)
			evidence.GET("", s.evidenceHandler.ListEvidence)
			evidence.GET("/investigation/:investigation_id", s.evidenceHandler.GetEvidenceByInvestigation)
			evidence.POST("/:id/files", s.evidenceHandler.UploadFile)
			evidence.GET("/:id/files/:file_id", s.evidenceHandler.DownloadFile)
			evidence.DELETE("/:id/files/:file_id", s.evidenceHandler.DeleteFile)
		}

		// Timeline routes
		timeline := v1.Group("/timeline")
		{
			timeline.POST("", s.timelineHandler.CreateEvent)
			timeline.GET("/:id", s.timelineHandler.GetEvent)
			timeline.PUT("/:id", s.timelineHandler.UpdateEvent)
			timeline.DELETE("/:id", s.timelineHandler.DeleteEvent)
			timeline.GET("", s.timelineHandler.ListEvents)
			timeline.GET("/investigation/:investigation_id", s.timelineHandler.GetEventsByInvestigation)
			timeline.POST("/bulk", s.timelineHandler.BulkCreateEvents)
			timeline.GET("/search", s.timelineHandler.SearchEvents)
		}

		// Workflow routes
		workflows := v1.Group("/workflows")
		{
			// Templates
			templates := workflows.Group("/templates")
			{
				templates.POST("", s.workflowHandler.CreateTemplate)
				templates.GET("/:id", s.workflowHandler.GetTemplate)
				templates.PUT("/:id", s.workflowHandler.UpdateTemplate)
				templates.DELETE("/:id", s.workflowHandler.DeleteTemplate)
				templates.GET("", s.workflowHandler.ListTemplates)
			}

			// Instances
			instances := workflows.Group("/instances")
			{
				instances.POST("", s.workflowHandler.CreateInstance)
				instances.GET("/:id", s.workflowHandler.GetInstance)
				instances.PUT("/:id/status", s.workflowHandler.UpdateInstanceStatus)
				instances.GET("/investigation/:investigation_id", s.workflowHandler.GetInstancesByInvestigation)
				instances.GET("/:instance_id/steps", s.workflowHandler.GetInstanceSteps)
			}

			// Steps
			steps := workflows.Group("/steps")
			{
				steps.PUT("/:step_id/start", s.workflowHandler.StartStep)
				steps.PUT("/:step_id/complete", s.workflowHandler.CompleteStep)
				steps.PUT("/:step_id/skip", s.workflowHandler.SkipStep)
			}

			// Management
			workflows.GET("/pending", s.workflowHandler.GetPendingSteps)
			workflows.GET("/overdue", s.workflowHandler.GetOverdueSteps)
			workflows.GET("/stats", s.workflowHandler.GetWorkflowStats)
		}

		// Collaboration routes
		collaboration := v1.Group("/collaboration")
		{
			// Comments
			comments := collaboration.Group("/comments")
			{
				comments.POST("", s.collaborationHandler.CreateComment)
				comments.GET("/:id", s.collaborationHandler.GetComment)
				comments.PUT("/:id", s.collaborationHandler.UpdateComment)
				comments.DELETE("/:id", s.collaborationHandler.DeleteComment)
				comments.GET("/:entity_type/:entity_id", s.collaborationHandler.GetCommentsByEntity)
			}

			// Assignments
			assignments := collaboration.Group("/assignments")
			{
				assignments.POST("", s.collaborationHandler.CreateAssignment)
				assignments.GET("/:id", s.collaborationHandler.GetAssignment)
				assignments.PUT("/:id", s.collaborationHandler.UpdateAssignment)
				assignments.GET("/user/:user_id", s.collaborationHandler.GetUserAssignments)
			}

			// Teams
			teams := collaboration.Group("/teams")
			{
				teams.POST("", s.collaborationHandler.CreateTeam)
				teams.GET("/:id", s.collaborationHandler.GetTeam)
				teams.PUT("/:id", s.collaborationHandler.UpdateTeam)
				teams.POST("/:team_id/members", s.collaborationHandler.AddTeamMember)
				teams.DELETE("/:team_id/members/:user_id", s.collaborationHandler.RemoveTeamMember)
				teams.GET("/user/:user_id", s.collaborationHandler.GetUserTeams)
			}

			// Notifications
			notifications := collaboration.Group("/notifications")
			{
				notifications.POST("", s.collaborationHandler.CreateNotification)
				notifications.GET("/user/:user_id", s.collaborationHandler.GetUserNotifications)
				notifications.PUT("/:id/read", s.collaborationHandler.MarkNotificationAsRead)
				notifications.PUT("/user/:user_id/read-all", s.collaborationHandler.MarkAllNotificationsAsRead)
			}

			// Statistics
			collaboration.GET("/stats", s.collaborationHandler.GetCollaborationStats)
			collaboration.GET("/stats/user/:user_id", s.collaborationHandler.GetUserActivityStats)
			collaboration.GET("/stats/team/:team_id", s.collaborationHandler.GetTeamActivityStats)
		}

		// Audit routes
		audit := v1.Group("/audit")
		{
			// Audit logs
			logs := audit.Group("/logs")
			{
				logs.GET("/:id", s.auditHandler.GetAuditLog)
				logs.GET("", s.auditHandler.ListAuditLogs)
				logs.GET("/:entity_type/:entity_id", s.auditHandler.GetAuditLogsByEntity)
				logs.GET("/user/:user_id", s.auditHandler.GetAuditLogsByUser)
			}

			// Chain of custody
			custody := audit.Group("/custody")
			{
				custody.GET("/evidence/:evidence_id", s.auditHandler.GetChainOfCustody)
				custody.POST("/evidence/:evidence_id/verify", s.auditHandler.VerifyChainOfCustody)
			}

			// Data integrity
			integrity := audit.Group("/integrity")
			{
				integrity.GET("/:entity_type/:entity_id", s.auditHandler.GetDataIntegrityChecks)
				integrity.POST("/:entity_type/:entity_id/verify", s.auditHandler.VerifyDataIntegrity)
			}

			// Access logs
			access := audit.Group("/access")
			{
				access.GET("/user/:user_id", s.auditHandler.GetUserAccessLogs)
				access.GET("/resource/:resource", s.auditHandler.GetResourceAccessLogs)
			}

			// Reports and summaries
			reports := audit.Group("/reports")
			{
				reports.GET("/compliance", s.auditHandler.GenerateComplianceReport)
				reports.GET("/summary", s.auditHandler.GetAuditSummary)
				reports.GET("/user/:user_id/activity", s.auditHandler.GetUserActivitySummary)
			}

			// Retention and archival
			retention := audit.Group("/retention")
			{
				retention.GET("/stats", s.auditHandler.GetAuditLogRetentionStats)
				retention.POST("/archive", s.auditHandler.ArchiveOldAuditLogs)
				retention.POST("/purge", s.auditHandler.PurgeArchivedLogs)
			}

			// Monitoring
			audit.GET("/suspicious", s.auditHandler.GetSuspiciousActivities)
		}
	}
}

// initGRPCServer initializes the gRPC server
func (s *Server) initGRPCServer() error {
	s.logger.Info("Initializing gRPC server")

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 4), // 4MB
		grpc.MaxSendMsgSize(1024 * 1024 * 4), // 4MB
	}

	s.grpcServer = grpc.NewServer(opts...)

	// Register health service
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// Enable reflection for development
	if s.config.Debug {
		reflection.Register(s.grpcServer)
	}

	s.logger.Info("gRPC server initialized", zap.Int("port", s.config.Server.GRPCPort))
	return nil
}

// Start starts both HTTP and gRPC servers
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting investigation toolkit server")

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Server.GRPCPort))
		if err != nil {
			s.logger.Fatal("Failed to listen for gRPC", zap.Error(err))
		}

		s.logger.Info("gRPC server listening", zap.String("address", lis.Addr().String()))
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// Start HTTP server
	go func() {
		s.logger.Info("HTTP server listening", zap.String("address", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Set health status to serving
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	s.logger.Info("Investigation toolkit server started successfully")
	
	// Wait for context cancellation
	<-ctx.Done()
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down investigation toolkit server")

	// Set health status to not serving
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to shutdown HTTP server gracefully", zap.Error(err))
	}

	// Shutdown gRPC server
	s.grpcServer.GracefulStop()

	// Close database connection
	if err := s.db.Close(); err != nil {
		s.logger.Error("Failed to close database connection", zap.Error(err))
	}

	s.logger.Info("Investigation toolkit server shutdown completed")
	return nil
}