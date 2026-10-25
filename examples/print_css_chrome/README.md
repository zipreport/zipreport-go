# Print CSS Chrome Example

CSS Paged Media features with Chromium using ZipReport.

## Features

- Use CSS @page rules with margin boxes
- Page counters (counter(page), counter(pages))
- Page breaks (break-before, break-after, break-inside)
- Widows and orphans control

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
cd examples/print_css_chrome
go run main.go output.pdf
```

## Template

The `template/` directory demonstrates native CSS Paged Media support in Chromium without requiring PagedJS.
