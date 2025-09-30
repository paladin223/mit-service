package metrics

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds all application metrics
type Metrics struct {
	// HTTP metrics
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	totalResponseTime  int64 // in milliseconds
	activeConnections  int64
	requestsPerSecond  float64
	avgResponseTime    float64
	lastRequestTime    time.Time

	// Inbox metrics
	totalTasks        int64
	completedTasks    int64
	failedTasks       int64
	totalTaskTime     int64 // in milliseconds
	tasksPerSecond    float64
	avgTaskTime       float64
	queueDepth        int64
	maxQueueDepth     int64
	workerUtilization float64
	lastTaskTime      time.Time

	// System metrics
	startTime      time.Time
	goroutineCount int
	memoryUsage    uint64
	cpuUsage       float64

	mu                sync.RWMutex
	lastMetricsUpdate time.Time
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:         time.Now(),
		lastMetricsUpdate: time.Now(),
	}
}

// HTTP Metrics

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(duration time.Duration, success bool) {
	atomic.AddInt64(&m.totalRequests, 1)
	atomic.AddInt64(&m.totalResponseTime, duration.Milliseconds())

	if success {
		atomic.AddInt64(&m.successfulRequests, 1)
	} else {
		atomic.AddInt64(&m.failedRequests, 1)
	}

	m.lastRequestTime = time.Now()
	m.updateHTTPMetrics()
}

// IncrementActiveConnections increments active connection count
func (m *Metrics) IncrementActiveConnections() {
	atomic.AddInt64(&m.activeConnections, 1)
}

// DecrementActiveConnections decrements active connection count
func (m *Metrics) DecrementActiveConnections() {
	atomic.AddInt64(&m.activeConnections, -1)
}

// updateHTTPMetrics updates calculated HTTP metrics
func (m *Metrics) updateHTTPMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastMetricsUpdate).Seconds()

	if elapsed >= 1.0 { // Update every second
		totalReqs := atomic.LoadInt64(&m.totalRequests)
		totalTime := atomic.LoadInt64(&m.totalResponseTime)

		if totalReqs > 0 {
			m.avgResponseTime = float64(totalTime) / float64(totalReqs)
		}

		// Calculate RPS over the last second
		timeSinceStart := now.Sub(m.startTime).Seconds()
		if timeSinceStart > 0 {
			m.requestsPerSecond = float64(totalReqs) / timeSinceStart
		}

		m.lastMetricsUpdate = now
	}
}

// Inbox Metrics

// RecordTaskExecution records a task execution
func (m *Metrics) RecordTaskExecution(duration time.Duration, success bool) {
	atomic.AddInt64(&m.totalTasks, 1)
	atomic.AddInt64(&m.totalTaskTime, duration.Milliseconds())

	if success {
		atomic.AddInt64(&m.completedTasks, 1)
	} else {
		atomic.AddInt64(&m.failedTasks, 1)
	}

	m.lastTaskTime = time.Now()
	m.updateTaskMetrics()
}

// SetQueueDepth sets the current queue depth
func (m *Metrics) SetQueueDepth(depth int64) {
	atomic.StoreInt64(&m.queueDepth, depth)

	// Track max queue depth
	for {
		current := atomic.LoadInt64(&m.maxQueueDepth)
		if depth <= current || atomic.CompareAndSwapInt64(&m.maxQueueDepth, current, depth) {
			break
		}
	}
}

// updateTaskMetrics updates calculated task metrics
func (m *Metrics) updateTaskMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastMetricsUpdate).Seconds()

	if elapsed >= 1.0 {
		totalTasks := atomic.LoadInt64(&m.totalTasks)
		totalTime := atomic.LoadInt64(&m.totalTaskTime)

		if totalTasks > 0 {
			m.avgTaskTime = float64(totalTime) / float64(totalTasks)
		}

		// Calculate TPS over the entire runtime
		timeSinceStart := now.Sub(m.startTime).Seconds()
		if timeSinceStart > 0 {
			m.tasksPerSecond = float64(totalTasks) / timeSinceStart
		}
	}
}

// UpdateSystemMetrics updates system-level metrics
func (m *Metrics) UpdateSystemMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.goroutineCount = runtime.NumGoroutine()
	m.memoryUsage = memStats.Alloc
}

// GetSnapshot returns a snapshot of all metrics
func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	m.UpdateSystemMetrics()

	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)

	return &MetricsSnapshot{
		// HTTP metrics
		TotalRequests:      atomic.LoadInt64(&m.totalRequests),
		SuccessfulRequests: atomic.LoadInt64(&m.successfulRequests),
		FailedRequests:     atomic.LoadInt64(&m.failedRequests),
		ActiveConnections:  atomic.LoadInt64(&m.activeConnections),
		RequestsPerSecond:  m.requestsPerSecond,
		AvgResponseTime:    m.avgResponseTime,

		// Inbox metrics
		TotalTasks:       atomic.LoadInt64(&m.totalTasks),
		CompletedTasks:   atomic.LoadInt64(&m.completedTasks),
		FailedTasksCount: atomic.LoadInt64(&m.failedTasks),
		TasksPerSecond:   m.tasksPerSecond,
		AvgTaskTime:      m.avgTaskTime,
		QueueDepth:       atomic.LoadInt64(&m.queueDepth),
		MaxQueueDepth:    atomic.LoadInt64(&m.maxQueueDepth),

		// System metrics
		Uptime:         uptime,
		GoroutineCount: m.goroutineCount,
		MemoryUsageMB:  float64(m.memoryUsage) / 1024 / 1024,

		// Timestamps
		LastRequestTime: m.lastRequestTime,
		LastTaskTime:    m.lastTaskTime,
		Timestamp:       time.Now(),
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	// HTTP metrics
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	ActiveConnections  int64   `json:"active_connections"`
	RequestsPerSecond  float64 `json:"requests_per_second"`
	AvgResponseTime    float64 `json:"avg_response_time_ms"`

	// Inbox metrics
	TotalTasks       int64   `json:"total_tasks"`
	CompletedTasks   int64   `json:"completed_tasks"`
	FailedTasksCount int64   `json:"failed_tasks_count"`
	TasksPerSecond   float64 `json:"tasks_per_second"`
	AvgTaskTime      float64 `json:"avg_task_time_ms"`
	QueueDepth       int64   `json:"queue_depth"`
	MaxQueueDepth    int64   `json:"max_queue_depth"`

	// System metrics
	Uptime         time.Duration `json:"uptime_seconds"`
	GoroutineCount int           `json:"goroutine_count"`
	MemoryUsageMB  float64       `json:"memory_usage_mb"`

	// Timestamps
	LastRequestTime time.Time `json:"last_request_time"`
	LastTaskTime    time.Time `json:"last_task_time"`
	Timestamp       time.Time `json:"timestamp"`
}

// HealthStatus represents the health status based on metrics
type HealthStatus struct {
	Status          string   `json:"status"` // "healthy", "warning", "critical"
	Score           int      `json:"score"`  // 0-100
	Issues          []string `json:"issues"`
	Recommendations []string `json:"recommendations"`
}

// GetHealthStatus analyzes metrics and returns health status
func (s *MetricsSnapshot) GetHealthStatus() *HealthStatus {
	status := &HealthStatus{
		Status:          "healthy",
		Score:           100,
		Issues:          []string{},
		Recommendations: []string{},
	}

	// Check response time (warning if > 100ms, critical if > 500ms)
	if s.AvgResponseTime > 500 {
		status.Status = "critical"
		status.Score -= 40
		status.Issues = append(status.Issues, "Very high response time (>500ms)")
		status.Recommendations = append(status.Recommendations, "Consider scaling up the service or optimizing database queries")
	} else if s.AvgResponseTime > 100 {
		if status.Status != "critical" {
			status.Status = "warning"
		}
		status.Score -= 20
		status.Issues = append(status.Issues, "High response time (>100ms)")
		status.Recommendations = append(status.Recommendations, "Monitor database performance and consider caching")
	}

	// Check queue depth (warning if > 100, critical if > 500)
	if s.QueueDepth > 500 {
		status.Status = "critical"
		status.Score -= 30
		status.Issues = append(status.Issues, "Very high queue depth (>500)")
		status.Recommendations = append(status.Recommendations, "Increase inbox worker count or check for processing bottlenecks")
	} else if s.QueueDepth > 100 {
		if status.Status != "critical" {
			status.Status = "warning"
		}
		status.Score -= 15
		status.Issues = append(status.Issues, "High queue depth (>100)")
		status.Recommendations = append(status.Recommendations, "Monitor task processing speed")
	}

	// Check error rate (warning if > 1%, critical if > 5%)
	if s.TotalRequests > 0 {
		errorRate := float64(s.FailedRequests) / float64(s.TotalRequests) * 100
		if errorRate > 5 {
			status.Status = "critical"
			status.Score -= 35
			status.Issues = append(status.Issues, "High error rate (>5%)")
			status.Recommendations = append(status.Recommendations, "Investigate application errors and system issues")
		} else if errorRate > 1 {
			if status.Status != "critical" {
				status.Status = "warning"
			}
			status.Score -= 10
			status.Issues = append(status.Issues, "Elevated error rate (>1%)")
			status.Recommendations = append(status.Recommendations, "Monitor error logs for patterns")
		}
	}

	// Check memory usage (warning if > 512MB, critical if > 1GB)
	if s.MemoryUsageMB > 1024 {
		status.Status = "critical"
		status.Score -= 25
		status.Issues = append(status.Issues, "Very high memory usage (>1GB)")
		status.Recommendations = append(status.Recommendations, "Check for memory leaks and consider scaling")
	} else if s.MemoryUsageMB > 512 {
		if status.Status != "critical" {
			status.Status = "warning"
		}
		status.Score -= 10
		status.Issues = append(status.Issues, "High memory usage (>512MB)")
		status.Recommendations = append(status.Recommendations, "Monitor memory usage trends")
	}

	// Check goroutine count (warning if > 1000, critical if > 5000)
	if s.GoroutineCount > 5000 {
		status.Status = "critical"
		status.Score -= 20
		status.Issues = append(status.Issues, "Very high goroutine count (>5000)")
		status.Recommendations = append(status.Recommendations, "Check for goroutine leaks")
	} else if s.GoroutineCount > 1000 {
		if status.Status != "critical" {
			status.Status = "warning"
		}
		status.Score -= 10
		status.Issues = append(status.Issues, "High goroutine count (>1000)")
		status.Recommendations = append(status.Recommendations, "Monitor goroutine usage")
	}

	return status
}

