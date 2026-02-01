# Simple Example

Basic ZipReport usage demonstrating template loading and PDF generation.

## Features

- Load a report from a template directory
- Provide template data
- Render to PDF using zipreport-server

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
cd examples/simple
go run main.go output.pdf
```

## Template

The `template/` directory contains the report template with:

- `index.html` - Main template
- `manifest.json` - Report metadata
- `css/` - Stylesheets
- `partials/` - Reusable template fragments
