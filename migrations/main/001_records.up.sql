-- Create main database for business records
CREATE TABLE IF NOT EXISTS records (
    id VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL
);

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_records_id ON records(id);
