-- Drop investigations table and related objects
DROP TRIGGER IF EXISTS update_investigations_updated_at ON investigations;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS investigations;