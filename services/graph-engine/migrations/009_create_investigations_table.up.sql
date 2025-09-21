-- Create investigations table
CREATE TABLE IF NOT EXISTS investigations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    entities TEXT[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    assigned_to VARCHAR(255),
    metadata JSONB,
    findings JSONB,
    evidence JSONB,
    tags TEXT[]
);

-- Create indexes for investigations
CREATE INDEX IF NOT EXISTS idx_investigations_status ON investigations(status);
CREATE INDEX IF NOT EXISTS idx_investigations_priority ON investigations(priority);
CREATE INDEX IF NOT EXISTS idx_investigations_created_by ON investigations(created_by);
CREATE INDEX IF NOT EXISTS idx_investigations_assigned_to ON investigations(assigned_to);
CREATE INDEX IF NOT EXISTS idx_investigations_created_at ON investigations(created_at);
CREATE INDEX IF NOT EXISTS idx_investigations_entities ON investigations USING GIN(entities);
CREATE INDEX IF NOT EXISTS idx_investigations_tags ON investigations USING GIN(tags);

-- Add comments
COMMENT ON TABLE investigations IS 'Stores investigation cases and their progress';
COMMENT ON COLUMN investigations.id IS 'Unique identifier for the investigation';
COMMENT ON COLUMN investigations.name IS 'Investigation name or title';
COMMENT ON COLUMN investigations.description IS 'Detailed description of the investigation';
COMMENT ON COLUMN investigations.status IS 'Current status (open, in_progress, closed, suspended)';
COMMENT ON COLUMN investigations.priority IS 'Investigation priority (low, medium, high, critical)';
COMMENT ON COLUMN investigations.entities IS 'Array of entity IDs involved in the investigation';
COMMENT ON COLUMN investigations.created_by IS 'User who created the investigation';
COMMENT ON COLUMN investigations.assigned_to IS 'User assigned to the investigation';
COMMENT ON COLUMN investigations.findings IS 'Investigation findings and conclusions';
COMMENT ON COLUMN investigations.evidence IS 'Evidence collected during investigation';
COMMENT ON COLUMN investigations.tags IS 'Tags for categorizing investigations';