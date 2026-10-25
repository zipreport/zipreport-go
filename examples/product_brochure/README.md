# Product Brochure Example

Software product brochure with feature charts using ZipReport.

## Features

- Software product brochure template with PagedJS
- Feature comparison charts using gonum/plot
- Pricing tiers and testimonials

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
cd examples/product_brochure
go run main.go output.pdf
```

## Template

The `template/` directory contains a product brochure template with:

- Competitive comparison charts
- Feature scores
- Pricing tiers
- Customer testimonials
