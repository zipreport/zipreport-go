# API Reference

## Package Overview

| Package | Description |
|---------|-------------|
| `pkg/api` | High-level API for report generation |
| `pkg/report` | Report types and loading |
| `pkg/template` | Template rendering with miya |
| `pkg/processor` | Backend implementations |
| `pkg/fileutils` | File system abstractions |

## pkg/api

High-level API for PDF and MIME generation.

### ZipReport

PDF generation via zipreport-server.

```go
// Create with server URL and API key
client := api.NewZipReport("http://localhost:6543", "api-key")

// Create with custom backend
client := api.NewZipReportWithBackend(customBackend)
```

#### Methods

```go
// CreateJob creates a ReportJob from a ReportFile
func (z *ZipReport) CreateJob(zpt *report.ReportFile) *report.ReportJob

// Render renders a report job with data and optional wrapper
func (z *ZipReport) Render(job *report.ReportJob, data map[string]interface{}, wrapper template.EnvironmentWrapper) *report.JobResult

// RenderDefaults renders with default job settings
func (z *ZipReport) RenderDefaults(zpt *report.ReportFile, data map[string]interface{}, wrapper template.EnvironmentWrapper) *report.JobResult
```

### MIMEReport

MIME email generation with embedded resources.

```go
client := api.NewMIMEReport()
result := client.RenderDefaults(zpt, data, nil)

// With options
client := api.NewMIMEReport(
    processor.WithSubject("Monthly Report"),
    processor.WithFrom("reports@example.com"),
    processor.WithTo("user@example.com", "other@example.com"),
)
```

### Quick Helpers

```go
// Render to PDF
result := api.RenderPDF(zpt, data, serverURL, apiKey)

// Render to MIME
result := api.RenderMIME(zpt, data)
```

## pkg/report

Report loading and job configuration.

### Loading Reports

```go
// Auto-detect file or directory
zpt, err := report.Load("report.zpt")
zpt, err := report.Load("./template")

// Load from file
zpt, err := report.LoadFile("report.zpt")

// Load from directory
zpt, err := report.LoadDir("./template")

// Load from bytes
zpt, err := report.LoadBytes(zipData)
```

### ReportFile

```go
// Get manifest
manifest := zpt.GetManifest()

// Check file existence
exists := zpt.Exists("style.css")

// Read file bytes
data, err := zpt.GetBytes("data.json")

// Get required parameters
params := zpt.Params()  // []string

// Check if JS event is enabled
usesJS := zpt.UseJSEvent()  // bool
```

### ReportJob

Configure rendering options:

```go
job := report.NewReportJob(zpt)

// Page settings
job.SetPageSize(report.PageA4)      // A3, A4, A5, Letter, Legal, Tabloid
job.SetLandscape(true)

// Margins
job.SetMargins(report.MarginStandard)   // standard, none, minimum
job.SetCustomMargins(&report.Margins{
    Top: 10, Right: 10, Bottom: 10, Left: 10,
})

// Timeouts
job.SetSettlingTime(200)    // milliseconds
job.SetJSTimeout(60)        // seconds
job.SetJobTimeout(120)      // seconds

// JavaScript event
job.UseJSEvent(true)
```

### Constants

```go
// Page sizes
report.PageA3
report.PageA4
report.PageA5
report.PageLetter
report.PageLegal
report.PageTabloid

// Margin styles
report.MarginStandard
report.MarginNone
report.MarginMinimum
```

### Building Reports

```go
// Build .zpt from directory
result := report.Build("./source", "output.zpt", nil)
if !result.Success() {
    log.Fatal(result.Error())
}

// Build with options
opts := report.DefaultBuildOptions()
opts.FollowSymlinks = true
opts.ExcludeHidden = true
result := report.Build("./source", "output.zpt", opts)

// Build to ZipFs (in-memory)
zfs, result := report.BuildZipFs("./source", nil)
```

### JobResult

```go
result := client.RenderDefaults(zpt, data, nil)

if result.Success {
    // result.Report contains PDF/MIME bytes
    os.WriteFile("output.pdf", result.Report, 0644)
} else {
    log.Printf("Error: %s", result.Error)
}
```

## pkg/template

Template rendering with miya.

### MiyaRender

```go
// Create renderer
renderer := template.NewMiyaRender(zpt)

// With options
renderer := template.NewMiyaRender(zpt,
    template.WithStrictParams(true),
    template.WithAutoEscape(true),
    template.WithWrapper(myWrapper),
)

// Render main template
output, err := renderer.Render(data)

// Render specific template
output, err := renderer.RenderTemplate("other.html", data)

// Render string directly
output, err := renderer.RenderString("Hello {{ name }}", data)

// Access environment for advanced configuration
env := renderer.GetEnvironment()
```

### EnvironmentWrapper

Customize the miya environment:

```go
// Using function wrapper
wrapper := template.FuncEnvironmentWrapper(func(env *miya.Environment) {
    env.AddFilter("currency", formatCurrency)
    env.AddGlobal("company", "ACME Corp")
})

renderer := template.NewMiyaRender(zpt, template.WithWrapper(wrapper))
```

Chaining multiple wrappers:

```go
chain := template.NewChainedWrapper(wrapper1, wrapper2)
chain.Add(wrapper3)

renderer := template.NewMiyaRender(zpt, template.WithWrapper(chain))
```

## pkg/processor

Backend implementations.

### ServerBackend

```go
// Create backend
backend := processor.NewServerBackend(url, apiKey)

// With options
backend := processor.NewServerBackend(url, apiKey,
    processor.WithAPIVersion(2),
    processor.WithTimeout(10 * time.Minute),
    processor.WithSecureSSL(true),
    processor.WithHTTPClient(customClient),
)

// Use with ZipReport
client := api.NewZipReportWithBackend(backend)
```

### MIMEProcessor

```go
proc := processor.NewMIMEProcessor()
result := proc.Process(job)

// With options
proc := processor.NewMIMEProcessor(
    processor.WithSubject("Report Title"),
    processor.WithFrom("sender@example.com"),
    processor.WithTo("recipient@example.com"),
)
```

Subject, From and To are trimmed and must not contain line breaks or other control characters (tabs are allowed; the render fails otherwise); a manifest title used as the default subject has line breaks collapsed. Non-ASCII subjects are encoded automatically; an ASCII subject longer than 989 characters is rejected. Only `src` references and `<link href>` targets that resolve to files in the report are embedded, never `.html`/`.htm` pages; anchor links are left unchanged.

## pkg/fileutils

File system abstractions.

### ZipFs

In-memory ZIP file system:

```go
// Create empty
zfs := fileutils.NewZipFs()

// From file
zfs, err := fileutils.NewZipFsFromFile("report.zpt")

// From bytes
zfs, err := fileutils.NewZipFsFromBytes(data)

// Operations
zfs.Add("file.txt", content, overwrite)
content, err := zfs.GetBytes("file.txt")
exists := zfs.Exists("file.txt")
zfs.Remove("file.txt")

// Save to bytes
data, err := zfs.Save()
```

### DiskFs

Disk-based file system:

```go
dfs, err := fileutils.NewDiskFs("/path/to/template")

// Read-only operations
content, err := dfs.GetBytes("file.txt")
exists := dfs.Exists("file.txt")
basePath := dfs.BasePath()
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/zipreport/miya"
    "github.com/zipreport/zipreport-go/pkg/api"
    "github.com/zipreport/zipreport-go/pkg/report"
    "github.com/zipreport/zipreport-go/pkg/template"
)

func main() {
    // Load template
    zpt, err := report.Load("invoice.zpt")
    if err != nil {
        log.Fatal(err)
    }

    // Template data
    data := map[string]interface{}{
        "invoice_number": "INV-2024-001",
        "customer": "John Doe",
        "items": []map[string]interface{}{
            {"name": "Widget", "qty": 5, "price": 10.00},
            {"name": "Gadget", "qty": 2, "price": 25.00},
        },
    }

    // Custom wrapper for currency formatting
    wrapper := template.FuncEnvironmentWrapper(func(env *miya.Environment) {
        env.AddFilter("currency", func(val interface{}, args ...interface{}) (interface{}, error) {
            if f, ok := val.(float64); ok {
                return fmt.Sprintf("$%.2f", f), nil
            }
            return val, nil
        })
    })

    // Create job with custom settings
    client := api.NewZipReport(os.Getenv("ZIPREPORT_URL"), os.Getenv("ZIPREPORT_API_KEY"))
    job := client.CreateJob(zpt)
    job.SetPageSize(report.PageLetter)
    job.SetMargins(report.MarginStandard)

    // Render
    result := client.Render(job, data, wrapper)
    if !result.Success {
        log.Fatal(result.Error)
    }

    os.WriteFile("invoice.pdf", result.Report, 0644)
    log.Println("Invoice generated successfully")
}
```
