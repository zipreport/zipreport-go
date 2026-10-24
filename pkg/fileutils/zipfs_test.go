package fileutils

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestNewZipFs(t *testing.T) {
	zfs := NewZipFs()
	if zfs == nil {
		t.Fatal("NewZipFs returned nil")
	}
	if zfs.files == nil {
		t.Error("files map is nil")
	}
	if zfs.cache == nil {
		t.Error("cache is nil")
	}
}

func TestZipFs_Add(t *testing.T) {
	zfs := NewZipFs()

	// Test adding a file
	err := zfs.Add("test.txt", []byte("hello"), false)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Test file exists
	if !zfs.Exists("test.txt") {
		t.Error("File should exist after Add")
	}

	// Test adding without overwrite fails
	err = zfs.Add("test.txt", []byte("world"), false)
	if err != ErrFileExists {
		t.Errorf("Expected ErrFileExists, got %v", err)
	}

	// Test adding with overwrite succeeds
	err = zfs.Add("test.txt", []byte("world"), true)
	if err != nil {
		t.Fatalf("Add with overwrite failed: %v", err)
	}

	content, err := zfs.GetBytes("test.txt")
	if err != nil {
		t.Fatalf("GetBytes failed: %v", err)
	}
	if string(content) != "world" {
		t.Errorf("Expected 'world', got '%s'", string(content))
	}

}

func TestZipFs_AddWithNestedPath(t *testing.T) {
	zfs := NewZipFs()

	err := zfs.Add("dir1/dir2/file.txt", []byte("nested"), false)
	if err != nil {
		t.Fatalf("Add nested file failed: %v", err)
	}

	// Parent directories should exist
	if !zfs.Exists("dir1") {
		t.Error("dir1 should exist")
	}
	if !zfs.IsDir("dir1") {
		t.Error("dir1 should be a directory")
	}
	if !zfs.Exists("dir1/dir2") {
		t.Error("dir1/dir2 should exist")
	}
	if !zfs.IsDir("dir1/dir2") {
		t.Error("dir1/dir2 should be a directory")
	}
}

func TestZipFs_Get(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("test.txt", []byte("hello world"), false)

	// Test Get
	rc, err := zfs.Get("test.txt")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(content))
	}

	// Test Get non-existent file
	_, err = zfs.Get("nonexistent.txt")
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestZipFs_GetBytes(t *testing.T) {
	zfs := NewZipFs()
	original := []byte("hello world")
	_ = zfs.Add("test.txt", original, false)

	content, err := zfs.GetBytes("test.txt")
	if err != nil {
		t.Fatalf("GetBytes failed: %v", err)
	}

	if string(content) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(content))
	}

	// Modify returned content shouldn't affect stored content
	content[0] = 'X'
	content2, _ := zfs.GetBytes("test.txt")
	if content2[0] != 'h' {
		t.Error("Modifying returned content affected stored content")
	}

	// Test non-existent file
	_, err = zfs.GetBytes("nonexistent.txt")
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestZipFs_Mkdir(t *testing.T) {
	zfs := NewZipFs()

	err := zfs.Mkdir("mydir")
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	if !zfs.Exists("mydir") {
		t.Error("Directory should exist")
	}
	if !zfs.IsDir("mydir") {
		t.Error("Should be a directory")
	}

	// Empty path should not error
	err = zfs.Mkdir("")
	if err != nil {
		t.Errorf("Mkdir with empty path should not error: %v", err)
	}
}

func TestZipFs_Exists(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("file.txt", []byte("data"), false)
	_ = zfs.Mkdir("dir")

	tests := []struct {
		path   string
		exists bool
	}{
		{"file.txt", true},
		{"dir", true},
		{"nonexistent", false},
		{"", true},  // root
		{".", true}, // root
	}

	for _, tt := range tests {
		if got := zfs.Exists(tt.path); got != tt.exists {
			t.Errorf("Exists(%q) = %v, want %v", tt.path, got, tt.exists)
		}
	}
}

func TestZipFs_IsDir(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("file.txt", []byte("data"), false)
	_ = zfs.Mkdir("dir")

	tests := []struct {
		path  string
		isDir bool
	}{
		{"file.txt", false},
		{"dir", true},
		{"", true},  // root
		{".", true}, // root
	}

	for _, tt := range tests {
		if got := zfs.IsDir(tt.path); got != tt.isDir {
			t.Errorf("IsDir(%q) = %v, want %v", tt.path, got, tt.isDir)
		}
	}
}

func TestZipFs_List(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("file1.txt", []byte("1"), false)
	_ = zfs.Add("file2.txt", []byte("2"), false)
	_ = zfs.Add("dir/file3.txt", []byte("3"), false)

	entries, err := zfs.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	sort.Strings(entries)
	expected := []string{"dir", "file1.txt", "file2.txt"}
	if len(entries) != len(expected) {
		t.Errorf("Expected %d entries, got %d: %v", len(expected), len(entries), entries)
	}

	// Test listing non-directory
	_, err = zfs.List("file1.txt")
	if err != ErrNotADirectory {
		t.Errorf("Expected ErrNotADirectory, got %v", err)
	}
}

func TestZipFs_ListFiles(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("file1.txt", []byte("1"), false)
	_ = zfs.Add("dir/file2.txt", []byte("2"), false)
	_ = zfs.Mkdir("emptydir")

	files, err := zfs.ListFiles("")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 1 || files[0] != "file1.txt" {
		t.Errorf("Expected [file1.txt], got %v", files)
	}
}

func TestZipFs_ListDirs(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("file1.txt", []byte("1"), false)
	_ = zfs.Add("dir1/file.txt", []byte("2"), false)
	_ = zfs.Mkdir("dir2")

	dirs, err := zfs.ListDirs("")
	if err != nil {
		t.Fatalf("ListDirs failed: %v", err)
	}

	sort.Strings(dirs)
	if len(dirs) != 2 {
		t.Errorf("Expected 2 dirs, got %v", dirs)
	}
}

func TestZipFs_Remove(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("test.txt", []byte("data"), false)

	err := zfs.Remove("test.txt")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if zfs.Exists("test.txt") {
		t.Error("File should not exist after Remove")
	}

	// Test removing non-existent file
	err = zfs.Remove("nonexistent.txt")
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestZipFs_Files(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("a.txt", []byte("a"), false)
	_ = zfs.Add("b.txt", []byte("b"), false)
	_ = zfs.Add("dir/c.txt", []byte("c"), false)

	files := zfs.Files()
	sort.Strings(files)

	expected := []string{"a.txt", "b.txt", "dir/c.txt"}
	if len(files) != len(expected) {
		t.Fatalf("Expected %d files, got %d", len(expected), len(files))
	}

	for i, f := range files {
		if f != expected[i] {
			t.Errorf("Expected %s at position %d, got %s", expected[i], i, f)
		}
	}
}

func TestZipFs_Save(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("test.txt", []byte("hello"), false)
	_ = zfs.Add("dir/nested.txt", []byte("world"), false)

	data, err := zfs.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify it's valid ZIP data
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Invalid ZIP data: %v", err)
	}

	fileCount := 0
	for _, f := range reader.File {
		if !f.FileInfo().IsDir() {
			fileCount++
		}
	}
	if fileCount != 2 {
		t.Errorf("Expected 2 files in ZIP, got %d", fileCount)
	}
}

func TestZipFs_SaveToFile(t *testing.T) {
	zfs := NewZipFs()
	_ = zfs.Add("test.txt", []byte("hello"), false)

	tmpFile := filepath.Join(t.TempDir(), "test.zip")
	err := zfs.SaveToFile(tmpFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists and is valid
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	_, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Saved file is not valid ZIP: %v", err)
	}
}

func TestNewZipFsFromBytes(t *testing.T) {
	// Create a ZIP in memory
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)
	w, _ := writer.Create("test.txt")
	_, _ = w.Write([]byte("hello from zip"))
	_ = writer.Close()

	// Load it
	zfs, err := NewZipFsFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("NewZipFsFromBytes failed: %v", err)
	}

	content, err := zfs.GetBytes("test.txt")
	if err != nil {
		t.Fatalf("GetBytes failed: %v", err)
	}

	if string(content) != "hello from zip" {
		t.Errorf("Expected 'hello from zip', got '%s'", string(content))
	}
}

func TestNewZipFsFromBytes_Invalid(t *testing.T) {
	_, err := NewZipFsFromBytes([]byte("not a zip"))
	if err == nil {
		t.Error("Expected error for invalid ZIP data")
	}
}

func TestNewZipFsFromFile(t *testing.T) {
	// Create a temporary ZIP file
	tmpFile := filepath.Join(t.TempDir(), "test.zip")

	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)
	w, _ := writer.Create("file.txt")
	_, _ = w.Write([]byte("file content"))
	_ = writer.Close()

	if err := os.WriteFile(tmpFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	zfs, err := NewZipFsFromFile(tmpFile)
	if err != nil {
		t.Fatalf("NewZipFsFromFile failed: %v", err)
	}

	content, err := zfs.GetBytes("file.txt")
	if err != nil {
		t.Fatalf("GetBytes failed: %v", err)
	}

	if string(content) != "file content" {
		t.Errorf("Expected 'file content', got '%s'", string(content))
	}
}

func TestNewZipFsFromFile_NotFound(t *testing.T) {
	_, err := NewZipFsFromFile("/nonexistent/file.zip")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestZipFs_PathNormalization(t *testing.T) {
	zfs := NewZipFs()

	// Add with various path formats
	_ = zfs.Add("./file1.txt", []byte("1"), false)
	_ = zfs.Add("/file2.txt", []byte("2"), false)
	_ = zfs.Add("dir//file3.txt", []byte("3"), false)

	// Should all be accessible with normalized paths
	if !zfs.Exists("file1.txt") {
		t.Error("file1.txt should exist")
	}
	if !zfs.Exists("file2.txt") {
		t.Error("file2.txt should exist")
	}
	if !zfs.Exists("dir/file3.txt") {
		t.Error("dir/file3.txt should exist")
	}
}

func TestZipFs_Clone_Independent(t *testing.T) {
	orig := NewZipFs()
	if err := orig.Add("a/b.txt", []byte("one"), false); err != nil {
		t.Fatal(err)
	}
	clone := orig.Clone()

	if err := clone.Add("new.txt", []byte("n"), false); err != nil {
		t.Fatal(err)
	}
	if orig.Exists("new.txt") {
		t.Error("write to clone leaked into original")
	}

	if err := orig.Add("a/b.txt", []byte("two"), true); err != nil {
		t.Fatal(err)
	}
	got, err := clone.GetBytes("a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "one" {
		t.Errorf("clone saw original's write: %q", got)
	}
	if !clone.IsDir("a") {
		t.Error("clone lost directory a")
	}
}

func TestZipFs_SaveDeterministic(t *testing.T) {
	names := []string{"z.txt", "a/b.txt", "m/n/o.txt", "c.txt"}
	for i := 0; i < 26; i++ {
		names = append(names, fmt.Sprintf("%c/x.txt", 'a'+i))
	}
	a := NewZipFs()
	for _, n := range names {
		if err := a.Add(n, []byte("content "+n), false); err != nil {
			t.Fatal(err)
		}
	}
	b := NewZipFs()
	for i := len(names) - 1; i >= 0; i-- {
		if err := b.Add(names[i], []byte("content "+names[i]), false); err != nil {
			t.Fatal(err)
		}
	}
	da, err := a.Save()
	if err != nil {
		t.Fatal(err)
	}
	db, err := b.Save()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(da, db) {
		t.Error("Save output differs for identical content")
	}

	r, err := zip.NewReader(bytes.NewReader(da), int64(len(da)))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range r.File {
		got = append(got, f.Name)
	}
	dirSet := map[string]bool{}
	for _, n := range names {
		for d := path.Dir(n); d != "."; d = path.Dir(d) {
			dirSet[d] = true
		}
	}
	var want, files []string
	for d := range dirSet {
		want = append(want, d)
	}
	sort.Strings(want)
	for i := range want {
		want[i] += "/"
	}
	files = append(files, names...)
	sort.Strings(files)
	want = append(want, files...)
	if !slices.Equal(got, want) {
		t.Errorf("entry order = %v, want %v", got, want)
	}
}

func zipWith(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func setMaxUncompressed(t *testing.T, n int64) {
	t.Helper()
	old := MaxUncompressedSize
	MaxUncompressedSize = n
	t.Cleanup(func() { MaxUncompressedSize = old })
}

func TestNewZipFsFromBytes_TooLarge(t *testing.T) {
	setMaxUncompressed(t, 1024)
	data := zipWith(t, map[string][]byte{
		"a.txt": bytes.Repeat([]byte("a"), 600),
		"b.txt": bytes.Repeat([]byte("b"), 600),
	})
	if _, err := NewZipFsFromBytes(data); !errors.Is(err, ErrZipTooLarge) {
		t.Fatalf("expected ErrZipTooLarge, got %v", err)
	}
}

func TestNewZipFsFromBytes_AtLimit(t *testing.T) {
	setMaxUncompressed(t, 1024)
	content := bytes.Repeat([]byte("a"), 1024)
	zfs, err := NewZipFsFromBytes(zipWith(t, map[string][]byte{"a.txt": content}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := zfs.GetBytes("a.txt")
	if err != nil || !bytes.Equal(got, content) {
		t.Errorf("content mismatch: len=%d err=%v", len(got), err)
	}
}

func TestNewZipFsFromFile_TooLarge(t *testing.T) {
	setMaxUncompressed(t, 1024)
	data := zipWith(t, map[string][]byte{
		"a.txt": bytes.Repeat([]byte("a"), 600),
		"b.txt": bytes.Repeat([]byte("b"), 600),
	})
	fn := filepath.Join(t.TempDir(), "big.zip")
	if err := os.WriteFile(fn, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewZipFsFromFile(fn); !errors.Is(err, ErrZipTooLarge) {
		t.Fatalf("expected ErrZipTooLarge, got %v", err)
	}
}

func TestNewZipFsFromBytes_MaxInt64Limit(t *testing.T) {
	setMaxUncompressed(t, math.MaxInt64)
	zfs, err := NewZipFsFromBytes(zipWith(t, map[string][]byte{"a.txt": bytes.Repeat([]byte("a"), 100)}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := zfs.GetBytes("a.txt")
	if len(got) != 100 {
		t.Errorf("expected 100 bytes, got %d", len(got))
	}
}

func TestNewZipFsFromBytes_NonPositiveLimitUsesDefault(t *testing.T) {
	data := zipWith(t, map[string][]byte{"a.txt": bytes.Repeat([]byte("a"), 100)})
	for _, n := range []int64{0, -1} {
		setMaxUncompressed(t, n)
		zfs, err := NewZipFsFromBytes(data)
		if err != nil {
			t.Fatalf("limit %d: unexpected error: %v", n, err)
		}
		got, _ := zfs.GetBytes("a.txt")
		if len(got) != 100 {
			t.Errorf("limit %d: expected 100 bytes, got %d", n, len(got))
		}
	}
}

func TestZipFs_Add_RejectsEscapingNames(t *testing.T) {
	for _, name := range []string{"../x", "..", "a/../../x", `..\x`, `a\..\..\x`, "C:/x", `\evil.txt`, `\\srv\share\x`, "", "."} {
		z := NewZipFs()
		err := z.Add(name, []byte("x"), true)
		if !errors.Is(err, ErrInvalidPath) {
			t.Errorf("Add(%q) err = %v, want ErrInvalidPath", name, err)
		}
		if name != "" && name != "." && !errors.Is(err, ErrUnportableName) {
			t.Errorf("Add(%q) err = %v, want ErrUnportableName", name, err)
		}
		if len(z.Files()) != 0 {
			t.Errorf("Add(%q) stored %v", name, z.Files())
		}
	}
	z := NewZipFs()
	if err := z.Add("1:1.png", []byte("x"), true); err != nil {
		t.Errorf("Add(1:1.png) err = %v", err)
	}
	if err := z.Add("a/../b", []byte("x"), true); err != nil {
		t.Errorf("Add(a/../b) err = %v", err)
	}
	if !z.Exists("b") {
		t.Error("a/../b should be stored as b")
	}
	if err := z.Mkdir("../d"); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("Mkdir(../d) err = %v, want ErrInvalidPath", err)
	}
}

func TestNewZipFsFromBytes_RejectsEscapingEntries(t *testing.T) {
	for _, evil := range []string{"../evil.txt", `..\evil.txt`, `\evil.txt`, "a/.."} {
		buf := &bytes.Buffer{}
		w := zip.NewWriter(buf)
		for _, n := range []string{"index.html", evil} {
			f, err := w.Create(n)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = f.Write([]byte("x"))
		}
		w.Close()
		if _, err := NewZipFsFromBytes(buf.Bytes()); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("entry %q: err = %v, want ErrInvalidPath", evil, err)
		}
	}
}

func TestNewZipFsFromBytes_TooManyEntries(t *testing.T) {
	files := make(map[string][]byte, maxZipEntries+1)
	for i := 0; i <= maxZipEntries; i++ {
		files[fmt.Sprintf("f%05d", i)] = []byte("x")
	}
	if _, err := NewZipFsFromBytes(zipWith(t, files)); !errors.Is(err, ErrZipTooLarge) {
		t.Fatalf("expected ErrZipTooLarge, got %v", err)
	}
}

func TestNewZipFsFromBytes_NameLimits(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"32 levels", strings.Repeat("a/", 32) + "f", false},
		{"33 levels", strings.Repeat("a/", 33) + "f", true},
		{"1024 bytes", strings.Repeat("n", 1024), false},
		{"1025 bytes", strings.Repeat("n", 1025), true},
		{"32 level folder entry", strings.Repeat("a/", 32), false},
		{"33 level folder entry", strings.Repeat("a/", 33), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte("x")
			if strings.HasSuffix(tt.entry, "/") {
				content = nil // folder entries carry no content
			}
			_, err := NewZipFsFromBytes(zipWith(t, map[string][]byte{tt.entry: content}))
			if tt.wantErr {
				if !errors.Is(err, ErrZipTooLarge) {
					t.Fatalf("expected ErrZipTooLarge, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
