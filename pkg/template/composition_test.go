package template

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	miyaruntime "github.com/zipreport/miya/runtime"
	"github.com/zipreport/zipreport-go/pkg/fileutils"
	"github.com/zipreport/zipreport-go/pkg/report"
)

var compositionFiles = map[string]string{
	"manifest.json":        `{"title":"T","author":"A","version":"1","description":"D"}`,
	"p.html":               "WRONG",
	"base.html":            "<main>{% block body %}base{% endblock %}</main>",
	"layouts/outer.html":   "<main>\n{% block head %}H{% endblock %}\n{% block body %}outer{% endblock %}\n</main>",
	"layouts/inner.html":   "{% extends 'layouts/outer.html' %}{% block body %}[inner {{ super() }}]{% endblock %}",
	"macros/m.html":        "{% macro tag(x) %}<b>{{ x }}</b>{% endmacro %}",
	"partials/p.html":      "<i>{{ name }}</i>",
	"partials/nested.html": `[{% include "partials/p.html" %}]`,
}

func compositionReport(t *testing.T, kind, index string) *report.ReportFile {
	t.Helper()
	var r *report.ReportFile
	switch kind {
	case "zip", "zipclone":
		zfs := fileutils.NewZipFs()
		for name, content := range compositionFiles {
			if err := zfs.Add(name, []byte(content), false); err != nil {
				t.Fatal(err)
			}
		}
		if err := zfs.Add("index.html", []byte(index), false); err != nil {
			t.Fatal(err)
		}
		b, err := zfs.Save()
		if err != nil {
			t.Fatal(err)
		}
		r, err = report.LoadBytes(b)
		if err != nil {
			t.Fatal(err)
		}
	case "dir", "dirclone":
		parent := t.TempDir()
		if err := os.WriteFile(filepath.Join(parent, "outside.html"), []byte("OUTSIDE"), 0o644); err != nil {
			t.Fatal(err)
		}
		tpl := filepath.Join(parent, "tpl")
		files := map[string]string{"index.html": index}
		for k, v := range compositionFiles {
			files[k] = v
		}
		for name, content := range files {
			p := filepath.Join(tpl, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if runtime.GOOS != "windows" {
			if err := os.Symlink("../../outside.html", filepath.Join(tpl, "partials", "link.html")); err != nil {
				t.Fatal(err)
			}
		}
		var err error
		r, err = report.LoadDir(tpl)
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown kind %q", kind)
	}
	if strings.HasSuffix(kind, "clone") {
		r = r.Clone()
	}
	return r
}

func TestTemplateComposition(t *testing.T) {
	cases := []struct {
		name    string
		index   string
		want    string
		errName string
	}{
		{"extends", `{% extends "base.html" %}{% block body %}child {{ name }}{% endblock %}`, "<main>child N&amp;</main>", ""},
		{"extends_deep", "{% extends 'layouts/inner.html' %}\n{% block head %}{% include 'partials/p.html' %}{% endblock %}", "<main>\n<i>N&amp;</i>[inner outer]</main>", ""},
		{"import", `{% import "macros/m.html" as m %}{{ m.tag(name)|safe }}`, "<b>N&amp;</b>", ""},
		{"from_import", `{% from "macros/m.html" import tag %}{{ tag(name)|safe }}`, "<b>N&amp;</b>", ""},
		{"nested_include", `{% include "partials/nested.html" %}`, "[<i>N&amp;</i>]", ""},
		{"leading_slash", `{% include "/partials/p.html" %}`, "<i>N&amp;</i>", ""},
		{"dot_slash", `{% include "./partials/p.html" %}`, "<i>N&amp;</i>", ""},
		{"missing", `{% include "nope.html" %}`, "", "nope.html"},
		{"outside", `{% include "../outside.html" %}`, "", "../outside.html"},
		{"symlink_outside", `{% include "partials/link.html" %}`, "", "partials/link.html"},
	}
	for _, c := range cases {
		for _, kind := range []string{"zip", "zipclone", "dir", "dirclone"} {
			if c.name == "symlink_outside" && (runtime.GOOS == "windows" || strings.HasPrefix(kind, "zip")) {
				continue
			}
			t.Run(c.name+"/"+kind, func(t *testing.T) {
				r := compositionReport(t, kind, c.index)
				out, err := NewMiyaRender(r).Render(map[string]interface{}{"name": "N&"})
				if c.errName == "" {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if out != c.want {
						t.Fatalf("output = %q, want %q", out, c.want)
					}
					return
				}
				if strings.Contains(out, "OUTSIDE") {
					t.Fatalf("output leaked outside file: %q", out)
				}
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "not found") || !strings.Contains(err.Error(), c.errName) {
					t.Fatalf("error = %v, want 'not found' and %q", err, c.errName)
				}
				if out != "" {
					t.Fatalf("output = %q, want empty", out)
				}
			})
		}
	}
}

func TestTemplateCompositionSelfInclude(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(64 << 20))
	for _, kind := range []string{"zip", "zipclone", "dir", "dirclone"} {
		t.Run(kind, func(t *testing.T) {
			r := compositionReport(t, kind, "x{% include \"index.html\" %}")
			_, err := NewMiyaRender(r).Render(nil)
			if !errors.Is(err, miyaruntime.ErrMaxRecursion) {
				t.Errorf("err = %v, want ErrMaxRecursion", err)
			}
		})
	}
}
