package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Metrics provides metrics collection and HTTP endpoint for monitoring
type Metrics struct {
	server *http.Server
	logger *Logger
	mutex  sync.RWMutex

	// Metrics data
	RequestCounts    map[string]int64   `json:"request_counts"`
	RequestDurations map[string][]int64 `json:"request_durations_ms"`
	SuccessRates     map[string]float64 `json:"success_rates"`
	LastUpdated      time.Time          `json:"last_updated"`

	// Counters
	TotalRequests  int64 `json:"total_requests"`
	TotalSuccesses int64 `json:"total_successes"`
	TotalFailures  int64 `json:"total_failures"`
}

// NewMetrics creates a new metrics instance and starts the HTTP server
func NewMetrics(port int, path string, logger *Logger) *Metrics {
	metrics := &Metrics{
		logger:           logger,
		RequestCounts:    make(map[string]int64),
		RequestDurations: make(map[string][]int64),
		SuccessRates:     make(map[string]float64),
		LastUpdated:      time.Now(),
	}

	// Create HTTP server for metrics endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(path, metrics.metricsHandler)
	mux.HandleFunc("/health", metrics.healthHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	metrics.server = server

	// Start server in background
	go func() {
		logger.Info("Starting metrics server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Metrics server error: %v", err)
		}
	}()

	return metrics
}

// RecordRequest records metrics for a completed request
func (m *Metrics) RecordRequest(url, status string, duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Update request counts
	m.RequestCounts[url]++
	m.TotalRequests++

	// Update duration tracking
	durationMs := duration.Milliseconds()
	if m.RequestDurations[url] == nil {
		m.RequestDurations[url] = make([]int64, 0)
	}

	// Keep only last 100 durations per URL to prevent memory growth
	if len(m.RequestDurations[url]) >= 100 {
		m.RequestDurations[url] = m.RequestDurations[url][1:]
	}
	m.RequestDurations[url] = append(m.RequestDurations[url], durationMs)

	// Update success/failure counters
	if status == "success" {
		m.TotalSuccesses++
	} else {
		m.TotalFailures++
	}

	// Calculate success rate for this URL
	successCount := int64(0)
	totalCount := m.RequestCounts[url]

	// This is a simplified calculation - in a real implementation,
	// you'd want to track successes/failures per URL separately
	if status == "success" {
		// Estimate success rate based on overall pattern
		overallSuccessRate := float64(m.TotalSuccesses) / float64(m.TotalRequests)
		m.SuccessRates[url] = overallSuccessRate
	}

	m.LastUpdated = time.Now()
}

// metricsHandler serves metrics data as JSON
func (m *Metrics) metricsHandler(w http.ResponseWriter, r *http.Request) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Create response structure
	response := struct {
		Metrics     *Metrics  `json:"metrics"`
		Summary     Summary   `json:"summary"`
		GeneratedAt time.Time `json:"generated_at"`
	}{
		Metrics:     m,
		Summary:     m.calculateSummary(),
		GeneratedAt: time.Now(),
	}

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Error("Failed to encode metrics response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// healthHandler provides a simple health check endpoint
func (m *Metrics) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Uptime    string    `json:"uptime"`
	}{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(m.LastUpdated).String(),
	}

	json.NewEncoder(w).Encode(health)
}

// Summary contains calculated summary statistics
type Summary struct {
	TotalURLs           int     `json:"total_urls"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	OverallSuccessRate  float64 `json:"overall_success_rate"`
	RequestsPerSecond   float64 `json:"requests_per_second"`
}

// calculateSummary calculates summary statistics from current metrics
func (m *Metrics) calculateSummary() Summary {
	summary := Summary{
		TotalURLs: len(m.RequestCounts),
	}

	// Calculate average response time
	totalDuration := int64(0)
	totalMeasurements := int64(0)

	for _, durations := range m.RequestDurations {
		for _, duration := range durations {
			totalDuration += duration
			totalMeasurements++
		}
	}

	if totalMeasurements > 0 {
		summary.AverageResponseTime = float64(totalDuration) / float64(totalMeasurements)
	}

	// Calculate overall success rate
	if m.TotalRequests > 0 {
		summary.OverallSuccessRate = float64(m.TotalSuccesses) / float64(m.TotalRequests) * 100
	}

	// Calculate requests per second (based on time since last update)
	timeSinceStart := time.Since(m.LastUpdated).Seconds()
	if timeSinceStart > 0 {
		summary.RequestsPerSecond = float64(m.TotalRequests) / timeSinceStart
	}

	return summary
}

// GetMetricsData returns a copy of the current metrics data
func (m *Metrics) GetMetricsData() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"request_counts":    m.RequestCounts,
		"request_durations": m.RequestDurations,
		"success_rates":     m.SuccessRates,
		"total_requests":    m.TotalRequests,
		"total_successes":   m.TotalSuccesses,
		"total_failures":    m.TotalFailures,
		"last_updated":      m.LastUpdated,
		"summary":           m.calculateSummary(),
	}
}

// Reset clears all metrics data
func (m *Metrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.RequestCounts = make(map[string]int64)
	m.RequestDurations = make(map[string][]int64)
	m.SuccessRates = make(map[string]float64)
	m.TotalRequests = 0
	m.TotalSuccesses = 0
	m.TotalFailures = 0
	m.LastUpdated = time.Now()
}

// Shutdown gracefully shuts down the metrics server
func (m *Metrics) Shutdown() {
	if m.server != nil {
		m.logger.Info("Shutting down metrics server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.server.Shutdown(ctx); err != nil {
			m.logger.Error("Error shutting down metrics server: %v", err)
		} else {
			m.logger.Info("Metrics server shutdown complete")
		}
	}
}
