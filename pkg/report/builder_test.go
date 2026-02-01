package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
)

func createValidReportDir(t *testing.T) string {
	dir := t.TempDir()

	manifest := `{
		"title": "Test Report",
		"author": "Test Author",
		"version": "1.0.0",
		"description": "A test report"
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	index := "<html><body>{{ name }}</body></html>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(index), 0644); err != nil {
		t.Fatal(err)
	}

	// Add a nested file
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "style.css"), []byte("body { color: red; }"), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestBuildResult(t *testing.T) {
	// Test successful result
	result := &BuildResult{success: true, errors: nil}
	if !result.Success() {
		t.Error("Success() should return true")
	}
	if len(result.Errors()) != 0 {
		t.Error("Errors() should be empty")
	}
	if result.Error() != "" {
		t.Error("Error() should be empty string")
	}

	// Test failed result
	result2 := &BuildResult{
		success: false,
		errors:  []string{"error1", "error2"},
	}
	if result2.Success() {
		t.Error("Success() should return false")
	}
	if len(result2.Errors()) != 2 {
		t.Errorf("Errors() length = %d, want 2", len(result2.Errors()))
	}
	if result2.Error() != "error1; error2" {
		t.Errorf("Error() = %q, want %q", result2.Error(), "error1; error2")
	}
}

func TestDefaultBuildOptions(t *testing.T) {
	opts := DefaultBuildOptions()

	if opts.FollowSymlinks {
		t.Error("FollowSymlinks should be false by default")
	}
	if !opts.ExcludeHidden {
		t.Error("ExcludeHidden should be true by default")
	}
}

func TestBuild_Success(t *testing.T) {
	srcDir := createValidReportDir(t)
	destFile := filepath.Join(t.TempDir(), "output.zpt")

	result := Build(srcDir, destFile, nil)

	if !result.Success() {
		t.Fatalf("Build failed: %s", result.Error())
	}

	// Verify output file exists
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("Output file not created")
	}

	// Verify it's a valid ZIP
	zfs, err := fileutils.NewZipFsFromFile(destFile)
	if err != nil {
		t.Fatalf("Output is not valid ZIP: %v", err)
	}

	// Verify contents
	if !zfs.Exists("manifest.json") {
		t.Error("manifest.json not in output")
	}
	if !zfs.Exists("index.html") {
		t.Error("index.html not in output")
	}
	if !zfs.Exists("assets/style.css") {
		t.Error("assets/style.css not in output")
	}
}

func TestBuild_SourceNotFound(t *testing.T) {
	result := Build("/nonexistent/directory", "/tmp/out.zpt", nil)

	if result.Success() {
		t.Error("Build should fail for non-existent source")
	}
}

func TestBuild_SourceNotDirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	_ = os.WriteFile(tmpFile, []byte("test"), 0644)

	result := Build(tmpFile, "/tmp/out.zpt", nil)

	if result.Success() {
		t.Error("Build should fail when source is not a directory")
	}
}

func TestBuild_WithOptions(t *testing.T) {
	srcDir := createValidReportDir(t)
	destFile := filepath.Join(t.TempDir(), "output.zpt")

	opts := &BuildOptions{
		FollowSymlinks: true,
		ExcludeHidden:  false,
	}

	result := Build(srcDir, destFile, opts)

	if !result.Success() {
		t.Fatalf("Build failed: %s", result.Error())
	}
}

func TestBuildZipFs_Success(t *testing.T) {
	srcDir := createValidReportDir(t)

	zfs, result := BuildZipFs(srcDir, nil)

	if !result.Success() {
		t.Fatalf("BuildZipFs failed: %s", result.Error())
	}
	if zfs == nil {
		t.Fatal("ZipFs is nil")
	}

	if !zfs.Exists("manifest.json") {
		t.Error("manifest.json not in ZipFs")
	}
	if !zfs.Exists("index.html") {
		t.Error("index.html not in ZipFs")
	}
}

func TestBuildZipFs_NoManifest(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644)

	_, result := BuildZipFs(dir, nil)

	if result.Success() {
		t.Error("BuildZipFs should fail without manifest.json")
	}
}

func TestBuildZipFs_NoIndex(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"title":"T","author":"A","version":"1","description":"D"}`
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644)

	_, result := BuildZipFs(dir, nil)

	if result.Success() {
		t.Error("BuildZipFs should fail without index.html")
	}
}

func TestBuildZipFs_InvalidManifest(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("not json"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644)

	_, result := BuildZipFs(dir, nil)

	if result.Success() {
		t.Error("BuildZipFs should fail with invalid manifest")
	}
}

func TestBuildZipFs_ExcludeHidden(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"title":"T","author":"A","version":"1","description":"D"}`
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644)
	_ = os.WriteFile(filepath.Join(dir, ".hidden"), []byte("hidden"), 0644)

	opts := &BuildOptions{ExcludeHidden: true}
	zfs, result := BuildZipFs(dir, opts)

	if !result.Success() {
		t.Fatalf("BuildZipFs failed: %s", result.Error())
	}

	if zfs.Exists(".hidden") {
		t.Error(".hidden should be excluded")
	}
}

func TestBuildZipFs_IncludeHidden(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"title":"T","author":"A","version":"1","description":"D"}`
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644)
	_ = os.WriteFile(filepath.Join(dir, ".hidden"), []byte("hidden"), 0644)

	opts := &BuildOptions{ExcludeHidden: false}
	zfs, result := BuildZipFs(dir, opts)

	if !result.Success() {
		t.Fatalf("BuildZipFs failed: %s", result.Error())
	}

	if !zfs.Exists(".hidden") {
		t.Error(".hidden should be included when ExcludeHidden is false")
	}
}

func TestBuildFromReportFile_ZipFs(t *testing.T) {
	// Create a report from ZipFs
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)
	_ = zfs.Add("index.html", []byte("<html></html>"), false)

	manifest := &Manifest{Title: "T", Author: "A", Version: "1", Description: "D"}
	rf := NewReportFile(zfs, manifest)

	newZfs, err := BuildFromReportFile(rf)
	if err != nil {
		t.Fatalf("BuildFromReportFile failed: %v", err)
	}
	if newZfs == nil {
		t.Fatal("newZfs is nil")
	}

	if !newZfs.Exists("index.html") {
		t.Error("index.html not in new ZipFs")
	}
}

func TestBuildFromReportFile_DiskFs(t *testing.T) {
	// Create a valid report directory
	dir := createValidReportDir(t)

	rf, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	newZfs, err := BuildFromReportFile(rf)
	if err != nil {
		t.Fatalf("BuildFromReportFile failed: %v", err)
	}
	if newZfs == nil {
		t.Fatal("newZfs is nil")
	}

	if !newZfs.Exists("index.html") {
		t.Error("index.html not in new ZipFs")
	}
}
