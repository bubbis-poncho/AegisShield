# Build stage
FROM python:3.11-slim AS builder

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir --user -r requirements.txt

# Final stage
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user
RUN useradd --create-home --shell /bin/bash appuser

# Copy Python packages from builder
COPY --from=builder /root/.local /home/appuser/.local

# Copy application code
COPY . .

# Create requirements.txt if it doesn't exist
RUN echo "fastapi==0.104.1" > requirements.txt && \
    echo "uvicorn[standard]==0.24.0" >> requirements.txt && \
    echo "scikit-learn==1.3.2" >> requirements.txt && \
    echo "pandas==2.1.3" >> requirements.txt && \
    echo "numpy==1.25.2" >> requirements.txt && \
    echo "grpcio==1.59.3" >> requirements.txt && \
    echo "grpcio-tools==1.59.3" >> requirements.txt && \
    echo "protobuf==4.25.1" >> requirements.txt && \
    echo "pydantic==2.5.0" >> requirements.txt

# Install any missing packages
RUN pip install --no-cache-dir -r requirements.txt

# Change ownership to non-root user
RUN chown -R appuser:appuser /app
USER appuser

# Make sure scripts are executable
RUN chmod +x /app/src/main.py

# Update PATH to include user packages
ENV PATH=/home/appuser/.local/bin:$PATH

# Expose port
EXPOSE 8061

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8061/health || exit 1

# Set environment variables
ENV PYTHONPATH=/app
ENV PORT=8061

# Run the application
CMD ["python", "src/main.py"]