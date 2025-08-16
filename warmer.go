package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// CacheWarmer represents the main cache warming engine
type CacheWarmer struct {
	config  *Config
	logger  *Logger
	client  *http.Client
	metrics *Metrics

	// Shutdown coordination
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Statistics
	stats Statistics
}

// Statistics holds runtime statistics for the cache warmer
type Statistics struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   int64 // in nanoseconds
	StartTime       time.Time
}

// NewCacheWarmer creates a new cache warmer instance
func NewCacheWarmer(config *Config, logger *Logger) *CacheWarmer {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Configure HTTP client
	client := &http.Client{
		Timeout: config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Control redirect behavior
			if !config.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			return nil
		},
	}

	// Initialize metrics if enabled
	var metrics *Metrics
	if config.Metrics.Enabled {
		metrics = NewMetrics(config.Metrics.Port, config.Metrics.Path, logger)
	}

	return &CacheWarmer{
		config:  config,
		logger:  logger,
		client:  client,
		metrics: metrics,
		ctx:     ctx,
		cancel:  cancel,
		stats: Statistics{
			StartTime: time.Now(),
		},
	}
}

// WarmCache performs the cache warming operation
func (cw *CacheWarmer) WarmCache() {
	cw.logger.Info("Starting cache warming with %d URLs and %d workers",
		len(cw.config.URLs), cw.config.Workers)

	// Reset statistics for this run
	atomic.StoreInt64(&cw.stats.TotalRequests, 0)
	atomic.StoreInt64(&cw.stats.SuccessRequests, 0)
	atomic.StoreInt64(&cw.stats.FailedRequests, 0)
	atomic.StoreInt64(&cw.stats.TotalDuration, 0)
	cw.stats.StartTime = time.Now()

	// Create work channel
	workChan := make(chan string, len(cw.config.URLs))

	// Start worker goroutines
	for i := 0; i < cw.config.Workers; i++ {
		cw.wg.Add(1)
		go cw.worker(i, workChan)
	}

	// Send URLs to workers
	for _, url := range cw.config.URLs {
		select {
		case workChan <- url:
		case <-cw.ctx.Done():
			cw.logger.Info("Cache warming cancelled")
			close(workChan)
			cw.wg.Wait()
			return
		}
	}

	// Close work channel to signal completion
	close(workChan)

	// Wait for all workers to complete
	cw.wg.Wait()

	// Print final statistics
	cw.printStatistics()
}

// worker processes URLs from the work channel
func (cw *CacheWarmer) worker(id int, workChan <-chan string) {
	defer cw.wg.Done()

	cw.logger.Debug("Worker %d started", id)

	for {
		select {
		case url, ok := <-workChan:
			if !ok {
				cw.logger.Debug("Worker %d finished", id)
				return
			}
			cw.processURL(id, url)
		case <-cw.ctx.Done():
			cw.logger.Debug("Worker %d cancelled", id)
			return
		}
	}
}

// processURL makes an HTTP request to the specified URL with retry logic
func (cw *CacheWarmer) processURL(workerID int, url string) {
	startTime := time.Now()
	var lastErr error

	// Increment total requests counter
	atomic.AddInt64(&cw.stats.TotalRequests, 1)

	// Retry logic
	for attempt := 0; attempt <= cw.config.RetryCount; attempt++ {
		if attempt > 0 {
			cw.logger.Debug("Worker %d retrying URL %s (attempt %d/%d)",
				workerID, url, attempt+1, cw.config.RetryCount+1)

			// Wait before retry
			select {
			case <-time.After(cw.config.RetryDelay):
			case <-cw.ctx.Done():
				return
			}
		}

		// Make the HTTP request
		success, err := cw.makeRequest(url)
		if success {
			duration := time.Since(startTime)
			atomic.AddInt64(&cw.stats.SuccessRequests, 1)
			atomic.AddInt64(&cw.stats.TotalDuration, int64(duration))

			cw.logger.Debug("Worker %d successfully warmed %s in %v",
				workerID, url, duration)

			// Update metrics if enabled
			if cw.metrics != nil {
				cw.metrics.RecordRequest(url, "success", duration)
			}
			return
		}

		lastErr = err
		cw.logger.Debug("Worker %d failed to warm %s: %v", workerID, url, err)
	}

	// All retries failed
	duration := time.Since(startTime)
	atomic.AddInt64(&cw.stats.FailedRequests, 1)
	atomic.AddInt64(&cw.stats.TotalDuration, int64(duration))

	cw.logger.Warn("Worker %d failed to warm %s after %d attempts: %v",
		workerID, url, cw.config.RetryCount+1, lastErr)

	// Update metrics if enabled
	if cw.metrics != nil {
		cw.metrics.RecordRequest(url, "failure", duration)
	}
}

// makeRequest performs a single HTTP request to the specified URL
func (cw *CacheWarmer) makeRequest(url string) (bool, error) {
	// Create request with context for cancellation
	req, err := http.NewRequestWithContext(cw.ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", cw.config.UserAgent)

	// Set custom headers
	for key, value := range cw.config.Headers {
		req.Header.Set(key, value)
	}

	// Make the request
	resp, err := cw.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check if status code is considered successful
	if !cw.config.IsSuccessCode(resp.StatusCode) {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read and discard response body to ensure complete request processing
	// This is important for cache warming as it ensures the full response is processed
	buffer := make([]byte, 4096)
	for {
		_, err := resp.Body.Read(buffer)
		if err != nil {
			break // EOF or other error, both are fine
		}
	}

	return true, nil
}

// printStatistics prints the current statistics
func (cw *CacheWarmer) printStatistics() {
	total := atomic.LoadInt64(&cw.stats.TotalRequests)
	success := atomic.LoadInt64(&cw.stats.SuccessRequests)
	failed := atomic.LoadInt64(&cw.stats.FailedRequests)
	totalDuration := time.Duration(atomic.LoadInt64(&cw.stats.TotalDuration))
	elapsed := time.Since(cw.stats.StartTime)

	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	avgDuration := time.Duration(0)
	if total > 0 {
		avgDuration = totalDuration / time.Duration(total)
	}

	cw.logger.Info("Cache warming completed:")
	cw.logger.Info("  Total requests: %d", total)
	cw.logger.Info("  Successful: %d (%.1f%%)", success, successRate)
	cw.logger.Info("  Failed: %d", failed)
	cw.logger.Info("  Total time: %v", elapsed)
	cw.logger.Info("  Average request time: %v", avgDuration)

	if total > 0 {
		requestsPerSecond := float64(total) / elapsed.Seconds()
		cw.logger.Info("  Requests per second: %.2f", requestsPerSecond)
	}
}

// GetStatistics returns the current statistics
func (cw *CacheWarmer) GetStatistics() Statistics {
	return Statistics{
		TotalRequests:   atomic.LoadInt64(&cw.stats.TotalRequests),
		SuccessRequests: atomic.LoadInt64(&cw.stats.SuccessRequests),
		FailedRequests:  atomic.LoadInt64(&cw.stats.FailedRequests),
		TotalDuration:   atomic.LoadInt64(&cw.stats.TotalDuration),
		StartTime:       cw.stats.StartTime,
	}
}

// Shutdown gracefully shuts down the cache warmer
func (cw *CacheWarmer) Shutdown() {
	cw.logger.Info("Shutting down cache warmer...")

	// Cancel context to stop all workers
	cw.cancel()

	// Wait for all workers to finish
	cw.wg.Wait()

	// Shutdown metrics server if enabled
	if cw.metrics != nil {
		cw.metrics.Shutdown()
	}

	cw.logger.Info("Cache warmer shutdown complete")
}
