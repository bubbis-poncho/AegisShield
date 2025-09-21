-- Drop triggers first
DROP TRIGGER IF EXISTS update_entities_updated_at ON entities;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_entities_standardized_name_trgm;
DROP INDEX IF EXISTS idx_entities_name_trgm;
DROP INDEX IF EXISTS idx_entities_attributes_gin;
DROP INDEX IF EXISTS idx_entities_identifiers_gin;
DROP INDEX IF EXISTS idx_entities_updated_at;
DROP INDEX IF EXISTS idx_entities_created_at;
DROP INDEX IF EXISTS idx_entities_confidence_score;
DROP INDEX IF EXISTS idx_entities_standardized_name;
DROP INDEX IF EXISTS idx_entities_name;
DROP INDEX IF EXISTS idx_entities_entity_type;

-- Drop table
DROP TABLE IF EXISTS entities;