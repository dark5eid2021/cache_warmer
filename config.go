package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the configuration for the cache warming tool
type Config struct {
	// URLs is the list of URLs to warm
	URLs []string `yaml:"urls"`

	// Workers is the number of concurrent workers
	Workers int `yaml:"workers"`

	// Timeout is the HTTP request timeout
	Timeout time.Duration `yaml:"timeout"`

	// RetryCount is the number of retries for failed requests
	RetryCount int `yaml:"retry_count"`

	// RetryDelay is the delay between retries
	RetryDelay time.Duration `yaml:"retry_delay"`

	// UserAgent is the User-Agent header to use for requests
	UserAgent string `yaml:"user_agent"`

	// Headers contains custom headers to include in requests
	Headers map[string]string `yaml:"headers"`

	// FollowRedirects determines if redirects should be followed
	FollowRedirects bool `yaml:"follow_redirects"`

	// MaxRedirects is the maximum number of redirects to follow
	MaxRedirects int `yaml:"max_redirects"`

	// SuccessCodes defines which HTTP status codes are considered successful
	SuccessCodes []int `yaml:"success_codes"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics"`
}

// MetricsConfig contains configuration for metrics collection
type MetricsConfig struct {
	// Enabled determines if metrics collection is enabled
	Enabled bool `yaml:"enabled"`

	// Port is the port to expose metrics on
	Port int `yaml:"port"`

	// Path is the path to expose metrics on
	Path string `yaml:"path"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		URLs:            []string{},
		Workers:         10,
		Timeout:         30 * time.Second,
		RetryCount:      3,
		RetryDelay:      1 * time.Second,
		UserAgent:       "Cache-Warmer/1.0",
		Headers:         make(map[string]string),
		FollowRedirects: true,
		MaxRedirects:    5,
		SuccessCodes:    []int{200, 201, 202, 204, 301, 302, 304},
		Metrics: MetricsConfig{
			Enabled: false,
			Port:    8080,
			Path:    "/metrics",
		},
	}
}

// LoadConfig loads configuration from file and applies command line overrides
func LoadConfig(configFile, urlsOverride string, workersOverride int, timeoutOverride time.Duration) (*Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// Load from file if it exists
	if configFile != "" {
		if err := config.LoadFromFile(configFile); err != nil {
			// If config file is explicitly specified but doesn't exist, return error
			if configFile != "config.yaml" {
				return nil, fmt.Errorf("failed to load config file %s: %v", configFile, err)
			}
			// If using default config file name and it doesn't exist, that's OK
			// We'll just use defaults
		}
	}

	// Apply command line overrides
	if urlsOverride != "" {
		urls := strings.Split(urlsOverride, ",")
		// Trim whitespace from each URL
		for i, u := range urls {
			urls[i] = strings.TrimSpace(u)
		}
		config.URLs = urls
	}

	if workersOverride > 0 {
		config.Workers = workersOverride
	}

	if timeoutOverride > 0 {
		config.Timeout = timeoutOverride
	}

	return config, nil
}

// LoadFromFile loads configuration from a YAML file
func (c *Config) LoadFromFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Create a temporary config to unmarshal into
	var fileConfig Config
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// Merge file config with current config (file config takes precedence)
	if len(fileConfig.URLs) > 0 {
		c.URLs = fileConfig.URLs
	}
	if fileConfig.Workers > 0 {
		c.Workers = fileConfig.Workers
	}
	if fileConfig.Timeout > 0 {
		c.Timeout = fileConfig.Timeout
	}
	if fileConfig.RetryCount > 0 {
		c.RetryCount = fileConfig.RetryCount
	}
	if fileConfig.RetryDelay > 0 {
		c.RetryDelay = fileConfig.RetryDelay
	}
	if fileConfig.UserAgent != "" {
		c.UserAgent = fileConfig.UserAgent
	}
	if len(fileConfig.Headers) > 0 {
		c.Headers = fileConfig.Headers
	}
	if fileConfig.MaxRedirects > 0 {
		c.MaxRedirects = fileConfig.MaxRedirects
	}
	if len(fileConfig.SuccessCodes) > 0 {
		c.SuccessCodes = fileConfig.SuccessCodes
	}

	// Set boolean values (these can be explicitly false)
	c.FollowRedirects = fileConfig.FollowRedirects

	// Merge metrics config
	if fileConfig.Metrics.Port > 0 {
		c.Metrics.Port = fileConfig.Metrics.Port
	}
	if fileConfig.Metrics.Path != "" {
		c.Metrics.Path = fileConfig.Metrics.Path
	}
	c.Metrics.Enabled = fileConfig.Metrics.Enabled

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check if we have at least one URL
	if len(c.URLs) == 0 {
		return fmt.Errorf("at least one URL must be specified")
	}

	// Validate each URL
	for i, urlStr := range c.URLs {
		if urlStr == "" {
			return fmt.Errorf("URL at index %d is empty", i)
		}

		// Parse URL to check if it's valid
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return fmt.Errorf("invalid URL at index %d (%s): %v", i, urlStr, err)
		}

		// Check if scheme is http or https
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("URL at index %d (%s) must use http or https scheme", i, urlStr)
		}

		// Check if host is present
		if parsedURL.Host == "" {
			return fmt.Errorf("URL at index %d (%s) must have a host", i, urlStr)
		}
	}

	// Validate workers count
	if c.Workers <= 0 {
		return fmt.Errorf("workers count must be positive, got %d", c.Workers)
	}

	if c.Workers > 1000 {
		return fmt.Errorf("workers count is too high (%d), maximum is 1000", c.Workers)
	}

	// Validate timeout
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}

	// Validate retry configuration
	if c.RetryCount < 0 {
		return fmt.Errorf("retry count must be non-negative, got %d", c.RetryCount)
	}

	if c.RetryDelay < 0 {
		return fmt.Errorf("retry delay must be non-negative, got %v", c.RetryDelay)
	}

	// Validate redirect configuration
	if c.MaxRedirects < 0 {
		return fmt.Errorf("max redirects must be non-negative, got %d", c.MaxRedirects)
	}

	// Validate success codes
	if len(c.SuccessCodes) == 0 {
		return fmt.Errorf("at least one success code must be specified")
	}

	for _, code := range c.SuccessCodes {
		if code < 100 || code >= 600 {
			return fmt.Errorf("invalid HTTP status code: %d", code)
		}
	}

	// Validate metrics configuration
	if c.Metrics.Enabled {
		if c.Metrics.Port <= 0 || c.Metrics.Port > 65535 {
			return fmt.Errorf("metrics port must be between 1 and 65535, got %d", c.Metrics.Port)
		}

		if c.Metrics.Path == "" {
			return fmt.Errorf("metrics path cannot be empty")
		}

		if !strings.HasPrefix(c.Metrics.Path, "/") {
			return fmt.Errorf("metrics path must start with '/', got %s", c.Metrics.Path)
		}
	}

	return nil
}

// IsSuccessCode checks if the given HTTP status code is considered successful
func (c *Config) IsSuccessCode(code int) bool {
	for _, successCode := range c.SuccessCodes {
		if code == successCode {
			return true
		}
	}
	return false
}
