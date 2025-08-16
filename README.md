# Cache Warmer

A high-performance, standalone cache warming tool written in Go. This tool helps pre-populate your application's cache by making HTTP requests to specified URLs, improving response times for your users.

## Features

- **Concurrent Processing**: Configurable number of worker goroutines for high throughput
- **Retry Logic**: Automatic retries with configurable delay for failed requests
- **Flexible Configuration**: YAML configuration files and command-line overrides
- **Metrics & Monitoring**: Built-in HTTP metrics endpoint for observability
- **Graceful Shutdown**: Proper signal handling for clean shutdowns
- **Multiple Run Modes**: Single run or continuous operation with intervals
- **Comprehensive Logging**: Structured logging with debug and verbose modes
- **Cross-Platform**: Builds for Linux, macOS, and Windows

## Quick Start

### 1. Build the Tool

```bash
# Build for your platform
make build

# Or build for all platforms
make build-all
```

### 2. Create Configuration

```bash
# Copy the example configuration
cp config.yaml.example config.yaml

# Edit the configuration file
vim config.yaml
```

### 3. Run Cache Warming

```bash
# Single run with configuration file
./build/cache-warmer -config config.yaml

# Single run with URLs from command line
./build/cache-warmer -urls "https://example.com,https://example.com/api"

# Continuous mode - run every 5 minutes
./build/cache-warmer -config config.yaml -interval 5m

# With verbose logging
./build/cache-warmer -config config.yaml -verbose
```

## Configuration

### Command Line Options

```
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
    Show help message
```

### Configuration File

The tool supports YAML configuration files. Here's a complete example:

```yaml
# List of URLs to warm
urls:
  - "https://example.com"
  - "https://example.com/api/health"
  - "https://example.com/api/products"

# Number of concurrent workers
workers: 10

# HTTP request timeout
timeout: 30s

# Retry configuration
retry_count: 3
retry_delay: 1s

# HTTP headers
user_agent: "Cache-Warmer/1.0"
headers:
  X-Cache-Warmer: "true"
  Authorization: "Bearer your-token"

# Redirect handling
follow_redirects: true
max_redirects: 5

# Success criteria
success_codes: [200, 201, 202, 204, 301, 302, 304]

# Metrics endpoint
metrics:
  enabled: true
  port: 8080
  path: "/metrics"
```

## Use Cases

### 1. Application Deployment

Warm your cache after deploying new code:

```bash
# Single run after deployment
./cache-warmer -config production-urls.yaml -workers 20
```

### 2. Scheduled Cache Warming

Use with cron for regular cache warming:

```bash
# Add to crontab - warm cache every hour
0 * * * * /path/to/cache-warmer -config /etc/cache-warmer/config.yaml
```

### 3. Load Testing Preparation

Warm cache before load testing:

```bash
# Warm with high concurrency
./cache-warmer -config loadtest-urls.yaml -workers 50 -timeout 10s
```

### 4. Continuous Background Warming

Run as a service for continuous warming:

```bash
# Run every 5 minutes
./cache-warmer -config config.yaml -interval 5m
```

## Monitoring and Metrics

When metrics are enabled, the tool exposes an HTTP endpoint with detailed statistics:

### Metrics Endpoint

```bash
# View metrics
curl http://localhost:8080/metrics

# Health check
curl http://localhost:8080/health
```

### Example Metrics Response

```json
{
  "metrics": {
    "request_counts": {
      "https://example.com": 150,
      "https://example.com/api": 150
    },
    "success_rates": {
      "https://example.com": 98.5,
      "https://example.com/api": 97.2
    },
    "total_requests": 300,
    "total_successes": 295,
    "total_failures": 5
  },
  "summary": {
    "total_urls": 2,
    "average_response_time_ms": 125.5,
    "overall_success_rate": 98.33,
    "requests_per_second": 12.5
  }
}
```

## Production Deployment

### As a Systemd Service

Create `/etc/systemd/system/cache-warmer.service`:

```ini
[Unit]
Description=Cache Warmer Service
After=network.target

[Service]
Type=simple
User=cache-warmer
WorkingDirectory=/opt/cache-warmer
ExecStart=/opt/cache-warmer/cache-warmer -config /etc/cache-warmer/config.yaml -interval 10m
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable cache-warmer
sudo systemctl start cache-warmer
```

### Docker Deployment

```dockerfile
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY cache-warmer .
COPY config.yaml .
CMD ["./cache-warmer", "-config", "config.yaml", "-interval", "5m"]
```

```bash
# Build and run
docker build -t cache-warmer .
docker run -d --name cache-warmer cache-warmer
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cache-warmer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cache-warmer
  template:
    metadata:
      labels:
        app: cache-warmer
    spec:
      containers:
      - name: cache-warmer
        image: cache-warmer:latest
        args: ["-config", "/config/config.yaml", "-interval", "10m"]
        volumeMounts:
        - name: config
          mountPath: /config
      volumes:
      - name: config
        configMap:
          name: cache-warmer-config
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile commands)

### Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd cache-warmer

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Run with example config
make run
```

### Development Commands

```bash
# Format code
make fmt

# Run linters
make check

# Build for all platforms
make build-all

# Run tests with coverage
make test-coverage

# Clean build artifacts
make clean
```

## Performance Tuning

### Worker Configuration

- **Low Load**: 5-10 workers for gentle cache warming
- **Medium Load**: 10-25 workers for balanced performance
- **High Load**: 25-100 workers for aggressive warming
- **Very High Load**: 100+ workers (monitor server capacity)

### Timeout Settings

- **Fast APIs**: 5-10 seconds
- **Standard APIs**: 15-30 seconds
- **Slow APIs**: 30-60 seconds
- **Heavy Pages**: 60+ seconds

### Memory Usage

The tool is designed to be memory-efficient:
- ~10MB base memory usage
- ~1KB per URL in configuration
- ~100KB per active worker
- Metrics data is bounded (last 100 measurements per URL)

## Troubleshooting

### Common Issues

**High failure rate**:
- Increase timeout duration
- Reduce number of workers
- Check retry configuration
- Verify URLs are accessible

**Memory usage**:
- Reduce number of workers
- Disable metrics if not needed
- Check for URL list size

**Performance issues**:
- Adjust worker count based on server capacity
- Tune timeout values
- Monitor server resources

### Debug Mode

Enable verbose logging for detailed information:

```bash
./cache-warmer -config config.yaml -verbose
```

### Health Checks

Monitor the health endpoint when metrics are enabled:

```bash
# Check if the service is running
curl -f http://localhost:8080/health || echo "Service is down"
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make check` to verify formatting and linting
6. Submit a pull request

## Support

For issues and questions:
- Check the troubleshooting section
- Review the configuration examples
- Open an issue on the repository