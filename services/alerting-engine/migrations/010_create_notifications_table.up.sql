-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(255) PRIMARY KEY,
    alert_id VARCHAR(255) NOT NULL,
    channel VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    subject TEXT,
    message TEXT NOT NULL,
    
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    
    -- Delivery details
    sent_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    
    -- Retry information
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    
    -- Error information
    error_message TEXT,
    error_code VARCHAR(100),
    
    -- Channel-specific data
    channel_data JSONB,
    
    -- Response tracking
    response_data JSONB,
    external_id VARCHAR(255),
    
    -- Metadata
    metadata JSONB,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT notifications_status_check CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'cancelled')),
    CONSTRAINT notifications_priority_check CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    CONSTRAINT notifications_channel_check CHECK (channel IN ('email', 'sms', 'slack', 'teams', 'webhook', 'pagerduty')),
    CONSTRAINT notifications_type_check CHECK (type IN ('alert', 'escalation', 'resolution', 'acknowledgment')),
    CONSTRAINT notifications_retry_count_positive CHECK (retry_count >= 0),
    CONSTRAINT notifications_max_retries_positive CHECK (max_retries >= 0),
    
    FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE
);

-- Create indexes for notifications table
CREATE INDEX IF NOT EXISTS idx_notifications_alert_id ON notifications(alert_id);
CREATE INDEX IF NOT EXISTS idx_notifications_channel ON notifications(channel);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications(priority);
CREATE INDEX IF NOT EXISTS idx_notifications_recipient ON notifications(recipient);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_sent_at ON notifications(sent_at);
CREATE INDEX IF NOT EXISTS idx_notifications_next_retry_at ON notifications(next_retry_at);
CREATE INDEX IF NOT EXISTS idx_notifications_external_id ON notifications(external_id);

-- Create composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_notifications_status_created ON notifications(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_channel_status ON notifications(channel, status);
CREATE INDEX IF NOT EXISTS idx_notifications_retry_status ON notifications(status, next_retry_at) 
    WHERE status = 'failed' AND next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_alert_channel ON notifications(alert_id, channel);

-- Create GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_notifications_channel_data ON notifications USING GIN (channel_data);
CREATE INDEX IF NOT EXISTS idx_notifications_response_data ON notifications USING GIN (response_data);
CREATE INDEX IF NOT EXISTS idx_notifications_metadata ON notifications USING GIN (metadata);

-- Create trigger for notifications
CREATE TRIGGER update_notifications_updated_at 
    BEFORE UPDATE ON notifications 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Add table comment
COMMENT ON TABLE notifications IS 'Stores notifications sent for alerts';
COMMENT ON COLUMN notifications.id IS 'Unique identifier for the notification';
COMMENT ON COLUMN notifications.alert_id IS 'ID of the alert this notification belongs to';
COMMENT ON COLUMN notifications.channel IS 'Notification channel (email, sms, slack, etc.)';
COMMENT ON COLUMN notifications.type IS 'Type of notification (alert, escalation, resolution, acknowledgment)';
COMMENT ON COLUMN notifications.status IS 'Current status of the notification';
COMMENT ON COLUMN notifications.channel_data IS 'Channel-specific configuration and data';
COMMENT ON COLUMN notifications.response_data IS 'Response data from the notification provider';
COMMENT ON COLUMN notifications.external_id IS 'External ID from the notification provider';
COMMENT ON COLUMN notifications.retry_count IS 'Number of retry attempts made';
COMMENT ON COLUMN notifications.next_retry_at IS 'Timestamp for the next retry attempt';