-- Create evidence table
CREATE TABLE IF NOT EXISTS evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investigation_id UUID NOT NULL REFERENCES investigations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    evidence_type VARCHAR(50) NOT NULL CHECK (evidence_type IN ('document', 'image', 'video', 'audio', 'transaction', 'communication', 'digital', 'physical', 'other')),
    source VARCHAR(100),
    collection_method VARCHAR(100),
    file_path VARCHAR(500),
    file_size BIGINT,
    file_hash VARCHAR(128),
    mime_type VARCHAR(100),
    collected_by UUID NOT NULL,
    collected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    chain_of_custody JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    tags TEXT[],
    is_authenticated BOOLEAN DEFAULT FALSE,
    authentication_method VARCHAR(100),
    authentication_date TIMESTAMP WITH TIME ZONE,
    authentication_by UUID,
    retention_date TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'under_review', 'authenticated', 'rejected', 'archived')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_evidence_investigation_id ON evidence(investigation_id);
CREATE INDEX IF NOT EXISTS idx_evidence_evidence_type ON evidence(evidence_type);
CREATE INDEX IF NOT EXISTS idx_evidence_collected_by ON evidence(collected_by);
CREATE INDEX IF NOT EXISTS idx_evidence_collected_at ON evidence(collected_at);
CREATE INDEX IF NOT EXISTS idx_evidence_status ON evidence(status);
CREATE INDEX IF NOT EXISTS idx_evidence_file_hash ON evidence(file_hash);
CREATE INDEX IF NOT EXISTS idx_evidence_tags ON evidence USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_evidence_metadata ON evidence USING GIN(metadata);
CREATE INDEX IF NOT EXISTS idx_evidence_chain_of_custody ON evidence USING GIN(chain_of_custody);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER update_evidence_updated_at
    BEFORE UPDATE ON evidence
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();