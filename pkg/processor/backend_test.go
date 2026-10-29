package processor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

// newTLSPDFServer returns a self-signed TLS server that counts requests carrying X-Auth-Key
func newTLSPDFServer(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Key") != "" {
			atomic.AddInt32(hits, 1)
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestServerBackend_SecureSSL_RejectsUntrustedCert(t *testing.T) {
	tests := []struct {
		name string
		opts []ServerOption
	}{
		{"secure only", []ServerOption{WithSecureSSL(true)}},
		{"secure then timeout", []ServerOption{WithSecureSSL(true), WithTimeout(time.Minute)}},
		{"timeout then secure", []ServerOption{WithTimeout(time.Minute), WithSecureSSL(true)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int32
			srv := newTLSPDFServer(t, &hits)
			b := NewServerBackend(srv.URL, "key", tt.opts...)
			_, err := b.Render([]byte("zpt"), map[string]interface{}{})
			if err == nil {
				t.Fatal("expected certificate error, got nil")
			}
			if !strings.Contains(err.Error(), "certificate") {
				t.Errorf("error = %v, want certificate error", err)
			}
			if n := atomic.LoadInt32(&hits); n != 0 {
				t.Errorf("server received %d authenticated requests, want 0", n)
			}
		})
	}
}

func TestServerBackend_DefaultSkipsVerification(t *testing.T) {
	var hits int32
	srv := newTLSPDFServer(t, &hits)
	b := NewServerBackend(srv.URL, "key")
	if _, err := b.Render([]byte("zpt"), map[string]interface{}{}); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Errorf("hits = %d, want 1", n)
	}
}

func TestServerBackend_WithTimeout_DoesNotMutateCallerClient(t *testing.T) {
	custom := &http.Client{Timeout: 3 * time.Second}
	b := NewServerBackend("http://x", "k", WithHTTPClient(custom), WithTimeout(time.Hour))

	if custom.Timeout != 3*time.Second {
		t.Errorf("caller client timeout = %v, want 3s", custom.Timeout)
	}
	if b.httpClient.Timeout != time.Hour {
		t.Errorf("backend timeout = %v, want 1h", b.httpClient.Timeout)
	}
	if b.httpClient == custom {
		t.Error("backend should hold a copy of the caller client")
	}
}

func TestServerBackend_WithTimeout_NilClient(t *testing.T) {
	b := NewServerBackend("http://x", "k", WithHTTPClient(nil), WithTimeout(time.Minute))

	if b.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if b.httpClient.Timeout != time.Minute {
		t.Errorf("timeout = %v, want 1m", b.httpClient.Timeout)
	}
}

func TestServerBackend_SecureSSL_LastWins(t *testing.T) {
	var hits int32
	srv := newTLSPDFServer(t, &hits)

	b := NewServerBackend(srv.URL, "key", WithSecureSSL(true), WithSecureSSL(false))
	if _, err := b.Render([]byte("zpt"), map[string]interface{}{}); err != nil {
		t.Errorf("true then false: Render failed: %v", err)
	}

	b = NewServerBackend(srv.URL, "key", WithSecureSSL(false), WithSecureSSL(true))
	_, err := b.Render([]byte("zpt"), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Errorf("false then true: error = %v, want certificate error", err)
	}
}

func TestServerBackend_CustomClientWinsOverSecureSSL(t *testing.T) {
	origT := &http.Transport{}
	custom := &http.Client{Transport: origT}
	b := NewServerBackend("http://x", "k", WithHTTPClient(custom), WithSecureSSL(false))

	if b.httpClient != custom {
		t.Error("custom client was replaced")
	}
	if custom.Transport != origT {
		t.Error("custom transport was replaced")
	}
	if origT.TLSClientConfig != nil {
		t.Error("custom transport was modified")
	}
}

func pdfServer(t *testing.T, size int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte(strings.Repeat("p", size)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestServerBackend_Render_ResponseTooLarge(t *testing.T) {
	srv := pdfServer(t, 2048)
	_, err := NewServerBackend(srv.URL, "k", WithMaxResponseSize(1024)).Render([]byte("z"), nil)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected ErrResponseTooLarge, got %v", err)
	}
}

func TestServerBackend_Render_ResponseAtLimit(t *testing.T) {
	srv := pdfServer(t, 1024)
	out, err := NewServerBackend(srv.URL, "k", WithMaxResponseSize(1024)).Render([]byte("z"), nil)
	if err != nil || len(out) != 1024 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
}

func TestServerBackend_Render_MaxInt64Limit(t *testing.T) {
	srv := pdfServer(t, 100)
	out, err := NewServerBackend(srv.URL, "k", WithMaxResponseSize(math.MaxInt64)).Render([]byte("z"), nil)
	if err != nil || len(out) != 100 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
}

func TestServerBackend_Render_ErrorBodyTruncated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 10000)))
	}))
	defer srv.Close()
	_, err := NewServerBackend(srv.URL, "k").Render([]byte("z"), nil)
	if err == nil || !strings.Contains(err.Error(), "status 500") || !strings.Contains(err.Error(), "(truncated)") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(err.Error()) >= 1200 {
		t.Errorf("error too long: %d", len(err.Error()))
	}
}

func TestServerBackend_Render_LargeErrorKeepsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
	}))
	defer srv.Close()
	_, err := NewServerBackend(srv.URL, "k", WithMaxResponseSize(1024)).Render([]byte("z"), nil)
	if err == nil || !strings.Contains(err.Error(), "status 502") || errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerBackend_Render_NonPositiveMaxUsesDefault(t *testing.T) {
	srv := pdfServer(t, 100)
	for _, n := range []int64{0, -1} {
		for _, custom := range []bool{false, true} {
			n, custom := n, custom
			t.Run(fmt.Sprintf("max=%d/customClient=%v", n, custom), func(t *testing.T) {
				opts := []ServerOption{WithMaxResponseSize(n)}
				if custom {
					opts = append(opts, WithHTTPClient(srv.Client()))
				}
				out, err := NewServerBackend(srv.URL, "k", opts...).Render([]byte("z"), nil)
				if err != nil || len(out) != 100 {
					t.Fatalf("len=%d err=%v", len(out), err)
				}
			})
		}
	}
}

func TestNewServerBackend_DefaultLimits(t *testing.T) {
	b := NewServerBackend("http://x", "k")
	if b.maxResponseSize != 512<<20 {
		t.Errorf("maxResponseSize = %d, want %d", b.maxResponseSize, int64(512<<20))
	}
	if b.httpClient.Timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want 5m", b.httpClient.Timeout)
	}
}

func TestServerBackend_Render_ErrorBodyReadIsBounded(t *testing.T) {
	const total = 64 << 20
	var written atomic.Int64
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		chunk := []byte(strings.Repeat("x", 32<<10))
		for written.Load() < total {
			n, err := w.Write(chunk)
			written.Add(int64(n))
			if err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	_, err := NewServerBackend(srv.URL, "k").Render([]byte("z"), nil)
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected status 500 error, got %v", err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("handler did not finish")
	}
	if written.Load() >= total {
		t.Fatalf("server wrote all %d bytes: error body read is unbounded", written.Load())
	}
}

func TestServerBackend_Render_RefusesCrossHostRedirect(t *testing.T) {
	for _, code := range []int{http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var hits int32
			b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&hits, 1)
				w.Header().Set("Content-Type", "application/pdf")
				_, _ = w.Write([]byte("%PDF-1.4"))
			}))
			t.Cleanup(b.Close)
			a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, b.URL+"/v2/render", code)
			}))
			t.Cleanup(a.Close)

			_, err := NewServerBackend(a.URL, "k").Render([]byte("zpt"), nil)
			if n := atomic.LoadInt32(&hits); n != 0 {
				t.Errorf("redirect target got %d requests, want 0", n)
			}
			if err == nil {
				t.Errorf("err = nil, want error")
			}
			if !errors.Is(err, ErrRedirectRefused) {
				t.Errorf("err = %v, want ErrRedirectRefused", err)
			}
		})
	}
}

func TestServerBackend_Render_FollowsSameHostRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/render" {
			http.Redirect(w, r, "/v2/render2", http.StatusTemporaryRedirect)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("X-Auth-Key") != "k" || len(body) == 0 {
			http.Error(w, "bad", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	t.Cleanup(srv.Close)

	if _, err := NewServerBackend(srv.URL, "k").Render([]byte("zpt"), nil); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
}

func TestCheckRedirect(t *testing.T) {
	req := func(u string) *http.Request { return httptest.NewRequest(http.MethodPost, u, nil) }
	tooMany := make([]*http.Request, 10)
	for i := range tooMany {
		tooMany[i] = req("http://h/a")
	}
	tests := []struct {
		name    string
		via     []*http.Request
		to      string
		wantErr bool
		refused bool
	}{
		{"same_host", []*http.Request{req("http://h:1/a")}, "http://h:1/b", false, false},
		{"different_port", []*http.Request{req("http://h:1/a")}, "http://h:2/a", true, true},
		{"different_hostname", []*http.Request{req("http://h/a")}, "http://other/a", true, true},
		{"https_to_http", []*http.Request{req("https://h/a")}, "http://h/a", true, true},
		{"http_to_https", []*http.Request{req("http://h/a")}, "https://h/a", false, false},
		{"too_many_redirects", tooMany, "http://h/b", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRedirect(req(tt.to), tt.via)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrRedirectRefused) != tt.refused {
				t.Errorf("errors.Is(err, ErrRedirectRefused) = %v, want %v (err = %v)", !tt.refused, tt.refused, err)
			}
		})
	}
}

func TestServerBackend_CustomClientKeepsRedirectPolicy(t *testing.T) {
	sentinel := errors.New("custom policy")
	tests := []struct {
		name string
		opts []ServerOption
	}{
		{"as_given", nil},
		{"with_timeout", []ServerOption{WithTimeout(time.Minute)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return sentinel }}
			b := NewServerBackend("http://a.example", "k", append([]ServerOption{WithHTTPClient(client)}, tt.opts...)...)
			next := httptest.NewRequest(http.MethodPost, "http://b.example/x", nil)
			via := []*http.Request{httptest.NewRequest(http.MethodPost, "http://a.example/x", nil)}
			if err := b.httpClient.CheckRedirect(next, via); !errors.Is(err, sentinel) {
				t.Errorf("err = %v, want sentinel", err)
			}
		})
	}
}

func TestServerBackend_Render_TrailingSlash(t *testing.T) {
	for _, tt := range []struct{ name, suffix string }{{"one_slash", "/"}, {"two_slashes", "//"}} {
		t.Run(tt.name, func(t *testing.T) {
			var path atomic.Value
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path.Store(r.URL.Path)
				w.Header().Set("Content-Type", "application/pdf")
				_, _ = w.Write([]byte("%PDF-1.4"))
			}))
			t.Cleanup(srv.Close)

			given := srv.URL + tt.suffix
			b := NewServerBackend(given, "k")
			if _, err := b.Render([]byte("zpt"), nil); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if got, _ := path.Load().(string); got != "/v2/render" {
				t.Errorf("path = %q, want /v2/render", got)
			}
			if b.URL() != given {
				t.Errorf("URL() = %q, want %q", b.URL(), given)
			}
		})
	}
}

// newBlockingPDFServer returns a server that signals on the returned channel once it has the
// request, then waits for the client to disconnect or for 2s before answering with a PDF.
func newBlockingPDFServer(t *testing.T) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	started := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// net/http only notices a client disconnect after the body has been read
		_, _ = io.Copy(io.Discard, r.Body)
		started <- struct{}{}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, started
}

func TestServerBackend_RenderContext_Cancel(t *testing.T) {
	srv, started := newBlockingPDFServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-started
		cancel()
	}()

	start := time.Now()
	_, err := NewServerBackend(srv.URL, "k").RenderContext(ctx, []byte("zpt"), nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("RenderContext took %v after cancel", d)
	}
}

func TestServerBackend_RenderContext_Deadline(t *testing.T) {
	srv, _ := newBlockingPDFServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := NewServerBackend(srv.URL, "k").RenderContext(ctx, []byte("zpt"), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("RenderContext took %v", d)
	}
}
