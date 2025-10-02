package models

import (
	"encoding/json"
	"errors"
	"time"
)

// Record represents a database record with id and JSON value
type Record struct {
	ID    string      `json:"id" db:"id"`
	Value interface{} `json:"value" db:"value"`
}

// InsertRequest represents the request payload for insert operation
type InsertRequest struct {
	ID    string                 `json:"id" binding:"required,min=1"`
	Value map[string]interface{} `json:"value" binding:"required"`
}

// UpdateRequest represents the request payload for update operation
type UpdateRequest struct {
	ID    string                 `json:"id" binding:"required,min=1"`
	Value map[string]interface{} `json:"value" binding:"required"`
}

// DeleteRequest represents the request payload for delete operation
type DeleteRequest struct {
	ID string `json:"id" binding:"required,min=1"`
}

// SuccessResponse represents a successful operation response
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// InboxTask represents a task in the inbox pattern for write operations
type InboxTask struct {
	ID        string          `json:"id" db:"id"`
	Operation string          `json:"operation" db:"operation"` // "insert", "update", "delete"
	Payload   json.RawMessage `json:"payload" db:"payload"`
	Status    string          `json:"status" db:"status"` // "pending", "processing", "completed", "failed"
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
	Retries   int             `json:"retries" db:"retries"`
	Error     string          `json:"error,omitempty" db:"error"`
}

// TaskStatus constants
const (
	TaskStatusPending    = "pending"
	TaskStatusProcessing = "processing"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
)

// TaskOperation constants
const (
	TaskOperationInsert = "insert"
	TaskOperationUpdate = "update"
	TaskOperationDelete = "delete"
)

// InsertTaskPayload represents the payload for insert task
type InsertTaskPayload struct {
	ID    string                 `json:"id"`
	Value map[string]interface{} `json:"value"`
}

// UpdateTaskPayload represents the payload for update task
type UpdateTaskPayload struct {
	ID    string                 `json:"id"`
	Value map[string]interface{} `json:"value"`
}

// DeleteTaskPayload represents the payload for delete task
type DeleteTaskPayload struct {
	ID string `json:"id"`
}

// TaskStats represents statistics about inbox tasks (deprecated - keeping for compatibility)
type TaskStats struct {
	TotalTasks      int `json:"total_tasks"`
	PendingTasks    int `json:"pending_tasks"`
	ProcessingTasks int `json:"processing_tasks"`
	CompletedTasks  int `json:"completed_tasks"`
	FailedTasks     int `json:"failed_tasks"`
}

// SimpleStats represents simple service statistics without inbox
type SimpleStats struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// TasksListResponse represents the response for tasks list
type TasksListResponse struct {
	Tasks  []*InboxTask `json:"tasks"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
	Stats  *TaskStats   `json:"stats,omitempty"`
}

// Common errors
var (
	ErrInvalidTaskOperation = errors.New("invalid task operation")
)
