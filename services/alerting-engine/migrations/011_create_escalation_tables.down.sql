-- Drop escalation tables
DROP TRIGGER IF EXISTS update_escalation_policies_updated_at ON escalation_policies;

-- Drop escalation_events indexes
DROP INDEX IF EXISTS idx_escalation_events_metadata;
DROP INDEX IF EXISTS idx_escalation_events_action_result;
DROP INDEX IF EXISTS idx_escalation_events_action_taken;
DROP INDEX IF EXISTS idx_escalation_events_policy_triggered;
DROP INDEX IF EXISTS idx_escalation_events_alert_level;
DROP INDEX IF EXISTS idx_escalation_events_triggered_by;
DROP INDEX IF EXISTS idx_escalation_events_triggered_at;
DROP INDEX IF EXISTS idx_escalation_events_level;
DROP INDEX IF EXISTS idx_escalation_events_type;
DROP INDEX IF EXISTS idx_escalation_events_policy_id;
DROP INDEX IF EXISTS idx_escalation_events_alert_id;

-- Drop escalation_policies indexes
DROP INDEX IF EXISTS idx_escalation_policies_tags;
DROP INDEX IF EXISTS idx_escalation_policies_metadata;
DROP INDEX IF EXISTS idx_escalation_policies_trigger_conditions;
DROP INDEX IF EXISTS idx_escalation_policies_rules;
DROP INDEX IF EXISTS idx_escalation_policies_created_by;
DROP INDEX IF EXISTS idx_escalation_policies_last_used;
DROP INDEX IF EXISTS idx_escalation_policies_updated_at;
DROP INDEX IF EXISTS idx_escalation_policies_created_at;
DROP INDEX IF EXISTS idx_escalation_policies_enabled;
DROP INDEX IF EXISTS idx_escalation_policies_name;

-- Drop tables
DROP TABLE IF EXISTS escalation_events;
DROP TABLE IF EXISTS escalation_policies;