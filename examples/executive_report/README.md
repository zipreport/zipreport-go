# Executive Report Example

Business report with dynamic charts using ZipReport.

## Features

- Generate charts dynamically at render time using gonum/plot
- Use ReportJob for custom configuration
- Enable JavaScript events for PagedJS

## Prerequisites

- Go 1.24+
- Running zipreport-server instance

## Configuration

Set environment variables (optional):

```bash
export ZIPREPORT_URL="http://127.0.0.1:6543"
export ZIPREPORT_KEY="your-api-key"
```

## Running

```bash
cd examples/executive_report
go run main.go output.pdf
```

## Template

The `template/` directory contains an executive report template with:

- Revenue charts
- Performance metrics
- Business summaries
