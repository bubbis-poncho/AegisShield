-- Migration: Drop widgets table
-- Version: 013

DROP TRIGGER IF EXISTS update_widgets_updated_at ON widgets;
DROP INDEX IF EXISTS idx_widgets_dashboard_id;
DROP INDEX IF EXISTS idx_widgets_type;
DROP INDEX IF EXISTS idx_widgets_created_at;
DROP INDEX IF EXISTS idx_widgets_is_visible;
DROP TABLE IF EXISTS widgets;