package interceptors

import (
	"context"
	"log/slog"
	"time"

	"github.com/aegisshield/entity-resolution/internal/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor logs gRPC requests and responses
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract metadata if available
		var traceID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("trace-id"); len(values) > 0 {
				traceID = values[0]
			}
		}

		logger.Info("gRPC request started",
			"method", info.FullMethod,
			"trace_id", traceID)

		// Call the handler
		resp, err := handler(ctx, req)

		// Log the result
		duration := time.Since(start)
		if err != nil {
			logger.Error("gRPC request failed",
				"method", info.FullMethod,
				"trace_id", traceID,
				"duration_ms", duration.Milliseconds(),
				"error", err)
		} else {
			logger.Info("gRPC request completed",
				"method", info.FullMethod,
				"trace_id", traceID,
				"duration_ms", duration.Milliseconds())
		}

		return resp, err
	}
}

// MetricsInterceptor collects metrics for gRPC requests
func MetricsInterceptor(collector *metrics.Collector) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record metrics based on method and outcome
		duration := time.Since(start)
		
		// You could expand this to track specific methods differently
		switch info.FullMethod {
		case "/entity_resolution.EntityResolution/ResolveEntity":
			if err != nil {
				collector.RecordResolutionError()
			}
		case "/entity_resolution.EntityResolution/ResolveBatch":
			// Batch-specific metrics would be handled in the service layer
		}

		return resp, err
	}
}

// ValidationInterceptor validates incoming requests
func ValidationInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Basic validation could be performed here
		// For now, we'll just pass through to the handler
		// More complex validation is done in the service layer

		return handler(ctx, req)
	}
}

// RecoveryInterceptor recovers from panics and returns appropriate errors
func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC handler panicked",
					"method", info.FullMethod,
					"panic", r)
				
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// RateLimitInterceptor implements rate limiting (placeholder)
func RateLimitInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: Implement actual rate limiting
		// This could use a token bucket algorithm or similar
		// For now, just pass through

		return handler(ctx, req)
	}
}

// AuthInterceptor handles authentication (placeholder)
func AuthInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for health checks
		if info.FullMethod == "/entity_resolution.EntityResolution/HealthCheck" {
			return handler(ctx, req)
		}

		// TODO: Implement actual authentication
		// This could validate JWT tokens, API keys, etc.
		// For now, just pass through

		return handler(ctx, req)
	}
}

// TimeoutInterceptor enforces request timeouts
func TimeoutInterceptor(timeout time.Duration, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Create context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Channel to receive handler result
		type result struct {
			resp interface{}
			err  error
		}
		resultChan := make(chan result, 1)

		// Run handler in goroutine
		go func() {
			resp, err := handler(timeoutCtx, req)
			resultChan <- result{resp: resp, err: err}
		}()

		// Wait for either completion or timeout
		select {
		case res := <-resultChan:
			return res.resp, res.err
		case <-timeoutCtx.Done():
			logger.Warn("gRPC request timed out",
				"method", info.FullMethod,
				"timeout", timeout)
			return nil, status.Errorf(codes.DeadlineExceeded, "request timeout")
		}
	}
}

// TracingInterceptor adds distributed tracing (placeholder)
func TracingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: Implement actual tracing with OpenTelemetry or similar
		// This would create spans, inject trace context, etc.

		// For now, just extract trace ID from metadata if present
		var traceID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("trace-id"); len(values) > 0 {
				traceID = values[0]
				// Add trace ID to context for downstream use
				ctx = context.WithValue(ctx, "trace_id", traceID)
			}
		}

		return handler(ctx, req)
	}
}

// StreamLoggingInterceptor logs streaming gRPC requests
func StreamLoggingInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		logger.Info("gRPC stream started",
			"method", info.FullMethod)

		// Call the handler
		err := handler(srv, stream)

		// Log the result
		duration := time.Since(start)
		if err != nil {
			logger.Error("gRPC stream failed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"error", err)
		} else {
			logger.Info("gRPC stream completed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds())
		}

		return err
	}
}

// StreamRecoveryInterceptor recovers from panics in streaming handlers
func StreamRecoveryInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC stream handler panicked",
					"method", info.FullMethod,
					"panic", r)
				
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, stream)
	}
}

// ChainUnaryInterceptors chains multiple unary interceptors
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	if len(interceptors) == 0 {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	if len(interceptors) == 1 {
		return interceptors[0]
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return chainUnaryInterceptors(interceptors, 0, info, handler)(ctx, req)
	}
}

func chainUnaryInterceptors(interceptors []grpc.UnaryServerInterceptor, curr int, info *grpc.UnaryServerInfo, finalHandler grpc.UnaryHandler) grpc.UnaryHandler {
	if curr == len(interceptors)-1 {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			return interceptors[curr](ctx, req, info, finalHandler)
		}
	}

	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return interceptors[curr](ctx, req, info, chainUnaryInterceptors(interceptors, curr+1, info, finalHandler))
	}
}

// ChainStreamInterceptors chains multiple stream interceptors
func ChainStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	if len(interceptors) == 0 {
		return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, stream)
		}
	}

	if len(interceptors) == 1 {
		return interceptors[0]
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return chainStreamInterceptors(interceptors, 0, info, handler)(srv, stream)
	}
}

func chainStreamInterceptors(interceptors []grpc.StreamServerInterceptor, curr int, info *grpc.StreamServerInfo, finalHandler grpc.StreamHandler) grpc.StreamHandler {
	if curr == len(interceptors)-1 {
		return func(srv interface{}, stream grpc.ServerStream) error {
			return interceptors[curr](srv, stream, info, finalHandler)
		}
	}

	return func(srv interface{}, stream grpc.ServerStream) error {
		return interceptors[curr](srv, stream, info, chainStreamInterceptors(interceptors, curr+1, info, finalHandler))
	}
}