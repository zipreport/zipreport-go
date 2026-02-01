# CLAUDE.md

This file provides guidance for Claude Code when working on this repository.

## Project Overview

ZipReport Go is a Go implementation of ZipReport - an HTML to PDF report generation library using Jinja2-compatible templates. It's a port of the Python ZipReport library.

## Repository Structure

```
zipreport-go/
├── cmd/zipreport/          # CLI application (build, info, list, debug commands)
├── pkg/
│   ├── api/                # High-level API (ZipReport, MIMEReport)
│   ├── fileutils/          # File system abstractions (ZipFs, DiskFs, PathCache)
│   ├── processor/          # Rendering backends (ServerBackend, MIMEProcessor)
│   ├── report/             # Report types (ReportFile, ReportJob, Manifest)
│   └── template/           # Miya template rendering and filters
├── examples/               # Standalone example modules (each has own go.mod)
├── output/                 # Generated output files from examples
└── .github/workflows/      # CI/CD pipelines
```

## Key Dependencies

- `github.com/zipreport/miya` - Jinja2-compatible template engine for Go
- `github.com/spf13/cobra` - CLI framework

## Common Commands

```bash
# Using Make (recommended)
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linter
make build          # Build CLI

# Manual commands
go test ./pkg/...
go test ./pkg/... -coverprofile=coverage.out
golangci-lint run
go build ./cmd/zipreport

# Run an example (from example directory)
cd examples/simple
ZIPREPORT_URL="http://localhost:6543" go run main.go output.pdf
```

## Architecture Notes

### File Systems
- `ZipFs` - In-memory ZIP file system for .zpt files
- `DiskFs` - Disk-based file system for template directories
- Both implement a common interface for file operations

### Processors
- `ServerBackend` - Renders PDFs via zipreport-server HTTP API
- `LocalBackend` - Stub for future local rendering (not yet implemented)
- `MIMEProcessor` - Generates MIME email messages with embedded resources

### Template Rendering
- Uses miya for Jinja2-compatible templates
- Custom filters: `png`, `gif`, `jpg`, `jpeg`, `svg` for dynamic image generation
- `EnvironmentWrapper` interface for customizing the template environment

## Testing Guidelines

- All core packages in `pkg/` should have tests
- Target coverage: 75%+ for library code
- Use table-driven tests where appropriate
- Mock external dependencies (HTTP servers, file systems)

## Examples

Examples are standalone Go modules in `examples/`. Each has:
- Own `go.mod` with `replace` directive for local development
- Local `template/` directory with report templates
- `README.md` with usage instructions

To run examples, a zipreport-server instance is needed (except for `newsletter` which generates MIME emails).

## Code Style

- Follow standard Go conventions
- Use `golangci-lint` for linting
- Keep functions focused and small
- Prefer explicit error handling over panics
