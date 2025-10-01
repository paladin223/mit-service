package handler

import (
	"encoding/json"
	"log"
	"mit-service/internal/metrics"
	"mit-service/internal/models"
	"mit-service/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler handles HTTP requests
type Handler struct {
	service *service.Service
	metrics *metrics.Metrics
}

// NewHandler creates a new handler instance
func NewHandler(service *service.Service, metrics *metrics.Metrics) *Handler {
	return &Handler{
		service: service,
		metrics: metrics,
	}
}

// validateID validates that ID is not empty or whitespace
func (h *Handler) validateID(id string) bool {
	return strings.TrimSpace(id) != ""
}

// Insert handles POST /insert requests
func (h *Handler) Insert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.InsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Insert: invalid request: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate ID
	if !h.validateID(req.ID) {
		h.writeErrorResponse(w, http.StatusBadRequest, "ID cannot be empty")
		return
	}

	// Validate Value
	if len(req.Value) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "Value cannot be empty")
		return
	}

	ctx := r.Context()
	if err := h.service.Insert(ctx, &req); err != nil {
		log.Printf("Insert: failed to insert record %s: %v", req.ID, err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to insert record: "+err.Error())
		return
	}

	log.Printf("Insert: queued insert task for record ID: %s", req.ID)
	h.writeJSONResponse(w, http.StatusCreated, models.SuccessResponse{
		Message: "Insert task queued successfully",
	})
}

// Update handles POST /update requests
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Update: invalid request: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate ID
	if !h.validateID(req.ID) {
		h.writeErrorResponse(w, http.StatusBadRequest, "ID cannot be empty")
		return
	}

	// Validate Value
	if len(req.Value) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "Value cannot be empty")
		return
	}

	ctx := r.Context()
	if err := h.service.Update(ctx, &req); err != nil {
		log.Printf("Update: failed to update record %s: %v", req.ID, err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update record: "+err.Error())
		return
	}

	// Success - no additional logging needed
	h.writeJSONResponse(w, http.StatusOK, models.SuccessResponse{
		Message: "Update task queued successfully",
	})
}

// Delete handles POST /delete requests
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Delete: invalid request: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate ID
	if !h.validateID(req.ID) {
		h.writeErrorResponse(w, http.StatusBadRequest, "ID cannot be empty")
		return
	}

	ctx := r.Context()
	if err := h.service.Delete(ctx, &req); err != nil {
		log.Printf("Delete: failed to delete record %s: %v", req.ID, err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete record: "+err.Error())
		return
	}

	// Success - no additional logging needed
	h.writeJSONResponse(w, http.StatusOK, models.SuccessResponse{
		Message: "Delete task queued successfully",
	})
}

// Get handles GET /get requests
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get ID from query parameters
	id := r.URL.Query().Get("id")
	if !h.validateID(id) {
		h.writeErrorResponse(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	ctx := r.Context()
	record, err := h.service.Get(ctx, id)
	if err != nil {
		log.Printf("Get: failed to get record %s: %v", id, err)
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, "Record not found")
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get record: "+err.Error())
		}
		return
	}

	// Success - no additional logging needed
	h.writeJSONResponse(w, http.StatusOK, record)
}

// Health handles GET /health requests
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "mit-service",
	})
}

// Tasks handles GET /tasks requests - shows current inbox tasks
func (h *Handler) Tasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	ctx := r.Context()
	response, err := h.service.GetTasks(ctx, status, limit, offset)
	if err != nil {
		log.Printf("Tasks: failed to get tasks: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get tasks: "+err.Error())
		return
	}

	log.Printf("Tasks: retrieved %d tasks (status: %s, limit: %d, offset: %d)",
		len(response.Tasks), status, limit, offset)
	h.writeJSONResponse(w, http.StatusOK, response)
}

// TaskStats handles GET /stats requests - shows inbox tasks statistics
func (h *Handler) TaskStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()
	stats, err := h.service.GetTaskStats(ctx)
	if err != nil {
		log.Printf("TaskStats: failed to get task stats: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get task stats: "+err.Error())
		return
	}

	log.Printf("TaskStats: total=%d, pending=%d, processing=%d, completed=%d, failed=%d",
		stats.TotalTasks, stats.PendingTasks, stats.ProcessingTasks, stats.CompletedTasks, stats.FailedTasks)
	h.writeJSONResponse(w, http.StatusOK, stats)
}

// Metrics handles GET /metrics requests - shows performance metrics
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	snapshot := h.metrics.GetSnapshot()

	log.Printf("Metrics: RPS=%.2f, avg_response=%.2fms, queue_depth=%d, goroutines=%d",
		snapshot.RequestsPerSecond, snapshot.AvgResponseTime, snapshot.QueueDepth, snapshot.GoroutineCount)

	h.writeJSONResponse(w, http.StatusOK, snapshot)
}

// Performance handles GET /performance requests - shows health status and recommendations
func (h *Handler) Performance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	snapshot := h.metrics.GetSnapshot()
	health := snapshot.GetHealthStatus()

	response := map[string]interface{}{
		"health":  health,
		"metrics": snapshot,
	}

	log.Printf("Performance: status=%s, score=%d, issues=%d",
		health.Status, health.Score, len(health.Issues))

	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse writes a JSON response with the given status code
func (h *Handler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeErrorResponse writes an error response with the given status code and message
func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSONResponse(w, statusCode, models.ErrorResponse{
		Error: message,
	})
}

// CORS middleware
func (h *Handler) enableCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

// Middleware wrapper to apply CORS
func (h *Handler) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		next(w, r)
	})
}

// PrometheusMetrics endpoint for Prometheus
func (h *Handler) PrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// Middleware wrapper for logging
func (h *Handler) withLogging(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
	})
}

// ResponseWriter wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Middleware wrapper for metrics
func (h *Handler) withMetrics(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Increment active connections
		h.metrics.IncrementActiveConnections()
		defer h.metrics.DecrementActiveConnections()

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Execute the request
		next(rw, r)

		// Record metrics with detailed information for Prometheus
		duration := time.Since(start)
		h.metrics.RecordHTTPRequestWithDetails(r.Method, r.URL.Path, rw.statusCode, duration)
	})
}
