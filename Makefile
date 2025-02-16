# Makefile
.PHONY: test build clean

# Default target
all: test build

# Build the application
build:
	go build -v ./...

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	go clean
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
