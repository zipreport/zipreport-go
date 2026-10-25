.PHONY: test test-verbose test-coverage lint build install clean help examples

# Default target
help:
	@echo "Available targets:"
	@echo "  test          - Run tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  build         - Build the CLI"
	@echo "  install       - Install the CLI"
	@echo "  clean         - Remove build artifacts"
	@echo "  examples      - Vet every example module"

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report: go tool cover -html=coverage.out"

# Run linter
lint:
	golangci-lint run

# Build CLI
build:
	go build -o bin/zipreport ./cmd/zipreport

# Install CLI
install:
	go install ./cmd/zipreport

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Vet every example module (each has its own go.mod)
examples:
	@for d in examples/*/; do \
		echo "vet $$d"; \
		(cd "$$d" && GOFLAGS=-mod=readonly go vet ./...) || exit 1; \
	done
