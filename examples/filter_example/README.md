# Filter Example

Dynamic image generation with ZipReport filters.

## Features

- Use image filters (png, gif, jpg, svg)
- Generate images dynamically at render time
- Pass callable functions to templates

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
cd examples/filter_example
go run main.go output.pdf
```

## Template

The `template/` directory demonstrates the image filters available in ZipReport for generating dynamic images.
