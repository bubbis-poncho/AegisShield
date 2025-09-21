package interceptors

import (
	"context"
	"log/slog"
	"time"

	"github.com/aegisshield/data-ingestion/internal/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor provides request/response logging for gRPC calls
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInterceptor, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		logger.Info("gRPC request started",
			"method", info.FullMethod,
			"start_time", start,
		)

		resp, err := handler(ctx, req)
		
		duration := time.Since(start)
		
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("gRPC request failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
				"code", st.Code(),
				"message", st.Message(),
			)
		} else {
			logger.Info("gRPC request completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return resp, err
	}
}

// MetricsInterceptor provides metrics collection for gRPC calls
func MetricsInterceptor(metrics *metrics.Collector) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInterceptor, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		resp, err := handler(ctx, req)
		
		duration := time.Since(start)
		
		// Record metrics based on method
		switch info.FullMethod {
		case "/dataingrestion.DataIngestionService/UploadFile":
			metrics.RecordHistogram("upload_file_duration_seconds", duration.Seconds())
			if err != nil {
				metrics.IncrementCounter("upload_file_errors_total")
			} else {
				metrics.IncrementCounter("upload_file_requests_total")
			}
			
		case "/dataingrestion.DataIngestionService/UploadFileStream":
			metrics.RecordHistogram("upload_file_stream_duration_seconds", duration.Seconds())
			if err != nil {
				metrics.IncrementCounter("upload_file_stream_errors_total")
			} else {
				metrics.IncrementCounter("upload_file_stream_requests_total")
			}
			
		case "/dataingrestion.DataIngestionService/ProcessTransactionStream":
			metrics.RecordHistogram("process_transaction_stream_duration_seconds", duration.Seconds())
			if err != nil {
				metrics.IncrementCounter("process_transaction_stream_errors_total")
			} else {
				metrics.IncrementCounter("process_transaction_stream_requests_total")
			}
			
		case "/dataingrestion.DataIngestionService/ValidateData":
			metrics.RecordHistogram("validate_data_duration_seconds", duration.Seconds())
			if err != nil {
				metrics.IncrementCounter("validate_data_errors_total")
			} else {
				metrics.IncrementCounter("validate_data_requests_total")
			}
		}

		return resp, err
	}
}

// ErrorHandlingInterceptor provides consistent error handling and conversion
func ErrorHandlingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInterceptor, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		
		if err != nil {
			// Convert internal errors to appropriate gRPC status codes
			if st, ok := status.FromError(err); ok {
				// Already a gRPC status error
				return resp, err
			}
			
			// Convert common errors to gRPC status codes
			switch {
			case err.Error() == "context canceled":
				return resp, status.Error(codes.Canceled, "Request was canceled")
			case err.Error() == "context deadline exceeded":
				return resp, status.Error(codes.DeadlineExceeded, "Request timeout")
			default:
				// Log the original error and return a generic internal error
				logger.Error("Internal server error",
					"method", info.FullMethod,
					"error", err,
				)
				return resp, status.Error(codes.Internal, "Internal server error")
			}
		}
		
		return resp, err
	}
}

// ValidationInterceptor provides request validation
func ValidationInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInterceptor, handler grpc.UnaryHandler) (interface{}, error) {
		// Basic request validation
		if req == nil {
			logger.Warn("Received nil request",
				"method", info.FullMethod,
			)
			return nil, status.Error(codes.InvalidArgument, "Request cannot be nil")
		}
		
		// Add method-specific validations here if needed
		// For now, we'll rely on the service-level validation
		
		return handler(ctx, req)
	}
}

// RecoveryInterceptor provides panic recovery for gRPC calls
func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInterceptor, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic recovered in gRPC handler",
					"method", info.FullMethod,
					"panic", r,
				)
				err = status.Error(codes.Internal, "Internal server error")
			}
		}()
		
		return handler(ctx, req)
	}
}

// StreamLoggingInterceptor provides request/response logging for gRPC streaming calls
func StreamLoggingInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		logger.Info("gRPC stream started",
			"method", info.FullMethod,
			"start_time", start,
		)

		err := handler(srv, stream)
		
		duration := time.Since(start)
		
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("gRPC stream failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
				"code", st.Code(),
				"message", st.Message(),
			)
		} else {
			logger.Info("gRPC stream completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return err
	}
}

// StreamMetricsInterceptor provides metrics collection for gRPC streaming calls
func StreamMetricsInterceptor(metrics *metrics.Collector) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		err := handler(srv, stream)
		
		duration := time.Since(start)
		
		// Record metrics based on method
		switch info.FullMethod {
		case "/dataingrestion.DataIngestionService/UploadFileStream":
			metrics.RecordHistogram("upload_file_stream_duration_seconds", duration.Seconds())
			if err != nil {
				metrics.IncrementCounter("upload_file_stream_errors_total")
			} else {
				metrics.IncrementCounter("upload_file_stream_requests_total")
			}
			
		case "/dataingrestion.DataIngestionService/ProcessTransactionStream":
			metrics.RecordHistogram("process_transaction_stream_duration_seconds", duration.Seconds())
			if err != nil {
				metrics.IncrementCounter("process_transaction_stream_errors_total")
			} else {
				metrics.IncrementCounter("process_transaction_stream_requests_total")
			}
		}

		return err
	}
}

// StreamRecoveryInterceptor provides panic recovery for gRPC streaming calls
func StreamRecoveryInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic recovered in gRPC stream handler",
					"method", info.FullMethod,
					"panic", r,
				)
				err = status.Error(codes.Internal, "Internal server error")
			}
		}()
		
		return handler(srv, stream)
	}
}

// StreamErrorHandlingInterceptor provides consistent error handling for streaming calls
func StreamErrorHandlingInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		
		if err != nil {
			// Convert internal errors to appropriate gRPC status codes
			if st, ok := status.FromError(err); ok {
				// Already a gRPC status error
				return err
			}
			
			// Convert common errors to gRPC status codes
			switch {
			case err.Error() == "context canceled":
				return status.Error(codes.Canceled, "Stream was canceled")
			case err.Error() == "context deadline exceeded":
				return status.Error(codes.DeadlineExceeded, "Stream timeout")
			default:
				// Log the original error and return a generic internal error
				logger.Error("Internal server error in stream",
					"method", info.FullMethod,
					"error", err,
				)
				return status.Error(codes.Internal, "Internal server error")
			}
		}
		
		return err
	}
}