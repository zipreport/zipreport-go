package fileutils

import (
	"sort"
	"sync"
	"testing"
)

func TestNewPathCache(t *testing.T) {
	pc := NewPathCache()
	if pc == nil {
		t.Fatal("NewPathCache returned nil")
	}
	if pc.root == nil {
		t.Error("root is nil")
	}
	if !pc.root.isDir {
		t.Error("root should be a directory")
	}
}

func TestPathCache_Add(t *testing.T) {
	pc := NewPathCache()

	// Add a file
	pc.Add("file.txt", false)
	if !pc.Exists("file.txt") {
		t.Error("file.txt should exist")
	}
	if !pc.IsFile("file.txt") {
		t.Error("file.txt should be a file")
	}
	if pc.IsDir("file.txt") {
		t.Error("file.txt should not be a directory")
	}

	// Add a directory
	pc.Add("mydir", true)
	if !pc.Exists("mydir") {
		t.Error("mydir should exist")
	}
	if !pc.IsDir("mydir") {
		t.Error("mydir should be a directory")
	}
}

func TestPathCache_AddNested(t *testing.T) {
	pc := NewPathCache()

	pc.Add("a/b/c/file.txt", false)

	// All parent directories should exist
	if !pc.IsDir("a") {
		t.Error("a should be a directory")
	}
	if !pc.IsDir("a/b") {
		t.Error("a/b should be a directory")
	}
	if !pc.IsDir("a/b/c") {
		t.Error("a/b/c should be a directory")
	}
	if !pc.IsFile("a/b/c/file.txt") {
		t.Error("a/b/c/file.txt should be a file")
	}
}

func TestPathCache_Remove(t *testing.T) {
	pc := NewPathCache()

	pc.Add("file.txt", false)
	if !pc.Exists("file.txt") {
		t.Error("file.txt should exist")
	}

	pc.Remove("file.txt")
	if pc.Exists("file.txt") {
		t.Error("file.txt should not exist after remove")
	}

	// Removing non-existent path should not panic
	pc.Remove("nonexistent")
	pc.Remove("")
}

func TestPathCache_Exists(t *testing.T) {
	pc := NewPathCache()

	pc.Add("file.txt", false)
	pc.Add("dir", true)

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
		if got := pc.Exists(tt.path); got != tt.exists {
			t.Errorf("Exists(%q) = %v, want %v", tt.path, got, tt.exists)
		}
	}
}

func TestPathCache_List(t *testing.T) {
	pc := NewPathCache()

	pc.Add("file1.txt", false)
	pc.Add("file2.txt", false)
	pc.Add("dir", true)

	entries := pc.List("")
	sort.Strings(entries)

	expected := []string{"dir", "file1.txt", "file2.txt"}
	if len(entries) != len(expected) {
		t.Fatalf("Expected %d entries, got %d: %v", len(expected), len(entries), entries)
	}

	for i, e := range entries {
		if e != expected[i] {
			t.Errorf("Expected %s at position %d, got %s", expected[i], i, e)
		}
	}
}

func TestPathCache_ListFiles(t *testing.T) {
	pc := NewPathCache()

	pc.Add("file1.txt", false)
	pc.Add("file2.txt", false)
	pc.Add("dir", true)

	files := pc.ListFiles("")
	sort.Strings(files)

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d: %v", len(files), files)
	}
	if files[0] != "file1.txt" || files[1] != "file2.txt" {
		t.Errorf("Unexpected files: %v", files)
	}
}

func TestPathCache_ListDirs(t *testing.T) {
	pc := NewPathCache()

	pc.Add("file.txt", false)
	pc.Add("dir1", true)
	pc.Add("dir2", true)

	dirs := pc.ListDirs("")
	sort.Strings(dirs)

	if len(dirs) != 2 {
		t.Fatalf("Expected 2 dirs, got %d: %v", len(dirs), dirs)
	}
	if dirs[0] != "dir1" || dirs[1] != "dir2" {
		t.Errorf("Unexpected dirs: %v", dirs)
	}
}

func TestPathCache_Clear(t *testing.T) {
	pc := NewPathCache()

	pc.Add("file1.txt", false)
	pc.Add("file2.txt", false)
	pc.Add("dir/file3.txt", false)

	pc.Clear()

	if pc.Exists("file1.txt") {
		t.Error("file1.txt should not exist after clear")
	}
	if pc.Exists("dir") {
		t.Error("dir should not exist after clear")
	}

	// Root should still be valid
	if !pc.Exists("") {
		t.Error("root should exist after clear")
	}
}

func TestPathCache_Concurrent(t *testing.T) {
	pc := NewPathCache()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			path := "file" + string(rune('0'+n%10)) + ".txt"
			pc.Add(path, false)
			pc.Exists(path)
			pc.List("")
		}(i)
	}
	wg.Wait()
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "."},
		{".", "."},
		{"/", ""},
		{"//", ""},
		{"file.txt", "file.txt"},
		{"/file.txt", "file.txt"},
		{"./file.txt", "file.txt"},
		{"dir/file.txt", "dir/file.txt"},
		{"/dir/file.txt", "dir/file.txt"},
		{"dir//file.txt", "dir/file.txt"},
		{"dir/./file.txt", "dir/file.txt"},
		{"dir/../file.txt", "file.txt"},
	}

	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{".", nil},
		{"file.txt", []string{"file.txt"}},
		{"dir/file.txt", []string{"dir", "file.txt"}},
		{"a/b/c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		got := splitPath(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("splitPath(%q) = %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i, part := range got {
			if part != tt.expected[i] {
				t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.input, i, part, tt.expected[i])
			}
		}
	}
}
