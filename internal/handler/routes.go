package handler

import (
	"mit-service/internal/metrics"
	"mit-service/internal/service"
	"net/http"
)

// SetupRoutes sets up HTTP routes using standard library
func SetupRoutes(service *service.Service, metrics *metrics.Metrics) *http.ServeMux {
	mux := http.NewServeMux()
	h := NewHandler(service, metrics)

	// Health check endpoint
	mux.HandleFunc("/health", h.withCORS(h.withMetrics(h.withLogging(h.Health))))

	// Monitoring endpoints
	mux.HandleFunc("/tasks", h.withCORS(h.withMetrics(h.withLogging(h.Tasks))))
	mux.HandleFunc("/stats", h.withCORS(h.withMetrics(h.withLogging(h.TaskStats))))
	mux.HandleFunc("/metrics", h.PrometheusMetrics) // No middleware to avoid recursive metrics
	mux.HandleFunc("/performance", h.withCORS(h.withMetrics(h.withLogging(h.Performance))))

	// API routes (root level as specified in requirements)
	mux.HandleFunc("/insert", h.withCORS(h.withMetrics(h.withLogging(h.Insert))))
	mux.HandleFunc("/update", h.withCORS(h.withMetrics(h.withLogging(h.Update))))
	mux.HandleFunc("/delete", h.withCORS(h.withMetrics(h.withLogging(h.Delete))))
	mux.HandleFunc("/get", h.withCORS(h.withMetrics(h.withLogging(h.Get))))

	return mux
}
