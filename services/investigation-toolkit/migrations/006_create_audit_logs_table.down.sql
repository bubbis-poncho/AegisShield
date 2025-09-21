-- Drop audit logs table and related functions
DROP FUNCTION IF EXISTS cleanup_old_audit_logs(INTEGER);
DROP TABLE IF EXISTS audit_logs;