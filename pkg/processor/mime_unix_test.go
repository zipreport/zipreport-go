//go:build unix

package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/report"
)

func TestMIMEProcessor_SymlinkToHiddenNotEmbedded(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"manifest.json": `{"title":"T","author":"A","version":"1","description":"D","report":"sub/out.html"}`,
		"index.html":    "x",
		".env":          "SECRET1",
		".git/config":   "SECRET2",
		"img/real.png":  "PNG",
	}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for link, target := range map[string]string{"config.txt": ".env", "assets": ".git", "logo.png": "img/real.png"} {
		if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
			t.Fatal(err)
		}
	}
	r, err := report.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	rendered := r.Clone()
	html := `<img src="../config.txt"><img src="../assets/config"><img src="../logo.png"><img src="../img/real.png">`
	if err := rendered.Add("sub/out.html", []byte(html), false); err != nil {
		t.Fatal(err)
	}

	result := processMIME(rendered)
	if !result.Success {
		t.Fatalf("Process failed: %s", result.Error)
	}
	parts := mimeParts(t, result.Report)
	if n := len(parts); n != 3 {
		t.Errorf("parts = %d, want 3", n)
	}
	for k, v := range parts {
		if strings.Contains(string(v), "SECRET") {
			t.Errorf("%s embeds a hidden file", k)
		}
		if k != "html" && string(v) != "PNG" {
			t.Errorf("part %s = %q, want PNG", k, v)
		}
	}
}
