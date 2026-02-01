package report

import (
	"testing"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
)

func createTestReportFile() *ReportFile {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html>test</html>"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"Test","author":"Tester","version":"1.0","description":"Test report","params":["name"]}`), false)

	manifest := &Manifest{
		Title:       "Test Report",
		Author:      "Test Author",
		Version:     "1.0.0",
		Description: "A test report",
		Params:      []string{"name", "value"},
		UseJSEvent:  true,
		Report:      "output.html",
	}

	return NewReportFile(zfs, manifest)
}

func TestNewReportFile(t *testing.T) {
	zfs := fileutils.NewZipFs()
	manifest := &Manifest{Title: "Test"}

	rf := NewReportFile(zfs, manifest)
	if rf == nil {
		t.Fatal("NewReportFile returned nil")
	}
	if rf.GetManifest() != manifest {
		t.Error("GetManifest returned wrong manifest")
	}
	if rf.GetFs() != zfs {
		t.Error("GetFs returned wrong file system")
	}
}

func TestReportFile_Getters(t *testing.T) {
	rf := createTestReportFile()

	if rf.Title() != "Test Report" {
		t.Errorf("Title() = %q, want %q", rf.Title(), "Test Report")
	}
	if rf.Author() != "Test Author" {
		t.Errorf("Author() = %q, want %q", rf.Author(), "Test Author")
	}
	if rf.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", rf.Version(), "1.0.0")
	}
	if rf.Description() != "A test report" {
		t.Errorf("Description() = %q, want %q", rf.Description(), "A test report")
	}
	if len(rf.Params()) != 2 {
		t.Errorf("Params() length = %d, want 2", len(rf.Params()))
	}
	if !rf.UseJSEvent() {
		t.Error("UseJSEvent() = false, want true")
	}
	if rf.ReportFilename() != "output.html" {
		t.Errorf("ReportFilename() = %q, want %q", rf.ReportFilename(), "output.html")
	}
}

func TestReportFile_ReportFilenameDefault(t *testing.T) {
	zfs := fileutils.NewZipFs()
	manifest := &Manifest{Title: "Test"}
	rf := NewReportFile(zfs, manifest)

	if rf.ReportFilename() != FileReport {
		t.Errorf("ReportFilename() = %q, want %q", rf.ReportFilename(), FileReport)
	}
}

func TestReportFile_GetParam(t *testing.T) {
	rf := createTestReportFile()

	tests := []struct {
		name       string
		defaultVal interface{}
		expected   interface{}
	}{
		{"title", "default", "Test Report"},
		{"author", "default", "Test Author"},
		{"version", "default", "1.0.0"},
		{"description", "default", "A test report"},
		{"useJSEvent", false, true},
		{"report", "default", "output.html"},
		{"unknown", "default", "default"},
	}

	for _, tt := range tests {
		got := rf.GetParam(tt.name, tt.defaultVal)
		if got != tt.expected {
			t.Errorf("GetParam(%q, %v) = %v, want %v", tt.name, tt.defaultVal, got, tt.expected)
		}
	}
}

func TestReportFile_GetParamParams(t *testing.T) {
	rf := createTestReportFile()

	params := rf.GetParam("params", nil)
	if params == nil {
		t.Fatal("GetParam(params) returned nil")
	}

	paramsList, ok := params.([]string)
	if !ok {
		t.Fatal("GetParam(params) is not []string")
	}

	if len(paramsList) != 2 {
		t.Errorf("GetParam(params) length = %d, want 2", len(paramsList))
	}
}

func TestReportFile_Exists(t *testing.T) {
	rf := createTestReportFile()

	if !rf.Exists("index.html") {
		t.Error("Exists(index.html) = false, want true")
	}
	if rf.Exists("nonexistent.html") {
		t.Error("Exists(nonexistent.html) = true, want false")
	}
}

func TestReportFile_Get(t *testing.T) {
	rf := createTestReportFile()

	rc, err := rf.Get("index.html")
	if err != nil {
		t.Fatalf("Get(index.html) error: %v", err)
	}
	defer rc.Close()

	// Test non-existent file
	_, err = rf.Get("nonexistent.html")
	if err == nil {
		t.Error("Get(nonexistent.html) should return error")
	}
}

func TestReportFile_GetBytes(t *testing.T) {
	rf := createTestReportFile()

	content, err := rf.GetBytes("index.html")
	if err != nil {
		t.Fatalf("GetBytes(index.html) error: %v", err)
	}
	if string(content) != "<html>test</html>" {
		t.Errorf("GetBytes(index.html) = %q, want %q", string(content), "<html>test</html>")
	}
}

func TestReportFile_Add(t *testing.T) {
	rf := createTestReportFile()

	err := rf.Add("new.txt", []byte("new content"), false)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	if !rf.Exists("new.txt") {
		t.Error("new.txt should exist after Add")
	}

	content, _ := rf.GetBytes("new.txt")
	if string(content) != "new content" {
		t.Errorf("Content = %q, want %q", string(content), "new content")
	}
}

func TestReportFile_Save(t *testing.T) {
	rf := createTestReportFile()

	data, err := rf.Save()
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	if len(data) == 0 {
		t.Error("Save returned empty data")
	}

	// Verify it's valid ZIP by loading it
	_, err = fileutils.NewZipFsFromBytes(data)
	if err != nil {
		t.Fatalf("Saved data is not valid ZIP: %v", err)
	}
}

func TestNewReportJob(t *testing.T) {
	rf := createTestReportFile()
	job := NewReportJob(rf)

	if job == nil {
		t.Fatal("NewReportJob returned nil")
	}
	if job.GetReport() != rf {
		t.Error("GetReport returned wrong report")
	}

	// Should inherit useJSEvent from manifest
	options := job.GetOptions()
	if options["js_event"] != true {
		t.Error("Should inherit useJSEvent from manifest")
	}
}

func TestReportJob_Setters(t *testing.T) {
	rf := createTestReportFile()
	job := NewReportJob(rf)

	// Test chaining
	job.SetPageSize(PageA4).
		SetMargins(MarginNone).
		SetLandscape(true).
		SetSettlingTime(500).
		SetJSTimeout(30).
		SetJobTimeout(120).
		UseJSEvent(false)

	if job.PageSize() != PageA4 {
		t.Error("PageSize not set correctly")
	}
	if job.Landscape() != true {
		t.Error("Landscape not set correctly")
	}

	options := job.GetOptions()
	if options["page_size"] != string(PageA4) {
		t.Errorf("page_size = %v, want %v", options["page_size"], PageA4)
	}
	if options["margins"] != string(MarginNone) {
		t.Errorf("margins = %v, want %v", options["margins"], MarginNone)
	}
	if options["landscape"] != true {
		t.Error("landscape should be true")
	}
	if options["settling_time"] != 500 {
		t.Errorf("settling_time = %v, want 500", options["settling_time"])
	}
	if options["timeout_js"] != 30 {
		t.Errorf("timeout_js = %v, want 30", options["timeout_js"])
	}
	if options["timeout_job"] != 120 {
		t.Errorf("timeout_job = %v, want 120", options["timeout_job"])
	}
	if options["js_event"] != false {
		t.Error("js_event should be false")
	}
}

func TestReportJob_CustomMargins(t *testing.T) {
	rf := createTestReportFile()
	job := NewReportJob(rf)

	margins := &Margins{
		Top:    10.5,
		Right:  20.0,
		Bottom: 15.5,
		Left:   25.0,
	}
	job.SetCustomMargins(margins)

	options := job.GetOptions()
	if options["margins"] != "custom" {
		t.Errorf("margins = %v, want custom", options["margins"])
	}
	if options["margin_top"] != "10.5" {
		t.Errorf("margin_top = %v, want 10.5", options["margin_top"])
	}
	if options["margin_right"] != "20.0" {
		t.Errorf("margin_right = %v, want 20.0", options["margin_right"])
	}
	if options["margin_bottom"] != "15.5" {
		t.Errorf("margin_bottom = %v, want 15.5", options["margin_bottom"])
	}
	if options["margin_left"] != "25.0" {
		t.Errorf("margin_left = %v, want 25.0", options["margin_left"])
	}
}

func TestJobResult(t *testing.T) {
	// Test successful result
	result := NewJobResult([]byte("PDF content"))
	if !result.Success {
		t.Error("NewJobResult should be successful")
	}
	if string(result.Report) != "PDF content" {
		t.Error("Report content mismatch")
	}

	// Test error result with string
	errResult := NewJobResultError("something failed")
	if errResult.Success {
		t.Error("NewJobResultError should not be successful")
	}
	if errResult.Error != "something failed" {
		t.Errorf("Error = %q, want %q", errResult.Error, "something failed")
	}

	// Test error result from error
	errResult2 := NewJobResultFromError(fileutils.ErrFileNotFound)
	if errResult2.Success {
		t.Error("NewJobResultFromError should not be successful")
	}
	if errResult2.Error != "file not found" {
		t.Errorf("Error = %q, want %q", errResult2.Error, "file not found")
	}
}
