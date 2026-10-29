# CLI Usage

The `zipreport` CLI tool provides commands for building, inspecting, and debugging report templates.

## Commands

### build

Build a .zpt file from a template directory.

```bash
zipreport build <source_dir> <output.zpt>
```

**Arguments:**

- `source_dir` - Path to the template directory
- `output.zpt` - Output filename

**Flags:**

- `-s, --symlinks` - Follow symlinks when building (symlinks must resolve inside the template folder; symlinked folders are not supported)

Builds fail when any file cannot be read, when a symlink is broken (with `-s`), when a symlinked folder is used (with `-s`), or when `index.html`/`manifest.json` is a symlink (without `-s`), when a symlink points outside the template folder (with `-s`), or when a file name is not portable (e.g. "a:b.css"). Otherwise, without `-s`, symlinks are skipped.

**Examples:**

```bash
# Build from directory
zipreport build ./my-template output.zpt

# Follow symlinks
zipreport build -s ./my-template output.zpt
```

**Requirements:**

The source directory must contain:

- `manifest.json` - Report metadata
- `index.html` - Main template file

### debug

Start a development HTTP server that renders the template on each request.

```bash
zipreport debug <source>
```

**Arguments:**

- `source` - Template directory or .zpt file

**Flags:**

- `-p, --port` - Port to listen on (default: 8000)
- `--host` - Interface to listen on (default: 127.0.0.1; use 0.0.0.0 to accept connections from other machines or from outside a container)

**Examples:**

```bash
# Debug a directory
zipreport debug ./my-template

# Debug a .zpt file
zipreport debug report.zpt

# Use custom port
zipreport debug -p 3000 ./my-template
```

The debug server:

- Reloads the template on each request (no restart needed)
- Serves static assets (CSS, JS, images)
- Never serves hidden files: a name with a segment starting with a dot, a symlink to one, or a spelling the folder does not list (another case, a Windows short name); a request for index.html, in any such spelling, is always rendered
- Uses data from `data.json` if present
- Shows render errors in the browser
- Has no authentication: only bind to other interfaces on trusted networks
- When bound to localhost, only answers requests addressed to localhost or a loopback IP; binding to any non-loopback address (e.g. `--host 0.0.0.0`) disables this check (needed behind port forwarders such as Codespaces)

### info

Display metadata from a .zpt file.

```bash
zipreport info <report.zpt>
```

**Arguments:**

- `report.zpt` - Path to the .zpt file

**Output:**

```
Report: report.zpt
----------------------------------------
Title:       My Report
Author:      John Doe
Version:     1.0.0
Description: A sample report
Parameters:  name, items
Use JS Event: false
```

Control and format characters in manifest fields, including non-ASCII spaces such as U+00A0, are shown escaped (for example `\x1b`, `\u00a0`), so a crafted .zpt cannot drive the terminal.

### list

List .zpt files in a directory.

```bash
zipreport list [directory]
```

**Arguments:**

- `directory` - Directory to search (default: current directory)

**Examples:**

```bash
# List in current directory
zipreport list

# List in specific directory
zipreport list ./reports
```

File names are printed with control characters, format characters (U+00A0, joiners) and invalid UTF-8 escaped. The escaping is unconditional, so piping the output into another command (`zipreport list | xargs zipreport info`) does not round-trip such names.

### render

Render a report to HTML or a MIME email, without a zipreport-server.

```bash
zipreport render <source> [flags]
```

**Arguments:**

- `source` - Path to a .zpt file or a template directory

**Flags:**

- `-d, --data` - JSON file with template data. Follows the `data.json` rules: a JSON object or `null`; a byte-order mark and an empty file are allowed (no data). Keys override `data.json`.
- `-f, --format` - Output format: `html` or `mime` (default `html`)
- `-o, --output` - Output file (default: stdout)
- `--subject` - MIME Subject header (default: the manifest title)
- `--from` - MIME From header
- `--to` - MIME To header; repeatable. A display name containing a comma must be quoted: `--to '"Doe, John" <j@example.com>'`

**Notes:**

- Manifest `params` are required, as in the library: pass them with `--data`.
- HTML output is the rendered `index.html` only, so relative asset links (`<img src="a.png">`) only resolve next to the template.
- Exit code is 1 on any error, and nothing is written.
- Without `-o` the rendered document is written to stdout unmodified, like `cat`. Do not render an untrusted report straight to a terminal.
- The image filters (`png`, `svg`, ...) cannot be fed from `--data`: they need Go callables or bytes, and JSON gives strings, so they fail with `invalid image generator type`.
- PDF output needs the library (`api.NewZipReport`).

**Examples:**

```bash
# HTML to a file
zipreport render report.zpt -d data.json -o report.html

# MIME email
zipreport render ./my-template -d data.json -f mime --from a@example.com --to b@example.com -o mail.eml
```

### version

Display the CLI version.

```bash
zipreport version
```

## Global Flags

- `--help, -h` - Show help for any command

## Template Structure

A valid template directory must contain:

```
template/
├── manifest.json    # Required: Report metadata
├── index.html       # Required: Main template
├── data.json        # Optional: Default data for debugging
├── style.css        # Optional: Stylesheets
├── script.js        # Optional: JavaScript
└── images/          # Optional: Static assets
    └── logo.png
```

### manifest.json

```json
{
    "title": "Report Title",
    "author": "Author Name",
    "version": "1.0.0",
    "description": "Report description",
    "params": ["param1", "param2"],
    "report": "report.html",
    "useJSEvent": false
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `title` | Yes | Report title |
| `author` | Yes | Author name |
| `version` | Yes | Version string |
| `description` | Yes | Report description |
| `params` | No | Required template parameters (default: `[]`) |
| `report` | No | Output filename, a relative path inside the report; a leading `/` is ignored (default: `report.html`) |
| `useJSEvent` | No | Wait for JS event (default: `false`) |

### data.json

Optional file containing default data for debugging:

```json
{
    "param1": "value1",
    "param2": ["item1", "item2"]
}
```

If present, it must contain a JSON object (an empty file or `null` is treated as no data; a UTF-8 byte-order mark is allowed); a malformed data.json makes every render fail with "invalid data.json".

## Examples

### Development Workflow

```bash
# 1. Create template directory
mkdir my-report
cd my-report

# 2. Create required files
echo '{"title":"My Report","author":"Me","version":"1.0.0","description":"Test","params":[]}' > manifest.json
echo '<h1>Hello World</h1>' > index.html

# 3. Start debug server
zipreport debug .

# 4. Edit templates, refresh browser to see changes

# 5. Build final .zpt
zipreport build . ../my-report.zpt

# 6. Verify
zipreport info ../my-report.zpt
```
