-- Migration: 002_create_data_jobs_table
-- Description: Create data processing jobs table
-- Up Migration

CREATE TABLE IF NOT EXISTS data_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES file_uploads(id) ON DELETE CASCADE,
    job_type VARCHAR(50) NOT NULL DEFAULT 'file_processing',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    processed_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_data_jobs_file_id ON data_jobs(file_id);
CREATE INDEX IF NOT EXISTS idx_data_jobs_status ON data_jobs(status);
CREATE INDEX IF NOT EXISTS idx_data_jobs_job_type ON data_jobs(job_type);
CREATE INDEX IF NOT EXISTS idx_data_jobs_created_at ON data_jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_data_jobs_started_at ON data_jobs(started_at);
CREATE INDEX IF NOT EXISTS idx_data_jobs_completed_at ON data_jobs(completed_at);

-- GIN index for JSONB metadata queries
CREATE INDEX IF NOT EXISTS idx_data_jobs_metadata ON data_jobs USING GIN(metadata);

-- Check constraints
ALTER TABLE data_jobs 
ADD CONSTRAINT chk_data_jobs_status 
CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled', 'paused'));

ALTER TABLE data_jobs 
ADD CONSTRAINT chk_data_jobs_job_type 
CHECK (job_type IN ('file_processing', 'transaction_processing', 'validation', 'enrichment', 'analysis'));

ALTER TABLE data_jobs 
ADD CONSTRAINT chk_data_jobs_counts 
CHECK (processed_count >= 0 AND error_count >= 0 AND total_count >= 0);

-- Comments
COMMENT ON TABLE data_jobs IS 'Tracks data processing jobs';
COMMENT ON COLUMN data_jobs.id IS 'Unique identifier for the job';
COMMENT ON COLUMN data_jobs.file_id IS 'Reference to the file being processed';
COMMENT ON COLUMN data_jobs.job_type IS 'Type of processing job';
COMMENT ON COLUMN data_jobs.status IS 'Current status of the job';
COMMENT ON COLUMN data_jobs.processed_count IS 'Number of records successfully processed';
COMMENT ON COLUMN data_jobs.error_count IS 'Number of records that failed processing';
COMMENT ON COLUMN data_jobs.total_count IS 'Total number of records to process';
COMMENT ON COLUMN data_jobs.started_at IS 'When the job started processing';
COMMENT ON COLUMN data_jobs.completed_at IS 'When the job completed (success or failure)';
COMMENT ON COLUMN data_jobs.error_message IS 'Error message if job failed';
COMMENT ON COLUMN data_jobs.metadata IS 'Additional job metadata as JSON';