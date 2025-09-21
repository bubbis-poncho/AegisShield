package interceptors

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/metrics"
)

// Interceptors contains gRPC interceptors for the graph engine service
type Interceptors struct {
	config  config.Config
	logger  *slog.Logger
	metrics *metrics.MetricsCollector
}

// NewInterceptors creates new gRPC interceptors
func NewInterceptors(
	config config.Config,
	logger *slog.Logger,
	metrics *metrics.MetricsCollector,
) *Interceptors {
	return &Interceptors{
		config:  config,
		logger:  logger,
		metrics: metrics,
	}
}

// UnaryServerInterceptor returns a unary server interceptor that combines multiple interceptors
func (i *Interceptors) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Chain interceptors: logging -> metrics -> validation -> recovery -> timeout -> handler
		return i.loggingUnaryInterceptor(
			i.metricsUnaryInterceptor(
				i.validationUnaryInterceptor(
					i.recoveryUnaryInterceptor(
						i.timeoutUnaryInterceptor(handler),
					),
				),
			),
		)(ctx, req, info)
	}
}

// StreamServerInterceptor returns a stream server interceptor that combines multiple interceptors
func (i *Interceptors) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Chain interceptors: logging -> metrics -> validation -> recovery -> handler
		return i.loggingStreamInterceptor(
			i.metricsStreamInterceptor(
				i.validationStreamInterceptor(
					i.recoveryStreamInterceptor(handler),
				),
			),
		)(srv, ss, info)
	}
}

// Logging interceptors

func (i *Interceptors) loggingUnaryInterceptor(handler grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (interface{}, error) {
		start := time.Now()
		
		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)
		userAgent := getMetadataValue(md, "user-agent")
		requestID := getMetadataValue(md, "x-request-id")
		
		i.logger.Info("gRPC request started",
			"method", info.FullMethod,
			"user_agent", userAgent,
			"request_id", requestID)

		resp, err := handler(ctx, req, info)
		
		duration := time.Since(start)
		
		if err != nil {
			st, _ := status.FromError(err)
			i.logger.Error("gRPC request failed",
				"method", info.FullMethod,
				"duration", duration,
				"status_code", st.Code(),
				"error", st.Message(),
				"request_id", requestID)
		} else {
			i.logger.Info("gRPC request completed",
				"method", info.FullMethod,
				"duration", duration,
				"request_id", requestID)
		}

		return resp, err
	}
}

func (i *Interceptors) loggingStreamInterceptor(handler grpc.StreamHandler) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo) error {
		start := time.Now()
		
		ctx := stream.Context()
		md, _ := metadata.FromIncomingContext(ctx)
		userAgent := getMetadataValue(md, "user-agent")
		requestID := getMetadataValue(md, "x-request-id")
		
		i.logger.Info("gRPC stream started",
			"method", info.FullMethod,
			"user_agent", userAgent,
			"request_id", requestID)

		err := handler(srv, stream, info)
		
		duration := time.Since(start)
		
		if err != nil {
			st, _ := status.FromError(err)
			i.logger.Error("gRPC stream failed",
				"method", info.FullMethod,
				"duration", duration,
				"status_code", st.Code(),
				"error", st.Message(),
				"request_id", requestID)
		} else {
			i.logger.Info("gRPC stream completed",
				"method", info.FullMethod,
				"duration", duration,
				"request_id", requestID)
		}

		return err
	}
}

// Metrics interceptors

func (i *Interceptors) metricsUnaryInterceptor(handler grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (interface{}, error) {
		start := time.Now()
		
		// Increment in-flight requests
		i.metrics.SetRequestsInFlight("grpc", info.FullMethod, 1)
		defer i.metrics.SetRequestsInFlight("grpc", info.FullMethod, 0)

		resp, err := handler(ctx, req, info)
		
		duration := time.Since(start)
		
		// Record metrics
		statusCode := "success"
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code().String()
		}
		
		i.metrics.IncrementRequests("grpc", info.FullMethod, statusCode)
		i.metrics.ObserveRequestDuration("grpc", info.FullMethod, duration)

		return resp, err
	}
}

func (i *Interceptors) metricsStreamInterceptor(handler grpc.StreamHandler) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo) error {
		start := time.Now()
		
		// Increment in-flight requests
		i.metrics.SetRequestsInFlight("grpc-stream", info.FullMethod, 1)
		defer i.metrics.SetRequestsInFlight("grpc-stream", info.FullMethod, 0)

		err := handler(srv, stream, info)
		
		duration := time.Since(start)
		
		// Record metrics
		statusCode := "success"
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code().String()
		}
		
		i.metrics.IncrementRequests("grpc-stream", info.FullMethod, statusCode)
		i.metrics.ObserveRequestDuration("grpc-stream", info.FullMethod, duration)

		return err
	}
}

// Validation interceptors

func (i *Interceptors) validationUnaryInterceptor(handler grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (interface{}, error) {
		// Validate request
		if err := i.validateRequest(req, info.FullMethod); err != nil {
			i.logger.Warn("Request validation failed",
				"method", info.FullMethod,
				"error", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return handler(ctx, req, info)
	}
}

func (i *Interceptors) validationStreamInterceptor(handler grpc.StreamHandler) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo) error {
		// Stream validation would be implemented here if needed
		return handler(srv, stream, info)
	}
}

// Recovery interceptors

func (i *Interceptors) recoveryUnaryInterceptor(handler grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				i.logger.Error("Panic recovered in gRPC handler",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(stack))
				
				// Return internal server error
				err = status.Error(codes.Internal, "Internal server error")
			}
		}()

		return handler(ctx, req, info)
	}
}

func (i *Interceptors) recoveryStreamInterceptor(handler grpc.StreamHandler) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				i.logger.Error("Panic recovered in gRPC stream handler",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(stack))
				
				// Return internal server error
				err = status.Error(codes.Internal, "Internal server error")
			}
		}()

		return handler(srv, stream, info)
	}
}

// Timeout interceptors

func (i *Interceptors) timeoutUnaryInterceptor(handler grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (interface{}, error) {
		// Set timeout based on method
		timeout := i.getTimeoutForMethod(info.FullMethod)
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		return handler(ctx, req, info)
	}
}

// Helper methods

// getMetadataValue extracts a value from gRPC metadata
func getMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// validateRequest validates incoming requests
func (i *Interceptors) validateRequest(req interface{}, method string) error {
	// Implement request validation logic based on the method and request type
	// This is a placeholder for method-specific validation
	
	switch method {
	case "/graph_engine.GraphEngine/AnalyzeSubGraph":
		// Validate AnalyzeSubGraph request
		return i.validateAnalyzeSubGraphRequest(req)
	case "/graph_engine.GraphEngine/FindPaths":
		// Validate FindPaths request
		return i.validateFindPathsRequest(req)
	case "/graph_engine.GraphEngine/CreateInvestigation":
		// Validate CreateInvestigation request
		return i.validateCreateInvestigationRequest(req)
	case "/graph_engine.GraphEngine/CalculateNetworkMetrics":
		// Validate CalculateNetworkMetrics request
		return i.validateCalculateNetworkMetricsRequest(req)
	default:
		return nil // No validation for unknown methods
	}
}

// validateAnalyzeSubGraphRequest validates analyze subgraph requests
func (i *Interceptors) validateAnalyzeSubGraphRequest(req interface{}) error {
	// Implementation would validate the specific request structure
	// This is a placeholder for actual validation logic
	return nil
}

// validateFindPathsRequest validates find paths requests
func (i *Interceptors) validateFindPathsRequest(req interface{}) error {
	// Implementation would validate the specific request structure
	// This is a placeholder for actual validation logic
	return nil
}

// validateCreateInvestigationRequest validates create investigation requests
func (i *Interceptors) validateCreateInvestigationRequest(req interface{}) error {
	// Implementation would validate the specific request structure
	// This is a placeholder for actual validation logic
	return nil
}

// validateCalculateNetworkMetricsRequest validates calculate network metrics requests
func (i *Interceptors) validateCalculateNetworkMetricsRequest(req interface{}) error {
	// Implementation would validate the specific request structure
	// This is a placeholder for actual validation logic
	return nil
}

// getTimeoutForMethod returns appropriate timeout for a gRPC method
func (i *Interceptors) getTimeoutForMethod(method string) time.Duration {
	// Define method-specific timeouts
	methodTimeouts := map[string]time.Duration{
		"/graph_engine.GraphEngine/AnalyzeSubGraph":           30 * time.Minute, // Long-running analysis
		"/graph_engine.GraphEngine/FindPaths":                5 * time.Minute,  // Path finding
		"/graph_engine.GraphEngine/CalculateNetworkMetrics":  15 * time.Minute, // Metrics calculation
		"/graph_engine.GraphEngine/CreateInvestigation":      30 * time.Second, // Quick operation
		"/graph_engine.GraphEngine/GetInvestigation":         10 * time.Second, // Quick read
		"/graph_engine.GraphEngine/GetAnalysisJob":           10 * time.Second, // Quick read
		"/graph_engine.GraphEngine/GetEntityNeighborhood":    2 * time.Minute,  // Neighborhood query
		"/graph_engine.GraphEngine/HealthCheck":              5 * time.Second,  // Health check
	}

	if timeout, exists := methodTimeouts[method]; exists {
		return timeout
	}

	// Default timeout
	return 10 * time.Minute
}

// AuthenticationInterceptor provides authentication validation (if needed)
func (i *Interceptors) AuthenticationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication for health checks
		if info.FullMethod == "/graph_engine.GraphEngine/HealthCheck" {
			return handler(ctx, req, info)
		}

		// Extract authentication metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Missing authentication metadata")
		}

		// Validate authentication token (implementation would depend on auth system)
		if err := i.validateAuthToken(md); err != nil {
			i.logger.Warn("Authentication failed",
				"method", info.FullMethod,
				"error", err)
			return nil, status.Error(codes.Unauthenticated, "Invalid authentication")
		}

		return handler(ctx, req, info)
	}
}

// validateAuthToken validates authentication tokens
func (i *Interceptors) validateAuthToken(md metadata.MD) error {
	// Implementation would validate JWT tokens or API keys
	// This is a placeholder for actual authentication logic
	
	authHeader := getMetadataValue(md, "authorization")
	if authHeader == "" {
		return status.Error(codes.Unauthenticated, "Missing authorization header")
	}

	// Placeholder validation
	if !i.config.Debug && authHeader != "Bearer valid-token" {
		return status.Error(codes.Unauthenticated, "Invalid token")
	}

	return nil
}

// RateLimitingInterceptor provides rate limiting (if needed)
func (i *Interceptors) RateLimitingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Implementation would check rate limits based on client IP or user ID
		// This is a placeholder for actual rate limiting logic
		
		md, _ := metadata.FromIncomingContext(ctx)
		clientIP := getMetadataValue(md, "x-forwarded-for")
		if clientIP == "" {
			clientIP = getMetadataValue(md, "x-real-ip")
		}

		// Check rate limit for client
		if i.isRateLimited(clientIP, info.FullMethod) {
			i.logger.Warn("Rate limit exceeded",
				"method", info.FullMethod,
				"client_ip", clientIP)
			return nil, status.Error(codes.ResourceExhausted, "Rate limit exceeded")
		}

		return handler(ctx, req, info)
	}
}

// isRateLimited checks if client is rate limited
func (i *Interceptors) isRateLimited(clientIP, method string) bool {
	// Implementation would check rate limiting based on client and method
	// This is a placeholder for actual rate limiting logic
	return false
}

// ContextEnrichmentInterceptor adds additional context to requests
func (i *Interceptors) ContextEnrichmentInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract metadata and enrich context
		md, _ := metadata.FromIncomingContext(ctx)
		
		// Add request ID if not present
		requestID := getMetadataValue(md, "x-request-id")
		if requestID == "" {
			requestID = generateRequestID()
			ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
		}

		// Add trace information
		traceID := getMetadataValue(md, "x-trace-id")
		if traceID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-trace-id", traceID)
		}

		return handler(ctx, req, info)
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Implementation would generate a unique ID (UUID, etc.)
	return "req-" + time.Now().Format("20060102150405")
}