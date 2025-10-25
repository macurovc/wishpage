.PHONY: help validate fmt lint test tidy clean check-deps fmt-css fmt-css-check lint-css lint-css-fix npm-install

# Default target - show available commands
help:
	@echo "Available commands:"
	@echo "  make validate    - Run all checks (format, lint, test, tidy)"
	@echo "  make fmt         - Format all Go code"
	@echo "  make fmt-css     - Format CSS files with Prettier"
	@echo "  make fmt-css-check - Check CSS formatting without modifying files"
	@echo "  make lint        - Run golangci-lint"
	@echo "  make lint-css    - Run Stylelint on CSS files"
	@echo "  make lint-css-fix - Run Stylelint and auto-fix CSS issues"
	@echo "  make test        - Run all tests with coverage"
	@echo "  make tidy        - Clean up go.mod and go.sum"
	@echo "  make clean       - Remove build artifacts and cache"
	@echo "  make check-deps  - Check if required tools are installed"
	@echo "  make npm-install - Install npm dependencies for CSS tools"

# Validate everything - the main command you'll want to use!
validate: check-deps fmt fmt-css tidy lint lint-css test
	@echo "✅ All validation checks passed!"

# Format all Go code
fmt:
	@echo "📝 Formatting Go code..."
	@go fmt ./...
	@echo "✅ Formatting complete"

# Format CSS files
fmt-css: npm-install
	@echo "📝 Formatting CSS files..."
	@npm run format:css
	@echo "✅ CSS formatting complete"

# Check CSS formatting without modifying files
fmt-css-check: npm-install
	@echo "🔍 Checking CSS formatting..."
	@npm run format:css:check
	@echo "✅ CSS formatting check complete"

# Run Go linter
lint:
	@echo "🔍 Running Go linter..."
	@golangci-lint run --config .golangci.yml
	@echo "✅ Go linting complete"

# Run CSS linter
lint-css: npm-install
	@echo "🔍 Running CSS linter..."
	@npm run lint:css
	@echo "✅ CSS linting complete"

# Run CSS linter with auto-fix
lint-css-fix: npm-install
	@echo "🔧 Running CSS linter with auto-fix..."
	@npm run lint:css:fix
	@echo "✅ CSS issues fixed"

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
	@rm -rf node_modules
	@echo "✅ Clean complete"

# Install npm dependencies
npm-install:
	@if [ ! -d "node_modules" ]; then \
		echo "📦 Installing npm dependencies..."; \
		npm install; \
	fi

# Check if required tools are installed
check-deps:
	@echo "🔧 Checking dependencies..."
	@which go > /dev/null || (echo "❌ go is not installed" && exit 1)
	@which golangci-lint > /dev/null || (echo "❌ golangci-lint is not installed. Install with: brew install golangci-lint" && exit 1)
	@which npm > /dev/null || (echo "❌ npm is not installed. Install Node.js from: https://nodejs.org/" && exit 1)
	@echo "✅ All required tools are installed"

