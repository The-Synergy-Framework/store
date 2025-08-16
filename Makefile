# Store Library Makefile
# Persistent storage framework for the Synergy Framework

# Variables
COVERAGE_DIR=./coverage
LINT_CONFIG=.golangci.yml
CORE_PATH=../core

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Store Library Development Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development tasks
.PHONY: build
build: ## Build/compile the library to check for compilation errors
	@echo "Building store library..."
	go build ./...
	@echo "Build successful - no compilation errors"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(COVERAGE_DIR)
	@go clean -cache -testcache
	@echo "Clean complete"

# Testing tasks
.PHONY: test
test: ## Run all tests
	@echo "Running all tests..."
	go test -v ./...

.PHONY: test-all
test-all: test-race test-coverage test-benchmark ## Run all tests including race detection, coverage, and benchmarks

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race -v ./...

.PHONY: test-short
test-short: ## Run only short tests
	@echo "Running short tests..."
	go test -short -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

.PHONY: test-benchmark
test-benchmark: ## Run benchmark tests
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Code quality tasks
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (if installed)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running linter with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

.PHONY: check
check: fmt vet lint ## Run all code quality checks

# Dependency management
.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies and check core module
	@echo "Verifying dependencies..."
	go mod verify
	@echo "Checking core module dependency..."
	@if [ -d "$(CORE_PATH)" ]; then \
		echo "✓ Core module found at $(CORE_PATH)"; \
	else \
		echo "✗ Core module not found at $(CORE_PATH)"; \
		echo "  Make sure core library is checked out at the same level"; \
		exit 1; \
	fi

# Security tasks
.PHONY: security
security: ## Run security checks
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/cosmos/gosec/v2/cmd/gosec@latest"; \
	fi

.PHONY: vulncheck
vulncheck: ## Check for vulnerabilities
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

.PHONY: security-audit
security-audit: security vulncheck ## Run comprehensive security audit

# Documentation tasks
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server on http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not found. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Development workflow
.PHONY: dev-setup
dev-setup: deps install-tools ## Setup development environment
	@echo "Setting up Store development environment..."
	@if [ ! -f $(LINT_CONFIG) ]; then \
		echo "Copying golangci-lint config from core..."; \
		cp $(CORE_PATH)/$(LINT_CONFIG) . 2>/dev/null || echo "Creating default config..."; \
	fi
	@echo "Store development environment ready"

.PHONY: pre-commit
pre-commit: fmt vet lint test-short ## Run pre-commit checks
	@echo "Pre-commit checks completed"

.PHONY: ci
ci: deps check test-all ## Run CI pipeline
	@echo "CI pipeline completed"

# Utility tasks
.PHONY: version
version: ## Show version information
	@echo "Version: $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')"
	@echo "Go version: $(shell go version)"
	@echo "Build time: $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')"

.PHONY: info
info: ## Show project information
	@echo "Project: Store Library - Persistent Storage"
	@echo "Module: $(shell go list -m)"
	@echo "Go version: $(shell go version | cut -d' ' -f3)"
	@echo "OS/Arch: $(shell go env GOOS)/$(shell go env GOARCH)"
	@echo "Core dependency: $(CORE_PATH)"
	@echo ""
	@echo "Store components:"
	@echo "  - SQL: driver registry, database, transactions"
	@echo "  - SQL: query builder, pagination, repositories"
	@echo "  - Filestore: interface and filesystem implementation"
	@echo "  - Future: Document store & KV abstractions"

# Cleanup tasks
.PHONY: clean-all
clean-all: clean ## Clean everything including go mod cache
	@echo "Cleaning everything..."
	go clean -modcache
	@echo "All clean"

# Install development tools
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/godoc@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Development tools installed"

# Watch mode for development
.PHONY: watch
watch: ## Watch for changes and run tests (requires fswatch)
	@echo "Watching for changes..."
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -o . | xargs -n1 -I{} make test-short; \
	else \
		echo "fswatch not found. Install with: brew install fswatch (macOS) or apt-get install fswatch (Ubuntu)"; \
	fi

.PHONY: watch-coverage
watch-coverage: ## Watch for changes and run tests with coverage
	@echo "Watching for changes with coverage..."
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -o . | xargs -n1 -I{} make test-coverage; \
	else \
		echo "fswatch not found. Install with: brew install fswatch (macOS) or apt-get install fswatch (Ubuntu)"; \
	fi

# Quick development shortcuts
.PHONY: quick
quick: fmt test-short ## Quick development check (format + short tests)

.PHONY: full
full: ci ## Full check (equivalent to CI pipeline)

# Default target
.DEFAULT_GOAL := help 