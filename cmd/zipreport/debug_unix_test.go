//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebugHandler_SymlinkToHiddenNotServed(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	for f, content := range map[string]string{".env": "TOPSECRET", ".git/config": "TOPSECRET", "a.css": "body{}"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for link, target := range map[string]string{"config.txt": ".env", "assets": ".git", "style.css": "a.css"} {
		if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"/config.txt", "/assets/config"} {
		if strings.Contains(debugGet(t, dir, p), "TOPSECRET") {
			t.Errorf("%s: hidden file served through a symlink", p)
		}
	}
	if body := debugGet(t, dir, "/style.css"); body != "body{}" {
		t.Errorf("/style.css: body = %q, want %q", body, "body{}")
	}
}
