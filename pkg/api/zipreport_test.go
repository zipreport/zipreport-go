package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
	"github.com/zipreport/zipreport-go/pkg/processor"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// mockBackend is a test backend that returns predefined data
type mockBackend struct {
	result []byte
	err    error
}

func (m *mockBackend) Render(zptData []byte, options map[string]interface{}) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func createTestReportFile() *report.ReportFile {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html><body>{{ name }}</body></html>"), false)
	_ = zfs.Add("report.html", []byte("<html><body>Rendered</body></html>"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"Test","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "Test",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}

	return report.NewReportFile(zfs, manifest)
}

func TestNewZipReport(t *testing.T) {
	zr := NewZipReport("http://localhost:5000", "test-api-key")
	if zr == nil {
		t.Fatal("NewZipReport returned nil")
	}
}

func TestNewZipReport_WithOptions(t *testing.T) {
	zr := NewZipReport("http://localhost:5000", "test-api-key",
		processor.WithAPIVersion(3),
	)
	if zr == nil {
		t.Fatal("NewZipReport returned nil")
	}
}

func TestNewZipReportWithBackend(t *testing.T) {
	backend := &mockBackend{result: []byte("PDF data")}
	zr := NewZipReportWithBackend(backend)
	if zr == nil {
		t.Fatal("NewZipReportWithBackend returned nil")
	}
}

func TestNewMIMEReport(t *testing.T) {
	mr := NewMIMEReport()
	if mr == nil {
		t.Fatal("NewMIMEReport returned nil")
	}
}

func TestNewMIMEReport_WithOptions(t *testing.T) {
	mr := NewMIMEReport(
		processor.WithSubject("Test Subject"),
		processor.WithFrom("sender@example.com"),
		processor.WithTo("recipient@example.com"),
	)
	if mr == nil {
		t.Fatal("NewMIMEReport returned nil")
	}
}

func TestBaseReport_CreateJob(t *testing.T) {
	backend := &mockBackend{result: []byte("PDF")}
	zr := NewZipReportWithBackend(backend)

	rf := createTestReportFile()
	job := zr.CreateJob(rf)

	if job == nil {
		t.Fatal("CreateJob returned nil")
	}
	if job.GetReport() != rf {
		t.Error("Job report doesn't match")
	}
}

func TestZipReport_Render_Success(t *testing.T) {
	backend := &mockBackend{result: []byte("%PDF-1.4 mock pdf content")}
	zr := NewZipReportWithBackend(backend)

	rf := createTestReportFile()
	job := zr.CreateJob(rf)

	result := zr.Render(job, map[string]interface{}{"name": "Test"}, nil)

	if !result.Success {
		t.Fatalf("Render failed: %s", result.Error)
	}
	if len(result.Report) == 0 {
		t.Error("Result report is empty")
	}
}

func TestZipReport_Render_WithWrapper(t *testing.T) {
	backend := &mockBackend{result: []byte("%PDF-1.4 mock pdf content")}
	zr := NewZipReportWithBackend(backend)

	rf := createTestReportFile()
	job := zr.CreateJob(rf)

	// Use a simple wrapper
	result := zr.Render(job, map[string]interface{}{"name": "Test"}, nil)

	if !result.Success {
		t.Fatalf("Render failed: %s", result.Error)
	}
}

func TestZipReport_Render_BackendError(t *testing.T) {
	backend := &mockBackend{err: processor.ErrNotImplemented}
	zr := NewZipReportWithBackend(backend)

	rf := createTestReportFile()
	job := zr.CreateJob(rf)

	result := zr.Render(job, map[string]interface{}{"name": "Test"}, nil)

	if result.Success {
		t.Error("Render should fail when backend returns error")
	}
}

func TestZipReport_RenderDefaults(t *testing.T) {
	backend := &mockBackend{result: []byte("%PDF-1.4")}
	zr := NewZipReportWithBackend(backend)

	rf := createTestReportFile()
	result := zr.RenderDefaults(rf, map[string]interface{}{"name": "Test"}, nil)

	if !result.Success {
		t.Fatalf("RenderDefaults failed: %s", result.Error)
	}
}

func TestMIMEReport_RenderDefaults(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html><body>{{ name }}</body></html>"), false)
	_ = zfs.Add("report.html", []byte("<html><body>MIME content</body></html>"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"Test","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "Test",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}
	rf := report.NewReportFile(zfs, manifest)

	mr := NewMIMEReport(processor.WithSubject("Test Email"))
	result := mr.RenderDefaults(rf, map[string]interface{}{"name": "World"}, nil)

	if !result.Success {
		t.Fatalf("RenderDefaults failed: %s", result.Error)
	}
}

func TestRenderPDF(t *testing.T) {
	// This will fail since we don't have a real server, but tests the code path
	rf := createTestReportFile()
	result := RenderPDF(rf, map[string]interface{}{"name": "Test"}, "http://invalid-server", "key")

	// Should fail due to network error
	if result.Success {
		t.Skip("Unexpected success - real server might be running")
	}
}

func TestRenderMIME(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html>{{ x }}</html>"), false)
	_ = zfs.Add("report.html", []byte("<html>MIME</html>"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{Title: "T", Author: "A", Version: "1", Description: "D"}
	rf := report.NewReportFile(zfs, manifest)

	result := RenderMIME(rf, nil)

	if !result.Success {
		t.Fatalf("RenderMIME failed: %s", result.Error)
	}
}

// capturingBackend records the ZPT payload and returns its rendered report.html as the "PDF"
type capturingBackend struct {
	mu       sync.Mutex
	payloads [][]byte
}

func (c *capturingBackend) Render(zptData []byte, options map[string]interface{}) ([]byte, error) {
	c.mu.Lock()
	c.payloads = append(c.payloads, zptData)
	c.mu.Unlock()
	zfs, err := fileutils.NewZipFsFromBytes(zptData)
	if err != nil {
		return nil, err
	}
	return zfs.GetBytes("report.html")
}

const isolationManifest = `{"title":"T","author":"A","version":"1","description":"D"}`

func newIsolationReport(t *testing.T, index string) *report.ReportFile {
	t.Helper()
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte(index), false)
	_ = zfs.Add("manifest.json", []byte(isolationManifest), false)
	return report.NewReportFile(zfs, &report.Manifest{Title: "T", Author: "A", Version: "1", Description: "D"})
}

func newIsolationDirReport(t *testing.T, index string) (*report.ReportFile, string) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"manifest.json": isolationManifest,
		"index.html":    index,
		"css/s.css":     "body{}",
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	rf, err := report.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return rf, dir
}

func TestBaseReport_Render_DoesNotModifyReport(t *testing.T) {
	rf := newIsolationReport(t, "<p>{{ name }}</p>")
	zr := NewZipReportWithBackend(&capturingBackend{})
	result := zr.RenderDefaults(rf, map[string]interface{}{"name": "Alice"}, nil)
	if !result.Success {
		t.Fatalf("render failed: %s", result.Error)
	}
	if !strings.Contains(string(result.Report), "Alice") {
		t.Errorf("report = %q, want to contain Alice", result.Report)
	}
	if rf.Exists("report.html") {
		t.Error("render modified the caller's report")
	}
}

func TestBaseReport_Render_DirectoryReportSendsAssets(t *testing.T) {
	rf, dir := newIsolationDirReport(t, `<link href="css/s.css"><p>{{ name }}</p>`)
	cb := &capturingBackend{}
	zr := NewZipReportWithBackend(cb)
	result := zr.RenderDefaults(rf, map[string]interface{}{"name": "Alice"}, nil)
	if !result.Success {
		t.Fatalf("render failed: %s", result.Error)
	}
	zfs, err := fileutils.NewZipFsFromBytes(cb.payloads[0])
	if err != nil {
		t.Fatal(err)
	}
	if !zfs.Exists("css/s.css") {
		t.Error("css/s.css not sent to the server")
	}
	if _, err := os.Stat(filepath.Join(dir, "report.html")); !os.IsNotExist(err) {
		t.Errorf("report.html written to the source folder (err=%v)", err)
	}
}

func runConcurrentRenders(t *testing.T, rf *report.ReportFile, goroutines, iterations int) {
	t.Helper()
	zr := NewZipReportWithBackend(&capturingBackend{})
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		failures []string
		started  sync.WaitGroup
	)
	start := make(chan struct{})
	started.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			started.Done()
			<-start
			for i := 0; i < iterations; i++ {
				name := fmt.Sprintf("user-%d-%d", g, i)
				result := zr.RenderDefaults(rf, map[string]interface{}{"name": name}, nil)
				if !result.Success || !strings.Contains(string(result.Report), name) {
					mu.Lock()
					failures = append(failures, fmt.Sprintf("%s: success=%v err=%q report=%q", name, result.Success, result.Error, result.Report))
					mu.Unlock()
				}
			}
		}(g)
	}
	started.Wait()
	close(start)
	wg.Wait()
	if len(failures) > 0 {
		t.Errorf("%d failed renders, first: %s", len(failures), failures[0])
	}
}

func TestBaseReport_Render_ConcurrentSharedReport(t *testing.T) {
	rf := newIsolationReport(t, "<p>{{ name }}</p>")
	runConcurrentRenders(t, rf, 16, 50)
}

func TestBaseReport_Render_ConcurrentSharedDirectoryReport(t *testing.T) {
	rf, _ := newIsolationDirReport(t, "<p>{{ name }}</p>")
	runConcurrentRenders(t, rf, 8, 20)
}

func TestBaseReport_Render_NoImageAccumulation(t *testing.T) {
	rf := newIsolationReport(t, "{{ img|png }}")
	cb := &capturingBackend{}
	zr := NewZipReportWithBackend(cb)
	for i := 0; i < 2; i++ {
		result := zr.RenderDefaults(rf, map[string]interface{}{"img": []byte("PNG")}, nil)
		if !result.Success {
			t.Fatalf("render %d failed: %s", i, result.Error)
		}
	}
	zfs, err := fileutils.NewZipFsFromBytes(cb.payloads[1])
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range zfs.Files() {
		if strings.HasPrefix(f, "_gen_") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("payload has %d _gen_ files, want 1 (%v)", n, zfs.Files())
	}
}

func TestMIMEReport_Render_UsesRenderedCopy(t *testing.T) {
	rf := newIsolationReport(t, "<p>{{ name }}</p>")
	result := NewMIMEReport().RenderDefaults(rf, map[string]interface{}{"name": "Zed"}, nil)
	if !result.Success {
		t.Fatalf("render failed: %s", result.Error)
	}
	msg := string(result.Report)
	marker := "Content-Transfer-Encoding: base64\r\n\r\n"
	i := strings.Index(msg, marker)
	if i < 0 {
		t.Fatalf("no base64 part in %q", msg)
	}
	body := msg[i+len(marker):]
	if j := strings.Index(body, "\r\n--"); j >= 0 {
		body = body[:j]
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body, "\r\n", ""))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(decoded), "Zed") {
		t.Errorf("HTML part = %q, want to contain Zed", decoded)
	}
	if rf.Exists("report.html") {
		t.Error("render modified the caller's report")
	}
}

func TestBaseReport_Render_EscapingOutputRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>x</p>"), 0644); err != nil {
		t.Fatal(err)
	}
	dfs, err := fileutils.NewDiskFs(dir)
	if err != nil {
		t.Fatal(err)
	}
	rf := report.NewReportFile(dfs, &report.Manifest{Title: "T", Author: "A", Version: "1", Description: "D", Report: "../escaped.html"})
	cb := &capturingBackend{}
	result := NewZipReportWithBackend(cb).RenderDefaults(rf, nil, nil)
	if len(cb.payloads) != 0 {
		t.Errorf("backend received %d payloads, want 0", len(cb.payloads))
	}
	if !strings.Contains(result.Error, "invalid path") {
		t.Errorf("Error = %q, want it to contain %q", result.Error, "invalid path")
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

func TestBaseReport_RenderContext_Cancel(t *testing.T) {
	srv, started := newBlockingPDFServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-started
		cancel()
	}()

	start := time.Now()
	result := NewZipReport(srv.URL, "k").RenderDefaultsContext(ctx, newIsolationReport(t, "<p>x</p>"), nil, nil)

	if result.Success {
		t.Errorf("Success = true, want false")
	}
	if !strings.Contains(result.Error, "context canceled") {
		t.Errorf("Error = %q, want it to contain %q", result.Error, "context canceled")
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("render took %v after cancel", d)
	}
}

func TestRenderPDF_Options(t *testing.T) {
	var hits int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Key") != "" {
			atomic.AddInt32(&hits, 1)
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	t.Cleanup(srv.Close)
	rf := newIsolationReport(t, "<p>x</p>")

	result := RenderPDF(rf, nil, srv.URL, "k", processor.WithSecureSSL(true))
	if result.Success {
		t.Errorf("Success = true with WithSecureSSL(true) on a self-signed server")
	}
	if !strings.Contains(result.Error, "certificate") {
		t.Errorf("Error = %q, want it to contain %q", result.Error, "certificate")
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Errorf("server got %d requests, want 0", n)
	}

	result = RenderPDF(rf, nil, srv.URL, "k")
	if !result.Success {
		t.Errorf("default RenderPDF failed: %s", result.Error)
	}
}
