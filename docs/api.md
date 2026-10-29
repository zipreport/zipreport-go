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

// RenderContext is Render with a context; cancelling it aborts the request to zipreport-server
func (z *ZipReport) RenderContext(ctx context.Context, job *report.ReportJob, data map[string]interface{}, wrapper template.EnvironmentWrapper) *report.JobResult

// RenderDefaultsContext is RenderDefaults with a context
func (z *ZipReport) RenderDefaultsContext(ctx context.Context, zpt *report.ReportFile, data map[string]interface{}, wrapper template.EnvironmentWrapper) *report.JobResult
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

// Same, verifying the server's TLS certificate
result := api.RenderPDF(zpt, data, serverURL, apiKey, processor.WithSecureSSL(true))

// Render to MIME
result := api.RenderMIME(zpt, data)
```

`RenderPDF` creates a new client per call. A service that renders repeatedly should create one `api.ZipReport` with `api.NewZipReport` and reuse it, so connections are reused.

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

### Rendering and concurrency

Rendering through `api.ZipReport` / `api.MIMEReport` never modifies the loaded `ReportFile`: each render reads through it and writes into a private copy (`ReportFile.Clone()`, an in-memory `fileutils.OverlayFs`). One loaded report can be reused and shared across goroutines, provided nothing modifies it (`Add`, writes to `GetManifest()` fields) at the same time.

When driving `template.MiyaRender` and a processor yourself, render into a copy and build the job from it:

```go
c := zpt.Clone()
if _, err := template.NewMiyaRender(c).Render(data); err != nil {
    return err
}
result := proc.Process(report.NewReportJob(c))
```

Filters added through an `EnvironmentWrapper` that write files into a `ReportFile` they captured write into the original, not the copy; use such filters with `MiyaRender` directly.

`Save()` and every render ship the report's own `manifest.json`, so changes made through `GetManifest()` are not sent; the in-memory manifest is only used when the report has no `manifest.json`.

`Save()` on a folder-loaded report packages every readable, non-hidden file in the folder (unreadable files and broken symlinks are skipped). A file name that is not portable inside a ZIP (a ".." segment, a drive-letter prefix such as "a:", or a leading backslash) fails Save() with fileutils.ErrUnportableName. It skips root-level `_gen_*` files left by older versions, and it fails if `index.html` is missing or the folder is nested deeper than 32 levels. It also fails with `fileutils.ErrZipTooLarge` past 10,000 files and folders (a folder reached through several symlinks counts once per path) or `fileutils.MaxUncompressedSize` bytes. The same limits, plus 1,024-byte names and 32 levels, apply to every `Save()`, not only folder reports. Large folders are read and sent on every render.

A `data.json` in the report supplies default data. It must contain a JSON object (empty files and `null` are treated as no data; a UTF-8 byte-order mark is allowed); anything else makes every render fail with `invalid data.json`.

Folder templates are contained: no read or write may leave the template folder, either through `..` or through a symlink that resolves outside it. Such paths fail with fileutils.ErrOutsideFolder (`LoadDir` names the file); includes and assets that resolve outside are not found. A broken symlink is rejected even if it points inside the folder. Containment is checked when a path is resolved, then the file is opened; a folder that an untrusted party can modify at the same time can race that check, so do not render from such a folder. Symlinks inside the folder are followed. The debug server and MIME embedding refuse a symlink whose target is hidden. `manifest.json`'s `report` must be a relative path inside the report. A leading "/" in manifest.json's report is ignored.

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
if !result.Success() {
    log.Fatal(result.Error()) // zfs is nil when the build failed
}
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

// Access environment for advanced configuration (before the render)
env := template.NewMiyaRender(zpt.Clone()).GetEnvironment()

// Render main template
output, err := template.NewMiyaRender(zpt.Clone()).Render(data)

// Render a specific template
output, err := template.NewMiyaRender(zpt.Clone()).RenderTemplate("other.html", data)

// Render a string directly
output, err := template.NewMiyaRender(zpt.Clone()).RenderString("Hello {{ name }}", data)
```

A `MiyaRender` renders once. A second call to `Render`, `RenderTemplate` or `RenderString` returns `template.ErrRendererUsed` (if filter registration failed, every call returns that error instead). Create a new renderer (cheap) for each render, including a retry after an error, and call `GetEnvironment`/`SetWrapper` before the render. This keeps template caches from carrying one render's data into the next.

`Render` writes the output (and any generated images) into the `ReportFile` it was given; pass `zpt.Clone()` to keep `zpt` unchanged.

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

// Verify the server's TLS certificate (verification is skipped by default)
backend := processor.NewServerBackend(url, apiKey,
    processor.WithAPIVersion(2),
    processor.WithTimeout(10 * time.Minute),
    processor.WithSecureSSL(true),
)

// Or bring your own client (used as given; WithSecureSSL is ignored,
// WithTimeout applies to a copy)
backend := processor.NewServerBackend(url, apiKey,
    processor.WithHTTPClient(customClient),
)
// set CheckRedirect: processor.CheckRedirect on customClient to keep the API key on the server's host

// Use with ZipReport
client := api.NewZipReportWithBackend(backend)
```

Responses larger than 512 MiB fail with `processor.ErrResponseTooLarge`; raise the limit with `processor.WithMaxResponseSize(n)` (`n <= 0` means the default). `api.RenderPDF` accepts the same options. `ErrResponseTooLarge` can be matched with `errors.Is` only when calling `ServerBackend.Render` or `RenderContext` directly; `ZipReport` reports it as text in `JobResult.Error`.

The default client refuses redirects when the host, port or hostname spelling changes, and on https to http (`processor.ErrRedirectRefused`). A client given through `WithHTTPClient` keeps its own policy, and Go's default forwards `X-Auth-Key` to the redirect target, so set `CheckRedirect: processor.CheckRedirect` on it.

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

Loading a `.zpt` fails with `fileutils.ErrZipTooLarge` when its total uncompressed size exceeds `fileutils.MaxUncompressedSize` (default 512 MiB). It also fails when the archive has more than 10,000 entries, or an entry name longer than 1,024 bytes or nested deeper than 32 folders. `MaxUncompressedSize` is process-wide: set it once at start-up, before any load. Loads running at the same time can each use up to the limit. `ZipFs.Save()`, and so `ReportFile.Save()`, `Build`/`BuildZipFs`/`BuildFromReportFile` and every PDF render, fail under the same limits, and `zfs.CheckLimits()` reports it without serialising. `ZipFs.Save`, `ReportFile.Save`, `CheckLimits` and `BuildFromReportFile` return an error matching `errors.Is(err, fileutils.ErrZipTooLarge)`; `Build`/`BuildZipFs` report it as text in `BuildResult`, and renders as text in `JobResult.Error`. `zfs.Clone()` returns an independent copy.

### DiskFs

Disk-based file system:

```go
dfs, err := fileutils.NewDiskFs("/path/to/template")

// Read-only operations
content, err := dfs.GetBytes("file.txt")
exists := dfs.Exists("file.txt")
basePath := dfs.BasePath()
```

### OverlayFs

Layers an in-memory `ZipFs` over another file system: reads fall through to the base, writes stay in memory. `ReportFile.Clone()` uses it.

```go
o := fileutils.NewOverlayFs(base)
o.Add("report.html", html, true)    // base is untouched
content, err := o.GetBytes("index.html") // read from base
```

`DiskFs.Snapshot()` copies a folder into a new `ZipFs` (hidden entries, root-level `_gen_*` files, special files, broken symlinks, unreadable files and symlinks leaving the folder are skipped; unportable names fail).

`fileutils.IsServable(fs, name)` reports whether a name may be handed to a client; the debug server and MIME embedding use it. The name must be canonical (relative, slash-separated, unchanged by `path.Clean`) with no hidden segment. On a folder on disk, every segment must also be spelled exactly as the folder lists it, and the file the name resolves to, symlinks followed, must have no hidden segment inside the folder. `ZipFs` and custom file systems are judged by name only.

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
    client := api.NewZipReport(os.Getenv("ZIPREPORT_URL"), os.Getenv("ZIPREPORT_KEY"))
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
