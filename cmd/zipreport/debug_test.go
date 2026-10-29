package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
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

// writeDebugTemplate writes a minimal valid template into dir
func writeDebugTemplate(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"manifest.json": `{"title":"T","author":"A","version":"1","description":"D"}`,
		"index.html":    `<p>ok</p>`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDebugHandler_SiblingDirNotServed(t *testing.T) {
	parent := t.TempDir()
	tpl := filepath.Join(parent, "tpl")
	writeDebugTemplate(t, tpl)
	secretDir := filepath.Join(parent, "tpl-secret")
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "s.txt"), []byte("TOPSECRET"), 0644); err != nil {
		t.Fatal(err)
	}

	h := &debugHandler{source: tpl}
	req := httptest.NewRequest("GET", "/x", nil)
	req.URL.Path = "/../tpl-secret/s.txt"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "<p>ok</p>") {
		t.Errorf("body = %q, want render fallthrough", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "TOPSECRET") {
		t.Error("sibling folder file was served")
	}
}

func TestDebugHandler_EmptyPath(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)

	h := &debugHandler{source: dir}
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = ""
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestNewDebugServer_Defaults(t *testing.T) {
	f := debugCmd.Flags().Lookup("host")
	if f == nil || f.DefValue != "127.0.0.1" {
		t.Fatalf("host flag = %+v, want default 127.0.0.1", f)
	}
	srv := newDebugServer("127.0.0.1", 8000, "x")
	if srv.Addr != "127.0.0.1:8000" {
		t.Errorf("Addr = %q", srv.Addr)
	}
	if srv.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout not set")
	}
}

func TestNewDebugServer_RejectsForeignHost(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	srv := newDebugServer("127.0.0.1", 8000, dir)

	serve := func(s *http.Server, host string) int {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = host
		rec := httptest.NewRecorder()
		s.Handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := serve(srv, "evil.example:8000"); got != 403 {
		t.Errorf("foreign host status = %d, want 403", got)
	}
	for _, host := range []string{"localhost:8000", "[::1]:8000", "[::1]", "localhost", "LOCALHOST:8000", "127.0.0.2:8000", ""} {
		if got := serve(srv, host); got != 200 {
			t.Errorf("host %q status = %d, want 200", host, got)
		}
	}
	if addr := newDebugServer("[::1]", 8000, dir).Addr; addr != "[::1]:8000" {
		t.Errorf("Addr = %q, want [::1]:8000", addr)
	}
	open := newDebugServer("0.0.0.0", 8000, dir)
	if got := serve(open, "192.168.1.5:8000"); got != 200 {
		t.Errorf("opted-in host status = %d, want 200", got)
	}
}

func TestDebugHandler_InvalidDataJSON(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{bad`), 0644); err != nil {
		t.Fatal(err)
	}
	h := &debugHandler{source: dir}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid data.json") {
		t.Errorf("body = %q, want invalid data.json", rec.Body.String())
	}
}

func TestDebugHandler_HiddenFileNotServed(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "css"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{".env", ".git/config", "css/.secret"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("TOPSECRET"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	h := &debugHandler{source: dir}
	for _, p := range []string{"/.env", "/.git/config", "/css/.secret"} {
		req := httptest.NewRequest("GET", "/x", nil)
		req.URL.Path = p
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("%s: status = %d", p, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "TOPSECRET") {
			t.Errorf("%s: hidden file served", p)
		}
	}
}

func TestRunDebug_UsesHardenedServer(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)

	var got *http.Server
	orig := serveDebug
	serveDebug = func(s *http.Server) error { got = s; return nil }
	oldHost, oldPort := debugHost, debugPort
	debugHost, debugPort = "127.0.0.1", 8000
	t.Cleanup(func() { serveDebug = orig; debugHost, debugPort = oldHost, oldPort })

	if _, err := captureStdout(t, func() error { return runDebug(debugCmd, []string{dir}) }); err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("serveDebug not called")
	}
	if got.Addr != "127.0.0.1:8000" {
		t.Errorf("Addr = %q", got.Addr)
	}
	if got.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout not set")
	}
	serve := func(host string) int {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = host
		rec := httptest.NewRecorder()
		got.Handler.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := serve("evil.example:8000"); c != 403 {
		t.Errorf("foreign host status = %d, want 403", c)
	}
	if c := serve("localhost:8000"); c != 200 {
		t.Errorf("localhost status = %d, want 200", c)
	}
}

func TestNewDebugServer_GuardOnEveryLoopbackBind(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	serve := func(s *http.Server, host string) int {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = host
		rec := httptest.NewRecorder()
		s.Handler.ServeHTTP(rec, req)
		return rec.Code
	}
	for _, bind := range []string{"127.0.0.1", "::1", "[::1]", "localhost"} {
		srv := newDebugServer(bind, 8000, dir)
		if got := serve(srv, "evil.example:8000"); got != 403 {
			t.Errorf("bind %q: foreign host status = %d, want 403", bind, got)
		}
		if got := serve(srv, "localhost:8000"); got != 200 {
			t.Errorf("bind %q: localhost status = %d, want 200", bind, got)
		}
	}
}

func TestDebugHandler_ServesStaticAsset(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "css"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "css", "a.css"), []byte("body{}"), 0644); err != nil {
		t.Fatal(err)
	}
	h := &debugHandler{source: dir}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/css/a.css", nil))
	if rec.Code != 200 || rec.Body.String() != "body{}" {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}

	// A missing asset falls back to the rendered report
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/missing.css", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<p>ok</p>") {
		t.Errorf("missing asset: status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestDebugHandler_LoadFailure(t *testing.T) {
	h := &debugHandler{source: filepath.Join(t.TempDir(), "nope")}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Failed to load report") {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestGetContentType(t *testing.T) {
	tests := map[string]string{
		".html":  "text/html; charset=utf-8",
		".css":   "text/css; charset=utf-8",
		".js":    "application/javascript; charset=utf-8",
		".json":  "application/json; charset=utf-8",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".svg":   "image/svg+xml",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".otf":   "font/otf",
		".pdf":   "application/pdf",
		".xyz":   "application/octet-stream",
		"":       "application/octet-stream",
	}
	for ext, want := range tests {
		if got := getContentType(ext); got != want {
			t.Errorf("getContentType(%q) = %q, want %q", ext, got, want)
		}
	}
}

// debugGet requests p (set verbatim, uncleaned) from a debug handler for source
func debugGet(t *testing.T, source, p string) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/x", nil)
	req.URL.Path = p
	rec := httptest.NewRecorder()
	(&debugHandler{source: source}).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%s: status = %d, body = %q", p, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func TestDebugHandler_IndexSourceNeverServed(t *testing.T) {
	const tmpl = `{{ "REN" ~ "DERED" }}`
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(tmpl), 0644); err != nil {
		t.Fatal(err)
	}

	z := fileutils.NewZipFs()
	if err := z.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false); err != nil {
		t.Fatal(err)
	}
	if err := z.Add("index.html", []byte(tmpl), false); err != nil {
		t.Fatal(err)
	}
	zpt := filepath.Join(t.TempDir(), "r.zpt")
	if err := z.SaveToFile(zpt); err != nil {
		t.Fatal(err)
	}

	for _, source := range []string{dir, zpt} {
		for _, p := range []string{"/index.html", "//index.html", "/./index.html"} {
			body := debugGet(t, source, p)
			if !strings.Contains(body, "RENDERED") || strings.Contains(body, "{{") {
				t.Errorf("%s %s: body = %q, want the rendered report", filepath.Base(source), p, body)
			}
		}
	}
}

func TestDebugHandler_CaseVariantOfIndexNotServed(t *testing.T) {
	dir := t.TempDir()
	writeDebugTemplate(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`{{ "REN" ~ "DERED" }}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "INDEX.HTML")); err != nil {
		t.Skip("folder is case-sensitive")
	}
	if body := debugGet(t, dir, "/INDEX.HTML"); strings.Contains(body, "{{") {
		t.Errorf("/INDEX.HTML: body = %q, want the rendered report", body)
	}
}
