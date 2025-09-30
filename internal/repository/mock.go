package repository

import (
	"context"
	"fmt"
	"mit-service/internal/models"
	"sync"
	"time"
)

// MockRepository implements Repository interface using in-memory storage
type MockRepository struct {
	records    map[string]*models.Record
	inboxTasks map[string]*models.InboxTask
	recordsMu  sync.RWMutex
	tasksMu    sync.RWMutex
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		records:    make(map[string]*models.Record),
		inboxTasks: make(map[string]*models.InboxTask),
	}
}

// Record operations

// Insert creates a new record
func (r *MockRepository) Insert(ctx context.Context, record *models.Record) error {
	r.recordsMu.Lock()
	defer r.recordsMu.Unlock()

	if _, exists := r.records[record.ID]; exists {
		return fmt.Errorf("record with id '%s' already exists", record.ID)
	}

	// Deep copy the record to avoid shared memory issues
	recordCopy := &models.Record{
		ID:    record.ID,
		Value: record.Value,
	}

	r.records[record.ID] = recordCopy
	return nil
}

// Update modifies an existing record
func (r *MockRepository) Update(ctx context.Context, record *models.Record) error {
	r.recordsMu.Lock()
	defer r.recordsMu.Unlock()

	if _, exists := r.records[record.ID]; !exists {
		return fmt.Errorf("record with id '%s' not found", record.ID)
	}

	// Deep copy the record to avoid shared memory issues
	recordCopy := &models.Record{
		ID:    record.ID,
		Value: record.Value,
	}

	r.records[record.ID] = recordCopy
	return nil
}

// Delete removes a record by ID
func (r *MockRepository) Delete(ctx context.Context, id string) error {
	r.recordsMu.Lock()
	defer r.recordsMu.Unlock()

	if _, exists := r.records[id]; !exists {
		return fmt.Errorf("record with id '%s' not found", id)
	}

	delete(r.records, id)
	return nil
}

// Get retrieves a record by ID
func (r *MockRepository) Get(ctx context.Context, id string) (*models.Record, error) {
	r.recordsMu.RLock()
	defer r.recordsMu.RUnlock()

	record, exists := r.records[id]
	if !exists {
		return nil, fmt.Errorf("record with id '%s' not found", id)
	}

	// Return a copy to avoid shared memory issues
	recordCopy := &models.Record{
		ID:    record.ID,
		Value: record.Value,
	}

	return recordCopy, nil
}

// Inbox operations

// CreateTask creates a new task in the inbox
func (r *MockRepository) CreateTask(ctx context.Context, task *models.InboxTask) error {
	r.tasksMu.Lock()
	defer r.tasksMu.Unlock()

	// Deep copy the task
	taskCopy := &models.InboxTask{
		ID:        task.ID,
		Operation: task.Operation,
		Payload:   make([]byte, len(task.Payload)),
		Status:    task.Status,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
		Retries:   task.Retries,
		Error:     task.Error,
	}
	copy(taskCopy.Payload, task.Payload)

	r.inboxTasks[task.ID] = taskCopy
	return nil
}

// GetPendingTasks retrieves pending tasks from the inbox
func (r *MockRepository) GetPendingTasks(ctx context.Context, limit int) ([]*models.InboxTask, error) {
	r.tasksMu.RLock()
	defer r.tasksMu.RUnlock()

	var pendingTasks []*models.InboxTask
	count := 0

	// Collect pending tasks (note: order is not guaranteed in map iteration)
	for _, task := range r.inboxTasks {
		if task.Status == models.TaskStatusPending && count < limit {
			// Return a copy to avoid shared memory issues
			taskCopy := &models.InboxTask{
				ID:        task.ID,
				Operation: task.Operation,
				Payload:   make([]byte, len(task.Payload)),
				Status:    task.Status,
				CreatedAt: task.CreatedAt,
				UpdatedAt: task.UpdatedAt,
				Retries:   task.Retries,
				Error:     task.Error,
			}
			copy(taskCopy.Payload, task.Payload)

			pendingTasks = append(pendingTasks, taskCopy)
			count++
		}
	}

	return pendingTasks, nil
}

// GetTasksByStatus retrieves tasks by status with pagination
func (r *MockRepository) GetTasksByStatus(ctx context.Context, status string, limit, offset int) ([]*models.InboxTask, error) {
	r.tasksMu.RLock()
	defer r.tasksMu.RUnlock()

	var filteredTasks []*models.InboxTask
	for _, task := range r.inboxTasks {
		if task.Status == status {
			// Return a copy to avoid shared memory issues
			taskCopy := r.copyTask(task)
			filteredTasks = append(filteredTasks, taskCopy)
		}
	}

	// Sort by created_at DESC (newest first)
	// Note: This is a simple implementation, in real scenarios you might want to use a more efficient sorting
	for i := 0; i < len(filteredTasks)-1; i++ {
		for j := i + 1; j < len(filteredTasks); j++ {
			if filteredTasks[i].CreatedAt.Before(filteredTasks[j].CreatedAt) {
				filteredTasks[i], filteredTasks[j] = filteredTasks[j], filteredTasks[i]
			}
		}
	}

	// Apply pagination
	start := offset
	if start >= len(filteredTasks) {
		return []*models.InboxTask{}, nil
	}

	end := start + limit
	if end > len(filteredTasks) {
		end = len(filteredTasks)
	}

	return filteredTasks[start:end], nil
}

// GetAllTasks retrieves all tasks with pagination
func (r *MockRepository) GetAllTasks(ctx context.Context, limit, offset int) ([]*models.InboxTask, error) {
	r.tasksMu.RLock()
	defer r.tasksMu.RUnlock()

	var allTasks []*models.InboxTask
	for _, task := range r.inboxTasks {
		taskCopy := r.copyTask(task)
		allTasks = append(allTasks, taskCopy)
	}

	// Sort by created_at DESC (newest first)
	for i := 0; i < len(allTasks)-1; i++ {
		for j := i + 1; j < len(allTasks); j++ {
			if allTasks[i].CreatedAt.Before(allTasks[j].CreatedAt) {
				allTasks[i], allTasks[j] = allTasks[j], allTasks[i]
			}
		}
	}

	// Apply pagination
	start := offset
	if start >= len(allTasks) {
		return []*models.InboxTask{}, nil
	}

	end := start + limit
	if end > len(allTasks) {
		end = len(allTasks)
	}

	return allTasks[start:end], nil
}

// GetTaskStats returns statistics about tasks by status
func (r *MockRepository) GetTaskStats(ctx context.Context) (*models.TaskStats, error) {
	r.tasksMu.RLock()
	defer r.tasksMu.RUnlock()

	stats := &models.TaskStats{
		TotalTasks: len(r.inboxTasks),
	}

	for _, task := range r.inboxTasks {
		switch task.Status {
		case models.TaskStatusPending:
			stats.PendingTasks++
		case models.TaskStatusProcessing:
			stats.ProcessingTasks++
		case models.TaskStatusCompleted:
			stats.CompletedTasks++
		case models.TaskStatusFailed:
			stats.FailedTasks++
		}
	}

	return stats, nil
}

// UpdateTaskStatus updates the status of a task
func (r *MockRepository) UpdateTaskStatus(ctx context.Context, taskID string, status string, errorMsg string) error {
	r.tasksMu.Lock()
	defer r.tasksMu.Unlock()

	task, exists := r.inboxTasks[taskID]
	if !exists {
		return fmt.Errorf("task with id '%s' not found", taskID)
	}

	task.Status = status
	task.UpdatedAt = time.Now()
	task.Error = errorMsg

	return nil
}

// IncrementTaskRetries increments the retry count for a task
func (r *MockRepository) IncrementTaskRetries(ctx context.Context, taskID string) error {
	r.tasksMu.Lock()
	defer r.tasksMu.Unlock()

	task, exists := r.inboxTasks[taskID]
	if !exists {
		return fmt.Errorf("task with id '%s' not found", taskID)
	}

	task.Retries++
	task.UpdatedAt = time.Now()

	return nil
}

// DeleteCompletedTasks removes completed tasks older than specified duration
func (r *MockRepository) DeleteCompletedTasks(ctx context.Context, olderThanHours int) error {
	r.tasksMu.Lock()
	defer r.tasksMu.Unlock()

	cutoffTime := time.Now().Add(-time.Duration(olderThanHours) * time.Hour)

	for id, task := range r.inboxTasks {
		if (task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusFailed) &&
			task.UpdatedAt.Before(cutoffTime) {
			delete(r.inboxTasks, id)
		}
	}

	return nil
}

// Close closes the repository (no-op for mock)
func (r *MockRepository) Close() error {
	return nil
}

// GetAllRecords returns all records (helper method for testing)
func (r *MockRepository) GetAllRecords() map[string]*models.Record {
	r.recordsMu.RLock()
	defer r.recordsMu.RUnlock()

	result := make(map[string]*models.Record)
	for id, record := range r.records {
		result[id] = &models.Record{
			ID:    record.ID,
			Value: record.Value,
		}
	}
	return result
}

// GetAllTasksForTesting returns all inbox tasks (helper method for testing)
func (r *MockRepository) GetAllTasksForTesting() map[string]*models.InboxTask {
	r.tasksMu.RLock()
	defer r.tasksMu.RUnlock()

	result := make(map[string]*models.InboxTask)
	for id, task := range r.inboxTasks {
		result[id] = r.copyTask(task)
	}
	return result
}

// copyTask creates a deep copy of a task
func (r *MockRepository) copyTask(task *models.InboxTask) *models.InboxTask {
	taskCopy := &models.InboxTask{
		ID:        task.ID,
		Operation: task.Operation,
		Payload:   make([]byte, len(task.Payload)),
		Status:    task.Status,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
		Retries:   task.Retries,
		Error:     task.Error,
	}
	copy(taskCopy.Payload, task.Payload)
	return taskCopy
}
