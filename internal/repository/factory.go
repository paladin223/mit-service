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

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(cfg *config.Config) (*RepositoryManager, error) {
	repo, err := NewRepository(cfg)
	if err != nil {
		return nil, err
	}

	return &RepositoryManager{
		Record: repo,
		Inbox:  repo,
	}, nil
}
