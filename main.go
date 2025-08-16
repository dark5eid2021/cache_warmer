package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Version information for the cache warming tool
const Version = "0.0.1"

func main() {
	// Define command line flags for configuration
	var (
		configFile = flag.String("config", "config.yaml", "Path to configuration file")
		urls       = flag.String("urls", "", "Comma-separated list of URLs to warm (overrides config file)")
		workers    = flag.Int("workers", 10, "Number of concurrent workers")
		interval   = flag.Duration("interval", 0, "Interval between warming cycles (0 = run once)")
		timeout    = flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		version    = flag.Bool("version", false, "Show version information")
		help       = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("Cache Warmer v%s\n", Version)
		return
	}

	// Handle help flag
	if *help {
		printUsage()
		return
	}

	// Initialize logger
	logger := NewLogger(*verbose)
	logger.Info("Starting Cache Warmer v%s", Version)

	// Load configuration
	config, err := LoadConfig(*configFile, *urls, *workers, *timeout)
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		logger.Error("Invalid configuration: %v", err)
		os.Exit(1)
	}

	logger.Info("Loaded configuration with %d URLs and %d workers",
		len(config.URLs), config.Workers)

	// Create cache warmer instance
	warmer := NewCacheWarmer(config, logger)

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the cache warmer
	if *interval > 0 {
		// Continuous mode - run at specified intervals
		logger.Info("Running in continuous mode with %v interval", *interval)
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()

		// Run initial warming
		warmer.WarmCache()

		for {
			select {
			case <-ticker.C:
				logger.Info("Starting scheduled cache warming cycle")
				warmer.WarmCache()
			case sig := <-sigChan:
				logger.Info("Received signal %v, shutting down gracefully", sig)
				warmer.Shutdown()
				return
			}
		}
	} else {
		// Single run mode
		logger.Info("Running in single execution mode")

		// Set up signal handler for graceful shutdown during single run
		go func() {
			<-sigChan
			logger.Info("Received shutdown signal, stopping cache warming")
			warmer.Shutdown()
		}()

		warmer.WarmCache()
		logger.Info("Cache warming completed")
	}
}

// printUsage displays comprehensive usage information
func printUsage() {
	fmt.Printf(`Cache Warmer v%s - A tool for preloading cache by making HTTP requests

USAGE:
    cache-warmer [OPTIONS]

OPTIONS:
    -config string
        Path to configuration file (default "config.yaml")
    -urls string
        Comma-separated list of URLs to warm (overrides config file)
    -workers int
        Number of concurrent workers (default 10)
    -interval duration
        Interval between warming cycles, 0 = run once (default 0)
        Examples: 5m, 1h, 30s
    -timeout duration
        HTTP request timeout (default 30s)
    -verbose
        Enable verbose logging
    -version
        Show version information
    -help
        Show this help message

EXAMPLES:
    # Run once with URLs from command line
    cache-warmer -urls "http://example.com,http://example.com/api"

    # Run continuously every 5 minutes
    cache-warmer -config myconfig.yaml -interval 5m -workers 20

    # Single run with verbose output
    cache-warmer -config config.yaml -verbose

CONFIGURATION FILE:
    The tool supports YAML configuration files. See config.yaml.example for format.
    Command line options override configuration file settings.

EXIT CODES:
    0 - Success
    1 - Configuration error
    2 - Runtime error
`, Version)
}
