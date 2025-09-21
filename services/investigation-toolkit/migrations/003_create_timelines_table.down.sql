-- Drop timelines table
DROP TRIGGER IF EXISTS update_timelines_updated_at ON timelines;
DROP TABLE IF EXISTS timelines;