# Cache Warmer Makefile
# Provides common build, test, and deployment commands

# Variables
BINARY_NAME=cache-warmer
VERSION=1.0.0
BUILD_DIR=build
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Cache Warmer v$(VERSION) - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build commands
.PHONY: build
build: ## Build the cache warmer binary
	@echo "Building cache warmer..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-all
build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	
	@echo "Built binaries for all platforms in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

# Development commands
.PHONY: run
run: ## Run the cache warmer with example configuration
	go run . -config config.yaml.example -verbose

.PHONY: run-once
run-once: ## Run cache warmer once with example URLs
	go run . -urls "https://httpbin.org/status/200,https://httpbin.org/delay/1" -verbose

.PHONY: run-continuous
run-continuous: ## Run cache warmer continuously every 30 seconds
	go run . -config config.yaml.example -interval 30s -verbose

# Testing commands
.PHONY: test
test: ## Run all tests
	go test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	go test -race -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code quality commands
.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint to be installed)
	golangci-lint run

.PHONY: check
check: fmt vet ## Run formatting and vetting

# Dependency management
.PHONY: deps
deps: ## Download dependencies
	go mod download

.PHONY: deps-update
deps-update: ## Update dependencies
	go mod tidy
	go get -u ./...

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	go mod verify

# Cleanup commands
.PHONY: clean
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Installation commands
.PHONY: install
install: build ## Install the binary to $GOPATH/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

.PHONY: uninstall
uninstall: ## Remove the binary from $GOPATH/bin
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Docker commands (optional)
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t cache-warmer:$(VERSION) .
	docker tag cache-warmer:$(VERSION) cache-warmer:latest

.PHONY: docker-run
docker-run: ## Run cache warmer in Docker
	docker run --rm -v $(PWD)/config.yaml.example:/app/config.yaml cache-warmer:latest

# Release commands
.PHONY: release
release: clean build-all ## Create a release build
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && \
	tar -czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar -czf $(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	zip $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release archives created in $(BUILD_DIR)/"

# Development utilities
.PHONY: example-config
example-config: ## Copy example config to config.yaml for local development
	cp config.yaml.example config.yaml
	@echo "Copied config.yaml.example to config.yaml"

.PHONY: watch
watch: ## Watch for changes and rebuild (requires 'entr' tool)
	find . -name "*.go" | entr -r make run

# Show project information
.PHONY: info
info: ## Show project information
	@echo "Cache Warmer v$(VERSION)"
	@echo "Go version: $(shell go version)"
	@echo "Build target: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "Source files: $(words $(GO_FILES)) Go files"