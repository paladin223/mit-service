package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mit-service/internal/metrics"
	"mit-service/internal/models"
	"mit-service/internal/repository"
	"sync"
	"time"
)

// InboxWorker processes tasks from the inbox using worker pattern
type InboxWorker struct {
	repo         *repository.RepositoryManager
	metrics      *metrics.Metrics
	workerCount  int
	batchSize    int
	pollInterval time.Duration
	maxRetries   int
	retryDelay   time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.RWMutex
}

// NewInboxWorker creates a new inbox worker
func NewInboxWorker(
	repo *repository.RepositoryManager,
	metrics *metrics.Metrics,
	workerCount int,
	batchSize int,
	pollInterval time.Duration,
	maxRetries int,
	retryDelay time.Duration,
) *InboxWorker {
	return &InboxWorker{
		repo:         repo,
		metrics:      metrics,
		workerCount:  workerCount,
		batchSize:    batchSize,
		pollInterval: pollInterval,
		maxRetries:   maxRetries,
		retryDelay:   retryDelay,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the inbox worker
func (w *InboxWorker) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return
	}

	w.running = true
	log.Printf("Starting inbox worker with %d workers", w.workerCount)

	// Start worker goroutines
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}

	// Start cleanup goroutine
	w.wg.Add(1)
	go w.cleanupWorker()
}

// Stop stops the inbox worker
func (w *InboxWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	log.Println("Stopping inbox worker...")
	w.running = false
	close(w.stopCh)
	w.wg.Wait()
	log.Println("Inbox worker stopped")
}

// worker processes tasks from the inbox
func (w *InboxWorker) worker(workerID int) {
	defer w.wg.Done()
	log.Printf("Worker %d started", workerID)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			log.Printf("Worker %d stopping", workerID)
			return
		case <-ticker.C:
			w.processTasks(workerID)
		}
	}
}

// processTasks retrieves and processes pending tasks
func (w *InboxWorker) processTasks(workerID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tasks, err := w.repo.Inbox.GetPendingTasks(ctx, w.batchSize)
	if err != nil {
		log.Printf("Worker %d: failed to get pending tasks: %v", workerID, err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	log.Printf("Worker %d: processing %d tasks", workerID, len(tasks))

	// Update queue depth metrics
	w.metrics.SetQueueDepth(int64(len(tasks)))

	// Log task details
	for _, task := range tasks {
		log.Printf("Worker %d: found task %s (operation: %s, status: %s, retries: %d, age: %v)",
			workerID, task.ID, task.Operation, task.Status, task.Retries,
			time.Since(task.CreatedAt).Round(time.Second))
	}

	for _, task := range tasks {
		w.processTask(ctx, workerID, task)
	}
}

// processTask processes a single task
func (w *InboxWorker) processTask(ctx context.Context, workerID int, task *models.InboxTask) {
	startTime := time.Now()
	log.Printf("Worker %d: starting processing task %s (operation: %s)", workerID, task.ID, task.Operation)

	// Task is already marked as processing by GetPendingTasks
	var processErr error

	// Process task based on operation
	switch task.Operation {
	case models.TaskOperationInsert:
		processErr = w.processInsertTask(ctx, task.Payload)
	case models.TaskOperationUpdate:
		processErr = w.processUpdateTask(ctx, task.Payload)
	case models.TaskOperationDelete:
		processErr = w.processDeleteTask(ctx, task.Payload)
	default:
		processErr = models.ErrInvalidTaskOperation
	}

	if processErr != nil {
		w.handleTaskError(ctx, workerID, task, processErr)
		// Record failed task metrics with operation details
		duration := time.Since(startTime)
		w.metrics.RecordTaskExecutionWithDetails(string(task.Operation), duration, false)
		return
	}

	// Mark task as completed
	updateErr := w.repo.Inbox.UpdateTaskStatus(ctx, task.ID, models.TaskStatusCompleted, "")
	if updateErr != nil {
		log.Printf("Worker %d: failed to update task %s status to completed: %v", workerID, task.ID, updateErr)
	} else {
		duration := time.Since(startTime)
		log.Printf("Worker %d: task %s completed successfully in %v", workerID, task.ID, duration.Round(time.Millisecond))
		// Record successful task metrics with operation details
		w.metrics.RecordTaskExecutionWithDetails(string(task.Operation), duration, true)
	}
}

// handleTaskError handles task processing errors
func (w *InboxWorker) handleTaskError(ctx context.Context, workerID int, task *models.InboxTask, processErr error) {
	log.Printf("Worker %d: task %s failed: %v", workerID, task.ID, processErr)

	// Increment retry count
	err := w.repo.Inbox.IncrementTaskRetries(ctx, task.ID)
	if err != nil {
		log.Printf("Worker %d: failed to increment retries for task %s: %v", workerID, task.ID, err)
	}

	// Check if max retries exceeded
	if task.Retries >= w.maxRetries {
		log.Printf("Worker %d: task %s exceeded max retries (%d), marking as failed", workerID, task.ID, w.maxRetries)
		err = w.repo.Inbox.UpdateTaskStatus(ctx, task.ID, models.TaskStatusFailed, processErr.Error())
		if err != nil {
			log.Printf("Worker %d: failed to update task %s status to failed: %v", workerID, task.ID, err)
		}
	} else {
		// Schedule retry by marking as pending again after delay
		go func() {
			time.Sleep(w.retryDelay)
			retryCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := w.repo.Inbox.UpdateTaskStatus(retryCtx, task.ID, models.TaskStatusPending, processErr.Error())
			if err != nil {
				log.Printf("Worker %d: failed to reschedule task %s for retry: %v", workerID, task.ID, err)
			} else {
				log.Printf("Worker %d: task %s scheduled for retry (attempt %d)", workerID, task.ID, task.Retries+2)
			}
		}()
	}
}

// cleanupWorker periodically cleans up completed and failed tasks
func (w *InboxWorker) cleanupWorker() {
	defer w.wg.Done()
	log.Println("Cleanup worker started")

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			log.Println("Cleanup worker stopping")
			return
		case <-ticker.C:
			w.cleanupOldTasks()
		}
	}
}

// processInsertTask processes an insert task
func (w *InboxWorker) processInsertTask(ctx context.Context, payload []byte) error {
	var taskPayload models.InsertTaskPayload
	if err := json.Unmarshal(payload, &taskPayload); err != nil {
		return fmt.Errorf("failed to unmarshal insert payload: %w", err)
	}

	record := &models.Record{
		ID:    taskPayload.ID,
		Value: taskPayload.Value,
	}

	if err := w.repo.Record.Insert(ctx, record); err != nil {
		// Check if error is due to duplicate key (idempotency check)
		if err.Error() == fmt.Sprintf("record with id '%s' already exists", record.ID) {
			// Record already exists, check if it has the same value (idempotent operation)
			existingRecord, getErr := w.repo.Record.Get(ctx, record.ID)
			if getErr != nil {
				return fmt.Errorf("failed to verify existing record: %w", getErr)
			}
			
			// Compare values to ensure idempotency
			existingValueJSON, _ := json.Marshal(existingRecord.Value)
			newValueJSON, _ := json.Marshal(record.Value)
			if string(existingValueJSON) == string(newValueJSON) {
				log.Printf("Record with ID %s already exists with same value (idempotent operation)", record.ID)
				return nil // Success - idempotent operation
			}
			
			// Values are different - this is a conflict
			return fmt.Errorf("record with id '%s' already exists but with different value", record.ID)
		}
		return fmt.Errorf("failed to insert record: %w", err)
	}

	log.Printf("Successfully inserted record with ID: %s", record.ID)
	return nil
}

// processUpdateTask processes an update task
func (w *InboxWorker) processUpdateTask(ctx context.Context, payload []byte) error {
	var taskPayload models.UpdateTaskPayload
	if err := json.Unmarshal(payload, &taskPayload); err != nil {
		return fmt.Errorf("failed to unmarshal update payload: %w", err)
	}

	record := &models.Record{
		ID:    taskPayload.ID,
		Value: taskPayload.Value,
	}

	if err := w.repo.Record.Update(ctx, record); err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	log.Printf("Successfully updated record with ID: %s", record.ID)
	return nil
}

// processDeleteTask processes a delete task
func (w *InboxWorker) processDeleteTask(ctx context.Context, payload []byte) error {
	var taskPayload models.DeleteTaskPayload
	if err := json.Unmarshal(payload, &taskPayload); err != nil {
		return fmt.Errorf("failed to unmarshal delete payload: %w", err)
	}

	if err := w.repo.Record.Delete(ctx, taskPayload.ID); err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	log.Printf("Successfully deleted record with ID: %s", taskPayload.ID)
	return nil
}

// cleanupOldTasks removes completed and failed tasks older than 24 hours
func (w *InboxWorker) cleanupOldTasks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get stats before cleanup
	statsBefore, err := w.repo.Inbox.GetTaskStats(ctx)
	if err != nil {
		log.Printf("Cleanup worker: failed to get stats before cleanup: %v", err)
		statsBefore = &models.TaskStats{} // Use empty stats
	}

	log.Printf("Cleanup worker: before cleanup - total: %d, completed: %d, failed: %d",
		statsBefore.TotalTasks, statsBefore.CompletedTasks, statsBefore.FailedTasks)

	err = w.repo.Inbox.DeleteCompletedTasks(ctx, 24)
	if err != nil {
		log.Printf("Cleanup worker: failed to delete old tasks: %v", err)
		return
	}

	// Get stats after cleanup
	statsAfter, err := w.repo.Inbox.GetTaskStats(ctx)
	if err != nil {
		log.Printf("Cleanup worker: failed to get stats after cleanup: %v", err)
	} else {
		deletedCount := statsBefore.TotalTasks - statsAfter.TotalTasks
		log.Printf("Cleanup worker: successfully cleaned up %d old tasks. Current stats: total: %d, pending: %d, processing: %d",
			deletedCount, statsAfter.TotalTasks, statsAfter.PendingTasks, statsAfter.ProcessingTasks)
	}
}
