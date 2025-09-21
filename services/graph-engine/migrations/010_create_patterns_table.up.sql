-- Create patterns table
CREATE TABLE IF NOT EXISTS patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_type VARCHAR(100) NOT NULL,
    entities TEXT[] NOT NULL,
    relationships TEXT[] NOT NULL,
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    severity VARCHAR(50) NOT NULL DEFAULT 'medium',
    description TEXT,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB,
    evidence JSONB,
    rule_id VARCHAR(100),
    analysis_job_id UUID,
    investigation_id UUID,
    is_verified BOOLEAN DEFAULT false,
    verified_by VARCHAR(255),
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for patterns
CREATE INDEX IF NOT EXISTS idx_patterns_type ON patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_patterns_severity ON patterns(severity);
CREATE INDEX IF NOT EXISTS idx_patterns_confidence ON patterns(confidence);
CREATE INDEX IF NOT EXISTS idx_patterns_detected_at ON patterns(detected_at);
CREATE INDEX IF NOT EXISTS idx_patterns_entities ON patterns USING GIN(entities);
CREATE INDEX IF NOT EXISTS idx_patterns_relationships ON patterns USING GIN(relationships);
CREATE INDEX IF NOT EXISTS idx_patterns_analysis_job_id ON patterns(analysis_job_id);
CREATE INDEX IF NOT EXISTS idx_patterns_investigation_id ON patterns(investigation_id);
CREATE INDEX IF NOT EXISTS idx_patterns_is_verified ON patterns(is_verified);

-- Add foreign key constraints
ALTER TABLE patterns 
ADD CONSTRAINT fk_patterns_analysis_job_id 
FOREIGN KEY (analysis_job_id) REFERENCES analysis_jobs(id) ON DELETE SET NULL;

-- Add comments
COMMENT ON TABLE patterns IS 'Stores detected patterns and anomalies in the graph';
COMMENT ON COLUMN patterns.id IS 'Unique identifier for the detected pattern';
COMMENT ON COLUMN patterns.pattern_type IS 'Type of pattern (triangle, star, chain, money_laundering, etc.)';
COMMENT ON COLUMN patterns.entities IS 'Array of entity IDs involved in the pattern';
COMMENT ON COLUMN patterns.relationships IS 'Array of relationship IDs forming the pattern';
COMMENT ON COLUMN patterns.confidence IS 'Confidence score of the pattern detection (0.0-1.0)';
COMMENT ON COLUMN patterns.severity IS 'Severity level (low, medium, high, critical)';
COMMENT ON COLUMN patterns.evidence IS 'Evidence supporting the pattern detection';
COMMENT ON COLUMN patterns.rule_id IS 'ID of the rule that detected this pattern';
COMMENT ON COLUMN patterns.is_verified IS 'Whether the pattern has been manually verified';