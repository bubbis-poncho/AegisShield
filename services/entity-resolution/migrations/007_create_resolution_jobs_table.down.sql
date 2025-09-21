-- Drop trigger
DROP TRIGGER IF EXISTS update_resolution_jobs_updated_at ON resolution_jobs;

-- Drop indexes
DROP INDEX IF EXISTS idx_resolution_jobs_metadata_gin;
DROP INDEX IF EXISTS idx_resolution_jobs_created_at;
DROP INDEX IF EXISTS idx_resolution_jobs_completed_at;
DROP INDEX IF EXISTS idx_resolution_jobs_started_at;
DROP INDEX IF EXISTS idx_resolution_jobs_status;

-- Drop table
DROP TABLE IF EXISTS resolution_jobs;