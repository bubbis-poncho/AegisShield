-- Drop trigger
DROP TRIGGER IF EXISTS update_entity_links_updated_at ON entity_links;

-- Drop indexes
DROP INDEX IF EXISTS idx_entity_links_unique;
DROP INDEX IF EXISTS idx_entity_links_properties_gin;
DROP INDEX IF EXISTS idx_entity_links_target_type;
DROP INDEX IF EXISTS idx_entity_links_source_type;
DROP INDEX IF EXISTS idx_entity_links_created_at;
DROP INDEX IF EXISTS idx_entity_links_confidence_score;
DROP INDEX IF EXISTS idx_entity_links_link_type;
DROP INDEX IF EXISTS idx_entity_links_target_entity_id;
DROP INDEX IF EXISTS idx_entity_links_source_entity_id;

-- Drop table
DROP TABLE IF EXISTS entity_links;