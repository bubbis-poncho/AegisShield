-- Migration: Drop dashboards table
-- Version: 012

DROP TRIGGER IF EXISTS update_dashboards_updated_at ON dashboards;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP INDEX IF EXISTS idx_dashboards_user_id;
DROP INDEX IF EXISTS idx_dashboards_created_at;
DROP INDEX IF EXISTS idx_dashboards_is_default;
DROP INDEX IF EXISTS idx_dashboards_is_public;
DROP TABLE IF EXISTS dashboards;