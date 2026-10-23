package report

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

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

func savedZip(t *testing.T, rf *ReportFile) *fileutils.ZipFs {
	t.Helper()
	data, err := rf.Save()
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	zfs, err := fileutils.NewZipFsFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	return zfs
}

func TestReportFile_Save_DiskFsIncludesAssets(t *testing.T) {
	dir := createValidReportDir(t)
	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	zfs := savedZip(t, rf)
	got, err := zfs.GetBytes("assets/style.css")
	if err != nil || string(got) != "body { color: red; }" {
		t.Errorf("assets/style.css = %q, %v", got, err)
	}
	if !zfs.Exists("index.html") || !zfs.Exists("manifest.json") {
		t.Error("index.html or manifest.json missing")
	}
}

func TestReportFile_Clone_DoesNotWriteToDisk(t *testing.T) {
	dir := createValidReportDir(t)
	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := rf.Clone()
	if err := c.Add("report.html", []byte("x"), true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "report.html")); !os.IsNotExist(err) {
		t.Errorf("report.html written to disk (err=%v)", err)
	}
	if !c.Exists("report.html") {
		t.Error("clone does not see its own write")
	}
}

func TestReportFile_Clone_SaveIncludesOverlayWrites(t *testing.T) {
	dir := createValidReportDir(t)
	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := rf.Clone()
	_ = c.Add("report.html", []byte("R"), true)
	_ = c.Add("_gen_1.png", []byte("P"), true)
	zfs := savedZip(t, c)
	for name, want := range map[string]string{"report.html": "R", "_gen_1.png": "P", "assets/style.css": "body { color: red; }"} {
		got, err := zfs.GetBytes(name)
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestReportFile_Clone_ZipFsIndependent(t *testing.T) {
	rf := createTestReportFile()
	c := rf.Clone()
	if err := c.Add("report.html", []byte("x"), true); err != nil {
		t.Fatal(err)
	}
	if rf.Exists("report.html") {
		t.Error("clone write reached the original")
	}
	if c.Title() != rf.Title() {
		t.Errorf("title = %q, want %q", c.Title(), rf.Title())
	}
}

func TestReportFile_Save_DiskFsWithoutManifestOnDisk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p></p>"), 0644); err != nil {
		t.Fatal(err)
	}
	dfs, err := fileutils.NewDiskFs(dir)
	if err != nil {
		t.Fatal(err)
	}
	rf := NewReportFile(dfs, &Manifest{Title: "T", Author: "A", Version: "1", Description: "D"})
	zfs := savedZip(t, rf)
	got, err := zfs.GetBytes("manifest.json")
	if err != nil || !strings.Contains(string(got), `"title": "T"`) {
		t.Errorf("manifest.json = %q, %v", got, err)
	}
}

// mapFsAdapter adapts an fs.FS (as an io/fs adapter for go:embed would) to FsInterface;
// List errors for "" and works for ".".
type mapFsAdapter struct{ m fstest.MapFS }

func (a mapFsAdapter) Get(name string) (io.ReadCloser, error) {
	b, err := a.GetBytes(name)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (a mapFsAdapter) GetBytes(name string) ([]byte, error) { return fs.ReadFile(a.m, name) }
func (a mapFsAdapter) Add(string, []byte, bool) error       { return errors.New("read-only") }
func (a mapFsAdapter) Mkdir(string) error                   { return errors.New("read-only") }
func (a mapFsAdapter) Remove(string) error                  { return errors.New("read-only") }
func (a mapFsAdapter) Exists(name string) bool              { _, err := fs.Stat(a.m, name); return err == nil }
func (a mapFsAdapter) IsDir(name string) bool {
	info, err := fs.Stat(a.m, name)
	return err == nil && info.IsDir()
}
func (a mapFsAdapter) List(p string) ([]string, error) {
	entries, err := fs.ReadDir(a.m, p)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
func (a mapFsAdapter) filtered(p string, dirs bool) ([]string, error) {
	entries, err := fs.ReadDir(a.m, p)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() == dirs {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
func (a mapFsAdapter) ListFiles(p string) ([]string, error) { return a.filtered(p, false) }
func (a mapFsAdapter) ListDirs(p string) ([]string, error)  { return a.filtered(p, true) }

type noListAdapter struct{ mapFsAdapter }

func (noListAdapter) List(string) ([]string, error) { return nil, errors.New("no listing") }

type emptyListAdapter struct{ mapFsAdapter }

func (emptyListAdapter) List(string) ([]string, error) { return nil, nil }

func adapterFiles() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<p></p>")},
		"manifest.json": {Data: []byte(`{"title":"T"}`)},
		"img/a.png":     {Data: []byte("png")},
	}
}

func TestReportFile_Save_IOFSAdapter(t *testing.T) {
	rf := NewReportFile(mapFsAdapter{adapterFiles()}, &Manifest{Title: "T"})
	zfs := savedZip(t, rf)
	for _, n := range []string{"index.html", "manifest.json", "img/a.png"} {
		if !zfs.Exists(n) {
			t.Errorf("%s missing", n)
		}
	}
}

func TestReportFile_Save_CustomFsWithoutListingFallsBack(t *testing.T) {
	rf := NewReportFile(noListAdapter{mapFsAdapter{adapterFiles()}}, &Manifest{Title: "T"})
	zfs := savedZip(t, rf)
	for _, n := range []string{"index.html", "manifest.json"} {
		if !zfs.Exists(n) {
			t.Errorf("%s missing", n)
		}
	}
}

func TestReportFile_Save_CustomFsListingIncompleteAddsKnownFiles(t *testing.T) {
	rf := NewReportFile(emptyListAdapter{mapFsAdapter{adapterFiles()}}, &Manifest{Title: "T"})
	zfs := savedZip(t, rf)
	for _, n := range []string{"index.html", "manifest.json"} {
		if !zfs.Exists(n) {
			t.Errorf("%s missing", n)
		}
	}
}

func TestReportFile_Save_UnreadableIndexFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	dir := createValidReportDir(t)
	idx := filepath.Join(dir, "index.html")
	if err := os.Chmod(idx, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(idx, 0644) })
	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rf.Save()
	if err == nil || !strings.Contains(err.Error(), "index.html") {
		t.Errorf("Save err = %v, want error mentioning index.html", err)
	}
}

func TestReportFile_Save_UnreadableUnusedFileSkipped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	dir := createValidReportDir(t)
	notes := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notes, []byte("n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(notes, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(notes, 0644) })
	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	zfs := savedZip(t, rf)
	if !zfs.Exists("assets/style.css") {
		t.Error("assets/style.css missing")
	}
}

func TestReportJob_WithReport(t *testing.T) {
	job := NewReportJob(createTestReportFile())
	job.SetPageSize(PageLetter).SetLandscape(true).SetCustomMargins(&Margins{Top: 5})

	other := createTestReportFile()
	c := job.WithReport(other)

	if c.GetReport() != other {
		t.Error("copy does not use the new report")
	}
	if job.GetReport() == other {
		t.Error("original job was modified")
	}
	if !reflect.DeepEqual(c.GetOptions(), job.GetOptions()) {
		t.Errorf("options differ: %v vs %v", c.GetOptions(), job.GetOptions())
	}
}

func TestReportJob_WithReport_CopiesMargins(t *testing.T) {
	job := NewReportJob(createTestReportFile())
	job.SetCustomMargins(&Margins{Top: 5})
	c := job.WithReport(createTestReportFile())

	c.customMargins.Top = 9
	if job.customMargins.Top != 5 {
		t.Errorf("original margins aliased: Top = %v", job.customMargins.Top)
	}
}

func TestBuildFromReportFile_UnreadableIndexFails(t *testing.T) {
	skipOnWindows(t)
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	dir := createValidReportDir(t)
	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(dir, "index.html")
	if err := os.Chmod(idx, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(idx, 0644) })
	zfs, err := BuildFromReportFile(rf)
	if err == nil || !strings.Contains(err.Error(), "index.html") {
		t.Fatalf("err = %v, want error mentioning index.html", err)
	}
	if zfs != nil {
		t.Error("zfs should be nil")
	}
}
