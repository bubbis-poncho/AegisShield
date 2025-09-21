-- Migration: 004_create_validation_errors_table
-- Description: Drop validation errors table
-- Down Migration

DROP INDEX IF EXISTS idx_validation_errors_metadata;
DROP INDEX IF EXISTS idx_validation_errors_warnings;
DROP INDEX IF EXISTS idx_validation_errors_errors;
DROP INDEX IF EXISTS idx_validation_errors_type_severity;
DROP INDEX IF EXISTS idx_validation_errors_severity_created;
DROP INDEX IF EXISTS idx_validation_errors_created_at;
DROP INDEX IF EXISTS idx_validation_errors_is_valid;
DROP INDEX IF EXISTS idx_validation_errors_quality_score;
DROP INDEX IF EXISTS idx_validation_errors_validated_at;
DROP INDEX IF EXISTS idx_validation_errors_field;
DROP INDEX IF EXISTS idx_validation_errors_severity;
DROP INDEX IF EXISTS idx_validation_errors_error_type;
DROP INDEX IF EXISTS idx_validation_errors_transaction_id;

DROP TABLE IF EXISTS validation_errors;