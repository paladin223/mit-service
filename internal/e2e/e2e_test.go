package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mit-service/internal/config"
	"mit-service/internal/handler"
	"mit-service/internal/metrics"
	"mit-service/internal/models"
	"mit-service/internal/repository"
	"mit-service/internal/service"
)

func TestE2E_InsertAndGet(t *testing.T) {
	// Setup test server with mock repository
	cfg := &config.Config{
		Repository: config.RepositoryConfig{Type: "mock"},
		InboxWorker: config.InboxWorkerConfig{
			WorkerCount:  1,
			BatchSize:    1,
			PollInterval: 100 * time.Millisecond,
			MaxRetries:   3,
			RetryDelay:   100 * time.Millisecond,
		},
	}

	repoManager, err := repository.NewRepositoryManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	appMetrics := metrics.NewMetrics()
	svc := service.NewService(repoManager, appMetrics)
	
	// Start inbox worker
	svc.StartInboxWorker(
		cfg.InboxWorker.WorkerCount,
		cfg.InboxWorker.BatchSize,
		cfg.InboxWorker.PollInterval,
		cfg.InboxWorker.MaxRetries,
		cfg.InboxWorker.RetryDelay,
	)
	defer svc.Close()

	mux := handler.SetupRoutes(svc, appMetrics)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test data
	testID := "test_user_123"
	testData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}

	// Test INSERT operation
	insertReq := models.InsertRequest{
		ID:    testID,
		Value: testData,
	}
	insertBody, _ := json.Marshal(insertReq)

	resp, err := http.Post(server.URL+"/insert", "application/json", bytes.NewBuffer(insertBody))
	if err != nil {
		t.Fatalf("Insert request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Wait for inbox processing
	time.Sleep(200 * time.Millisecond)

	// Test GET operation
	getResp, err := http.Get(server.URL + "/get?id=" + testID)
	if err != nil {
		t.Fatalf("Get request failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", getResp.StatusCode)
	}

	var record models.Record
	if err := json.NewDecoder(getResp.Body).Decode(&record); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if record.ID != testID {
		t.Errorf("Expected ID %s, got %s", testID, record.ID)
	}

	// Check record data
	recordData, ok := record.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("Record value is not a map")
	}

	if recordData["name"] != testData["name"] {
		t.Errorf("Expected name %v, got %v", testData["name"], recordData["name"])
	}
}

func TestE2E_UpdateRecord(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Repository: config.RepositoryConfig{Type: "mock"},
		InboxWorker: config.InboxWorkerConfig{
			WorkerCount:  1,
			BatchSize:    1,
			PollInterval: 100 * time.Millisecond,
			MaxRetries:   3,
			RetryDelay:   100 * time.Millisecond,
		},
	}

	repoManager, err := repository.NewRepositoryManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	appMetrics := metrics.NewMetrics()
	svc := service.NewService(repoManager, appMetrics)
	svc.StartInboxWorker(
		cfg.InboxWorker.WorkerCount,
		cfg.InboxWorker.BatchSize,
		cfg.InboxWorker.PollInterval,
		cfg.InboxWorker.MaxRetries,
		cfg.InboxWorker.RetryDelay,
	)
	defer svc.Close()

	mux := handler.SetupRoutes(svc, appMetrics)
	server := httptest.NewServer(mux)
	defer server.Close()

	testID := "test_update_123"

	// Insert initial record
	insertData := map[string]interface{}{
		"name": "Initial Name",
		"age":  25,
	}
	insertReq := models.InsertRequest{ID: testID, Value: insertData}
	insertBody, _ := json.Marshal(insertReq)
	
	resp, _ := http.Post(server.URL+"/insert", "application/json", bytes.NewBuffer(insertBody))
	resp.Body.Close()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Update record
	updateData := map[string]interface{}{
		"name": "Updated Name",
		"age":  26,
		"city": "New York",
	}
	updateReq := models.UpdateRequest{ID: testID, Value: updateData}
	updateBody, _ := json.Marshal(updateReq)

	updateResp, err := http.Post(server.URL+"/update", "application/json", bytes.NewBuffer(updateBody))
	if err != nil {
		t.Fatalf("Update request failed: %v", err)
	}
	updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", updateResp.StatusCode)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify update
	getResp, _ := http.Get(server.URL + "/get?id=" + testID)
	defer getResp.Body.Close()

	var record models.Record
	json.NewDecoder(getResp.Body).Decode(&record)

	recordData := record.Value.(map[string]interface{})
	if recordData["name"] != "Updated Name" {
		t.Errorf("Expected updated name, got %v", recordData["name"])
	}

	if recordData["city"] != "New York" {
		t.Errorf("Expected city New York, got %v", recordData["city"])
	}
}

func TestE2E_GetNonExistentRecord(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Repository: config.RepositoryConfig{Type: "mock"},
	}

	repoManager, _ := repository.NewRepositoryManager(cfg)
	appMetrics := metrics.NewMetrics()
	svc := service.NewService(repoManager, appMetrics)

	mux := handler.SetupRoutes(svc, appMetrics)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Try to get non-existent record
	resp, err := http.Get(server.URL + "/get?id=nonexistent")
	if err != nil {
		t.Fatalf("Get request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestE2E_TaskStats(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Repository: config.RepositoryConfig{Type: "mock"},
		InboxWorker: config.InboxWorkerConfig{
			WorkerCount:  1,
			BatchSize:    1,
			PollInterval: 100 * time.Millisecond,
			MaxRetries:   3,
			RetryDelay:   100 * time.Millisecond,
		},
	}

	repoManager, _ := repository.NewRepositoryManager(cfg)
	appMetrics := metrics.NewMetrics()
	svc := service.NewService(repoManager, appMetrics)
	svc.StartInboxWorker(
		cfg.InboxWorker.WorkerCount,
		cfg.InboxWorker.BatchSize,
		cfg.InboxWorker.PollInterval,
		cfg.InboxWorker.MaxRetries,
		cfg.InboxWorker.RetryDelay,
	)
	defer svc.Close()

	mux := handler.SetupRoutes(svc, appMetrics)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Insert a few records
	for i := 0; i < 3; i++ {
		insertReq := models.InsertRequest{
			ID:    fmt.Sprintf("test_%d", i),
			Value: map[string]interface{}{"data": i},
		}
		insertBody, _ := json.Marshal(insertReq)
		resp, _ := http.Post(server.URL+"/insert", "application/json", bytes.NewBuffer(insertBody))
		resp.Body.Close()
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check stats
	statsResp, err := http.Get(server.URL + "/stats")
	if err != nil {
		t.Fatalf("Stats request failed: %v", err)
	}
	defer statsResp.Body.Close()

	if statsResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", statsResp.StatusCode)
	}

	var stats models.TaskStats
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode stats: %v", err)
	}

	if stats.TotalTasks < 3 {
		t.Errorf("Expected at least 3 total tasks, got %d", stats.TotalTasks)
	}

	if stats.CompletedTasks < 3 {
		t.Errorf("Expected at least 3 completed tasks, got %d", stats.CompletedTasks)
	}
}

func TestE2E_HealthCheck(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Repository: config.RepositoryConfig{Type: "mock"},
	}

	repoManager, _ := repository.NewRepositoryManager(cfg)
	appMetrics := metrics.NewMetrics()
	svc := service.NewService(repoManager, appMetrics)

	mux := handler.SetupRoutes(svc, appMetrics)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test health endpoint
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("Expected healthy status, got %v", health["status"])
	}

	if health["service"] != "mit-service" {
		t.Errorf("Expected mit-service, got %v", health["service"])
	}
}
