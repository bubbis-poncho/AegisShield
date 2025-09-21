-- Create escalation_policies table
CREATE TABLE IF NOT EXISTS escalation_policies (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    enabled BOOLEAN DEFAULT true,
    
    -- Escalation configuration
    rules JSONB NOT NULL,
    default_action JSONB,
    
    -- Conditions for triggering escalation
    trigger_conditions JSONB,
    
    -- Metadata
    metadata JSONB,
    tags TEXT[],
    
    -- Audit fields
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Statistics
    usage_count BIGINT DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT escalation_policies_name_unique UNIQUE (name)
);

-- Create escalation_events table for tracking escalation history
CREATE TABLE IF NOT EXISTS escalation_events (
    id VARCHAR(255) PRIMARY KEY,
    alert_id VARCHAR(255) NOT NULL,
    escalation_policy_id VARCHAR(255),
    
    event_type VARCHAR(50) NOT NULL,
    escalation_level INTEGER NOT NULL,
    
    -- Event details
    triggered_by VARCHAR(255),
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Action taken
    action_taken JSONB,
    action_result JSONB,
    
    -- Metadata
    metadata JSONB,
    
    CONSTRAINT escalation_events_type_check CHECK (event_type IN ('escalated', 'resolved', 'acknowledged', 'timeout')),
    CONSTRAINT escalation_events_level_positive CHECK (escalation_level >= 0),
    
    FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE,
    FOREIGN KEY (escalation_policy_id) REFERENCES escalation_policies(id) ON DELETE SET NULL
);

-- Create indexes for escalation_policies table
CREATE INDEX IF NOT EXISTS idx_escalation_policies_name ON escalation_policies(name);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_enabled ON escalation_policies(enabled);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_created_at ON escalation_policies(created_at);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_updated_at ON escalation_policies(updated_at);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_last_used ON escalation_policies(last_used_at);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_created_by ON escalation_policies(created_by);

-- Create GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_escalation_policies_rules ON escalation_policies USING GIN (rules);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_trigger_conditions ON escalation_policies USING GIN (trigger_conditions);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_metadata ON escalation_policies USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_tags ON escalation_policies USING GIN (tags);

-- Create indexes for escalation_events table
CREATE INDEX IF NOT EXISTS idx_escalation_events_alert_id ON escalation_events(alert_id);
CREATE INDEX IF NOT EXISTS idx_escalation_events_policy_id ON escalation_events(escalation_policy_id);
CREATE INDEX IF NOT EXISTS idx_escalation_events_type ON escalation_events(event_type);
CREATE INDEX IF NOT EXISTS idx_escalation_events_level ON escalation_events(escalation_level);
CREATE INDEX IF NOT EXISTS idx_escalation_events_triggered_at ON escalation_events(triggered_at);
CREATE INDEX IF NOT EXISTS idx_escalation_events_triggered_by ON escalation_events(triggered_by);

-- Create composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_escalation_events_alert_level ON escalation_events(alert_id, escalation_level);
CREATE INDEX IF NOT EXISTS idx_escalation_events_policy_triggered ON escalation_events(escalation_policy_id, triggered_at DESC);

-- Create GIN indexes for JSONB columns in escalation_events
CREATE INDEX IF NOT EXISTS idx_escalation_events_action_taken ON escalation_events USING GIN (action_taken);
CREATE INDEX IF NOT EXISTS idx_escalation_events_action_result ON escalation_events USING GIN (action_result);
CREATE INDEX IF NOT EXISTS idx_escalation_events_metadata ON escalation_events USING GIN (metadata);

-- Create triggers
CREATE TRIGGER update_escalation_policies_updated_at 
    BEFORE UPDATE ON escalation_policies 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Add table comments
COMMENT ON TABLE escalation_policies IS 'Stores escalation policies for alert management';
COMMENT ON COLUMN escalation_policies.id IS 'Unique identifier for the escalation policy';
COMMENT ON COLUMN escalation_policies.rules IS 'JSON configuration for escalation rules and steps';
COMMENT ON COLUMN escalation_policies.trigger_conditions IS 'JSON conditions that trigger this escalation policy';
COMMENT ON COLUMN escalation_policies.usage_count IS 'Number of times this policy has been used';

COMMENT ON TABLE escalation_events IS 'Stores escalation events and history';
COMMENT ON COLUMN escalation_events.id IS 'Unique identifier for the escalation event';
COMMENT ON COLUMN escalation_events.alert_id IS 'ID of the alert that was escalated';
COMMENT ON COLUMN escalation_events.escalation_level IS 'Level of escalation (0 = initial, 1+ = escalated)';
COMMENT ON COLUMN escalation_events.action_taken IS 'JSON data describing the action taken during escalation';
COMMENT ON COLUMN escalation_events.action_result IS 'JSON data describing the result of the escalation action';