-- Create resolution_jobs table for tracking batch processing jobs
CREATE TABLE IF NOT EXISTS resolution_jobs (
    id UUID PRIMARY KEY,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    progress INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure valid status values
    CONSTRAINT chk_resolution_jobs_status 
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    
    -- Ensure valid progress values
    CONSTRAINT chk_resolution_jobs_progress 
        CHECK (progress >= 0 AND progress <= total),
    
    -- Ensure valid counts
    CONSTRAINT chk_resolution_jobs_counts 
        CHECK (error_count >= 0 AND success_count >= 0)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_resolution_jobs_status ON resolution_jobs(status);
CREATE INDEX IF NOT EXISTS idx_resolution_jobs_started_at ON resolution_jobs(started_at);
CREATE INDEX IF NOT EXISTS idx_resolution_jobs_completed_at ON resolution_jobs(completed_at);
CREATE INDEX IF NOT EXISTS idx_resolution_jobs_created_at ON resolution_jobs(created_at);

-- Create GIN index for JSONB metadata
CREATE INDEX IF NOT EXISTS idx_resolution_jobs_metadata_gin ON resolution_jobs USING GIN(metadata);

-- Add trigger to automatically update updated_at timestamp
CREATE TRIGGER update_resolution_jobs_updated_at 
    BEFORE UPDATE ON resolution_jobs 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();