package fileutils

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestNewDiskFs(t *testing.T) {
	dir := t.TempDir()

	dfs, err := NewDiskFs(dir)
	if err != nil {
		t.Fatalf("NewDiskFs failed: %v", err)
	}

	if dfs.BasePath() != dir {
		t.Errorf("BasePath = %q, want %q", dfs.BasePath(), dir)
	}
}

func TestNewDiskFs_NotFound(t *testing.T) {
	_, err := NewDiskFs("/nonexistent/directory")
	if err == nil {
		t.Error("NewDiskFs should fail for non-existent directory")
	}
}

func TestNewDiskFs_NotADirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	_ = os.WriteFile(tmpFile, []byte("test"), 0644)

	_, err := NewDiskFs(tmpFile)
	if err != ErrNotADirectory {
		t.Errorf("Expected ErrNotADirectory, got %v", err)
	}
}

func TestDiskFs_Add(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	err := dfs.Add("test.txt", []byte("hello"), false)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Verify file exists
	content, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("Content = %q, want %q", string(content), "hello")
	}
}

func TestDiskFs_Add_Overwrite(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	_ = dfs.Add("test.txt", []byte("original"), false)

	// Without overwrite should fail
	err := dfs.Add("test.txt", []byte("new"), false)
	if err != ErrFileExists {
		t.Errorf("Expected ErrFileExists, got %v", err)
	}

	// With overwrite should succeed
	err = dfs.Add("test.txt", []byte("new"), true)
	if err != nil {
		t.Fatalf("Add with overwrite failed: %v", err)
	}

	content, _ := dfs.GetBytes("test.txt")
	if string(content) != "new" {
		t.Errorf("Content = %q, want %q", string(content), "new")
	}
}

func TestDiskFs_Add_NestedPath(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	err := dfs.Add("a/b/c/file.txt", []byte("nested"), false)
	if err != nil {
		t.Fatalf("Add nested failed: %v", err)
	}

	if !dfs.Exists("a/b/c/file.txt") {
		t.Error("Nested file should exist")
	}
}

func TestDiskFs_Get(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	dfs, _ := NewDiskFs(dir)

	rc, err := dfs.Get("test.txt")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("Content = %q, want %q", string(content), "hello world")
	}
}

func TestDiskFs_Get_NotFound(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	_, err := dfs.Get("nonexistent.txt")
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestDiskFs_GetBytes(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "test.txt"), []byte("test content"), 0644)

	dfs, _ := NewDiskFs(dir)

	content, err := dfs.GetBytes("test.txt")
	if err != nil {
		t.Fatalf("GetBytes failed: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Content = %q, want %q", string(content), "test content")
	}
}

func TestDiskFs_GetBytes_NotFound(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	_, err := dfs.GetBytes("nonexistent.txt")
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestDiskFs_Mkdir(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	err := dfs.Mkdir("newdir")
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	if !dfs.IsDir("newdir") {
		t.Error("newdir should be a directory")
	}
}

func TestDiskFs_Mkdir_Nested(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	err := dfs.Mkdir("a/b/c")
	if err != nil {
		t.Fatalf("Mkdir nested failed: %v", err)
	}

	if !dfs.IsDir("a/b/c") {
		t.Error("a/b/c should be a directory")
	}
}

func TestDiskFs_Exists(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	dfs, _ := NewDiskFs(dir)

	if !dfs.Exists("file.txt") {
		t.Error("file.txt should exist")
	}
	if !dfs.Exists("subdir") {
		t.Error("subdir should exist")
	}
	if dfs.Exists("nonexistent") {
		t.Error("nonexistent should not exist")
	}
}

func TestDiskFs_IsDir(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	dfs, _ := NewDiskFs(dir)

	if dfs.IsDir("file.txt") {
		t.Error("file.txt should not be a directory")
	}
	if !dfs.IsDir("subdir") {
		t.Error("subdir should be a directory")
	}
	if dfs.IsDir("nonexistent") {
		t.Error("nonexistent should not be a directory")
	}
}

func TestDiskFs_List(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("1"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("2"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	dfs, _ := NewDiskFs(dir)

	entries, err := dfs.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	sort.Strings(entries)
	expected := []string{"file1.txt", "file2.txt", "subdir"}
	if len(entries) != len(expected) {
		t.Errorf("Expected %d entries, got %d: %v", len(expected), len(entries), entries)
	}
}

func TestDiskFs_ListFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("1"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("2"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	dfs, _ := NewDiskFs(dir)

	files, err := dfs.ListFiles("")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	sort.Strings(files)
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}
}

func TestDiskFs_ListDirs(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("1"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, "dir1"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "dir2"), 0755)

	dfs, _ := NewDiskFs(dir)

	dirs, err := dfs.ListDirs("")
	if err != nil {
		t.Fatalf("ListDirs failed: %v", err)
	}

	sort.Strings(dirs)
	if len(dirs) != 2 {
		t.Errorf("Expected 2 dirs, got %d: %v", len(dirs), dirs)
	}
}

func TestDiskFs_Remove(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("1"), 0644)

	dfs, _ := NewDiskFs(dir)

	err := dfs.Remove("file.txt")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if dfs.Exists("file.txt") {
		t.Error("file.txt should not exist after Remove")
	}
}

func TestDiskFs_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	dfs, _ := NewDiskFs(dir)

	// Create a file outside the base directory
	outsideFile := filepath.Join(filepath.Dir(dir), "outside.txt")
	_ = os.WriteFile(outsideFile, []byte("secret"), 0644)
	defer os.Remove(outsideFile)

	// Try to access it via path traversal
	_, err := dfs.GetBytes("../outside.txt")
	if err != ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for path traversal, got %v", err)
	}

	// Try to write via path traversal
	err = dfs.Add("../outside.txt", []byte("hacked"), true)
	if err != ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for path traversal write, got %v", err)
	}
}

func TestDiskFs_Snapshot_CopiesNestedAndSkipsHiddenAndGen(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"index.html":      "i",
		"css/s.css":       "c",
		".DS_Store":       "x",
		".git/x":          "x",
		"_gen_7.png":      "x",
		"sub/_gen_ok.png": "k",
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	dfs, err := NewDiskFs(dir)
	if err != nil {
		t.Fatal(err)
	}
	zfs, err := dfs.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"index.html", "css/s.css", "sub/_gen_ok.png"} {
		if !zfs.Exists(n) {
			t.Errorf("%s missing from snapshot", n)
		}
	}
	for _, n := range []string{".DS_Store", ".git/x", "_gen_7.png"} {
		if zfs.Exists(n) {
			t.Errorf("%s should not be in snapshot", n)
		}
	}
}

func TestDiskFs_SiblingPrefixEscape(t *testing.T) {
	parent := t.TempDir()
	base := filepath.Join(parent, "tpl")
	secretDir := filepath.Join(parent, "tpl-secret")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "s.txt"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	dfs, err := NewDiskFs(base)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := dfs.GetBytes("../tpl-secret/s.txt"); err != ErrInvalidPath {
		t.Errorf("GetBytes err = %v, want ErrInvalidPath", err)
	}
	if dfs.Exists("../tpl-secret/s.txt") {
		t.Error("Exists returned true for sibling-folder file")
	}
	if err := dfs.Add("../tpl-secret/new.txt", []byte("x"), true); err != ErrInvalidPath {
		t.Errorf("Add err = %v, want ErrInvalidPath", err)
	}
	if _, err := os.Stat(filepath.Join(secretDir, "new.txt")); !os.IsNotExist(err) {
		t.Errorf("new.txt written outside base (err=%v)", err)
	}
}

func TestDiskFs_DotDotPrefixedNameInsideBase(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "..notes"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	dfs, _ := NewDiskFs(dir)

	got, err := dfs.GetBytes("..notes")
	if err != nil {
		t.Fatalf("GetBytes failed: %v", err)
	}
	if string(got) != "ok" {
		t.Errorf("content = %q, want ok", got)
	}
}

func TestDiskFs_SubdirAndRootStillResolve(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b.txt"), []byte("bee"), 0644); err != nil {
		t.Fatal(err)
	}
	dfs, _ := NewDiskFs(dir)

	if !dfs.Exists("") {
		t.Error("Exists(\"\") = false, want true")
	}
	got, err := dfs.GetBytes("a/../a/b.txt")
	if err != nil {
		t.Fatalf("GetBytes failed: %v", err)
	}
	if string(got) != "bee" {
		t.Errorf("content = %q, want bee", got)
	}
}
