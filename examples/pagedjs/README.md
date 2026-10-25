# PagedJS Example

PagedJS pagination with dynamic charts using ZipReport.

## Features

- Use PagedJS for advanced pagination
- Generate charts dynamically using gonum/plot
- Use JS event waiting for PagedJS rendering

## Prerequisites

- Go 1.24+
- Running zipreport-server instance

## Configuration

Set environment variables (optional):

```bash
export ZIPREPORT_URL="https://127.0.0.1:6543"
export ZIPREPORT_KEY="your-api-key"
```

## Running

```bash
cd examples/pagedjs
go run main.go output.pdf
```

## Template

The `template/` directory contains a PagedJS-enabled template that demonstrates:

- Page numbering
- Headers and footers
- Dynamic chart generation
