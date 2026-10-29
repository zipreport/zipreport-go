package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zipreport/zipreport-go/pkg/processor"
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

// setRenderFlags assigns the render flag globals and resets them after the test
func setRenderFlags(t *testing.T, data, format, output, subject, from string, to ...string) {
	t.Helper()
	renderData, renderFormat, renderOutput, renderSubject, renderFrom, renderTo = data, format, output, subject, from, to
	t.Cleanup(func() {
		renderData, renderFormat, renderOutput, renderSubject, renderFrom, renderTo = "", "html", "", "", "", nil
	})
}

// writeRenderTemplate writes a template folder and a data file outside of it
func writeRenderTemplate(t *testing.T) (src, dataFile string) {
	t.Helper()
	src = t.TempDir()
	files := map[string]string{
		"manifest.json": `{"title":"Render Test","author":"A","version":"1","description":"D","params":["name"]}`,
		"index.html":    `<p>{{ name }} {{ n > 2 }}</p><img src="a.png">`,
		"a.png":         "PNG",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(src, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	dataFile = filepath.Join(t.TempDir(), "values.json")
	if err := os.WriteFile(dataFile, []byte(`{"name":"<b>World</b>","n":3}`), 0644); err != nil {
		t.Fatal(err)
	}
	return src, dataFile
}

func TestRunRender_HTMLFromFolder(t *testing.T) {
	src, dataFile := writeRenderTemplate(t)
	setRenderFlags(t, dataFile, "html", "", "", "")
	out, err := captureStdout(t, func() error { return runRender(renderCmd, []string{src}) })
	if err != nil {
		t.Fatalf("runRender: %v", err)
	}
	if want := `<p>&lt;b&gt;World&lt;/b&gt; true</p><img src="a.png">`; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if _, err := os.Stat(filepath.Join(src, "report.html")); err == nil {
		t.Errorf("render wrote into the template folder")
	}
}

func TestRunRender_MIMEToFile(t *testing.T) {
	src, dataFile := writeRenderTemplate(t)
	zpt := filepath.Join(t.TempDir(), "r.zpt")
	if res := report.Build(src, zpt, nil); !res.Success() {
		t.Fatalf("build: %v", res)
	}
	mailFile := filepath.Join(t.TempDir(), "mail.eml")
	setRenderFlags(t, dataFile, "mime", mailFile, "Hello", "from@example.com", "a@example.com", `"Doe, John" <j@example.com>`)
	if err := runRender(renderCmd, []string{zpt}); err != nil {
		t.Fatalf("runRender: %v", err)
	}
	raw, err := os.ReadFile(mailFile)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := processor.ParseMIMEMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := msg.Header.Get("Subject"); got != "Hello" {
		t.Errorf("Subject = %q, want Hello", got)
	}
	if got := msg.Header.Get("From"); got != "from@example.com" {
		t.Errorf("From = %q, want from@example.com", got)
	}
	to, err := msg.Header.AddressList("To")
	if err != nil || len(to) != 2 {
		t.Fatalf("To = %v, %v, want 2 addresses", to, err)
	}
	if to[1].Name != "Doe, John" {
		t.Errorf("To[1].Name = %q, want %q", to[1].Name, "Doe, John")
	}
	if !bytes.Contains(raw, []byte("Content-ID: <")) {
		t.Errorf("a.png was not embedded")
	}
}

func TestRunRender_FailureWritesNothing(t *testing.T) {
	src, dataFile := writeRenderTemplate(t)
	outFile := filepath.Join(t.TempDir(), "mail.eml")
	setRenderFlags(t, dataFile, "mime", outFile, "bad\nsubject", "")
	err := runRender(renderCmd, []string{src})
	if err == nil || !strings.Contains(err.Error(), "control character") {
		t.Fatalf("error = %v, want it to contain %q", err, "control character")
	}
	if _, err := os.Stat(outFile); err == nil {
		t.Errorf("output file written despite failure")
	}
}

func TestRunRender_InputErrors(t *testing.T) {
	src, dataFile := writeRenderTemplate(t)
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	missingParam := `missing required parameters: [name]`
	cases := []struct {
		name, data, format, source, want string
	}{
		{"unknown format", dataFile, "pdf", src, `unknown format "pdf"`},
		{"missing data file", filepath.Join(dir, "nope.json"), "html", src, "cannot read data file"},
		{"data not an object", write("arr.json", `[1]`), "html", src, "invalid data file"},
		{"malformed data", write("bad.json", `{`), "html", src, "invalid data file"},
		{"missing param", "", "html", src, missingParam},
		{"empty data file is no data", write("empty.json", "\xef\xbb\xbf \n"), "html", src, missingParam},
		{"null data is no data", write("null.json", `null`), "html", src, missingParam},
		{"missing source", dataFile, "html", filepath.Join(dir, "nope"), "failed to load report"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setRenderFlags(t, c.data, c.format, "", "", "")
			out, err := captureStdout(t, func() error { return runRender(renderCmd, []string{c.source}) })
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %v, want it to contain %q", err, c.want)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty", out)
			}
		})
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "zipreport")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-buildvcs=false", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func TestCLI_Binary_Render(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e binary test in short mode")
	}
	bin := buildTestBinary(t)
	src, dataFile := writeRenderTemplate(t)

	out, err := exec.Command(bin, "render", src, "--data", dataFile, "--format", "mime",
		"--to", `"Doe, John" <j@example.com>`, "--to", "b@example.com").Output()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	msg, err := processor.ParseMIMEMessage(out)
	if err != nil {
		t.Fatal(err)
	}
	to, err := msg.Header.AddressList("To")
	if err != nil || len(to) != 2 {
		t.Fatalf("To = %v, %v, want 2 addresses", to, err)
	}
	if to[0].Name != "Doe, John" || to[1].Address != "b@example.com" {
		t.Errorf("To = %v, want Doe, John and b@example.com", to)
	}

	exitOne := func(t *testing.T, err error) {
		t.Helper()
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %v", err)
		}
	}

	var stderr bytes.Buffer
	c := exec.Command(bin, "render", src)
	c.Stderr = &stderr
	exitOne(t, c.Run())
	if !strings.Contains(stderr.String(), "missing required parameters") {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), "missing required parameters")
	}

	stderr.Reset()
	c = exec.Command(bin, "bild", "x")
	c.Stderr = &stderr
	exitOne(t, c.Run())
	if want := "\n\nDid you mean this?\n\tbuild\n"; !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), want)
	}
}

func TestCLI_Binary_EscapesErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e binary test in short mode")
	}
	bin := buildTestBinary(t)

	write := func(t *testing.T, manifest, index string) string {
		t.Helper()
		src := t.TempDir()
		if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, "index.html"), []byte(index), 0644); err != nil {
			t.Fatal(err)
		}
		return src
	}
	paramManifest, err := json.Marshal(map[string]interface{}{
		"title": "T", "author": "A", "version": "1", "description": "D",
		"params": []string{"n\x1b[2J"},
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct{ name, manifest, index, want string }{
		{"template error", `{"title":"T","author":"A","version":"1","description":"D"}`, "{% extends \"a\x1b[2J.html\" %}", `parent template a\x1b[2J.html:`},
		{"manifest param", string(paramManifest), "x", `[n\x1b[2J]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := write(t, c.manifest, c.index)
			var stderr bytes.Buffer
			cmd := exec.Command(bin, "render", src)
			cmd.Stderr = &stderr
			var ee *exec.ExitError
			if err := cmd.Run(); !errors.As(err, &ee) || ee.ExitCode() != 1 {
				t.Fatalf("expected exit code 1, got %v", err)
			}
			if strings.Contains(stderr.String(), "\x1b") {
				t.Errorf("stderr contains raw %q", "\x1b")
			}
			if !strings.Contains(stderr.String(), c.want) {
				t.Errorf("stderr missing %q: %q", c.want, stderr.String())
			}
		})
	}
}
