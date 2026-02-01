# ZipReport Go Examples

This directory contains standalone examples demonstrating various ZipReport features.

## Structure

Each example is an independent Go module with its own `go.mod`, `go.sum`, and local `template/` directory containing the report template.

## Examples

| Example | Description |
|---------|-------------|
| [simple](./simple) | Basic PDF generation from template |
| [newsletter](./newsletter) | MIME email generation |
| [filter_example](./filter_example) | Dynamic image generation with filters |
| [print_css_chrome](./print_css_chrome) | CSS Paged Media with Chromium |
| [pagedjs](./pagedjs) | PagedJS pagination with dynamic charts |
| [analytics_dashboard](./analytics_dashboard) | Single-page dashboard with charts |
| [executive_report](./executive_report) | Business report with charts |
| [product_brochure](./product_brochure) | Product brochure with comparison charts |

## Running an Example

Each example can be run independently:

```bash
cd examples/simple
go run main.go output.pdf
```

## Prerequisites

- Go 1.21+
- For PDF generation: a running zipreport-server instance
- For MIME generation: no server required

## Configuration

Most examples use environment variables for configuration:

```bash
export ZIPREPORT_URL="https://127.0.0.1:6543"
export ZIPREPORT_KEY="your-api-key"
```

## Development

When developing with the local zipreport-go library, each example's `go.mod` includes a `replace` directive:

```go
replace github.com/zipreport/zipreport-go => ../..
```

For production use, remove this line and run `go mod tidy` to fetch the published version.
