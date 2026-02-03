# Installation

## CLI Tool

### Using Go install (recommended)

```bash
go install github.com/zipreport/zipreport-go/cmd/zipreport@latest
```

### Pre-built binaries

Download the latest release for your platform from the [Releases page](https://github.com/zipreport/zipreport-go/releases).

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux    | x86_64       | `zipreport_linux_amd64.tar.gz` |
| Linux    | ARM64        | `zipreport_linux_arm64.tar.gz` |
| macOS    | x86_64       | `zipreport_darwin_amd64.tar.gz` |
| macOS    | Apple Silicon| `zipreport_darwin_arm64.tar.gz` |
| Windows  | x86_64       | `zipreport_windows_amd64.zip` |

#### Linux/macOS

```bash
# Download and extract
tar -xzf zipreport_linux_amd64.tar.gz

# Move to PATH
sudo mv zipreport /usr/local/bin/

# Verify installation
zipreport --version
```

#### Windows (PowerShell)

```powershell
# Extract
Expand-Archive zipreport_windows_amd64.zip

# Move to PATH
Move-Item zipreport.exe C:\Windows\System32\

# Verify installation
zipreport --version
```

### From source

```bash
git clone https://github.com/zipreport/zipreport-go
cd zipreport-go
go install ./cmd/zipreport
```

## Library

Add ZipReport Go to your Go project:

```bash
go get github.com/zipreport/zipreport-go
```

## zipreport-server

PDF generation requires a running zipreport-server instance. The server uses headless Chrome to render HTML to PDF.

### Docker (recommended)

```bash
docker run -p 6543:6543 zipreport/zipreport-server
```

### Configuration

Set the server URL via environment variable:

```bash
export ZIPREPORT_URL="http://localhost:6543"
export ZIPREPORT_API_KEY="your-api-key"  # if authentication is enabled
```

Or configure in code:

```go
client := api.NewZipReport("http://localhost:6543", "your-api-key")
```

## Verifying Installation

### CLI

```bash
# Check version
zipreport version

# Build a test report
zipreport build ./my-template output.zpt

# Display help
zipreport --help
```

### Library

```go
package main

import (
    "fmt"
    "github.com/zipreport/zipreport-go/pkg/report"
)

func main() {
    // Try loading a report
    zpt, err := report.Load("./my-template")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    manifest := zpt.GetManifest()
    fmt.Printf("Loaded: %s v%s\n", manifest.Title, manifest.Version)
}
```
