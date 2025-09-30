package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mit-service/internal/models"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// PostgresRepository implements Repository interface using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	repo := &PostgresRepository{db: db}

	// Initialize database schema
	if err := repo.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return repo, nil
}

// initSchema creates the required tables
func (r *PostgresRepository) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS records (
			id VARCHAR(255) PRIMARY KEY,
			value JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS inbox_tasks (
			id VARCHAR(255) PRIMARY KEY,
			operation VARCHAR(50) NOT NULL,
			payload JSONB NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			retries INTEGER DEFAULT 0,
			error TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_tasks_status ON inbox_tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_tasks_created_at ON inbox_tasks(created_at)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	return nil
}

// Record operations

// Insert creates a new record
func (r *PostgresRepository) Insert(ctx context.Context, record *models.Record) error {
	valueJSON, err := json.Marshal(record.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	query := `INSERT INTO records (id, value) VALUES ($1, $2)`
	_, err = r.db.ExecContext(ctx, query, record.ID, valueJSON)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("record with id '%s' already exists", record.ID)
		}
		return fmt.Errorf("failed to insert record: %w", err)
	}

	return nil
}

// Update modifies an existing record
func (r *PostgresRepository) Update(ctx context.Context, record *models.Record) error {
	valueJSON, err := json.Marshal(record.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	query := `UPDATE records SET value = $2, updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, record.ID, valueJSON)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("record with id '%s' not found", record.ID)
	}

	return nil
}

// Delete removes a record by ID
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM records WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("record with id '%s' not found", id)
	}

	return nil
}

// Get retrieves a record by ID
func (r *PostgresRepository) Get(ctx context.Context, id string) (*models.Record, error) {
	query := `SELECT id, value FROM records WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var record models.Record
	var valueJSON []byte

	err := row.Scan(&record.ID, &valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("record with id '%s' not found", id)
		}
		return nil, fmt.Errorf("failed to scan record: %w", err)
	}

	if err := json.Unmarshal(valueJSON, &record.Value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return &record, nil
}

// Inbox operations

// CreateTask creates a new task in the inbox
func (r *PostgresRepository) CreateTask(ctx context.Context, task *models.InboxTask) error {
	query := `INSERT INTO inbox_tasks (id, operation, payload, status, created_at, updated_at, retries) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.Operation, task.Payload, task.Status,
		task.CreatedAt, task.UpdatedAt, task.Retries)

	if err != nil {
		return fmt.Errorf("failed to create inbox task: %w", err)
	}

	return nil
}

// GetPendingTasks retrieves pending tasks from the inbox
func (r *PostgresRepository) GetPendingTasks(ctx context.Context, limit int) ([]*models.InboxTask, error) {
	query := `SELECT id, operation, payload, status, created_at, updated_at, retries, error
			  FROM inbox_tasks 
			  WHERE status = $1 
			  ORDER BY created_at ASC 
			  LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, models.TaskStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.InboxTask
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// GetTasksByStatus retrieves tasks by status with pagination
func (r *PostgresRepository) GetTasksByStatus(ctx context.Context, status string, limit, offset int) ([]*models.InboxTask, error) {
	query := `SELECT id, operation, payload, status, created_at, updated_at, retries, error
			  FROM inbox_tasks 
			  WHERE status = $1 
			  ORDER BY created_at DESC 
			  LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by status: %w", err)
	}
	defer rows.Close()

	var tasks []*models.InboxTask
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// GetAllTasks retrieves all tasks with pagination
func (r *PostgresRepository) GetAllTasks(ctx context.Context, limit, offset int) ([]*models.InboxTask, error) {
	query := `SELECT id, operation, payload, status, created_at, updated_at, retries, error
			  FROM inbox_tasks 
			  ORDER BY created_at DESC 
			  LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query all tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.InboxTask
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// GetTaskStats returns statistics about tasks by status
func (r *PostgresRepository) GetTaskStats(ctx context.Context) (*models.TaskStats, error) {
	query := `SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
				COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing,
				COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
				COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
			  FROM inbox_tasks`

	row := r.db.QueryRowContext(ctx, query)

	var stats models.TaskStats
	err := row.Scan(&stats.TotalTasks, &stats.PendingTasks, &stats.ProcessingTasks,
		&stats.CompletedTasks, &stats.FailedTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get task stats: %w", err)
	}

	return &stats, nil
}

// Helper function to scan task from rows
func (r *PostgresRepository) scanTask(scanner interface{}) (*models.InboxTask, error) {
	var task models.InboxTask
	var errorStr sql.NullString

	type Scanner interface {
		Scan(dest ...interface{}) error
	}

	s := scanner.(Scanner)
	err := s.Scan(&task.ID, &task.Operation, &task.Payload, &task.Status,
		&task.CreatedAt, &task.UpdatedAt, &task.Retries, &errorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}

	if errorStr.Valid {
		task.Error = errorStr.String
	}

	return &task, nil
}

// UpdateTaskStatus updates the status of a task
func (r *PostgresRepository) UpdateTaskStatus(ctx context.Context, taskID string, status string, errorMsg string) error {
	query := `UPDATE inbox_tasks 
			  SET status = $2, updated_at = NOW(), error = $3
			  WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, taskID, status, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// IncrementTaskRetries increments the retry count for a task
func (r *PostgresRepository) IncrementTaskRetries(ctx context.Context, taskID string) error {
	query := `UPDATE inbox_tasks 
			  SET retries = retries + 1, updated_at = NOW()
			  WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to increment task retries: %w", err)
	}

	return nil
}

// DeleteCompletedTasks removes completed tasks older than specified duration
func (r *PostgresRepository) DeleteCompletedTasks(ctx context.Context, olderThanHours int) error {
	query := `DELETE FROM inbox_tasks 
			  WHERE status IN ($1, $2) 
			  AND updated_at < NOW() - INTERVAL '%d hours'`

	_, err := r.db.ExecContext(ctx, fmt.Sprintf(query, olderThanHours),
		models.TaskStatusCompleted, models.TaskStatusFailed)
	if err != nil {
		return fmt.Errorf("failed to delete completed tasks: %w", err)
	}

	return nil
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}
