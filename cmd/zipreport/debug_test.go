package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebugHandler_DoesNotWriteToSource(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"manifest.json": `{"title":"T","author":"A","version":"1","description":"D"}`,
		"index.html":    `<p>{{ name }}</p>`,
		"data.json":     `{"name":"Bob"}`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	h := &debugHandler{source: dir}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Bob") {
		t.Errorf("body = %q, want to contain Bob", rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "report.html")); !os.IsNotExist(err) {
		t.Errorf("report.html written to the source folder (err=%v)", err)
	}
}
