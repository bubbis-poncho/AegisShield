-- Drop workflow tables
DROP TRIGGER IF EXISTS update_workflow_steps_updated_at ON workflow_steps;
DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
DROP TABLE IF EXISTS workflow_step_history;
DROP TABLE IF EXISTS workflow_steps;
DROP TABLE IF EXISTS workflows;