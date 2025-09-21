-- Migration: 002_create_data_jobs_table
-- Description: Drop data processing jobs table
-- Down Migration

DROP INDEX IF EXISTS idx_data_jobs_metadata;
DROP INDEX IF EXISTS idx_data_jobs_completed_at;
DROP INDEX IF EXISTS idx_data_jobs_started_at;
DROP INDEX IF EXISTS idx_data_jobs_created_at;
DROP INDEX IF EXISTS idx_data_jobs_job_type;
DROP INDEX IF EXISTS idx_data_jobs_status;
DROP INDEX IF EXISTS idx_data_jobs_file_id;

DROP TABLE IF EXISTS data_jobs;