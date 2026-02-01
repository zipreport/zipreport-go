package processor

import (
	"strings"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
	"github.com/zipreport/zipreport-go/pkg/report"
)

func createTestReport() *report.ReportFile {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html><body>Test</body></html>"), false)
	_ = zfs.Add("report.html", []byte(`<html><body><img src="image.png"/></body></html>`), false)
	_ = zfs.Add("image.png", []byte("PNG data"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"Test Report","author":"Tester","version":"1.0","description":"Test"}`), false)

	manifest := &report.Manifest{
		Title:       "Test Report",
		Author:      "Test Author",
		Version:     "1.0.0",
		Description: "A test report",
	}

	return report.NewReportFile(zfs, manifest)
}

func TestNewMIMEProcessor(t *testing.T) {
	p := NewMIMEProcessor()
	if p == nil {
		t.Fatal("NewMIMEProcessor returned nil")
	}
}

func TestMIMEProcessor_WithOptions(t *testing.T) {
	p := NewMIMEProcessor(
		WithSubject("Test Subject"),
		WithFrom("sender@example.com"),
		WithTo("recipient@example.com", "another@example.com"),
	)

	if p.subject != "Test Subject" {
		t.Errorf("subject = %q, want %q", p.subject, "Test Subject")
	}
	if p.from != "sender@example.com" {
		t.Errorf("from = %q, want %q", p.from, "sender@example.com")
	}
	if len(p.to) != 2 {
		t.Errorf("to length = %d, want 2", len(p.to))
	}
}

func TestMIMEProcessor_Process(t *testing.T) {
	r := createTestReport()
	job := report.NewReportJob(r)

	p := NewMIMEProcessor(
		WithSubject("Test Email"),
		WithFrom("sender@example.com"),
		WithTo("recipient@example.com"),
	)

	result := p.Process(job)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}

	mimeData := string(result.Report)

	// Check headers
	if !strings.Contains(mimeData, "From: sender@example.com") {
		t.Error("MIME should contain From header")
	}
	if !strings.Contains(mimeData, "To: recipient@example.com") {
		t.Error("MIME should contain To header")
	}
	if !strings.Contains(mimeData, "Subject: Test Email") {
		t.Error("MIME should contain Subject header")
	}
	if !strings.Contains(mimeData, "MIME-Version: 1.0") {
		t.Error("MIME should contain MIME-Version header")
	}
	if !strings.Contains(mimeData, "multipart/related") {
		t.Error("MIME should be multipart/related")
	}
}

func TestMIMEProcessor_Process_NoReportHtml(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html>test</html>"), false)
	manifest := &report.Manifest{Title: "Test"}
	r := report.NewReportFile(zfs, manifest)

	job := report.NewReportJob(r)
	p := NewMIMEProcessor()

	result := p.Process(job)
	if result.Success {
		t.Error("Process should fail when report.html is missing")
	}
}

func TestMIMEProcessor_DefaultSubject(t *testing.T) {
	r := createTestReport()
	job := report.NewReportJob(r)

	p := NewMIMEProcessor() // No subject set

	result := p.Process(job)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}

	mimeData := string(result.Report)
	// Should use report title as subject
	if !strings.Contains(mimeData, "Subject: Test Report") {
		t.Error("MIME should use report title as default subject")
	}
}

func TestExtractResources(t *testing.T) {
	html := `
		<html>
		<head>
			<link href="style.css" rel="stylesheet">
		</head>
		<body>
			<img src="image.png">
			<img src='another.jpg'>
			<a href="https://example.com">External</a>
			<a href="//cdn.example.com/lib.js">Protocol-relative</a>
			<img src="data:image/png;base64,ABC123">
			<a href="#section">Anchor</a>
			<img src="cid:existing@msg">
		</body>
		</html>
	`

	resources := extractResources(html)

	expected := map[string]bool{
		"style.css":   true,
		"image.png":   true,
		"another.jpg": true,
	}

	if len(resources) != len(expected) {
		t.Errorf("Expected %d resources, got %d: %v", len(expected), len(resources), resources)
	}

	for _, r := range resources {
		if !expected[r] {
			t.Errorf("Unexpected resource: %s", r)
		}
	}
}

func TestExtractResources_Dedup(t *testing.T) {
	html := `
		<img src="image.png">
		<img src="image.png">
		<img src="image.png">
	`

	resources := extractResources(html)

	if len(resources) != 1 {
		t.Errorf("Expected 1 unique resource, got %d: %v", len(resources), resources)
	}
}

func TestIsExternalURL(t *testing.T) {
	tests := []struct {
		url      string
		external bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"//cdn.example.com", true},
		{"image.png", false},
		{"css/style.css", false},
		{"/absolute/path.js", false},
	}

	for _, tt := range tests {
		got := isExternalURL(tt.url)
		if got != tt.external {
			t.Errorf("isExternalURL(%q) = %v, want %v", tt.url, got, tt.external)
		}
	}
}

func TestGetMIMEType(t *testing.T) {
	tests := []struct {
		filename string
		contains string
	}{
		{"image.png", "image/png"},
		{"style.css", "text/css"},
		{"script.js", "javascript"},
		{"photo.jpg", "image/jpeg"},
		{"unknown.zzz", "application/octet-stream"},
	}

	for _, tt := range tests {
		got := getMIMEType(tt.filename)
		if !strings.Contains(got, tt.contains) {
			t.Errorf("getMIMEType(%q) = %q, should contain %q", tt.filename, got, tt.contains)
		}
	}
}

func TestGenerateBoundary(t *testing.T) {
	boundary := generateBoundary()
	if boundary == "" {
		t.Error("generateBoundary returned empty string")
	}
}

func TestParseMIMEMessage(t *testing.T) {
	r := createTestReport()
	job := report.NewReportJob(r)

	p := NewMIMEProcessor(
		WithSubject("Test"),
		WithFrom("sender@example.com"),
	)

	result := p.Process(job)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}

	msg, err := ParseMIMEMessage(result.Report)
	if err != nil {
		t.Fatalf("ParseMIMEMessage failed: %v", err)
	}

	if msg.Header.Get("Subject") != "Test" {
		t.Errorf("Subject = %q, want %q", msg.Header.Get("Subject"), "Test")
	}
	if msg.Header.Get("From") != "sender@example.com" {
		t.Errorf("From = %q, want %q", msg.Header.Get("From"), "sender@example.com")
	}
}
