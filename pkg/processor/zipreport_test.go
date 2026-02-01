package processor

import (
	"errors"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
	"github.com/zipreport/zipreport-go/pkg/report"
)

// mockBackend is a test backend for ZipReportProcessor
type mockBackend struct {
	result   []byte
	err      error
	received struct {
		zptData []byte
		options map[string]interface{}
	}
}

func (m *mockBackend) Render(zptData []byte, options map[string]interface{}) ([]byte, error) {
	m.received.zptData = zptData
	m.received.options = options
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func createTestReportForProcessor() *report.ReportFile {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html><body>Test</body></html>"), false)
	_ = zfs.Add("report.html", []byte("<html><body>Rendered</body></html>"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "T",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}

	return report.NewReportFile(zfs, manifest)
}

func TestNewZipReportProcessor(t *testing.T) {
	backend := &mockBackend{result: []byte("PDF")}
	p := NewZipReportProcessor(backend)

	if p == nil {
		t.Fatal("NewZipReportProcessor returned nil")
	}
	if p.backend != backend {
		t.Error("Backend not set correctly")
	}
}

func TestZipReportProcessor_Backend(t *testing.T) {
	backend := &mockBackend{result: []byte("PDF")}
	p := NewZipReportProcessor(backend)

	if p.Backend() != backend {
		t.Error("Backend() returned wrong backend")
	}
}

func TestZipReportProcessor_Process_Success(t *testing.T) {
	backend := &mockBackend{result: []byte("%PDF-1.4 content")}
	p := NewZipReportProcessor(backend)

	rf := createTestReportForProcessor()
	job := report.NewReportJob(rf)

	result := p.Process(job)

	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	if string(result.Report) != "%PDF-1.4 content" {
		t.Errorf("Result = %q, want %q", string(result.Report), "%PDF-1.4 content")
	}
}

func TestZipReportProcessor_Process_WithOptions(t *testing.T) {
	backend := &mockBackend{result: []byte("PDF")}
	p := NewZipReportProcessor(backend)

	rf := createTestReportForProcessor()
	job := report.NewReportJob(rf)
	job.SetLandscape(true)
	job.SetPageSize(report.PageA4)

	result := p.Process(job)

	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}

	// Verify options were passed to backend
	if backend.received.options["landscape"] != true {
		t.Error("landscape option not passed to backend")
	}
	if backend.received.options["page_size"] != string(report.PageA4) {
		t.Error("page_size option not passed to backend")
	}
}

func TestZipReportProcessor_Process_BackendError(t *testing.T) {
	backend := &mockBackend{err: errors.New("rendering failed")}
	p := NewZipReportProcessor(backend)

	rf := createTestReportForProcessor()
	job := report.NewReportJob(rf)

	result := p.Process(job)

	if result.Success {
		t.Error("Process should fail when backend returns error")
	}
	if result.Error == "" {
		t.Error("Error message should not be empty")
	}
}
