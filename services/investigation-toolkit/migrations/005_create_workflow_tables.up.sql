-- Create workflows table for investigation workflow definitions and instances
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    workflow_type VARCHAR(50) NOT NULL CHECK (workflow_type IN ('template', 'instance')),
    template_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    investigation_id UUID REFERENCES investigations(id) ON DELETE CASCADE,
    definition JSONB NOT NULL,
    current_step VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'suspended', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    variables JSONB DEFAULT '{}',
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT workflows_template_consistency CHECK (
        (workflow_type = 'template' AND investigation_id IS NULL) OR
        (workflow_type = 'instance' AND investigation_id IS NOT NULL)
    ),
    CONSTRAINT workflows_completion_consistency CHECK (
        (status IN ('completed', 'failed', 'cancelled') AND completed_at IS NOT NULL) OR
        (status NOT IN ('completed', 'failed', 'cancelled') AND completed_at IS NULL)
    )
);

-- Create workflow_steps table for tracking individual workflow step executions
CREATE TABLE IF NOT EXISTS workflow_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    step_name VARCHAR(100) NOT NULL,
    step_type VARCHAR(50) NOT NULL CHECK (step_type IN ('manual', 'automated', 'approval', 'decision', 'notification', 'data_collection')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'skipped', 'cancelled')),
    assigned_to UUID,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    input_data JSONB DEFAULT '{}',
    output_data JSONB DEFAULT '{}',
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT workflow_steps_retry_consistency CHECK (retry_count <= max_retries),
    CONSTRAINT workflow_steps_completion_consistency CHECK (
        (status IN ('completed', 'failed', 'skipped', 'cancelled') AND completed_at IS NOT NULL) OR
        (status NOT IN ('completed', 'failed', 'skipped', 'cancelled') AND completed_at IS NULL)
    )
);

-- Create workflow_step_history table for audit trail of step changes
CREATE TABLE IF NOT EXISTS workflow_step_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_step_id UUID NOT NULL REFERENCES workflow_steps(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL CHECK (action IN ('created', 'started', 'completed', 'failed', 'assigned', 'reassigned', 'cancelled', 'retry')),
    previous_status VARCHAR(20),
    new_status VARCHAR(20),
    performed_by UUID NOT NULL,
    reason TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_workflows_workflow_type ON workflows(workflow_type);
CREATE INDEX IF NOT EXISTS idx_workflows_template_id ON workflows(template_id);
CREATE INDEX IF NOT EXISTS idx_workflows_investigation_id ON workflows(investigation_id);
CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
CREATE INDEX IF NOT EXISTS idx_workflows_created_by ON workflows(created_by);
CREATE INDEX IF NOT EXISTS idx_workflows_created_at ON workflows(created_at);
CREATE INDEX IF NOT EXISTS idx_workflows_definition ON workflows USING GIN(definition);
CREATE INDEX IF NOT EXISTS idx_workflows_variables ON workflows USING GIN(variables);

CREATE INDEX IF NOT EXISTS idx_workflow_steps_workflow_id ON workflow_steps(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_step_name ON workflow_steps(step_name);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_step_type ON workflow_steps(step_type);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_status ON workflow_steps(status);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_assigned_to ON workflow_steps(assigned_to);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_due_date ON workflow_steps(due_date);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_created_at ON workflow_steps(created_at);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_input_data ON workflow_steps USING GIN(input_data);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_output_data ON workflow_steps USING GIN(output_data);

CREATE INDEX IF NOT EXISTS idx_workflow_step_history_workflow_step_id ON workflow_step_history(workflow_step_id);
CREATE INDEX IF NOT EXISTS idx_workflow_step_history_action ON workflow_step_history(action);
CREATE INDEX IF NOT EXISTS idx_workflow_step_history_performed_by ON workflow_step_history(performed_by);
CREATE INDEX IF NOT EXISTS idx_workflow_step_history_created_at ON workflow_step_history(created_at);

-- Create triggers to update updated_at timestamp
CREATE TRIGGER update_workflows_updated_at
    BEFORE UPDATE ON workflows
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_steps_updated_at
    BEFORE UPDATE ON workflow_steps
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();