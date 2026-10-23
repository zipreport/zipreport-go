package processor

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout is the HTTP client timeout used when WithTimeout is not given.
const defaultTimeout = 5 * time.Minute

// defaultMaxResponseSize is the response size limit used when WithMaxResponseSize is not given or <= 0.
const defaultMaxResponseSize int64 = 512 << 20

// maxErrorBodySize bounds how much of an error response is included in the returned error.
const maxErrorBodySize = 1024

// ErrResponseTooLarge is returned when the server response exceeds the configured limit.
var ErrResponseTooLarge = errors.New("response exceeds size limit")

// ServerBackend renders PDFs via zipreport-server HTTP API.
type ServerBackend struct {
	url        string
	apiKey     string
	apiVersion int
	httpClient *http.Client
	secureSSL  bool
	timeout    time.Duration
	timeoutSet bool

	maxResponseSize int64
}

// ServerOption configures a ServerBackend.
type ServerOption func(*ServerBackend)

// WithAPIVersion sets the API version.
func WithAPIVersion(version int) ServerOption {
	return func(b *ServerBackend) {
		b.apiVersion = version
	}
}

// WithHTTPClient sets a custom HTTP client. The client is used as given:
// WithSecureSSL has no effect on it, and WithTimeout applies to a copy.
func WithHTTPClient(client *http.Client) ServerOption {
	return func(b *ServerBackend) {
		b.httpClient = client
	}
}

// WithSecureSSL controls TLS certificate verification. Verification is
// skipped by default; pass true to enable it.
func WithSecureSSL(secure bool) ServerOption {
	return func(b *ServerBackend) {
		b.secureSSL = secure
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ServerOption {
	return func(b *ServerBackend) {
		b.timeout = timeout
		b.timeoutSet = true
	}
}

// WithMaxResponseSize sets the maximum accepted response size in bytes (<= 0 means the default, 512 MiB).
func WithMaxResponseSize(n int64) ServerOption {
	return func(b *ServerBackend) {
		b.maxResponseSize = n
	}
}

// NewServerBackend creates a new ServerBackend.
func NewServerBackend(url, apiKey string, opts ...ServerOption) *ServerBackend {
	b := &ServerBackend{
		url:        url,
		apiKey:     apiKey,
		apiVersion: 2,
		secureSSL:  false, // matching Python behavior
		timeout:    defaultTimeout,

		maxResponseSize: defaultMaxResponseSize,
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.maxResponseSize <= 0 {
		b.maxResponseSize = defaultMaxResponseSize
	}

	if b.httpClient != nil {
		if b.timeoutSet {
			// Copy so the caller's client is not modified
			c := *b.httpClient
			c.Timeout = b.timeout
			b.httpClient = &c
		}
		return b
	}

	// Always set TLSClientConfig: a non-nil value keeps both modes on HTTP/1.1, as before
	b.httpClient = &http.Client{
		Timeout: b.timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !b.secureSSL},
		},
	}
	return b
}

// Render sends the ZPT to zipreport-server and returns the PDF.
func (b *ServerBackend) Render(zptData []byte, options map[string]interface{}) ([]byte, error) {
	// Build multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add ZPT file as "report" field (matching Python implementation)
	part, err := writer.CreateFormFile("report", "report.zpt")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(zptData); err != nil {
		return nil, fmt.Errorf("failed to write zpt data: %w", err)
	}

	// Add options as individual form fields (matching Python implementation)
	for k, v := range options {
		var strVal string
		switch val := v.(type) {
		case bool:
			strVal = fmt.Sprintf("%t", val)
		case string:
			strVal = val
		default:
			strVal = fmt.Sprintf("%v", val)
		}
		if err := writer.WriteField(k, strVal); err != nil {
			return nil, fmt.Errorf("failed to write field %s: %w", k, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Build request
	endpoint := fmt.Sprintf("%s/v%d/render", b.url, b.apiVersion)
	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Auth-Key", b.apiKey)

	// Execute request
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status, reading at most a bounded error body
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize+1))
		suffix := ""
		if len(body) > maxErrorBodySize {
			body = body[:maxErrorBodySize]
			suffix = "... (truncated)"
		}
		return nil, fmt.Errorf("server error (status %d): %s%s", resp.StatusCode, strings.ToValidUTF8(string(body), ""), suffix)
	}

	// Check content type
	if resp.Header.Get("Content-Type") != "application/pdf" {
		return nil, fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}

	// Read response
	responseData, err := io.ReadAll(io.LimitReader(resp.Body, saturatingInc(b.maxResponseSize)))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if int64(len(responseData)) > b.maxResponseSize {
		return nil, fmt.Errorf("%w (limit %d bytes)", ErrResponseTooLarge, b.maxResponseSize)
	}

	return responseData, nil
}

// URL returns the server URL.
func (b *ServerBackend) URL() string {
	return b.url
}

// APIVersion returns the API version.
func (b *ServerBackend) APIVersion() int {
	return b.apiVersion
}

// saturatingInc returns n+1, or n when n+1 would overflow.
func saturatingInc(n int64) int64 {
	if n == math.MaxInt64 {
		return n
	}
	return n + 1
}
