package main

import (
	"context"
	"log"
	"mit-service/internal/config"
	"mit-service/internal/handler"
	"mit-service/internal/metrics"
	"mit-service/internal/repository"
	"mit-service/internal/service"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	log.Printf("Starting MIT Service with repository type: %s", cfg.Repository.Type)

	// Initialize repository
	repoManager, err := repository.NewRepositoryManager(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}

	log.Println("Repository initialized successfully")

	// Initialize metrics
	appMetrics := metrics.NewMetrics()
	log.Println("Metrics initialized successfully")

	// Initialize service
	svc := service.NewService(repoManager, appMetrics)

	// Start inbox worker
	svc.StartInboxWorker(
		cfg.InboxWorker.WorkerCount,
		cfg.InboxWorker.BatchSize,
		cfg.InboxWorker.PollInterval,
		cfg.InboxWorker.MaxRetries,
		cfg.InboxWorker.RetryDelay,
	)

	log.Printf("Inbox worker started with %d workers", cfg.InboxWorker.WorkerCount)

	// Setup HTTP routes
	mux := handler.SetupRoutes(svc, appMetrics)

	// Create HTTP server
	server := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        mux,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Log service endpoints
	log.Println("Service endpoints:")
	log.Printf("  Health check:  http://localhost:%s/health", cfg.Server.Port)
	log.Printf("  Performance:   http://localhost:%s/performance", cfg.Server.Port)
	log.Printf("  Metrics:       http://localhost:%s/metrics", cfg.Server.Port)
	log.Printf("  Task stats:    http://localhost:%s/stats", cfg.Server.Port)
	log.Printf("  Task list:     http://localhost:%s/tasks?status=<status>&limit=<limit>&offset=<offset>", cfg.Server.Port)
	log.Printf("  Insert:        POST http://localhost:%s/insert", cfg.Server.Port)
	log.Printf("  Update:        POST http://localhost:%s/update", cfg.Server.Port)
	log.Printf("  Delete:        POST http://localhost:%s/delete", cfg.Server.Port)
	log.Printf("  Get:           GET  http://localhost:%s/get?id=<record_id>", cfg.Server.Port)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Stop service and cleanup
	svc.Close()

	// Close repository connections
	if repoManager.Record != nil {
		if err := repoManager.Record.Close(); err != nil {
			log.Printf("Error closing record repository: %v", err)
		}
	}

	if repoManager.Inbox != nil {
		if err := repoManager.Inbox.Close(); err != nil {
			log.Printf("Error closing inbox repository: %v", err)
		}
	}

	log.Println("Server shutdown completed")
}
