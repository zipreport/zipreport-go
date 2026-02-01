# Analytics Dashboard Example

Single-page dashboard with multiple chart types using ZipReport.

## Features

- Single-page dashboard layout with PagedJS
- Multiple chart types (line, bar)
- Dynamic chart generation using gonum/plot

## Prerequisites

- Go 1.21+
- Running zipreport-server instance

## Configuration

Set environment variables (optional):

```bash
export ZIPREPORT_URL="https://127.0.0.1:6543"
export ZIPREPORT_KEY="your-api-key"
```

## Running

```bash
cd examples/analytics_dashboard
go run main.go output.pdf
```

## Template

The `template/` directory contains a dashboard template with:

- Traffic charts
- Device breakdown
- Hourly activity
- Traffic sources
