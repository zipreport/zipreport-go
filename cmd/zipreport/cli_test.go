package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/report"
)

// captureStdout runs fn and returns what it wrote to os.Stdout
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Drain concurrently so output larger than the pipe buffer cannot deadlock
	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()

	defer w.Close()
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	runErr := fn()
	_ = w.Close()
	return <-done, runErr
}

// writeCLITemplate writes a valid template; writeDebugTemplate uses a different manifest
func writeCLITemplate(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"manifest.json": `{"title":"CLI Test","author":"Tester","version":"1.0","description":"D","params":["name"]}`,
		"index.html":    `<p>{{ name }}</p>`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunBuild_Success(t *testing.T) {
	src := t.TempDir()
	writeCLITemplate(t, src)
	out := filepath.Join(t.TempDir(), "r.zpt")

	stdout, err := captureStdout(t, func() error { return runBuild(buildCmd, []string{src, out}) })
	if err != nil {
		t.Fatalf("runBuild: %v", err)
	}
	if !strings.Contains(stdout, "Successfully built") {
		t.Errorf("stdout missing success message: %q", stdout)
	}
	if _, err := report.LoadFile(out); err != nil {
		t.Errorf("built file not loadable: %v", err)
	}
}

func TestRunBuild_MissingIndexFails(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "r.zpt")

	stdout, err := captureStdout(t, func() error { return runBuild(buildCmd, []string{src, out}) })
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(err.Error(), "build failed:") || !strings.Contains(err.Error(), "index.html") {
		t.Errorf("unexpected error: %v", err)
	}
	if strings.Contains(stdout, "Successfully built") {
		t.Errorf("stdout claims success: %q", stdout)
	}
	if _, statErr := os.Stat(out); statErr == nil {
		t.Error("output file should not exist")
	}
}

func TestRunInfo_PrintsManifest(t *testing.T) {
	src := t.TempDir()
	writeCLITemplate(t, src)
	out := filepath.Join(t.TempDir(), "r.zpt")
	if res := report.Build(src, out, nil); !res.Success() {
		t.Fatalf("build: %v", res.Error())
	}

	stdout, err := captureStdout(t, func() error { return runInfo(infoCmd, []string{out}) })
	if err != nil {
		t.Fatalf("runInfo: %v", err)
	}
	for _, want := range []string{"Title:       CLI Test", "Author:      Tester", "Parameters:  name"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunInfo_MissingFile(t *testing.T) {
	_, err := captureStdout(t, func() error {
		return runInfo(infoCmd, []string{filepath.Join(t.TempDir(), "nope.zpt")})
	})
	if err == nil || !strings.Contains(err.Error(), "failed to load report") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunList_FindsZptFiles(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.zpt", "b.zpt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	stdout, err := captureStdout(t, func() error { return runList(listCmd, []string{dir}) })
	if err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"Found 2 .zpt file(s)", "  a.zpt", "  b.zpt"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "c.txt") {
		t.Errorf("stdout lists c.txt:\n%s", stdout)
	}
}

func TestRunList_Empty(t *testing.T) {
	stdout, err := captureStdout(t, func() error { return runList(listCmd, []string{t.TempDir()}) })
	if err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(stdout, "No .zpt files found") {
		t.Errorf("unexpected stdout: %q", stdout)
	}
}

func TestRunList_NotADirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := captureStdout(t, func() error { return runList(listCmd, []string{f}) })
	if err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVersionCmd(t *testing.T) {
	stdout, _ := captureStdout(t, func() error { versionCmd.Run(versionCmd, nil); return nil })
	if stdout != "zipreport version "+currentVersion()+"\n" {
		t.Errorf("unexpected stdout: %q", stdout)
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name    string
		ldflags string
		info    *debug.BuildInfo
		want    string
	}{
		{"ldflags win", "1.2.3", &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}}, "1.2.3"},
		{"module version", "", &debug.BuildInfo{Main: debug.Module{Version: "v1.1.0"}}, "1.1.0"},
		{"devel", "", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "dev"},
		{"empty", "", &debug.BuildInfo{}, "dev"},
		{"nil info", "", nil, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.ldflags, tt.info); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCLI_Binary_VersionFromBuildInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e binary test in short mode")
	}
	bin := filepath.Join(t.TempDir(), "zipreport")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		if strings.Contains(string(out), "error obtaining VCS status") {
			t.Skipf("VCS status unavailable: %s", out)
		}
		t.Fatalf("go build: %v\n%s", err, out)
	}

	mout, err := exec.Command("go", "version", "-m", bin).CombinedOutput()
	if err != nil {
		t.Fatalf("go version -m: %v\n%s", err, mout)
	}
	modVer, found := "", false
	for _, line := range strings.Split(string(mout), "\n") {
		f := strings.Fields(line)
		if len(f) >= 1 && f[0] == "mod" {
			found = true
			if len(f) >= 3 {
				modVer = f[2]
			}
			break
		}
	}
	if !found {
		t.Fatalf("no mod line in go version -m output:\n%s", mout)
	}
	want := "dev"
	if modVer != "" && modVer != "(devel)" {
		want = strings.TrimPrefix(modVer, "v")
	}

	vout, err := exec.Command(bin, "version").Output()
	if err != nil {
		t.Fatalf("version command: %v", err)
	}
	if string(vout) != "zipreport version "+want+"\n" {
		t.Errorf("unexpected version output: %q, want version %q", vout, want)
	}
}

func TestCLI_Binary_ExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e binary test in short mode")
	}
	bin := filepath.Join(t.TempDir(), "zipreport")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-buildvcs=false", "-ldflags", "-X main.version=9.9.9-e2e", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	src := t.TempDir()
	writeCLITemplate(t, src)
	out := filepath.Join(t.TempDir(), "r.zpt")
	if err := exec.Command(bin, "build", src, out).Run(); err != nil {
		t.Fatalf("build command: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("output not created: %v", err)
	}

	var stderr bytes.Buffer
	c := exec.Command(bin, "build", filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "o.zpt"))
	c.Stderr = &stderr
	err := c.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}
	if !strings.Contains(stderr.String(), "build failed") {
		t.Errorf("stderr missing 'build failed': %q", stderr.String())
	}

	vout, err := exec.Command(bin, "version").Output()
	if err != nil {
		t.Fatalf("version command: %v", err)
	}
	if string(vout) != "zipreport version 9.9.9-e2e\n" {
		t.Errorf("unexpected version output: %q", vout)
	}
}

func TestRunBuild_SymlinkedIndexFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	src := t.TempDir()
	writeCLITemplate(t, src)
	if err := os.Rename(filepath.Join(src, "index.html"), filepath.Join(src, "real_index.html")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real_index.html", filepath.Join(src, "index.html")); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "r.zpt")

	_, err := captureStdout(t, func() error { return runBuild(buildCmd, []string{src, out}) })
	if err == nil || !strings.Contains(err.Error(), "index.html") {
		t.Fatalf("err = %v, want mention of index.html", err)
	}
	if _, statErr := os.Stat(out); statErr == nil {
		t.Error("output file was created")
	}
}
