# Quick Start

This guide walks you through creating and rendering your first PDF report.

## 1. Create a Template Directory

Create a directory structure for your template:

```
my-report/
├── manifest.json
├── index.html
└── style.css (optional)
```

### manifest.json

The manifest describes your report:

```json
{
    "title": "My Report",
    "author": "Your Name",
    "version": "1.0.0",
    "description": "A sample report",
    "params": ["name", "items"]
}
```

Required fields:

- `title` - Report title
- `author` - Author name
- `version` - Version string
- `description` - Report description
- `params` - List of required template parameters (optional; default `[]`)

Optional fields:

- `report` - Output filename, a relative path inside the report; a leading `/` is ignored (defaults to `report.html`)
- `useJSEvent` - Wait for JavaScript event before rendering (default: `false`)

### index.html

The main template using Jinja2 syntax:

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{ title }}</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <h1>Hello, {{ name }}!</h1>

    <h2>Items</h2>
    <ul>
    {% for item in items %}
        <li>{{ item }}</li>
    {% endfor %}
    </ul>
</body>
</html>
```

### style.css

```css
body {
    font-family: Arial, sans-serif;
    margin: 40px;
}

h1 {
    color: #333;
}
```

## 2. Build the Template

Package your template into a .zpt file:

```bash
zipreport build ./my-report output.zpt
```

## 3. Preview in Browser

Use the debug server for development:

```bash
zipreport debug ./my-report
# Or with a .zpt file
zipreport debug output.zpt
```

Open http://localhost:8000 in your browser to preview.

## 4. Render to PDF

### Using the Library

```go
package main

import (
    "os"
    "log"

    "github.com/zipreport/zipreport-go/pkg/api"
    "github.com/zipreport/zipreport-go/pkg/report"
)

func main() {
    // Load the report template
    zpt, err := report.Load("output.zpt")
    if err != nil {
        log.Fatal(err)
    }

    // Prepare template data
    data := map[string]interface{}{
        "name": "World",
        "items": []string{"First item", "Second item", "Third item"},
    }

    // Create client and render
    client := api.NewZipReport("http://localhost:6543", "")
    result := client.RenderDefaults(zpt, data, nil)

    if !result.Success {
        log.Fatalf("Render failed: %s", result.Error)
    }

    // Save PDF
    if err := os.WriteFile("output.pdf", result.Report, 0644); err != nil {
        log.Fatal(err)
    }

    log.Println("PDF saved to output.pdf")
}
```

### Using Quick Helper

For simple cases:

```go
result := api.RenderPDF(zpt, data, "http://localhost:6543", "")
```

## 5. Generate MIME Email

For email-ready output with embedded images:

```go
package main

import (
    "os"

    "github.com/zipreport/zipreport-go/pkg/api"
    "github.com/zipreport/zipreport-go/pkg/report"
)

func main() {
    zpt, _ := report.Load("output.zpt")

    data := map[string]interface{}{
        "name": "World",
        "items": []string{"Item 1", "Item 2"},
    }

    client := api.NewMIMEReport()
    result := client.RenderDefaults(zpt, data, nil)

    if result.Success {
        os.WriteFile("email.eml", result.Report, 0644)
    }
}
```

## Next Steps

- [CLI Usage](cli.md) - Learn all CLI commands
- [API Reference](api.md) - Detailed API documentation
- [Template Filters](filters.md) - Dynamic image generation
- [Examples](examples.md) - More complex examples
