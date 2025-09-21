-- Migration: 001_create_file_uploads_table
-- Description: Drop file uploads table
-- Down Migration

DROP INDEX IF EXISTS idx_file_uploads_metadata;
DROP INDEX IF EXISTS idx_file_uploads_file_name;
DROP INDEX IF EXISTS idx_file_uploads_uploaded_at;
DROP INDEX IF EXISTS idx_file_uploads_created_at;
DROP INDEX IF EXISTS idx_file_uploads_status;

DROP TABLE IF EXISTS file_uploads;