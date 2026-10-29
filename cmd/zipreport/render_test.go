package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/report"
)

// writeHostileManifestZpt builds a .zpt whose manifest fields all carry control characters
func writeHostileManifestZpt(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	manifest, err := json.Marshal(map[string]interface{}{
		"title":       "T\x1b[2Jitle",
		"author":      "Au\xe2\x80\xaethor",
		"version":     "1\x07",
		"description": "De\rsc",
		"params":      []string{"p\x1b]0;x\x07"},
		"report":      "out\x1b.html",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), manifest, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "index.html"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "hostile.zpt")
	if res := report.Build(src, out, nil); !res.Success() {
		t.Fatalf("build: %v", res)
	}
	return out
}

func TestRunInfo_EscapesControlCharacters(t *testing.T) {
	zpt := writeHostileManifestZpt(t)
	out, err := captureStdout(t, func() error { return runInfo(infoCmd, []string{zpt}) })
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"\x1b", "\x07", "\r", "\xe2\x80\xae"} {
		if strings.Contains(out, raw) {
			t.Errorf("stdout contains raw %q", raw)
		}
	}
	for _, want := range []string{
		`Title:       T\x1b[2Jitle`,
		`Author:      Au\u202ethor`,
		`Version:     1\a`,
		`Description: De\rsc`,
		`Parameters:  p\x1b]0;x\a`,
		`Output File: out\x1b.html`,
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("stdout missing %q", want)
		}
	}
}

func TestRunList_EscapesControlCharacters(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file names cannot contain control characters")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a\x1b[2J.zpt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error { return runList(listCmd, []string{dir}) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("stdout contains raw %q", "\x1b")
	}
	if want := `  a\x1b[2J.zpt`; !strings.Contains(out, want+"\n") {
		t.Errorf("stdout missing %q", want)
	}
}
