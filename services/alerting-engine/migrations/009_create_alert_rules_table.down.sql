-- Drop alert_rules table
DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;
DROP INDEX IF EXISTS idx_alert_rules_tags;
DROP INDEX IF EXISTS idx_alert_rules_metadata;
DROP INDEX IF EXISTS idx_alert_rules_actions;
DROP INDEX IF EXISTS idx_alert_rules_conditions;
DROP INDEX IF EXISTS idx_alert_rules_category_enabled;
DROP INDEX IF EXISTS idx_alert_rules_enabled_severity;
DROP INDEX IF EXISTS idx_alert_rules_enabled_type;
DROP INDEX IF EXISTS idx_alert_rules_created_by;
DROP INDEX IF EXISTS idx_alert_rules_last_evaluation;
DROP INDEX IF EXISTS idx_alert_rules_updated_at;
DROP INDEX IF EXISTS idx_alert_rules_created_at;
DROP INDEX IF EXISTS idx_alert_rules_category;
DROP INDEX IF EXISTS idx_alert_rules_type;
DROP INDEX IF EXISTS idx_alert_rules_severity;
DROP INDEX IF EXISTS idx_alert_rules_enabled;
DROP INDEX IF EXISTS idx_alert_rules_name;
DROP TABLE IF EXISTS alert_rules;