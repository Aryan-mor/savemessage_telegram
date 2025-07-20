.PHONY: run test cover lint clean build help

# Default target
help:
	@echo "Available commands:"
	@echo "  run     - Run the bot"
	@echo "  test    - Run all tests with verbose output"
	@echo "  cover   - Run tests with coverage report"
	@echo "  lint    - Run golangci-lint"
	@echo "  clean   - Clean build artifacts and coverage files"
	@echo "  build   - Build the bot binary"
	@echo "  deps    - Install dependencies"

# Run the bot
run:
	go run main.go

# Run all tests with verbose output
test:
	go test ./internal/... -v

# Run tests with coverage report
cover:
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -1

# Run golangci-lint
lint:
	golangci-lint run

# Clean build artifacts and coverage files
clean:
	rm -rf coverage.out coverage.html
	rm -rf main bot_modular
	rm -rf cmd/modular/modular

# Build the bot binary
build:
	go build -o bot_modular main.go

# Build modular version
build-modular:
	go build -o cmd/modular/modular cmd/modular/main.go

# Install dependencies
deps:
	go mod tidy
	go mod download

# Install development tools
dev-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run tests with race detection
test-race:
	go test -race ./internal/...

# Run benchmarks
bench:
	go test -bench=. ./internal/...

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./... 