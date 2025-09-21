# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
COPY ../../shared/go.mod ../../shared/go.sum ../../shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .
COPY ../../shared ../../shared

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates and other runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy any necessary config files
COPY --from=builder /app/config ./config

# Copy migration files if they exist
COPY --from=builder /app/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appuser /root/
USER appuser

# Expose port
EXPOSE 8062

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8062/health || exit 1

# Set environment variables
ENV GIN_MODE=release
ENV PORT=8062

# Run the application
CMD ["./main"]