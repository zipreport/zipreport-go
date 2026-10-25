# Development

This guide covers setting up a development environment, running tests, and contributing to ZipReport Go.

## Prerequisites

- Go 1.24+
- golangci-lint v2 (CI pins v2.12.2; the v2 config in `.golangci.yml` is rejected by v1)
- zipreport-server (for testing PDF generation)

## Getting Started

```bash
# Clone the repository
git clone https://github.com/zipreport/zipreport-go
cd zipreport-go

# Download dependencies
go mod download

# Build CLI
make build

# Run tests
make test
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make test` | Run tests |
| `make test-verbose` | Run tests with verbose output |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run golangci-lint |
| `make build` | Build CLI to `bin/` |
| `make install` | Install CLI to GOPATH |
| `make clean` | Remove build artifacts |

## Running Tests

### All Tests

```bash
make test
# or
go test ./...
```

`go test -short ./...` skips the CLI end-to-end test, which builds the binary and needs `go` on `PATH`.

### With Coverage

```bash
make test-coverage
# or
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### View HTML Coverage

```bash
go tool cover -html=coverage.out
```

### Specific Package

```bash
go test ./pkg/api/...
go test ./pkg/report/...
go test ./pkg/template/...
```

### Verbose Output

```bash
make test-verbose
# or
go test -v ./...
```

## Linting

```bash
make lint
# or
golangci-lint run
```

## Project Structure

```
zipreport-go/
├── cmd/zipreport/          # CLI application
│   ├── main.go             # Entry point
│   ├── build.go            # build command
│   ├── debug.go            # debug command
│   ├── info.go             # info command
│   ├── list.go             # list command
│   └── version.go          # version command
├── pkg/
│   ├── api/                # High-level API
│   │   ├── zipreport.go    # ZipReport, MIMEReport
│   │   └── zipreport_test.go
│   ├── fileutils/          # File system abstractions
│   │   ├── interface.go    # FsInterface definition
│   │   ├── zipfs.go        # In-memory ZIP filesystem
│   │   ├── diskfs.go       # Disk-based filesystem
│   │   ├── pathcache.go    # Path caching
│   │   ├── overlayfs.go    # In-memory writes over a read-only base
│   │   ├── doc.go          # Package documentation
│   │   └── *_test.go
│   ├── processor/          # Rendering backends
│   │   ├── interface.go    # Backend interfaces
│   │   ├── backend_server.go # zipreport-server backend
│   │   ├── backend_local.go  # Local backend (stub)
│   │   ├── mime.go         # MIME processor
│   │   ├── zipreport.go    # ZipReportProcessor
│   │   └── *_test.go
│   ├── report/             # Report types
│   │   ├── reportfile.go   # ReportFile type
│   │   ├── job.go          # ReportJob configuration
│   │   ├── const.go        # Constants
│   │   ├── loader.go       # Report loading
│   │   ├── builder.go      # .zpt building
│   │   └── *_test.go
│   └── template/           # Template rendering
│       ├── render.go       # MiyaRender
│       ├── environment.go  # EnvironmentWrapper
│       ├── filters.go      # Built-in filters
│       ├── loader.go       # Template loader
│       └── *_test.go
├── examples/               # Standalone examples
├── docs/                   # Documentation
└── output/                 # Example outputs
```

## Testing Guidelines

### Test Coverage

Target: 75%+ for library code in `pkg/`.

Current coverage can be checked with:

```bash
make test-coverage
```

### Test Patterns

Use table-driven tests:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty", "", ""},
        {"simple", "hello", "HELLO"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Function(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Mocking

Mock external dependencies:

```go
// Mock HTTP server
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/pdf")
    w.Write([]byte("%PDF-1.4..."))
}))
defer server.Close()

backend := processor.NewServerBackend(server.URL, "test-key")
```

## Adding New Features

1. **Create a branch**

```bash
git checkout -b feature/my-feature
```

2. **Write tests first**

```bash
# Create test file
touch pkg/mypackage/myfeature_test.go
```

3. **Implement feature**

4. **Run tests and linting**

```bash
make test lint
```

5. **Update documentation**

6. **Create pull request**

## Code Style

- Follow standard Go conventions
- Run `golangci-lint` before committing
- Keep functions focused and small
- Prefer explicit error handling over panics
- Use meaningful variable names
- Add comments for exported functions

## Commit Messages

Use clear, descriptive commit messages:

```
Add currency filter for template rendering

- Implement currency filter with locale support
- Add tests for various number formats
- Update documentation
```

## Pull Request Process

1. Ensure tests pass: `make test`
2. Ensure linting passes: `make lint`
3. Update documentation if needed
4. Add tests for new functionality
5. Request review

## CI/CD

GitHub Actions runs on every push and pull request:

- **Test** - Runs all tests with coverage
- **Lint** - Runs golangci-lint
- **Build Check** - Builds for all platforms

## Releasing

Releases are created via GitHub Actions when a tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers:

- Building binaries for all platforms
- Creating GitHub release with binaries
- Updating Go package registry
