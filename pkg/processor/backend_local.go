package processor

import "errors"

// ErrNotImplemented is returned when a feature is not yet implemented.
var ErrNotImplemented = errors.New("not yet implemented")

// LocalBackend renders PDFs locally using a headless browser.
// This is a stub for future implementation.
type LocalBackend struct {
	// Configuration options for local rendering
	chromePath string
	timeout    int // seconds
}

// LocalOption configures a LocalBackend.
type LocalOption func(*LocalBackend)

// WithChromePath sets the path to the Chrome/Chromium executable.
func WithChromePath(path string) LocalOption {
	return func(b *LocalBackend) {
		b.chromePath = path
	}
}

// WithLocalTimeout sets the rendering timeout in seconds.
func WithLocalTimeout(seconds int) LocalOption {
	return func(b *LocalBackend) {
		b.timeout = seconds
	}
}

// NewLocalBackend creates a new LocalBackend.
func NewLocalBackend(opts ...LocalOption) *LocalBackend {
	b := &LocalBackend{
		timeout: 120,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Render converts HTML to PDF locally.
// This is a stub that returns an error until implemented.
func (b *LocalBackend) Render(zptData []byte, options map[string]interface{}) ([]byte, error) {
	// Future implementation:
	// 1. Extract zptData (ZIP) to temp dir or memory
	// 2. Read report.html from the extracted content
	// 3. Use chromedp/wkhtmltopdf/etc. to convert HTML to PDF
	// 4. Apply options (page size, margins, landscape, etc.)
	// 5. Return PDF bytes
	return nil, ErrNotImplemented
}
