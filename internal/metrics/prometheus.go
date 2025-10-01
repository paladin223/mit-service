package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

// PrometheusMetrics holds Prometheus metrics
type PrometheusMetrics struct {
	// HTTP metrics
	httpRequestsTotal      *prometheus.CounterVec
	httpRequestDuration    *prometheus.HistogramVec
	httpActiveConnections  prometheus.Gauge

	// Task metrics
	tasksTotal             *prometheus.CounterVec
	taskDuration           *prometheus.HistogramVec
	queueDepth             prometheus.Gauge
	maxQueueDepth          prometheus.Gauge

	// System metrics
	goroutineCount         prometheus.Gauge
	memoryUsage            prometheus.Gauge
	uptimeSeconds          prometheus.Gauge
}

// NewPrometheusMetrics creates a new Prometheus metrics instance
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		httpRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "mit_service_http_requests_total",
			Help: "Total number of HTTP requests",
		}, []string{"method", "endpoint", "status"}),

		httpRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mit_service_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "endpoint"}),

		httpActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "mit_service_http_active_connections",
			Help: "Number of active HTTP connections",
		}),

		tasksTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "mit_service_tasks_total",
			Help: "Total number of tasks processed",
		}, []string{"operation", "status"}),

		taskDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mit_service_task_duration_seconds", 
			Help:    "Task processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),

		queueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "mit_service_queue_depth",
			Help: "Current queue depth",
		}),

		maxQueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "mit_service_max_queue_depth",
			Help: "Maximum queue depth observed",
		}),

		goroutineCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "mit_service_goroutines",
			Help: "Number of goroutines",
		}),

		memoryUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "mit_service_memory_usage_bytes",
			Help: "Memory usage in bytes",
		}),

		uptimeSeconds: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "mit_service_uptime_seconds",
			Help: "Service uptime in seconds",
		}),
	}
}

// RecordHTTPRequest records an HTTP request metric
func (pm *PrometheusMetrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	status := "success"
	if statusCode >= 400 {
		status = "error"
	}

	pm.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	pm.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// SetActiveConnections sets the active connections count
func (pm *PrometheusMetrics) SetActiveConnections(count int64) {
	pm.httpActiveConnections.Set(float64(count))
}

// RecordTask records a task processing metric
func (pm *PrometheusMetrics) RecordTask(operation, status string, duration time.Duration) {
	pm.tasksTotal.WithLabelValues(operation, status).Inc()
	pm.taskDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// SetQueueMetrics sets queue-related metrics
func (pm *PrometheusMetrics) SetQueueMetrics(current, max int64) {
	pm.queueDepth.Set(float64(current))
	pm.maxQueueDepth.Set(float64(max))
}

// SetSystemMetrics sets system-level metrics
func (pm *PrometheusMetrics) SetSystemMetrics(goroutines int, memoryBytes uint64, uptimeDuration time.Duration) {
	pm.goroutineCount.Set(float64(goroutines))
	pm.memoryUsage.Set(float64(memoryBytes))
	pm.uptimeSeconds.Set(uptimeDuration.Seconds())
}
