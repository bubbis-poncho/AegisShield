-- Create analysis_jobs table
CREATE TABLE IF NOT EXISTS analysis_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    parameters JSONB,
    results JSONB,
    created_by VARCHAR(255),
    entity_ids TEXT[] NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for analysis_jobs
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_type ON analysis_jobs(type);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_status ON analysis_jobs(status);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_started_at ON analysis_jobs(started_at);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_created_by ON analysis_jobs(created_by);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_entity_ids ON analysis_jobs USING GIN(entity_ids);

-- Add comments
COMMENT ON TABLE analysis_jobs IS 'Stores graph analysis job information and progress';
COMMENT ON COLUMN analysis_jobs.id IS 'Unique identifier for the analysis job';
COMMENT ON COLUMN analysis_jobs.type IS 'Type of analysis (subgraph, pathfinding, metrics, etc.)';
COMMENT ON COLUMN analysis_jobs.status IS 'Current status (pending, running, completed, failed)';
COMMENT ON COLUMN analysis_jobs.progress IS 'Current progress count';
COMMENT ON COLUMN analysis_jobs.total IS 'Total work units to complete';
COMMENT ON COLUMN analysis_jobs.parameters IS 'Analysis configuration parameters';
COMMENT ON COLUMN analysis_jobs.results IS 'Analysis results and findings';
COMMENT ON COLUMN analysis_jobs.entity_ids IS 'Array of entity IDs being analyzed';
COMMENT ON COLUMN analysis_jobs.metadata IS 'Additional job metadata and context';