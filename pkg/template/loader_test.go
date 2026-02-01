package template

import (
	"testing"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
	"github.com/zipreport/zipreport-go/pkg/report"
)

func createTestReportForLoader() *report.ReportFile {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html>Index</html>"), false)
	_ = zfs.Add("templates/partial.html", []byte("<div>Partial</div>"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "T",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}

	return report.NewReportFile(zfs, manifest)
}

func TestNewReportFileLoader(t *testing.T) {
	r := createTestReportForLoader()
	loader := NewReportFileLoader(r)

	if loader == nil {
		t.Fatal("NewReportFileLoader returned nil")
	}
	if loader.report != r {
		t.Error("report not set correctly")
	}
}

func TestReportFileLoader_GetSource_Success(t *testing.T) {
	r := createTestReportForLoader()
	loader := NewReportFileLoader(r)

	source, err := loader.GetSource("index.html")
	if err != nil {
		t.Fatalf("GetSource failed: %v", err)
	}

	if source != "<html>Index</html>" {
		t.Errorf("source = %q, want %q", source, "<html>Index</html>")
	}
}

func TestReportFileLoader_GetSource_NestedPath(t *testing.T) {
	r := createTestReportForLoader()
	loader := NewReportFileLoader(r)

	source, err := loader.GetSource("templates/partial.html")
	if err != nil {
		t.Fatalf("GetSource failed: %v", err)
	}

	if source != "<div>Partial</div>" {
		t.Errorf("source = %q, want %q", source, "<div>Partial</div>")
	}
}

func TestReportFileLoader_GetSource_NotFound(t *testing.T) {
	r := createTestReportForLoader()
	loader := NewReportFileLoader(r)

	_, err := loader.GetSource("nonexistent.html")
	if err == nil {
		t.Error("GetSource should fail for non-existent file")
	}
}

func TestReportFileLoader_Exists(t *testing.T) {
	r := createTestReportForLoader()
	loader := NewReportFileLoader(r)

	if !loader.Exists("index.html") {
		t.Error("Exists should return true for index.html")
	}
	if !loader.Exists("templates/partial.html") {
		t.Error("Exists should return true for templates/partial.html")
	}
	if loader.Exists("nonexistent.html") {
		t.Error("Exists should return false for non-existent file")
	}
}

func TestReportFileLoader_IsCached(t *testing.T) {
	r := createTestReportForLoader()
	loader := NewReportFileLoader(r)

	// IsCached always returns false for this loader
	if loader.IsCached("index.html") {
		t.Error("IsCached should return false")
	}
	if loader.IsCached("nonexistent.html") {
		t.Error("IsCached should return false for non-existent file")
	}
}

func TestReportFileLoader_ListTemplates(t *testing.T) {
	r := createTestReportForLoader()
	loader := NewReportFileLoader(r)

	templates, err := loader.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	// Should return at least index.html which exists in our test report
	if len(templates) == 0 {
		t.Error("ListTemplates returned empty list")
	}

	// Check that at least index.html is in the list
	found := false
	for _, tmpl := range templates {
		if tmpl == "index.html" {
			found = true
			break
		}
	}
	if !found {
		t.Error("index.html not found in templates list")
	}
}
