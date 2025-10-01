-- Create inbox_tasks table for the inbox pattern
CREATE TABLE IF NOT EXISTS inbox_tasks (
    id VARCHAR(255) PRIMARY KEY,
    operation VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    retries INTEGER DEFAULT 0,
    error TEXT
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_inbox_tasks_status ON inbox_tasks(status);
CREATE INDEX IF NOT EXISTS idx_inbox_tasks_created_at ON inbox_tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_inbox_tasks_status_created_at ON inbox_tasks(status, created_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

-- Create trigger for automatic updated_at updates on inbox_tasks
CREATE TRIGGER update_inbox_tasks_updated_at 
    BEFORE UPDATE ON inbox_tasks 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
