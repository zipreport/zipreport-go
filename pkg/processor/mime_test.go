package processor

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
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

// mimeParts parses a generated message. It returns the decoded HTML part under key "html"
// and each other part under its Content-ID header value exactly as written (e.g. "<resource0@zipreport>").
func mimeParts(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	_, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType: %v", err)
	}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	parts := make(map[string][]byte)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		raw, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(raw), "\r\n", ""))
		if err != nil {
			t.Fatalf("base64 decode: %v", err)
		}
		key := part.Header.Get("Content-ID")
		if key == "" {
			key = "html"
		}
		parts[key] = decoded
	}
	return parts
}

// newTestReport builds a ZipFs-backed report from files and a manifest.
func newTestReport(files map[string]string, m *report.Manifest) *report.ReportFile {
	zfs := fileutils.NewZipFs()
	for name, content := range files {
		_ = zfs.Add(name, []byte(content), false)
	}
	return report.NewReportFile(zfs, m)
}

func processMIME(r *report.ReportFile, opts ...MIMEOption) *report.JobResult {
	return NewMIMEProcessor(opts...).Process(report.NewReportJob(r))
}

func parsedHeader(t *testing.T, data []byte, key string) string {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	return msg.Header.Get(key)
}

func TestMIMEProcessor_Base64LinesWrapped(t *testing.T) {
	r := createTestReport()
	big := bytes.Repeat([]byte{0xAB}, 3000)
	if err := r.Add("image.png", big, true); err != nil {
		t.Fatal(err)
	}
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	for _, line := range strings.Split(string(result.Report), "\r\n") {
		if len(line) > 76 {
			t.Fatalf("line longer than 76 characters: %d", len(line))
		}
	}
	if got := mimeParts(t, result.Report)["<resource0@zipreport>"]; !bytes.Equal(got, big) {
		t.Errorf("image part does not round-trip: got %d bytes", len(got))
	}
}

func TestMIMEProcessor_NonASCIISubjectEncoded(t *testing.T) {
	result := processMIME(createTestReport(), WithSubject("Relatório mensal"))
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	if !strings.Contains(string(result.Report), "Subject: =?utf-8?q?") {
		t.Error("subject should be RFC 2047 encoded")
	}
	got, err := new(mime.WordDecoder).DecodeHeader(parsedHeader(t, result.Report, "Subject"))
	if err != nil || got != "Relatório mensal" {
		t.Errorf("decoded subject = %q, %v", got, err)
	}
}

func TestMIMEProcessor_DefaultSubjectNonASCIITitle(t *testing.T) {
	r := newTestReport(map[string]string{"report.html": "<p>x</p>"}, &report.Manifest{Title: "Relatório"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	if !strings.Contains(string(result.Report), "Subject: =?utf-8?q?") {
		t.Error("default subject should be RFC 2047 encoded")
	}
	got, err := new(mime.WordDecoder).DecodeHeader(parsedHeader(t, result.Report, "Subject"))
	if err != nil || got != "Relatório" {
		t.Errorf("decoded subject = %q, %v", got, err)
	}
}

func TestMIMEProcessor_LongSubjectFolded(t *testing.T) {
	title := strings.Repeat("ó", 600)
	r := newTestReport(map[string]string{"report.html": "<p>x</p>"}, &report.Manifest{Title: title})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	if !strings.Contains(string(result.Report), "Subject: =?utf-8?q?") {
		t.Error("subject should be RFC 2047 encoded")
	}
	for _, line := range strings.Split(string(result.Report), "\r\n") {
		if len(line) > 998 {
			t.Fatalf("line longer than 998 characters: %d", len(line))
		}
	}
	got, err := new(mime.WordDecoder).DecodeHeader(parsedHeader(t, result.Report, "Subject"))
	if err != nil || got != title {
		t.Errorf("decoded subject mismatch: %v", err)
	}
}

func TestMIMEProcessor_RejectsControlCharsInCallerHeaders(t *testing.T) {
	tests := []struct {
		name string
		opt  MIMEOption
	}{
		{"subject", WithSubject("Hi\r\nBcc: x@evil.test")},
		{"from", WithFrom("a@b.c\nBcc: x@evil.test")},
		{"to", WithTo("ok@b.c", "x@b.c\r\nBcc: y@evil.test")},
		{"nul", WithFrom("a@b.c\x00")},
		{"del", WithSubject("a\x7fb")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processMIME(createTestReport(), tt.opt)
			if result.Success {
				t.Fatal("expected failure")
			}
			if !strings.Contains(result.Error, "control character") {
				t.Errorf("unexpected error: %s", result.Error)
			}
		})
	}
	t.Run("tab allowed", func(t *testing.T) {
		result := processMIME(createTestReport(), WithSubject("a\tb"))
		if !result.Success {
			t.Errorf("tab should be allowed: %s", result.Error)
		}
	})
}

func TestMIMEProcessor_TitleLineBreaksCollapsed(t *testing.T) {
	r := newTestReport(map[string]string{"report.html": "<p>x</p>"}, &report.Manifest{Title: "Monthly\nReport"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	if !strings.Contains(string(result.Report), "Subject: Monthly Report") {
		t.Error("title line break should be collapsed to a space")
	}
}

func TestMIMEProcessor_TrailingNewlineInAddressTrimmed(t *testing.T) {
	result := processMIME(createTestReport(), WithTo("a@b.c\n"))
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	if !strings.Contains(string(result.Report), "To: a@b.c\r\n") {
		t.Error("trailing newline should be trimmed from address")
	}
}

func TestMIMEProcessor_AnchorsAndSchemesUntouched(t *testing.T) {
	r := newTestReport(map[string]string{
		"report.html": `<a href="mailto:a@b.c">m</a><a href="tel:+351">t</a><a href="page2.html">p</a><a href="data.json">d</a><img src="image.png">`,
		"page2.html":  "<p>2</p>",
		"data.json":   "{}",
		"image.png":   "PNG",
	}, &report.Manifest{Title: "T"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	parts := mimeParts(t, result.Report)
	html := string(parts["html"])
	for _, want := range []string{`href="mailto:a@b.c"`, `href="tel:+351"`, `href="page2.html"`, `href="data.json"`, `src="cid:resource0@zipreport"`} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML should contain %s, got %s", want, html)
		}
	}
	if len(parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(parts))
	}
}

func TestMIMEProcessor_SameFileEmbeddedOnce(t *testing.T) {
	r := newTestReport(map[string]string{
		"report.html": `<img src="image.png"><img src="./image.png"><img src="/image.png">`,
		"image.png":   "PNG",
	}, &report.Manifest{Title: "T"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	parts := mimeParts(t, result.Report)
	if len(parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(parts))
	}
	if n := strings.Count(string(parts["html"]), `src="cid:resource0@zipreport"`); n != 3 {
		t.Errorf("expected 3 cid references, got %d in %s", n, parts["html"])
	}
}

func TestMIMEProcessor_ResolvesRelativeToReportFile(t *testing.T) {
	r := newTestReport(map[string]string{
		"sub/out.html":  `<img src="img/a.png">`,
		"sub/img/a.png": "RIGHT",
		"img/a.png":     "WRONG",
	}, &report.Manifest{Title: "T", Report: "sub/out.html"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	if got := string(mimeParts(t, result.Report)["<resource0@zipreport>"]); got != "RIGHT" {
		t.Errorf("image content = %q, want RIGHT", got)
	}
}

func TestMIMEProcessor_UsesManifestReportFilename(t *testing.T) {
	r := newTestReport(map[string]string{
		"out.html":    "<p>NEW</p>",
		"report.html": "<p>STALE</p>",
	}, &report.Manifest{Title: "T", Report: "out.html"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	html := string(mimeParts(t, result.Report)["html"])
	if !strings.Contains(html, "NEW") || strings.Contains(html, "STALE") {
		t.Errorf("unexpected HTML: %s", html)
	}
}

func TestExtractResources_OnlyEmbeddable(t *testing.T) {
	html := `<a href="page.html"><a href="mailto:x@y"><link rel="stylesheet" href="s.css"><link rel="canonical" href="report.html"><img src="x.png"><script src="javascript:void(0)">`
	got := extractResources(html)
	want := map[string]bool{"x.png": true, "s.css": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected resource %q", g)
		}
	}
}

func TestMIMEProcessor_AnchorToEmbeddedImageUntouched(t *testing.T) {
	r := newTestReport(map[string]string{
		"report.html": `<a href="image.png"><img src="image.png"></a><p title='image.png'>image.png</p>`,
		"image.png":   "PNG",
	}, &report.Manifest{Title: "T"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	html := string(mimeParts(t, result.Report)["html"])
	for _, want := range []string{`<a href="image.png">`, `src="cid:resource0@zipreport"`, `title='image.png'`} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML should contain %s, got %s", want, html)
		}
	}
}

func TestMIMEProcessor_OverlongASCIISubjectRejected(t *testing.T) {
	result := processMIME(createTestReport(), WithSubject(strings.Repeat("a", 1200)))
	if result.Success || !strings.Contains(result.Error, "subject too long") {
		t.Errorf("expected subject too long error, got success=%v err=%q", result.Success, result.Error)
	}
	if result := processMIME(createTestReport(), WithSubject(strings.Repeat("a", 900))); !result.Success {
		t.Errorf("900 char subject should succeed: %s", result.Error)
	}
}

func TestMIMEProcessor_HiddenFileNotEmbedded(t *testing.T) {
	r := newTestReport(map[string]string{
		"report.html": `<img src=".env"><img src="sub/.env"><img src="image.png">`,
		".env":        "SECRET",
		"sub/.env":    "SECRET2",
		"image.png":   "PNG",
	}, &report.Manifest{Title: "T"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	parts := mimeParts(t, result.Report)
	if len(parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(parts))
	}
	for k, v := range parts {
		if strings.Contains(string(v), "SECRET") || strings.Contains(string(v), "SECRET2") {
			t.Errorf("part %s contains hidden file content", k)
		}
	}
}

func TestMIMEProcessor_SchemeRefsNotEmbedded(t *testing.T) {
	html := `<img src="data:image/png;base64,AAAA"><img src="cid:x@y"><link rel="stylesheet" href="https://cdn.example/s.css"><img src="image.png">`
	r := newTestReport(map[string]string{
		"report.html": html,
		"image.png":   "PNG",
		"cid:x@y":     "TRAP",
	}, &report.Manifest{Title: "T", Report: "report.html"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("failed: %s", result.Error)
	}
	parts := mimeParts(t, result.Report)
	if len(parts) != 2 {
		t.Errorf("parts = %d, want 2", len(parts))
	}
	body := string(parts["html"])
	for _, want := range []string{`src="data:image/png;base64,AAAA"`, `src="cid:x@y"`, `href="https://cdn.example/s.css"`} {
		if !strings.Contains(body, want) {
			t.Errorf("html missing %s:\n%s", want, body)
		}
	}
}

func TestMIMEProcessor_RootRelativeRefFromSubfolder(t *testing.T) {
	r := newTestReport(map[string]string{
		"sub/out.html":  `<img src="/img/a.png">`,
		"img/a.png":     "RIGHT",
		"sub/img/a.png": "WRONG",
	}, &report.Manifest{Title: "T", Report: "sub/out.html"})
	result := processMIME(r)
	if !result.Success {
		t.Fatalf("failed: %s", result.Error)
	}
	parts := mimeParts(t, result.Report)
	found := false
	for k, v := range parts {
		if k == "html" {
			continue
		}
		found = true
		if string(v) != "RIGHT" {
			t.Errorf("embedded part %q = %q, want RIGHT", k, v)
		}
	}
	if !found {
		t.Error("no embedded part")
	}
}
