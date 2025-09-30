package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mit-service/internal/metrics"
	"mit-service/internal/models"
	"mit-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

// Service provides business logic for the application
type Service struct {
	repo    *repository.RepositoryManager
	worker  *InboxWorker
	metrics *metrics.Metrics
}

// NewService creates a new service instance
func NewService(repo *repository.RepositoryManager, metrics *metrics.Metrics) *Service {
	return &Service{
		repo:    repo,
		metrics: metrics,
	}
}

// StartInboxWorker starts the inbox pattern worker
func (s *Service) StartInboxWorker(workerCount int, batchSize int, pollInterval time.Duration, maxRetries int, retryDelay time.Duration) {
	s.worker = NewInboxWorker(s.repo, s.metrics, workerCount, batchSize, pollInterval, maxRetries, retryDelay)
	s.worker.Start()
}

// StopInboxWorker stops the inbox pattern worker
func (s *Service) StopInboxWorker() {
	if s.worker != nil {
		s.worker.Stop()
	}
}

// Insert creates a new record asynchronously using inbox pattern
func (s *Service) Insert(ctx context.Context, req *models.InsertRequest) error {
	payload, err := json.Marshal(&models.InsertTaskPayload{
		ID:    req.ID,
		Value: req.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal insert payload: %w", err)
	}

	task := &models.InboxTask{
		ID:        uuid.New().String(),
		Operation: models.TaskOperationInsert,
		Payload:   payload,
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Retries:   0,
	}

	if err := s.repo.Inbox.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create insert task: %w", err)
	}

	return nil
}

// Update modifies an existing record asynchronously using inbox pattern
func (s *Service) Update(ctx context.Context, req *models.UpdateRequest) error {
	payload, err := json.Marshal(&models.UpdateTaskPayload{
		ID:    req.ID,
		Value: req.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal update payload: %w", err)
	}

	task := &models.InboxTask{
		ID:        uuid.New().String(),
		Operation: models.TaskOperationUpdate,
		Payload:   payload,
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Retries:   0,
	}

	if err := s.repo.Inbox.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create update task: %w", err)
	}

	return nil
}

// Delete removes a record asynchronously using inbox pattern
func (s *Service) Delete(ctx context.Context, req *models.DeleteRequest) error {
	payload, err := json.Marshal(&models.DeleteTaskPayload{
		ID: req.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal delete payload: %w", err)
	}

	task := &models.InboxTask{
		ID:        uuid.New().String(),
		Operation: models.TaskOperationDelete,
		Payload:   payload,
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Retries:   0,
	}

	if err := s.repo.Inbox.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create delete task: %w", err)
	}

	return nil
}

// Get retrieves a record synchronously (read operations are not queued)
func (s *Service) Get(ctx context.Context, id string) (*models.Record, error) {
	record, err := s.repo.Record.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	return record, nil
}

// GetTasks retrieves tasks with optional filtering and pagination
func (s *Service) GetTasks(ctx context.Context, status string, limit, offset int) (*models.TasksListResponse, error) {
	// Set default values
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var tasks []*models.InboxTask
	var err error

	if status == "" {
		tasks, err = s.repo.Inbox.GetAllTasks(ctx, limit, offset)
	} else {
		tasks, err = s.repo.Inbox.GetTasksByStatus(ctx, status, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Get stats
	stats, err := s.repo.Inbox.GetTaskStats(ctx)
	if err != nil {
		log.Printf("Failed to get task stats: %v", err)
		// Don't fail the request if stats retrieval fails
	} else {
		// Update queue depth metrics
		s.metrics.SetQueueDepth(int64(stats.PendingTasks + stats.ProcessingTasks))
	}

	response := &models.TasksListResponse{
		Tasks:  tasks,
		Total:  len(tasks),
		Limit:  limit,
		Offset: offset,
		Stats:  stats,
	}

	return response, nil
}

// GetTaskStats retrieves statistics about inbox tasks
func (s *Service) GetTaskStats(ctx context.Context) (*models.TaskStats, error) {
	stats, err := s.repo.Inbox.GetTaskStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get task stats: %w", err)
	}
	return stats, nil
}

// Close closes the service and its dependencies
func (s *Service) Close() error {
	s.StopInboxWorker()
	return nil
}
