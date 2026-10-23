package report

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
)

func createTestZipBytes() []byte {
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)

	// Add manifest.json
	w, _ := writer.Create("manifest.json")
	_, _ = w.Write([]byte(`{
		"title": "Test Report",
		"author": "Test Author",
		"version": "1.0.0",
		"description": "A test report",
		"params": ["name"]
	}`))

	// Add index.html
	w, _ = writer.Create("index.html")
	_, _ = w.Write([]byte("<html><body>{{ name }}</body></html>"))

	_ = writer.Close()
	return buf.Bytes()
}

func createTestDir(t *testing.T) string {
	dir := t.TempDir()

	manifest := `{
		"title": "Test Report",
		"author": "Test Author",
		"version": "1.0.0",
		"description": "A test report",
		"params": []
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	index := "<html><body>Test</body></html>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(index), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestLoadBytes(t *testing.T) {
	data := createTestZipBytes()

	rf, err := LoadBytes(data)
	if err != nil {
		t.Fatalf("LoadBytes failed: %v", err)
	}

	if rf.Title() != "Test Report" {
		t.Errorf("Title = %q, want %q", rf.Title(), "Test Report")
	}
	if rf.Author() != "Test Author" {
		t.Errorf("Author = %q, want %q", rf.Author(), "Test Author")
	}
}

func TestLoadBytes_InvalidZip(t *testing.T) {
	_, err := LoadBytes([]byte("not a zip"))
	if err == nil {
		t.Error("LoadBytes should fail for invalid ZIP")
	}
}

func TestLoadBytes_NoManifest(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)
	w, _ := writer.Create("index.html")
	_, _ = w.Write([]byte("<html></html>"))
	_ = writer.Close()

	_, err := LoadBytes(buf.Bytes())
	if err != ErrManifestNotFound {
		t.Errorf("Expected ErrManifestNotFound, got %v", err)
	}
}

func TestLoadBytes_NoIndex(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)
	w, _ := writer.Create("manifest.json")
	_, _ = w.Write([]byte(`{"title":"T","author":"A","version":"1","description":"D"}`))
	_ = writer.Close()

	_, err := LoadBytes(buf.Bytes())
	if err != ErrIndexNotFound {
		t.Errorf("Expected ErrIndexNotFound, got %v", err)
	}
}

func TestLoadFile(t *testing.T) {
	data := createTestZipBytes()
	tmpFile := filepath.Join(t.TempDir(), "test.zpt")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	rf, err := LoadFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if rf.Title() != "Test Report" {
		t.Errorf("Title = %q, want %q", rf.Title(), "Test Report")
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/file.zpt")
	if err == nil {
		t.Error("LoadFile should fail for non-existent file")
	}
}

func TestLoadDir(t *testing.T) {
	dir := createTestDir(t)

	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	if rf.Title() != "Test Report" {
		t.Errorf("Title = %q, want %q", rf.Title(), "Test Report")
	}
}

func TestLoadDir_NotFound(t *testing.T) {
	_, err := LoadDir("/nonexistent/directory")
	if err == nil {
		t.Error("LoadDir should fail for non-existent directory")
	}
}

func TestLoadDir_NoManifest(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644)

	_, err := LoadDir(dir)
	if err != ErrManifestNotFound {
		t.Errorf("Expected ErrManifestNotFound, got %v", err)
	}
}

func TestLoadDir_NoIndex(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"title":"T","author":"A","version":"1","description":"D"}`
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644)

	_, err := LoadDir(dir)
	if err != ErrIndexNotFound {
		t.Errorf("Expected ErrIndexNotFound, got %v", err)
	}
}

func TestLoad_AutoDetect(t *testing.T) {
	// Test with file
	data := createTestZipBytes()
	tmpFile := filepath.Join(t.TempDir(), "test.zpt")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	rf, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load(file) failed: %v", err)
	}
	if rf.Title() != "Test Report" {
		t.Error("Load(file) returned wrong report")
	}

	// Test with directory
	dir := createTestDir(t)
	rf, err = Load(dir)
	if err != nil {
		t.Fatalf("Load(dir) failed: %v", err)
	}
	if rf.Title() != "Test Report" {
		t.Error("Load(dir) returned wrong report")
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Error("Load should fail for non-existent path")
	}
}

func TestLoadZipFs(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)
	_ = zfs.Add("index.html", []byte("<html></html>"), false)

	rf, err := LoadZipFs(zfs)
	if err != nil {
		t.Fatalf("LoadZipFs failed: %v", err)
	}

	if rf.Title() != "T" {
		t.Errorf("Title = %q, want %q", rf.Title(), "T")
	}
}

func TestParseManifest_Valid(t *testing.T) {
	data := []byte(`{
		"title": "Test",
		"author": "Author",
		"version": "1.0",
		"description": "Desc",
		"params": ["a", "b"],
		"useJSEvent": true,
		"report": "output.html"
	}`)

	manifest, err := parseManifest(data)
	if err != nil {
		t.Fatalf("parseManifest failed: %v", err)
	}

	if manifest.Title != "Test" {
		t.Errorf("Title = %q, want %q", manifest.Title, "Test")
	}
	if len(manifest.Params) != 2 {
		t.Errorf("Params length = %d, want 2", len(manifest.Params))
	}
	if !manifest.UseJSEvent {
		t.Error("UseJSEvent should be true")
	}
	if manifest.Report != "output.html" {
		t.Errorf("Report = %q, want %q", manifest.Report, "output.html")
	}
}

func TestParseManifest_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"no title", `{"author":"A","version":"1","description":"D"}`},
		{"no author", `{"title":"T","version":"1","description":"D"}`},
		{"no version", `{"title":"T","author":"A","description":"D"}`},
		{"no description", `{"title":"T","author":"A","version":"1"}`},
	}

	for _, tt := range tests {
		_, err := parseManifest([]byte(tt.json))
		if err == nil {
			t.Errorf("%s: parseManifest should fail", tt.name)
		}
	}
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	_, err := parseManifest([]byte("not json"))
	if err == nil {
		t.Error("parseManifest should fail for invalid JSON")
	}
}

func TestParseManifest_NilParams(t *testing.T) {
	data := []byte(`{
		"title": "Test",
		"author": "Author",
		"version": "1.0",
		"description": "Desc"
	}`)

	manifest, err := parseManifest(data)
	if err != nil {
		t.Fatalf("parseManifest failed: %v", err)
	}

	if manifest.Params == nil {
		t.Error("Params should not be nil")
	}
	if len(manifest.Params) != 0 {
		t.Errorf("Params length = %d, want 0", len(manifest.Params))
	}
}

func TestValidateManifest(t *testing.T) {
	valid := []byte(`{"title":"T","author":"A","version":"1","description":"D"}`)
	if err := ValidateManifest(valid); err != nil {
		t.Errorf("ValidateManifest should pass for valid manifest: %v", err)
	}

	invalid := []byte(`{"title":"T"}`)
	if err := ValidateManifest(invalid); err == nil {
		t.Error("ValidateManifest should fail for invalid manifest")
	}
}

func TestParseManifest_ReportMustBeLocal(t *testing.T) {
	build := func(report *string) []byte {
		m := map[string]any{"title": "T", "author": "A", "version": "1", "description": "D"}
		if report != nil {
			m["report"] = *report
		}
		b, _ := json.Marshal(m)
		return b
	}
	for _, r := range []string{"../x.html", "/abs.html", "a/../../x.html", `..\x.html`, "..", ".", "a/.."} {
		r := r
		if _, err := parseManifest(build(&r)); !errors.Is(err, ErrInvalidManifest) {
			t.Errorf("report %q: err = %v, want ErrInvalidManifest", r, err)
		}
	}
	for _, r := range []string{"out.html", "sub/out.html"} {
		r := r
		if _, err := parseManifest(build(&r)); err != nil {
			t.Errorf("report %q: unexpected error %v", r, err)
		}
	}
	if _, err := parseManifest(build(nil)); err != nil {
		t.Errorf("absent report: unexpected error %v", err)
	}
}
