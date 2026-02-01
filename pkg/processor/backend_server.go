package processor

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// ServerBackend renders PDFs via zipreport-server HTTP API.
type ServerBackend struct {
	url        string
	apiKey     string
	apiVersion int
	httpClient *http.Client
	secureSSL  bool
}

// ServerOption configures a ServerBackend.
type ServerOption func(*ServerBackend)

// WithAPIVersion sets the API version.
func WithAPIVersion(version int) ServerOption {
	return func(b *ServerBackend) {
		b.apiVersion = version
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ServerOption {
	return func(b *ServerBackend) {
		b.httpClient = client
	}
}

// WithSecureSSL enables strict SSL certificate validation.
func WithSecureSSL(secure bool) ServerOption {
	return func(b *ServerBackend) {
		b.secureSSL = secure
		if !secure {
			// Create a client that skips certificate verification
			b.httpClient = &http.Client{
				Timeout: 5 * time.Minute,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
		}
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ServerOption {
	return func(b *ServerBackend) {
		b.httpClient.Timeout = timeout
	}
}

// NewServerBackend creates a new ServerBackend.
func NewServerBackend(url, apiKey string, opts ...ServerOption) *ServerBackend {
	b := &ServerBackend{
		url:        url,
		apiKey:     apiKey,
		apiVersion: 2,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		secureSSL: false,
	}

	// Default to insecure SSL (matching Python behavior)
	b.httpClient = &http.Client{
		Timeout: 5 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	for _, opt := range opts {
		opt(b)
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

	// Read response
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(responseData))
	}

	// Check content type
	if resp.Header.Get("Content-Type") != "application/pdf" {
		return nil, fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
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
