# Newsletter Example

MIME email generation with ZipReport.

## Features

- Generate MIME email messages from templates
- Embed resources in the email
- Use the MIMEReport processor

## Prerequisites

- Go 1.21+

## Running

```bash
cd examples/newsletter
go run main.go output.eml
```

The generated `.eml` file can be opened with any email client.

## Template

The `template/` directory contains the newsletter template with embedded images and styles.
