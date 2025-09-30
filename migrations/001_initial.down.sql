-- Drop triggers
DROP TRIGGER IF EXISTS update_inbox_tasks_updated_at ON inbox_tasks;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_inbox_tasks_status;
DROP INDEX IF EXISTS idx_inbox_tasks_created_at;
DROP INDEX IF EXISTS idx_inbox_tasks_status_created_at;

-- Drop tables
DROP TABLE IF EXISTS inbox_tasks;
DROP TABLE IF EXISTS records;
