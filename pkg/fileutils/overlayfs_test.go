package fileutils

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestOverlayFs_ReadsThroughAndIsolatesWrites(t *testing.T) {
	base := NewZipFs()
	if err := base.Add("index.html", []byte("base"), false); err != nil {
		t.Fatal(err)
	}
	o := NewOverlayFs(base)

	got, err := o.GetBytes("index.html")
	if err != nil || string(got) != "base" {
		t.Fatalf("GetBytes = %q, %v", got, err)
	}

	if err := o.Add("report.html", []byte("r"), true); err != nil {
		t.Fatal(err)
	}
	if base.Exists("report.html") {
		t.Error("write reached base")
	}
	if !o.Exists("report.html") {
		t.Error("overlay write not visible")
	}

	if err := o.Add("index.html", []byte("over"), true); err != nil {
		t.Fatal(err)
	}
	got, _ = o.GetBytes("index.html")
	if string(got) != "over" {
		t.Errorf("overlay read = %q, want over", got)
	}
	got, _ = base.GetBytes("index.html")
	if string(got) != "base" {
		t.Errorf("base read = %q, want base", got)
	}
}

func TestOverlayFs_DiskBaseNotWritten(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("disk"), 0644); err != nil {
		t.Fatal(err)
	}
	dfs, err := NewDiskFs(dir)
	if err != nil {
		t.Fatal(err)
	}
	o := NewOverlayFs(dfs)
	if err := o.Add("report.html", []byte("r"), true); err != nil {
		t.Fatal(err)
	}
	if err := o.Add("_gen_1.png", []byte("p"), true); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"report.html", "_gen_1.png"} {
		if _, err := os.Stat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Errorf("%s was written to disk (err=%v)", n, err)
		}
	}
	got, err := o.GetBytes("index.html")
	if err != nil || string(got) != "disk" {
		t.Errorf("GetBytes = %q, %v", got, err)
	}
}

func TestOverlayFs_ListMerges(t *testing.T) {
	base := NewZipFs()
	_ = base.Add("a.txt", []byte("a"), false)
	_ = base.Add("img/x.png", []byte("x"), false)
	o := NewOverlayFs(base)
	_ = o.Add("b.txt", []byte("b"), false)
	_ = o.Add("img/y.png", []byte("y"), false)

	names, err := o.List("")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	want := []string{"a.txt", "b.txt", "img"}
	if len(names) != len(want) {
		t.Fatalf("List = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("List = %v, want %v", names, want)
		}
	}

	files, err := o.ListFiles("img")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) != 2 || files[0] != "x.png" || files[1] != "y.png" {
		t.Errorf("ListFiles(img) = %v", files)
	}
}

func TestOverlayFs_AddWithoutOverwrite(t *testing.T) {
	base := NewZipFs()
	_ = base.Add("a.txt", []byte("a"), false)
	o := NewOverlayFs(base)
	if err := o.Add("a.txt", []byte("b"), false); !errors.Is(err, ErrFileExists) {
		t.Errorf("err = %v, want ErrFileExists", err)
	}
}

func TestOverlayFs_RemoveBaseFileFails(t *testing.T) {
	base := NewZipFs()
	_ = base.Add("a.txt", []byte("a"), false)
	o := NewOverlayFs(base)
	if err := o.Remove("a.txt"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("err = %v, want ErrReadOnly", err)
	}
	_ = o.Add("n.txt", []byte("n"), false)
	if err := o.Remove("n.txt"); err != nil {
		t.Errorf("Remove overlay file: %v", err)
	}
	if o.Exists("n.txt") {
		t.Error("n.txt still exists")
	}
}

func TestOverlayFs_ImplementsFsInterface(t *testing.T) {
	var _ FsInterface = NewOverlayFs(NewZipFs())
}

type rootlessFs struct{ *ZipFs }

func (r rootlessFs) Exists(name string) bool {
	if isRoot(name) {
		return false
	}
	return r.ZipFs.Exists(name)
}

func (r rootlessFs) IsDir(name string) bool {
	if isRoot(name) {
		return false
	}
	return r.ZipFs.IsDir(name)
}

func (r rootlessFs) List(p string) ([]string, error) {
	if isRoot(p) {
		return nil, ErrFileNotFound
	}
	return r.ZipFs.List(p)
}

func TestOverlayFs_RootComesFromBase(t *testing.T) {
	o := NewOverlayFs(rootlessFs{NewZipFs()})
	if err := o.Add("x.txt", []byte("x"), false); err != nil {
		t.Fatal(err)
	}
	if o.Exists("") {
		t.Error("Exists(\"\") = true, want false")
	}
	if o.IsDir("") {
		t.Error("IsDir(\"\") = true, want false")
	}
	if _, err := o.List(""); err == nil {
		t.Error("List(\"\") succeeded, want error")
	}
}

func TestOverlayFs_Accessors(t *testing.T) {
	base := NewZipFs()
	o := NewOverlayFs(base)
	if o.Base() != FsInterface(base) {
		t.Error("Base() is not the base passed in")
	}
	_ = o.Add("w.txt", []byte("w"), false)
	if !o.Overlay().Exists("w.txt") {
		t.Error("Overlay() does not hold the write")
	}
}

func TestOverlayFs_GetPrefersOverlay(t *testing.T) {
	base := NewZipFs()
	_ = base.Add("a.txt", []byte("base"), false)
	_ = base.Add("b.txt", []byte("base-b"), false)
	o := NewOverlayFs(base)
	_ = o.Add("a.txt", []byte("over"), true)

	for name, want := range map[string]string{"a.txt": "over", "b.txt": "base-b"} {
		rc, err := o.Get(name)
		if err != nil {
			t.Fatalf("Get(%s): %v", name, err)
		}
		got, _ := io.ReadAll(rc)
		_ = rc.Close()
		if string(got) != want {
			t.Errorf("Get(%s) = %q, want %q", name, got, want)
		}
	}
	if _, err := o.Get("missing.txt"); err == nil {
		t.Error("Get(missing.txt) succeeded")
	}
}

func TestOverlayFs_MkdirAndListDirs(t *testing.T) {
	base := NewZipFs()
	_ = base.Mkdir("css")
	_ = base.Add("img/x.png", []byte("x"), false)
	o := NewOverlayFs(base)
	if err := o.Mkdir("fonts"); err != nil {
		t.Fatal(err)
	}
	if err := o.Mkdir("img"); err != nil {
		t.Fatal(err)
	}
	if base.Exists("fonts") {
		t.Error("Mkdir reached base")
	}
	if !o.IsDir("fonts") {
		t.Error("overlay dir not visible")
	}

	dirs, err := o.ListDirs("")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(dirs)
	want := []string{"css", "fonts", "img"}
	if len(dirs) != len(want) {
		t.Fatalf("ListDirs = %v, want %v", dirs, want)
	}
	for i := range want {
		if dirs[i] != want[i] {
			t.Fatalf("ListDirs = %v, want %v", dirs, want)
		}
	}
}
