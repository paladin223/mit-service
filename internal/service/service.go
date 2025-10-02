package service

import (
	"context"
	"fmt"
	"log"
	"mit-service/internal/metrics"
	"mit-service/internal/models"
	"mit-service/internal/repository"
	"time"
)

// Service provides business logic for the application
type Service struct {
	repo    *repository.RepositoryManager
	metrics *metrics.Metrics
}

// NewService creates a new service instance
func NewService(repo *repository.RepositoryManager, metrics *metrics.Metrics) *Service {
	return &Service{
		repo:    repo,
		metrics: metrics,
	}
}

// Insert creates a new record synchronously
func (s *Service) Insert(ctx context.Context, req *models.InsertRequest) error {
	startTime := time.Now()
	
	record := &models.Record{
		ID:    req.ID,
		Value: req.Value,
	}

	if err := s.repo.Record.Insert(ctx, record); err != nil {
		// Record execution metrics for failure
		duration := time.Since(startTime)
		s.metrics.RecordTaskExecutionWithDetails("insert", duration, false)
		return fmt.Errorf("failed to insert record: %w", err)
	}

	// Record successful execution metrics
	duration := time.Since(startTime)
	s.metrics.RecordTaskExecutionWithDetails("insert", duration, true)
	
	log.Printf("Successfully inserted record with ID: %s in %v", record.ID, duration.Round(time.Millisecond))
	return nil
}

// Update modifies an existing record synchronously
func (s *Service) Update(ctx context.Context, req *models.UpdateRequest) error {
	startTime := time.Now()
	
	record := &models.Record{
		ID:    req.ID,
		Value: req.Value,
	}

	if err := s.repo.Record.Update(ctx, record); err != nil {
		// Record execution metrics for failure
		duration := time.Since(startTime)
		s.metrics.RecordTaskExecutionWithDetails("update", duration, false)
		return fmt.Errorf("failed to update record: %w", err)
	}

	// Record successful execution metrics
	duration := time.Since(startTime)
	s.metrics.RecordTaskExecutionWithDetails("update", duration, true)
	
	log.Printf("Successfully updated record with ID: %s in %v", record.ID, duration.Round(time.Millisecond))
	return nil
}

// Delete removes a record synchronously
func (s *Service) Delete(ctx context.Context, req *models.DeleteRequest) error {
	startTime := time.Now()
	
	if err := s.repo.Record.Delete(ctx, req.ID); err != nil {
		// Record execution metrics for failure
		duration := time.Since(startTime)
		s.metrics.RecordTaskExecutionWithDetails("delete", duration, false)
		return fmt.Errorf("failed to delete record: %w", err)
	}

	// Record successful execution metrics
	duration := time.Since(startTime)
	s.metrics.RecordTaskExecutionWithDetails("delete", duration, true)
	
	log.Printf("Successfully deleted record with ID: %s in %v", req.ID, duration.Round(time.Millisecond))
	return nil
}

// Get retrieves a record by ID
func (s *Service) Get(ctx context.Context, id string) (*models.Record, error) {
	startTime := time.Now()
	
	record, err := s.repo.Record.Get(ctx, id)
	if err != nil {
		// Record execution metrics for failure
		duration := time.Since(startTime)
		s.metrics.RecordTaskExecutionWithDetails("get", duration, false)
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	// Record successful execution metrics
	duration := time.Since(startTime)
	s.metrics.RecordTaskExecutionWithDetails("get", duration, true)

	return record, nil
}

// GetStats returns simple operation statistics (no inbox stats)
func (s *Service) GetStats(ctx context.Context) (*models.SimpleStats, error) {
	// Since we don't have inbox anymore, return simple stats
	// You can extend this with actual database record counts if needed
	return &models.SimpleStats{
		Status:    "healthy",
		Timestamp: time.Now(),
	}, nil
}

// Close closes the service and its dependencies
func (s *Service) Close() error {
	log.Println("Service shutdown completed")
	return nil
}