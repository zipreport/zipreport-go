# Examples

Each example is a standalone Go module in the `examples/` directory with its own dependencies and local templates.

## Running Examples

1. Start zipreport-server (required for PDF generation):

```bash
docker run -p 6543:6543 zipreport/zipreport-server
```

2. Navigate to an example directory and run:

```bash
cd examples/simple
export ZIPREPORT_URL="http://localhost:6543"
go run main.go output.pdf
```

## Available Examples

### simple

Basic PDF generation from a template.

```bash
cd examples/simple
go run main.go output.pdf
```

Demonstrates:

- Loading a template from directory
- Passing data to templates
- Rendering to PDF

### newsletter

MIME email generation with embedded images.

```bash
cd examples/newsletter
go run main.go output.eml
```

Demonstrates:

- MIMEReport for email generation
- Embedded images and styling
- No server required

### filter_example

Dynamic image generation using filters.

```bash
cd examples/filter_example
go run main.go output.pdf
```

Demonstrates:

- Using `png` filter for dynamic charts
- Solid-colour PNG/GIF/JPG/SVG images generated with the standard library
- Passing data to image generators

### print_css_chrome

CSS Paged Media for print layouts.

```bash
cd examples/print_css_chrome
go run main.go output.pdf
```

Demonstrates:

- CSS `@page` rules
- Page breaks and headers/footers
- Print-specific styling

### pagedjs

PagedJS for advanced print layouts with dynamic charts.

```bash
cd examples/pagedjs
go run main.go output.pdf
```

Demonstrates:

- PagedJS library integration
- JavaScript event waiting
- Complex page layouts

### analytics_dashboard

Dashboard with multiple charts and data visualization.

```bash
cd examples/analytics_dashboard
go run main.go output.pdf
```

Demonstrates:

- Multiple chart types
- Grid layouts
- Data-driven visualizations

### executive_report

Professional business report with sections.

```bash
cd examples/executive_report
go run main.go output.pdf
```

Demonstrates:

- Multi-section reports
- Tables and data presentation
- Professional styling

### product_brochure

Marketing brochure with product showcase.

```bash
cd examples/product_brochure
go run main.go output.pdf
```

Demonstrates:

- Product layouts
- Image handling
- Marketing-style design

## Example Structure

Each example follows this structure:

```
example-name/
├── go.mod          # Module definition with replace directive
├── go.sum          # Dependencies
├── main.go         # Example code
├── README.md       # Usage instructions
└── template/       # Local template files
    ├── manifest.json
    ├── index.html
    └── ...
```

### go.mod with Local Development

```go
module github.com/zipreport/zipreport-go/examples/simple

go 1.24

require github.com/zipreport/zipreport-go v1.0.0

replace github.com/zipreport/zipreport-go => ../..
```

## Creating Your Own Example

1. Create a new directory in `examples/`:

```bash
mkdir examples/my-example
cd examples/my-example
```

2. Initialize the module:

```bash
go mod init github.com/zipreport/zipreport-go/examples/my-example
```

3. Add replace directive to `go.mod`:

```go
require github.com/zipreport/zipreport-go v1.0.0
replace github.com/zipreport/zipreport-go => ../..
```

4. Create template directory:

```bash
mkdir template
```

5. Add template files:

```bash
# manifest.json
echo '{
    "title": "My Example",
    "author": "Me",
    "version": "1.0.0",
    "description": "My example report",
    "params": ["name"]
}' > template/manifest.json

# index.html
echo '<h1>Hello {{ name }}</h1>' > template/index.html
```

6. Create main.go:

```go
package main

import (
    "fmt"
    "os"

    "github.com/zipreport/zipreport-go/pkg/api"
    "github.com/zipreport/zipreport-go/pkg/report"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run main.go <output.pdf>")
        os.Exit(1)
    }

    zpt, err := report.LoadDir("template")
    if err != nil {
        fmt.Printf("Failed to load template: %v\n", err)
        os.Exit(1)
    }

    data := map[string]interface{}{
        "name": "World",
    }

    serverURL := os.Getenv("ZIPREPORT_URL")
    if serverURL == "" {
        serverURL = "http://localhost:6543"
    }

    client := api.NewZipReport(serverURL, "")
    result := client.RenderDefaults(zpt, data, nil)

    if !result.Success {
        fmt.Printf("Render failed: %s\n", result.Error)
        os.Exit(1)
    }

    if err := os.WriteFile(os.Args[1], result.Report, 0644); err != nil {
        fmt.Printf("Failed to save PDF: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("PDF saved to %s\n", os.Args[1])
}
```

7. Run:

```bash
go run main.go output.pdf
```

## Output Directory

Example outputs are saved to the `output/` directory in the project root:

```
output/
├── simple.pdf
├── newsletter.eml
├── filter_example.pdf
├── print_css_chrome.pdf
├── pagedjs.pdf
├── analytics_dashboard.pdf
├── executive_report.pdf
└── product_brochure.pdf
```
