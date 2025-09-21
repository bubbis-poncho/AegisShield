-- Create entities table for storing resolved entities
CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(100) NOT NULL,
    name VARCHAR(500),
    standardized_name VARCHAR(500),
    identifiers JSONB DEFAULT '{}',
    attributes JSONB DEFAULT '{}',
    confidence_score DECIMAL(5,4) NOT NULL DEFAULT 1.0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_entities_entity_type ON entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name);
CREATE INDEX IF NOT EXISTS idx_entities_standardized_name ON entities(standardized_name);
CREATE INDEX IF NOT EXISTS idx_entities_confidence_score ON entities(confidence_score);
CREATE INDEX IF NOT EXISTS idx_entities_created_at ON entities(created_at);
CREATE INDEX IF NOT EXISTS idx_entities_updated_at ON entities(updated_at);

-- Create GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_entities_identifiers_gin ON entities USING GIN(identifiers);
CREATE INDEX IF NOT EXISTS idx_entities_attributes_gin ON entities USING GIN(attributes);

-- Enable fuzzy text search extensions if not already enabled
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;

-- Create indexes for fuzzy text matching
CREATE INDEX IF NOT EXISTS idx_entities_name_trgm ON entities USING GIN(name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_entities_standardized_name_trgm ON entities USING GIN(standardized_name gin_trgm_ops);

-- Add trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_entities_updated_at 
    BEFORE UPDATE ON entities 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();