.PHONY: build run clean dev install

# Build variables
BINARY_NAME=logvoyant
BUILD_DIR=./bin
MAIN_PATH=./cmd/logvoyant

# Go build flags
LDFLAGS=-ldflags "-s -w"

# Build for current platform
build: deps
	@echo "🔨 Building LogVoyant..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all:
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "✅ Multi-platform build complete"

# Run in development mode
dev:
	@echo "🚀 Starting LogVoyant in dev mode..."
	go run $(MAIN_PATH) start --port 3100

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
watch:
	@echo "👀 Starting with hot reload..."
	air

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed and go.sum generated"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f logvoyant.db
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Install locally
install: build
	@echo "📥 Installing LogVoyant..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"

# Uninstall
uninstall:
	@echo "🗑️  Uninstalling LogVoyant..."
	rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Uninstalled"

# Format code
fmt:
	@echo "✨ Formatting code..."
	go fmt ./...
	@echo "✅ Format complete"

# Lint code (requires golangci-lint)
lint:
	@echo "🔍 Linting code..."
	golangci-lint run
	@echo "✅ Lint complete"

# Run with sample log file
demo:
	@echo "🎬 Starting demo with sample logs..."
	@echo "[INFO] Sample log line 1" > /tmp/demo.log
	@echo "[ERROR] Connection timeout to database" >> /tmp/demo.log
	@echo "[WARN] High memory usage detected" >> /tmp/demo.log
	go run $(MAIN_PATH) start /tmp/demo.log

# Docker build (optional)
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t logvoyant:latest .
	@echo "✅ Docker build complete"

# Show help
help:
	@echo "LogVoyant - Build Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build         Build for current platform"
	@echo "  make build-all     Build for all platforms"
	@echo "  make dev           Run in development mode"
	@echo "  make watch         Run with hot reload (requires air)"
	@echo "  make test          Run tests"
	@echo "  make clean         Remove build artifacts"
	@echo "  make install       Install to /usr/local/bin"
	@echo "  make demo          Run with sample logs"
	@echo "  make help          Show this help"