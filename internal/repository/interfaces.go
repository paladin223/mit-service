package repository

import (
	"context"
	"mit-service/internal/models"
)

// RecordRepository defines the interface for record operations
type RecordRepository interface {
	// Insert creates a new record
	Insert(ctx context.Context, record *models.Record) error

	// Update modifies an existing record
	Update(ctx context.Context, record *models.Record) error

	// Delete removes a record by ID
	Delete(ctx context.Context, id string) error

	// Get retrieves a record by ID
	Get(ctx context.Context, id string) (*models.Record, error)

	// Close closes the repository connection
	Close() error
}

// InboxRepository defines the interface for inbox pattern operations
type InboxRepository interface {
	// CreateTask creates a new task in the inbox
	CreateTask(ctx context.Context, task *models.InboxTask) error

	// GetPendingTasks retrieves pending tasks from the inbox
	GetPendingTasks(ctx context.Context, limit int) ([]*models.InboxTask, error)

	// GetTasksByStatus retrieves tasks by status with pagination
	GetTasksByStatus(ctx context.Context, status string, limit, offset int) ([]*models.InboxTask, error)

	// GetAllTasks retrieves all tasks with pagination
	GetAllTasks(ctx context.Context, limit, offset int) ([]*models.InboxTask, error)

	// GetTaskStats returns statistics about tasks by status
	GetTaskStats(ctx context.Context) (*models.TaskStats, error)

	// UpdateTaskStatus updates the status of a task
	UpdateTaskStatus(ctx context.Context, taskID string, status string, errorMsg string) error

	// IncrementTaskRetries increments the retry count for a task
	IncrementTaskRetries(ctx context.Context, taskID string) error

	// DeleteCompletedTasks removes completed tasks older than specified duration
	DeleteCompletedTasks(ctx context.Context, olderThanHours int) error

	// Close closes the repository connection
	Close() error
}

// Repository combines all repository interfaces
type Repository interface {
	RecordRepository
	InboxRepository
}

// RepositoryManager provides access to all repositories
type RepositoryManager struct {
	Record RecordRepository
	Inbox  InboxRepository
}
