# ZipReport Go

A Go implementation of ZipReport - HTML to PDF report generation using Jinja2-compatible templates.

## Features

- **Jinja2-compatible templating** via [miya](https://github.com/zipreport/miya)
- **PDF generation** via zipreport-server
- **MIME email generation** with embedded resources
- **Dynamic image filters** (png, gif, jpg, svg)
- **CLI tool** for building and debugging templates

## Overview

ZipReport Go allows you to create professional PDF reports from HTML templates. It uses the miya template engine which provides Jinja2-compatible syntax, making it easy to port existing Python ZipReport templates.

### How It Works

1. **Create a template** - Write your report as an HTML template with Jinja2 syntax
2. **Package as .zpt** - Use the CLI to bundle templates into a single .zpt file
3. **Render with data** - Pass your data to the API to generate PDFs or MIME emails

### Key Concepts

- **ReportFile (.zpt)** - A ZIP archive containing the template, manifest, and assets
- **Manifest** - JSON file describing the report metadata and required parameters
- **Template** - HTML with Jinja2 syntax (extends, includes, loops, filters)
- **Backend** - The rendering engine (zipreport-server for PDF, MIME processor for email)

## Quick Links

- [Installation](installation.md) - Get started with ZipReport Go
- [Quick Start](quickstart.md) - Create your first report
- [CLI Usage](cli.md) - Command line tool reference
- [API Reference](api.md) - Library API documentation
- [Examples](examples.md) - Working examples

## Requirements

- Go 1.24+
- zipreport-server (for PDF generation)

## License

MIT License - see [LICENSE](https://github.com/zipreport/zipreport-go/blob/main/LICENSE) for details.
