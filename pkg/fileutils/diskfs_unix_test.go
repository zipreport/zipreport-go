//go:build unix

package fileutils

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func writeTestFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func snapshotOf(t *testing.T, dir string) *ZipFs {
	t.Helper()
	dfs, err := NewDiskFs(dir)
	if err != nil {
		t.Fatal(err)
	}
	zfs, err := dfs.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	return zfs
}

func TestDiskFs_Snapshot_SymlinkedRoot(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	writeTestFile(t, filepath.Join(real, "index.html"), "i")
	link := filepath.Join(tmp, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if !snapshotOf(t, link).Exists("index.html") {
		t.Error("index.html missing")
	}
}

func TestDiskFs_Snapshot_DotNamedRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".tpl")
	writeTestFile(t, filepath.Join(dir, "index.html"), "i")
	if !snapshotOf(t, dir).Exists("index.html") {
		t.Error("index.html missing")
	}
}

func TestDiskFs_Snapshot_SymlinkInsideFollowed(t *testing.T) {
	real := t.TempDir()
	writeTestFile(t, filepath.Join(real, "shared", "h.css"), "h")
	if err := os.Symlink("shared", filepath.Join(real, "partials")); err != nil {
		t.Fatal(err)
	}
	zfs := snapshotOf(t, real)
	for _, n := range []string{"partials/h.css", "shared/h.css"} {
		if !zfs.Exists(n) {
			t.Errorf("%s missing", n)
		}
	}
}

func TestDiskFs_Snapshot_SymlinkOutsideSkipped(t *testing.T) {
	real := t.TempDir()
	writeTestFile(t, filepath.Join(real, "index.html"), "i")
	other := t.TempDir()
	writeTestFile(t, filepath.Join(other, "secret.txt"), "s")
	if err := os.Symlink(other, filepath.Join(real, "ext")); err != nil {
		t.Fatal(err)
	}
	zfs := snapshotOf(t, real)
	if zfs.Exists("ext/secret.txt") {
		t.Error("file outside the folder was included")
	}
}

func TestDiskFs_Snapshot_SpecialAndBrokenSkipped(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "index.html"), "i")
	if err := os.Symlink("missing.pdf", filepath.Join(dir, "old.pdf")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "a.fifo"), 0644); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(dir, "s.sock")
	hasSock := len(sock) <= 100
	if hasSock {
		l, err := net.Listen("unix", sock)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = l.Close() })
	}

	done := make(chan *ZipFs, 1)
	errc := make(chan error, 1)
	go func() {
		dfs, err := NewDiskFs(dir)
		if err != nil {
			errc <- err
			return
		}
		zfs, err := dfs.Snapshot()
		if err != nil {
			errc <- err
			return
		}
		done <- zfs
	}()
	select {
	case zfs := <-done:
		if !zfs.Exists("index.html") {
			t.Error("index.html missing")
		}
		for _, n := range []string{"old.pdf", "a.fifo", "s.sock"} {
			if zfs.Exists(n) {
				t.Errorf("%s should be skipped", n)
			}
		}
	case err := <-errc:
		t.Fatalf("Snapshot: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Snapshot did not finish within 5s")
	}
}

func TestDiskFs_Snapshot_SymlinkLoopSkipped(t *testing.T) {
	real := t.TempDir()
	writeTestFile(t, filepath.Join(real, "index.html"), "i")
	if err := os.MkdirAll(filepath.Join(real, "a"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("..", filepath.Join(real, "a", "b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".", filepath.Join(real, "x")); err != nil {
		t.Fatal(err)
	}
	if !snapshotOf(t, real).Exists("index.html") {
		t.Error("index.html missing")
	}
}

func TestDiskFs_Snapshot_UnreadableSkipped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "index.html"), "i")
	writeTestFile(t, filepath.Join(dir, "notes.txt"), "n")
	writeTestFile(t, filepath.Join(dir, "private", "x.txt"), "x")
	if err := os.Chmod(filepath.Join(dir, "notes.txt"), 0000); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, "private"), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(dir, "private"), 0755)
		_ = os.Chmod(filepath.Join(dir, "notes.txt"), 0644)
	})
	zfs := snapshotOf(t, dir)
	if !zfs.Exists("index.html") {
		t.Error("index.html missing")
	}
	for _, n := range []string{"notes.txt", "private/x.txt"} {
		if zfs.Exists(n) {
			t.Errorf("%s should be skipped", n)
		}
	}
}

func TestDiskFs_Snapshot_UnportableNamesSkipped(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "index.html"), "x")
	writeTestFile(t, filepath.Join(dir, "a:b.css"), "x")
	writeTestFile(t, filepath.Join(dir, "1:1.png"), "x")
	zfs := snapshotOf(t, dir)
	if !zfs.Exists("index.html") || !zfs.Exists("1:1.png") {
		t.Errorf("missing expected files: %v", zfs.Files())
	}
	if zfs.Exists("a:b.css") {
		t.Errorf("a:b.css should be skipped: %v", zfs.Files())
	}
}

func TestDiskFs_SymlinkOutsideRejected(t *testing.T) {
	outside := t.TempDir()
	writeTestFile(t, filepath.Join(outside, "secret.txt"), "secret")
	writeTestFile(t, filepath.Join(outside, "sub", "x.txt"), "x")
	base := t.TempDir()
	for link, target := range map[string]string{
		"ext.txt": filepath.Join(outside, "secret.txt"),
		"extdir":  filepath.Join(outside, "sub"),
		"dang":    filepath.Join(outside, "missing.txt"),
	} {
		if err := os.Symlink(target, filepath.Join(base, link)); err != nil {
			t.Fatal(err)
		}
	}
	dfs, err := NewDiskFs(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"ext.txt", "extdir/x.txt"} {
		if _, err := dfs.GetBytes(n); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("GetBytes(%q) err = %v, want ErrInvalidPath", n, err)
		}
	}
	if dfs.Exists("ext.txt") {
		t.Error("Exists(ext.txt) should be false")
	}
	for _, n := range []string{"extdir/new.txt", "dang"} {
		if err := dfs.Add(n, []byte("w"), true); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("Add(%q) err = %v, want ErrInvalidPath", n, err)
		}
	}
	for _, p := range []string{filepath.Join(outside, "sub", "new.txt"), filepath.Join(outside, "missing.txt")} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s was created outside the base", p)
		}
	}
}

func TestDiskFs_SymlinkInsideAllowed(t *testing.T) {
	base := t.TempDir()
	writeTestFile(t, filepath.Join(base, "real.txt"), "r")
	writeTestFile(t, filepath.Join(base, "sub", "x.txt"), "x")
	if err := os.Symlink("real.txt", filepath.Join(base, "alias.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub", filepath.Join(base, "adir")); err != nil {
		t.Fatal(err)
	}
	dfs, err := NewDiskFs(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"alias.txt", "adir/x.txt"} {
		if _, err := dfs.GetBytes(n); err != nil {
			t.Errorf("GetBytes(%q): %v", n, err)
		}
	}
	if err := dfs.Add("adir/y.txt", []byte("y"), true); err != nil {
		t.Errorf("Add(adir/y.txt): %v", err)
	}
}

func TestDiskFs_SymlinkedBaseAllowed(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	writeTestFile(t, filepath.Join(real, "index.html"), "x")
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	dfs, err := NewDiskFs(link)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dfs.GetBytes("index.html"); err != nil {
		t.Errorf("GetBytes: %v", err)
	}
	if err := dfs.Add("out.html", []byte("o"), true); err != nil {
		t.Errorf("Add: %v", err)
	}
}

func TestDiskFs_Snapshot_DepthLimit(t *testing.T) {
	chain := func(n int) string {
		root := t.TempDir()
		p := root
		for i := 0; i < n; i++ {
			p = filepath.Join(p, "d")
		}
		writeTestFile(t, filepath.Join(p, "f.txt"), "x")
		return root
	}

	deep := chain(40)
	dfs, err := NewDiskFs(deep)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dfs.Snapshot(); err == nil || !strings.Contains(err.Error(), "deeper than 32") {
		t.Errorf("40 levels: err = %v, want 'deeper than 32'", err)
	}

	ok := chain(30)
	zfs := snapshotOf(t, ok)
	if !zfs.Exists(strings.Repeat("d/", 30) + "f.txt") {
		t.Error("30-level file missing from snapshot")
	}
}
