# ZipReport Go

[![Go Reference](https://pkg.go.dev/badge/github.com/zipreport/zipreport-go.svg)](https://pkg.go.dev/github.com/zipreport/zipreport-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/zipreport/zipreport-go)](https://goreportcard.com/report/github.com/zipreport/zipreport-go)
[![CI](https://github.com/zipreport/zipreport-go/actions/workflows/ci.yml/badge.svg)](https://github.com/zipreport/zipreport-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/zipreport/zipreport-go)](https://github.com/zipreport/zipreport-go/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Documentation](https://img.shields.io/badge/docs-zipreport.github.io-blue)](https://zipreport.github.io/zipreport-go/)

A Go implementation of ZipReport - HTML to PDF report generation using Jinja2-compatible templates.

## Features

- **Jinja2-compatible templating** via [miya](https://github.com/zipreport/miya)
- **PDF generation** via zipreport-server
- **MIME email generation** with embedded resources
- **Dynamic image filters** (png, gif, jpg, svg)
- **CLI tool** for building and debugging templates

## Installation

### CLI Tool

**Using Go install (recommended):**
```bash
go install github.com/zipreport/zipreport-go/cmd/zipreport@latest
```

**Pre-built binaries:**

Download the latest release for your platform from the [Releases page](https://github.com/zipreport/zipreport-go/releases).

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux    | x86_64       | `zipreport_linux_amd64.tar.gz` |
| Linux    | ARM64        | `zipreport_linux_arm64.tar.gz` |
| macOS    | x86_64       | `zipreport_darwin_amd64.tar.gz` |
| macOS    | Apple Silicon| `zipreport_darwin_arm64.tar.gz` |
| Windows  | x86_64       | `zipreport_windows_amd64.zip` |

Extract and move to your PATH:
```bash
# Linux/macOS
tar -xzf zipreport_linux_amd64.tar.gz
sudo mv zipreport /usr/local/bin/

# Windows (PowerShell)
Expand-Archive zipreport_windows_amd64.zip
Move-Item zipreport.exe C:\Windows\System32\
```

**From source:**
```bash
git clone https://github.com/zipreport/zipreport-go
cd zipreport-go
go install ./cmd/zipreport
```

### Library

```bash
go get github.com/zipreport/zipreport-go
```

## Quick Start

```go
package main

import (
    "os"
    "github.com/zipreport/zipreport-go/pkg/api"
    "github.com/zipreport/zipreport-go/pkg/report"
)

func main() {
    // Load report template
    zpt, _ := report.Load("report.zpt")

    // Template data
    data := map[string]interface{}{
        "title": "My Report",
        "items": []string{"Item 1", "Item 2", "Item 3"},
    }

    // Render to PDF
    client := api.NewZipReport("https://localhost:6543", "api-key")
    result := client.RenderDefaults(zpt, data, nil)

    if result.Success {
        os.WriteFile("output.pdf", result.Report, 0644)
    }
}
```

## CLI Usage

```bash
# Build a .zpt file from a template directory
zipreport build ./my-template output.zpt

# List .zpt files in a directory
zipreport list ./reports

# Display report metadata
zipreport info report.zpt

# Start debug server for template development
zipreport debug ./my-template
```

## Project Structure

```
zipreport-go/
├── cmd/zipreport/     # CLI application
├── pkg/
│   ├── fileutils/     # File system abstractions (ZipFs, DiskFs)
│   ├── report/        # Report types (ReportFile, ReportJob)
│   ├── template/      # Miya template rendering
│   ├── processor/     # PDF/MIME backends
│   └── api/           # High-level API
└── examples/          # Standalone usage examples
```

## Examples

Each example is a standalone Go module with its own dependencies and local template files.

| Example | Description |
|---------|-------------|
| [simple](./examples/simple) | Basic PDF generation |
| [newsletter](./examples/newsletter) | MIME email generation |
| [filter_example](./examples/filter_example) | Dynamic image filters |
| [print_css_chrome](./examples/print_css_chrome) | CSS Paged Media |
| [pagedjs](./examples/pagedjs) | PagedJS with dynamic charts |
| [analytics_dashboard](./examples/analytics_dashboard) | Dashboard with charts |
| [executive_report](./examples/executive_report) | Business report |
| [product_brochure](./examples/product_brochure) | Product brochure |

Run an example:

```bash
cd examples/simple
export ZIPREPORT_URL="http://localhost:6543"
go run main.go output.pdf
```

## Packages

### `pkg/api`

High-level API for report generation:

```go
// PDF via zipreport-server
client := api.NewZipReport(url, apiKey)
result := client.RenderDefaults(zpt, data, nil)

// With custom backend
client := api.NewZipReportWithBackend(customBackend)

// MIME email
client := api.NewMIMEReport()
result := client.RenderDefaults(zpt, data, nil)
```

### `pkg/report`

Report loading and job configuration:

```go
// Load from file or directory
zpt, err := report.Load("report.zpt")
zpt, err := report.LoadDir("./template")

// Configure rendering job
job := report.NewReportJob(zpt)
job.SetPageSize(report.PageA4)
job.SetMargins(report.MarginStandard)
job.SetLandscape(false)
job.UseJSEvent(true)

// Build .zpt from directory
result := report.Build("./source", "output.zpt", nil)
```

### `pkg/template`

Template rendering with miya:

```go
renderer := template.NewMiyaRender(zpt)
output, err := renderer.Render(data)

// With custom environment wrapper
wrapper := template.FuncEnvironmentWrapper(func(env *miya.Environment) {
    env.AddFilter("currency", formatCurrency)
    env.AddGlobal("company", "ACME Corp")
})
renderer := template.NewMiyaRender(zpt, template.WithWrapper(wrapper))
```

### `pkg/processor`

Backend abstraction for rendering:

```go
// Server backend (zipreport-server)
backend := processor.NewServerBackend(url, apiKey)

// Future: local backend
backend := processor.NewLocalBackend()

// Create processor
proc := processor.NewZipReportProcessor(backend)
result := proc.Process(job)
```

## Dynamic Image Filters

Generate images at render time:

```go
// In Go
data := map[string]interface{}{
    "chart_func": func(args interface{}) []byte {
        // Generate PNG bytes
        return pngBytes
    },
    "chart_data": chartConfig,
}
```

```html
<!-- In template -->
{{ chart_func | png(data=chart_data, width=400, height=300) }}
```

Supported filters: `png`, `gif`, `jpg`, `jpeg`, `svg`

## Development

### Using Make

```bash
make help           # Show available targets
make test           # Run tests
make test-verbose   # Run tests with verbose output
make test-coverage  # Run tests with coverage report
make lint           # Run golangci-lint
make build          # Build CLI to bin/
make install        # Install CLI
make clean          # Remove build artifacts
```

### Manual Commands

```bash
# Run tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Build CLI
go build ./cmd/zipreport

# Run linter
golangci-lint run
```

## Requirements

- Go 1.24+
- zipreport-server (for PDF generation)

## Sponsors

This project is sponsored by [Blackshield](https://blackshield.io).

## Changelog

See [CHANGELOG.md](CHANGELOG.md).

## License

SPDX-License-Identifier: MIT

[MIT](LICENSE)
