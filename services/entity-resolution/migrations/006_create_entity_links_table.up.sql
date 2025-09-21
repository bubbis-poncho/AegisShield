-- Create entity_links table for storing relationships between entities
CREATE TABLE IF NOT EXISTS entity_links (
    id UUID PRIMARY KEY,
    source_entity_id UUID NOT NULL,
    target_entity_id UUID NOT NULL,
    link_type VARCHAR(100) NOT NULL,
    properties JSONB DEFAULT '{}',
    confidence_score DECIMAL(5,4) NOT NULL DEFAULT 1.0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key constraints
    CONSTRAINT fk_entity_links_source_entity 
        FOREIGN KEY (source_entity_id) REFERENCES entities(id) ON DELETE CASCADE,
    CONSTRAINT fk_entity_links_target_entity 
        FOREIGN KEY (target_entity_id) REFERENCES entities(id) ON DELETE CASCADE,
    
    -- Ensure no self-links
    CONSTRAINT chk_entity_links_no_self_link 
        CHECK (source_entity_id != target_entity_id),
    
    -- Ensure valid confidence score
    CONSTRAINT chk_entity_links_confidence_score 
        CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_entity_links_source_entity_id ON entity_links(source_entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_target_entity_id ON entity_links(target_entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_link_type ON entity_links(link_type);
CREATE INDEX IF NOT EXISTS idx_entity_links_confidence_score ON entity_links(confidence_score);
CREATE INDEX IF NOT EXISTS idx_entity_links_created_at ON entity_links(created_at);

-- Create composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_entity_links_source_type ON entity_links(source_entity_id, link_type);
CREATE INDEX IF NOT EXISTS idx_entity_links_target_type ON entity_links(target_entity_id, link_type);

-- Create GIN index for JSONB properties
CREATE INDEX IF NOT EXISTS idx_entity_links_properties_gin ON entity_links USING GIN(properties);

-- Create unique constraint to prevent duplicate links
CREATE UNIQUE INDEX IF NOT EXISTS idx_entity_links_unique 
    ON entity_links(source_entity_id, target_entity_id, link_type);

-- Add trigger to automatically update updated_at timestamp
CREATE TRIGGER update_entity_links_updated_at 
    BEFORE UPDATE ON entity_links 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();