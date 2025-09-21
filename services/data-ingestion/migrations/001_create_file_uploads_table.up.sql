-- Migration: 001_create_file_uploads_table
-- Description: Create file uploads table for tracking uploaded files
-- Up Migration

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS file_uploads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    content_type VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    storage_path TEXT,
    uploaded_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_file_uploads_status ON file_uploads(status);
CREATE INDEX IF NOT EXISTS idx_file_uploads_created_at ON file_uploads(created_at);
CREATE INDEX IF NOT EXISTS idx_file_uploads_uploaded_at ON file_uploads(uploaded_at);
CREATE INDEX IF NOT EXISTS idx_file_uploads_file_name ON file_uploads(file_name);

-- GIN index for JSONB metadata queries
CREATE INDEX IF NOT EXISTS idx_file_uploads_metadata ON file_uploads USING GIN(metadata);

-- Check constraints
ALTER TABLE file_uploads 
ADD CONSTRAINT chk_file_uploads_status 
CHECK (status IN ('pending', 'uploading', 'uploaded', 'processing', 'processed', 'failed', 'deleted'));

ALTER TABLE file_uploads 
ADD CONSTRAINT chk_file_uploads_file_size 
CHECK (file_size >= 0);

-- Comments
COMMENT ON TABLE file_uploads IS 'Tracks uploaded files and their metadata';
COMMENT ON COLUMN file_uploads.id IS 'Unique identifier for the file upload';
COMMENT ON COLUMN file_uploads.file_name IS 'Original filename';
COMMENT ON COLUMN file_uploads.file_size IS 'File size in bytes';
COMMENT ON COLUMN file_uploads.content_type IS 'MIME content type';
COMMENT ON COLUMN file_uploads.status IS 'Current status of the file upload';
COMMENT ON COLUMN file_uploads.storage_path IS 'Path where file is stored';
COMMENT ON COLUMN file_uploads.uploaded_at IS 'When the file upload completed';
COMMENT ON COLUMN file_uploads.error_message IS 'Error message if upload failed';
COMMENT ON COLUMN file_uploads.metadata IS 'Additional metadata as JSON';