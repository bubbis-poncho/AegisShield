-- Create alert_rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    expression TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL,
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    type VARCHAR(50) NOT NULL DEFAULT 'threshold',
    category VARCHAR(100),
    
    enabled BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1,
    
    -- Rule configuration
    conditions JSONB NOT NULL,
    actions JSONB,
    throttle_duration INTERVAL,
    escalation_policy_id VARCHAR(255),
    
    -- Template settings
    title_template TEXT,
    description_template TEXT,
    
    -- Metadata
    metadata JSONB,
    tags TEXT[],
    
    -- Audit fields
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Statistics
    last_evaluation_at TIMESTAMP WITH TIME ZONE,
    evaluation_count BIGINT DEFAULT 0,
    match_count BIGINT DEFAULT 0,
    last_match_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT alert_rules_severity_check CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT alert_rules_priority_check CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    CONSTRAINT alert_rules_type_check CHECK (type IN ('threshold', 'anomaly', 'pattern', 'custom')),
    CONSTRAINT alert_rules_name_unique UNIQUE (name),
    CONSTRAINT alert_rules_version_positive CHECK (version > 0)
);

-- Create indexes for alert_rules table
CREATE INDEX IF NOT EXISTS idx_alert_rules_name ON alert_rules(name);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_alert_rules_severity ON alert_rules(severity);
CREATE INDEX IF NOT EXISTS idx_alert_rules_type ON alert_rules(type);
CREATE INDEX IF NOT EXISTS idx_alert_rules_category ON alert_rules(category);
CREATE INDEX IF NOT EXISTS idx_alert_rules_created_at ON alert_rules(created_at);
CREATE INDEX IF NOT EXISTS idx_alert_rules_updated_at ON alert_rules(updated_at);
CREATE INDEX IF NOT EXISTS idx_alert_rules_last_evaluation ON alert_rules(last_evaluation_at);
CREATE INDEX IF NOT EXISTS idx_alert_rules_created_by ON alert_rules(created_by);

-- Create composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled_type ON alert_rules(enabled, type);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled_severity ON alert_rules(enabled, severity);
CREATE INDEX IF NOT EXISTS idx_alert_rules_category_enabled ON alert_rules(category, enabled);

-- Create GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_alert_rules_conditions ON alert_rules USING GIN (conditions);
CREATE INDEX IF NOT EXISTS idx_alert_rules_actions ON alert_rules USING GIN (actions);
CREATE INDEX IF NOT EXISTS idx_alert_rules_metadata ON alert_rules USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_alert_rules_tags ON alert_rules USING GIN (tags);

-- Create trigger for alert_rules
CREATE TRIGGER update_alert_rules_updated_at 
    BEFORE UPDATE ON alert_rules 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Add table comment
COMMENT ON TABLE alert_rules IS 'Stores alert rules for the alerting engine';
COMMENT ON COLUMN alert_rules.id IS 'Unique identifier for the rule';
COMMENT ON COLUMN alert_rules.expression IS 'Rule expression for evaluation';
COMMENT ON COLUMN alert_rules.conditions IS 'JSON configuration for rule conditions';
COMMENT ON COLUMN alert_rules.actions IS 'JSON configuration for rule actions';
COMMENT ON COLUMN alert_rules.throttle_duration IS 'Minimum time between alert generations';
COMMENT ON COLUMN alert_rules.evaluation_count IS 'Total number of times this rule has been evaluated';
COMMENT ON COLUMN alert_rules.match_count IS 'Total number of times this rule has matched';