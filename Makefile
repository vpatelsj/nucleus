.PHONY: build install cleanup clean test help

# Build the nucleus binary
build:
	@echo "Building nucleus..."
	@go build -o bin/nucleus cmd/nucleus/main.go
	@echo "✓ Build complete: bin/nucleus"

# Install Kubernetes master node with Cilium CNI
install: build
	@echo "Starting Kubernetes installation..."
	@sudo ./bin/nucleus install

# Cleanup Kubernetes installation
cleanup: build
	@echo "Starting cleanup..."
	@sudo ./bin/nucleus cleanup

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run || go vet ./...

# Display help
help:
	@echo "Nucleus - Kubernetes master node installer with Cilium CNI"
	@echo ""
	@echo "Available targets:"
	@echo "  make build    - Build the nucleus binary"
	@echo "  make install  - Install Kubernetes master node"
	@echo "  make cleanup  - Remove Kubernetes installation"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make test     - Run tests"
	@echo "  make fmt      - Format code"
	@echo "  make lint     - Run linter"
	@echo "  make help     - Display this help message"
