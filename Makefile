.PHONY: help validate fmt lint test tidy clean check-deps

# Default target - show available commands
help:
	@echo "Available commands:"
	@echo "  make validate    - Run all checks (format, lint, test, tidy)"
	@echo "  make fmt         - Format all Go code"
	@echo "  make lint        - Run golangci-lint"
	@echo "  make test        - Run all tests with coverage"
	@echo "  make tidy        - Clean up go.mod and go.sum"
	@echo "  make clean       - Remove build artifacts and cache"
	@echo "  make check-deps  - Check if required tools are installed"

# Validate everything - the main command you'll want to use!
validate: check-deps fmt tidy lint test
	@echo "✅ All validation checks passed!"

# Format all Go code
fmt:
	@echo "📝 Formatting Go code..."
	@go fmt ./...
	@echo "✅ Formatting complete"

# Run linter
lint:
	@echo "🔍 Running linter..."
	@golangci-lint run --config .golangci.yml
	@echo "✅ Linting complete"

# Run tests with coverage
test:
	@echo "🧪 Running tests..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "📊 Coverage report:"
	@go tool cover -func=coverage.out | tail -1
	@echo "✅ Tests complete"

# Clean up and verify go.mod
tidy:
	@echo "🧹 Tidying go.mod..."
	@go mod tidy
	@go mod verify
	@echo "✅ go.mod is clean"

# Clean build artifacts
clean:
	@echo "🗑️  Cleaning build artifacts..."
	@go clean -cache -testcache -modcache
	@rm -f coverage.out
	@echo "✅ Clean complete"

# Check if required tools are installed
check-deps:
	@echo "🔧 Checking dependencies..."
	@which go > /dev/null || (echo "❌ go is not installed" && exit 1)
	@which golangci-lint > /dev/null || (echo "❌ golangci-lint is not installed. Install with: brew install golangci-lint" && exit 1)
	@echo "✅ All required tools are installed"

