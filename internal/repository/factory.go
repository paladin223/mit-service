package repository

import (
	"fmt"
	"mit-service/internal/config"
)

// NewRepository creates a new repository based on configuration
func NewRepository(cfg *config.Config) (Repository, error) {
	switch cfg.Repository.Type {
	case "postgres":
		repo, err := NewPostgresRepository(cfg.Database.ConnectionString())
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres repository: %w", err)
		}
		return repo, nil

	case "mock":
		return NewMockRepository(), nil

	default:
		return nil, fmt.Errorf("unsupported repository type: %s", cfg.Repository.Type)
	}
}

// NewRepositoryManager creates a new repository manager with separate DBs
func NewRepositoryManager(cfg *config.Config) (*RepositoryManager, error) {
	switch cfg.Repository.Type {
	case "postgres":
		// Create separate repositories for main and inbox DBs
		recordRepo, err := NewPostgresRecordRepository(cfg.Database.ConnectionString())
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres record repository: %w", err)
		}

		inboxRepo, err := NewPostgresInboxRepository(cfg.InboxDB.ConnectionString())
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres inbox repository: %w", err)
		}

		return &RepositoryManager{
			Record: recordRepo,
			Inbox:  inboxRepo,
		}, nil

	case "mock":
		repo := NewMockRepository()
		return &RepositoryManager{
			Record: repo,
			Inbox:  repo,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported repository type: %s", cfg.Repository.Type)
	}
}
