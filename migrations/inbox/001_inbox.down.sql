-- Drop inbox database objects
DROP TRIGGER IF EXISTS update_inbox_tasks_updated_at ON inbox_tasks;
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;
DROP TABLE IF EXISTS inbox_tasks CASCADE;
