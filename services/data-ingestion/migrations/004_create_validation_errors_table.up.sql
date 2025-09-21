-- Migration: 004_create_validation_errors_table
-- Description: Create validation errors table for tracking data validation issues
-- Up Migration

CREATE TABLE IF NOT EXISTS validation_errors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID REFERENCES transactions(id) ON DELETE CASCADE,
    error_type VARCHAR(50) NOT NULL,
    error_message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL,
    field VARCHAR(100),
    validated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    errors JSONB,
    warnings JSONB,
    quality_score DECIMAL(3,2) DEFAULT 0.00,
    is_valid BOOLEAN DEFAULT TRUE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_validation_errors_transaction_id ON validation_errors(transaction_id);
CREATE INDEX IF NOT EXISTS idx_validation_errors_error_type ON validation_errors(error_type);
CREATE INDEX IF NOT EXISTS idx_validation_errors_severity ON validation_errors(severity);
CREATE INDEX IF NOT EXISTS idx_validation_errors_field ON validation_errors(field);
CREATE INDEX IF NOT EXISTS idx_validation_errors_validated_at ON validation_errors(validated_at);
CREATE INDEX IF NOT EXISTS idx_validation_errors_quality_score ON validation_errors(quality_score);
CREATE INDEX IF NOT EXISTS idx_validation_errors_is_valid ON validation_errors(is_valid);
CREATE INDEX IF NOT EXISTS idx_validation_errors_created_at ON validation_errors(created_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_validation_errors_severity_created ON validation_errors(severity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_validation_errors_type_severity ON validation_errors(error_type, severity);

-- GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_validation_errors_errors ON validation_errors USING GIN(errors);
CREATE INDEX IF NOT EXISTS idx_validation_errors_warnings ON validation_errors USING GIN(warnings);
CREATE INDEX IF NOT EXISTS idx_validation_errors_metadata ON validation_errors USING GIN(metadata);

-- Check constraints
ALTER TABLE validation_errors 
ADD CONSTRAINT chk_validation_errors_severity 
CHECK (severity IN ('INFO', 'WARNING', 'ERROR', 'CRITICAL'));

ALTER TABLE validation_errors 
ADD CONSTRAINT chk_validation_errors_error_type 
CHECK (error_type IN ('VALIDATION', 'BUSINESS_RULE', 'DATA_QUALITY', 'SCHEMA', 'FORMAT', 'COMPLIANCE'));

ALTER TABLE validation_errors 
ADD CONSTRAINT chk_validation_errors_quality_score 
CHECK (quality_score >= 0.00 AND quality_score <= 1.00);

-- Comments
COMMENT ON TABLE validation_errors IS 'Stores validation errors and warnings for transactions';
COMMENT ON COLUMN validation_errors.id IS 'Unique identifier for the validation record';
COMMENT ON COLUMN validation_errors.transaction_id IS 'Reference to the transaction being validated';
COMMENT ON COLUMN validation_errors.error_type IS 'Type of validation error';
COMMENT ON COLUMN validation_errors.error_message IS 'Human-readable error message';
COMMENT ON COLUMN validation_errors.severity IS 'Severity level of the validation issue';
COMMENT ON COLUMN validation_errors.field IS 'Field that failed validation (if applicable)';
COMMENT ON COLUMN validation_errors.validated_at IS 'When the validation was performed';
COMMENT ON COLUMN validation_errors.errors IS 'Detailed error information as JSON';
COMMENT ON COLUMN validation_errors.warnings IS 'Warning information as JSON';
COMMENT ON COLUMN validation_errors.quality_score IS 'Overall data quality score (0.00 to 1.00)';
COMMENT ON COLUMN validation_errors.is_valid IS 'Whether the data passed validation';
COMMENT ON COLUMN validation_errors.metadata IS 'Additional validation metadata as JSON';