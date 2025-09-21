-- Migration: Create widgets table
-- Version: 013

CREATE TABLE IF NOT EXISTS widgets (
    id VARCHAR(36) PRIMARY KEY,
    dashboard_id VARCHAR(36) NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    position JSONB NOT NULL DEFAULT '{}',
    size JSONB NOT NULL DEFAULT '{}',
    config JSONB NOT NULL DEFAULT '{}',
    data_source JSONB NOT NULL DEFAULT '{}',
    refresh_rate INTEGER DEFAULT 30,
    is_visible BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (dashboard_id) REFERENCES dashboards(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_widgets_dashboard_id ON widgets(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_widgets_type ON widgets(type);
CREATE INDEX IF NOT EXISTS idx_widgets_created_at ON widgets(created_at);
CREATE INDEX IF NOT EXISTS idx_widgets_is_visible ON widgets(is_visible) WHERE is_visible = TRUE;

-- Add updated_at trigger
CREATE TRIGGER update_widgets_updated_at 
    BEFORE UPDATE ON widgets 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();