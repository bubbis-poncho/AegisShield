-- Drop collaboration tables
DROP TRIGGER IF EXISTS update_collaboration_comments_updated_at ON collaboration_comments;
DROP TRIGGER IF EXISTS update_collaboration_updated_at ON collaboration;
DROP TABLE IF EXISTS collaboration_comments;
DROP TABLE IF EXISTS collaboration;