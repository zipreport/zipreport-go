package processor

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewServerBackend(t *testing.T) {
	b := NewServerBackend("http://localhost:5000", "test-key")

	if b == nil {
		t.Fatal("NewServerBackend returned nil")
	}
	if b.url != "http://localhost:5000" {
		t.Errorf("url = %q, want %q", b.url, "http://localhost:5000")
	}
	if b.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", b.apiKey, "test-key")
	}
	if b.apiVersion != 2 {
		t.Errorf("apiVersion = %d, want 2", b.apiVersion)
	}
}

func TestServerBackend_WithAPIVersion(t *testing.T) {
	b := NewServerBackend("http://localhost:5000", "key", WithAPIVersion(3))

	if b.apiVersion != 3 {
		t.Errorf("apiVersion = %d, want 3", b.apiVersion)
	}
}

func TestServerBackend_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 10 * time.Second}
	b := NewServerBackend("http://localhost:5000", "key", WithHTTPClient(customClient))

	if b.httpClient != customClient {
		t.Error("Custom HTTP client not set")
	}
}

func TestServerBackend_WithSecureSSL(t *testing.T) {
	b := NewServerBackend("http://localhost:5000", "key", WithSecureSSL(false))

	if b.secureSSL != false {
		t.Error("secureSSL should be false")
	}
}

func TestServerBackend_WithTimeout(t *testing.T) {
	b := NewServerBackend("http://localhost:5000", "key", WithTimeout(30*time.Second))

	if b.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want %v", b.httpClient.Timeout, 30*time.Second)
	}
}

func TestServerBackend_URL(t *testing.T) {
	b := NewServerBackend("http://example.com", "key")

	if b.URL() != "http://example.com" {
		t.Errorf("URL() = %q, want %q", b.URL(), "http://example.com")
	}
}

func TestServerBackend_APIVersion(t *testing.T) {
	b := NewServerBackend("http://example.com", "key", WithAPIVersion(5))

	if b.APIVersion() != 5 {
		t.Errorf("APIVersion() = %d, want 5", b.APIVersion())
	}
}

func TestServerBackend_Render_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v2/render" {
			t.Errorf("Path = %s, want /v2/render", r.URL.Path)
		}
		if r.Header.Get("X-Auth-Key") != "test-key" {
			t.Errorf("X-Auth-Key = %q, want %q", r.Header.Get("X-Auth-Key"), "test-key")
		}

		// Return PDF
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 test"))
	}))
	defer server.Close()

	b := NewServerBackend(server.URL, "test-key")
	result, err := b.Render([]byte("ZPT data"), nil)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "%PDF-1.4 test" {
		t.Errorf("Result = %q, want %q", string(result), "%PDF-1.4 test")
	}
}

func TestServerBackend_Render_WithOptions(t *testing.T) {
	var receivedLandscape, receivedFormat string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(10 << 20)
		receivedLandscape = r.FormValue("landscape")
		receivedFormat = r.FormValue("format")

		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("PDF"))
	}))
	defer server.Close()

	b := NewServerBackend(server.URL, "key")
	_, err := b.Render([]byte("ZPT"), map[string]interface{}{
		"landscape": true,
		"format":    "A4",
	})

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if receivedLandscape != "true" {
		t.Errorf("landscape = %q, want %q", receivedLandscape, "true")
	}
	if receivedFormat != "A4" {
		t.Errorf("format = %q, want %q", receivedFormat, "A4")
	}
}

func TestServerBackend_Render_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal error"))
	}))
	defer server.Close()

	b := NewServerBackend(server.URL, "key")
	_, err := b.Render([]byte("ZPT"), nil)

	if err == nil {
		t.Error("Render should fail on server error")
	}
}

func TestServerBackend_Render_WrongContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>Error</html>"))
	}))
	defer server.Close()

	b := NewServerBackend(server.URL, "key")
	_, err := b.Render([]byte("ZPT"), nil)

	if err == nil {
		t.Error("Render should fail on wrong content type")
	}
}

func TestServerBackend_Render_ConnectionError(t *testing.T) {
	b := NewServerBackend("http://localhost:99999", "key")
	_, err := b.Render([]byte("ZPT"), nil)

	if err == nil {
		t.Error("Render should fail on connection error")
	}
}

// LocalBackend tests

func TestNewLocalBackend(t *testing.T) {
	b := NewLocalBackend()

	if b == nil {
		t.Fatal("NewLocalBackend returned nil")
	}
	if b.timeout != 120 {
		t.Errorf("timeout = %d, want 120", b.timeout)
	}
}

func TestLocalBackend_WithChromePath(t *testing.T) {
	b := NewLocalBackend(WithChromePath("/usr/bin/chromium"))

	if b.chromePath != "/usr/bin/chromium" {
		t.Errorf("chromePath = %q, want %q", b.chromePath, "/usr/bin/chromium")
	}
}

func TestLocalBackend_WithLocalTimeout(t *testing.T) {
	b := NewLocalBackend(WithLocalTimeout(60))

	if b.timeout != 60 {
		t.Errorf("timeout = %d, want 60", b.timeout)
	}
}

func TestLocalBackend_Render_NotImplemented(t *testing.T) {
	b := NewLocalBackend()

	_, err := b.Render([]byte("ZPT"), nil)

	if err != ErrNotImplemented {
		t.Errorf("err = %v, want ErrNotImplemented", err)
	}
}
