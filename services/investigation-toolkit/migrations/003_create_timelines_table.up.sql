-- Create timelines table for investigation timeline events
CREATE TABLE IF NOT EXISTS timelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investigation_id UUID NOT NULL REFERENCES investigations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('transaction', 'communication', 'meeting', 'document', 'investigation_action', 'system_event', 'other')),
    event_date TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_minutes INTEGER,
    location VARCHAR(255),
    participants TEXT[],
    related_evidence_ids UUID[],
    external_references JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    tags TEXT[],
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT timelines_duration_positive CHECK (duration_minutes IS NULL OR duration_minutes > 0)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_timelines_investigation_id ON timelines(investigation_id);
CREATE INDEX IF NOT EXISTS idx_timelines_event_type ON timelines(event_type);
CREATE INDEX IF NOT EXISTS idx_timelines_event_date ON timelines(event_date);
CREATE INDEX IF NOT EXISTS idx_timelines_created_by ON timelines(created_by);
CREATE INDEX IF NOT EXISTS idx_timelines_created_at ON timelines(created_at);
CREATE INDEX IF NOT EXISTS idx_timelines_participants ON timelines USING GIN(participants);
CREATE INDEX IF NOT EXISTS idx_timelines_related_evidence_ids ON timelines USING GIN(related_evidence_ids);
CREATE INDEX IF NOT EXISTS idx_timelines_tags ON timelines USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_timelines_metadata ON timelines USING GIN(metadata);
CREATE INDEX IF NOT EXISTS idx_timelines_external_references ON timelines USING GIN(external_references);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER update_timelines_updated_at
    BEFORE UPDATE ON timelines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();