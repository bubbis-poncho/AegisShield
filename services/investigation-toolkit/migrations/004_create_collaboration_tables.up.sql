-- Create collaboration table for team collaboration on investigations
CREATE TABLE IF NOT EXISTS collaboration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investigation_id UUID NOT NULL REFERENCES investigations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('lead_investigator', 'investigator', 'analyst', 'reviewer', 'observer', 'consultant')),
    permissions JSONB DEFAULT '{}',
    assigned_by UUID NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    removed_at TIMESTAMP WITH TIME ZONE,
    removed_by UUID,
    is_active BOOLEAN DEFAULT TRUE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT collaboration_unique_active_user_investigation UNIQUE (investigation_id, user_id, is_active),
    CONSTRAINT collaboration_removed_consistency CHECK (
        (removed_at IS NULL AND removed_by IS NULL AND is_active = TRUE) OR
        (removed_at IS NOT NULL AND removed_by IS NOT NULL AND is_active = FALSE)
    )
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_collaboration_investigation_id ON collaboration(investigation_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_user_id ON collaboration(user_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_role ON collaboration(role);
CREATE INDEX IF NOT EXISTS idx_collaboration_assigned_by ON collaboration(assigned_by);
CREATE INDEX IF NOT EXISTS idx_collaboration_assigned_at ON collaboration(assigned_at);
CREATE INDEX IF NOT EXISTS idx_collaboration_is_active ON collaboration(is_active);
CREATE INDEX IF NOT EXISTS idx_collaboration_permissions ON collaboration USING GIN(permissions);

-- Create collaboration_comments table for discussion threads
CREATE TABLE IF NOT EXISTS collaboration_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investigation_id UUID NOT NULL REFERENCES investigations(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES collaboration_comments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    comment_type VARCHAR(50) NOT NULL DEFAULT 'general' CHECK (comment_type IN ('general', 'question', 'finding', 'recommendation', 'status_update', 'evidence_comment')),
    mentioned_users UUID[],
    attachments JSONB DEFAULT '[]',
    is_internal BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT collaboration_comments_content_not_empty CHECK (LENGTH(TRIM(content)) > 0)
);

-- Create indexes for collaboration_comments
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_investigation_id ON collaboration_comments(investigation_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_parent_id ON collaboration_comments(parent_comment_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_user_id ON collaboration_comments(user_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_comment_type ON collaboration_comments(comment_type);
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_created_at ON collaboration_comments(created_at);
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_mentioned_users ON collaboration_comments USING GIN(mentioned_users);
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_attachments ON collaboration_comments USING GIN(attachments);
CREATE INDEX IF NOT EXISTS idx_collaboration_comments_deleted_at ON collaboration_comments(deleted_at);

-- Create triggers to update updated_at timestamp
CREATE TRIGGER update_collaboration_updated_at
    BEFORE UPDATE ON collaboration
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_collaboration_comments_updated_at
    BEFORE UPDATE ON collaboration_comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();