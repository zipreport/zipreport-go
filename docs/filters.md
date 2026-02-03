# Template Filters

ZipReport Go provides built-in filters for dynamic content generation, particularly for generating images at render time.

## Image Filters

Generate images dynamically during template rendering. The generated images are automatically embedded in the report.

### Supported Formats

| Filter | MIME Type | Description |
|--------|-----------|-------------|
| `png` | image/png | PNG images |
| `gif` | image/gif | GIF images |
| `jpg` | image/jpeg | JPEG images |
| `jpeg` | image/jpeg | JPEG images (alias) |
| `svg` | image/svg+xml | SVG vector graphics |

### Basic Usage

In your Go code, define a function that generates image bytes:

```go
data := map[string]interface{}{
    "generate_chart": func(args interface{}) []byte {
        // Generate and return PNG bytes
        return createChartPNG(args)
    },
}
```

In your template:

```html
{{ generate_chart | png }}
```

### With Arguments

Pass data and attributes to the filter:

```html
<!-- With data argument -->
{{ generate_chart | png(data=chart_config) }}

<!-- With dimensions -->
{{ generate_chart | png(data=chart_config, width=400, height=300) }}

<!-- With alt text and CSS class -->
{{ generate_chart | png(data=chart_config, alt="Sales Chart", class="chart-img") }}
```

### Filter Arguments

| Argument | Type | Description |
|----------|------|-------------|
| `data` | any | Data passed to the generator function |
| `width` | int | Image width in pixels |
| `height` | int | Image height in pixels |
| `alt` | string | Alt text for the img tag |
| `class` | string | CSS class for the img tag |

### Generator Function Signatures

The image generator can use any of these signatures:

```go
// Simple: returns bytes only
func(data interface{}) []byte

// With error: returns bytes and error
func(data interface{}) ([]byte, error)

// No data: uses no arguments
func() []byte

// With error, no data
func() ([]byte, error)
```

### Example: Chart Generation with gonum/plot

```go
package main

import (
    "bytes"

    "gonum.org/v1/plot"
    "gonum.org/v1/plot/plotter"
    "gonum.org/v1/plot/vg"

    "github.com/zipreport/zipreport-go/pkg/api"
    "github.com/zipreport/zipreport-go/pkg/report"
)

func createBarChart(data interface{}) []byte {
    config := data.(map[string]interface{})

    p := plot.New()
    p.Title.Text = config["title"].(string)

    values := plotter.Values{}
    for _, v := range config["values"].([]float64) {
        values = append(values, v)
    }

    bars, _ := plotter.NewBarChart(values, vg.Points(20))
    p.Add(bars)

    // Render to PNG
    w, _ := p.WriterTo(4*vg.Inch, 3*vg.Inch, "png")
    var buf bytes.Buffer
    w.WriteTo(&buf)

    return buf.Bytes()
}

func main() {
    zpt, _ := report.Load("dashboard.zpt")

    data := map[string]interface{}{
        "chart_func": createBarChart,
        "chart_data": map[string]interface{}{
            "title": "Monthly Sales",
            "values": []float64{100, 150, 200, 175, 225},
        },
    }

    client := api.NewZipReport("http://localhost:6543", "")
    result := client.RenderDefaults(zpt, data, nil)
    // ...
}
```

Template:

```html
<div class="chart">
    {{ chart_func | png(data=chart_data, width=400, height=300, alt="Sales Chart") }}
</div>
```

### Example: SVG Generation

```go
func createSVGBadge(data interface{}) []byte {
    text := data.(string)
    svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="30">
        <rect width="100" height="30" fill="#4CAF50" rx="5"/>
        <text x="50" y="20" text-anchor="middle" fill="white" font-size="14">%s</text>
    </svg>`, text)
    return []byte(svg)
}

data := map[string]interface{}{
    "badge": createSVGBadge,
    "status": "Active",
}
```

Template:

```html
{{ badge | svg(data=status) }}
```

## JSON Filter

Convert values to JSON strings for embedding in JavaScript:

```html
<script>
    const data = {{ my_data | json }};
    console.log(data);
</script>
```

## Custom Filters

Add custom filters using an EnvironmentWrapper:

```go
import (
    "fmt"
    "strings"

    "github.com/zipreport/miya"
    "github.com/zipreport/zipreport-go/pkg/template"
)

wrapper := template.FuncEnvironmentWrapper(func(env *miya.Environment) {
    // Currency formatter
    env.AddFilter("currency", func(val interface{}, args ...interface{}) (interface{}, error) {
        if f, ok := val.(float64); ok {
            return fmt.Sprintf("$%.2f", f), nil
        }
        return val, nil
    })

    // Uppercase filter
    env.AddFilter("shout", func(val interface{}, args ...interface{}) (interface{}, error) {
        if s, ok := val.(string); ok {
            return strings.ToUpper(s), nil
        }
        return val, nil
    })
})

renderer := template.NewMiyaRender(zpt, template.WithWrapper(wrapper))
```

Template:

```html
<p>Price: {{ price | currency }}</p>
<p>{{ message | shout }}</p>
```

## Global Variables

Add global variables available in all templates:

```go
wrapper := template.FuncEnvironmentWrapper(func(env *miya.Environment) {
    env.AddGlobal("company_name", "ACME Corp")
    env.AddGlobal("current_year", 2024)
    env.AddGlobal("config", map[string]interface{}{
        "theme": "dark",
        "version": "2.0",
    })
})
```

Template:

```html
<footer>
    &copy; {{ current_year }} {{ company_name }}
</footer>
```

## Chaining Wrappers

Combine multiple wrappers:

```go
currencyWrapper := template.FuncEnvironmentWrapper(func(env *miya.Environment) {
    env.AddFilter("currency", currencyFilter)
})

dateWrapper := template.FuncEnvironmentWrapper(func(env *miya.Environment) {
    env.AddFilter("date", dateFilter)
})

chain := template.NewChainedWrapper(currencyWrapper, dateWrapper)
renderer := template.NewMiyaRender(zpt, template.WithWrapper(chain))
```
