package template

import (
	"errors"
	"runtime/debug"
	"strings"
	"sync"
	"testing"

	"github.com/zipreport/miya"
	miyaruntime "github.com/zipreport/miya/runtime"
	"github.com/zipreport/zipreport-go/pkg/fileutils"
	"github.com/zipreport/zipreport-go/pkg/report"
)

func createTestReportForRender() *report.ReportFile {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("<html><body>{{ name }}</body></html>"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"Test","author":"A","version":"1","description":"D","params":["name"]}`), false)

	manifest := &report.Manifest{
		Title:       "Test",
		Author:      "Author",
		Version:     "1.0",
		Description: "Desc",
		Params:      []string{"name"},
	}

	return report.NewReportFile(zfs, manifest)
}

func TestDefaultRenderOptions(t *testing.T) {
	opts := DefaultRenderOptions()

	if !opts.StrictParams {
		t.Error("StrictParams should be true by default")
	}
	if !opts.AutoEscape {
		t.Error("AutoEscape should be true by default")
	}
	if !opts.TrimBlocks {
		t.Error("TrimBlocks should be true by default")
	}
	if !opts.LstripBlocks {
		t.Error("LstripBlocks should be true by default")
	}
}

func TestNewMiyaRender(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r)

	if mr == nil {
		t.Fatal("NewMiyaRender returned nil")
	}
	if mr.env == nil {
		t.Error("Environment is nil")
	}
	if mr.report != r {
		t.Error("Report not set correctly")
	}
}

func TestNewMiyaRender_WithOptions(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r,
		WithStrictParams(false),
		WithAutoEscape(false),
	)

	if mr.options.StrictParams {
		t.Error("StrictParams should be false")
	}
	if mr.options.AutoEscape {
		t.Error("AutoEscape should be false")
	}
}

func TestMiyaRender_GetEnvironment(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r)

	env := mr.GetEnvironment()
	if env == nil {
		t.Error("GetEnvironment returned nil")
	}
}

func TestMiyaRender_Render(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r)

	output, err := mr.Render(map[string]interface{}{
		"name": "World",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(output, "World") {
		t.Errorf("Output should contain 'World': %s", output)
	}
}

func TestMiyaRender_Render_StrictParams(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r, WithStrictParams(true))

	// Missing required param should fail
	_, err := mr.Render(map[string]interface{}{})
	if err == nil {
		t.Error("Render should fail with missing required param")
	}
	if !strings.Contains(err.Error(), "missing required parameters") {
		t.Errorf("Error should mention missing parameters: %v", err)
	}
}

func TestMiyaRender_Render_NoStrictParams(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r, WithStrictParams(false))

	// Missing required param should not fail
	_, err := mr.Render(map[string]interface{}{})
	if err != nil {
		t.Errorf("Render should not fail with StrictParams=false: %v", err)
	}
}

func TestMiyaRender_RenderTemplate(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("{{ greeting }}"), false)
	_ = zfs.Add("other.html", []byte("Other: {{ value }}"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "T",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}
	r := report.NewReportFile(zfs, manifest)

	mr := NewMiyaRender(r)

	output, err := mr.RenderTemplate("other.html", map[string]interface{}{
		"value": "Test",
	})
	if err != nil {
		t.Fatalf("RenderTemplate failed: %v", err)
	}

	if !strings.Contains(output, "Other: Test") {
		t.Errorf("Output should contain 'Other: Test': %s", output)
	}
}

func TestMiyaRender_RenderString(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r)

	output, err := mr.RenderString("Hello, {{ name }}!", map[string]interface{}{
		"name": "World",
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if output != "Hello, World!" {
		t.Errorf("Output = %q, want %q", output, "Hello, World!")
	}
}

func TestMiyaRender_WithWrapper(t *testing.T) {
	r := createTestReportForRender()

	wrapper := FuncEnvironmentWrapper(func(env *miya.Environment) {
		env.AddGlobal("custom_var", "CustomValue")
	})

	mr := NewMiyaRender(r, WithWrapper(wrapper))

	output, err := mr.RenderString("{{ custom_var }}", nil)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if output != "CustomValue" {
		t.Errorf("Output = %q, want %q", output, "CustomValue")
	}
}

func TestMiyaRender_SetWrapper(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r)

	wrapper := FuncEnvironmentWrapper(func(env *miya.Environment) {
		env.AddGlobal("test", "value")
	})
	mr.SetWrapper(wrapper)

	output, err := mr.RenderString("{{ test }}", nil)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if output != "value" {
		t.Errorf("Output = %q, want %q", output, "value")
	}
}

func TestMiyaRender_LoadDefaultData(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("{{ name }} - {{ extra }}"), false)
	_ = zfs.Add("data.json", []byte(`{"name": "FromJSON", "extra": "ExtraValue"}`), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "T",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}
	r := report.NewReportFile(zfs, manifest)
	mr := NewMiyaRender(r)

	// Without providing data, should use data.json
	output, err := mr.Render(nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(output, "FromJSON") {
		t.Errorf("Output should contain data from data.json: %s", output)
	}
}

func TestMiyaRender_ProvidedDataOverridesDefault(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("{{ name }}"), false)
	_ = zfs.Add("data.json", []byte(`{"name": "FromJSON"}`), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "T",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}
	r := report.NewReportFile(zfs, manifest)
	mr := NewMiyaRender(r)

	// Provided data should override data.json
	output, err := mr.Render(map[string]interface{}{
		"name": "Provided",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(output, "Provided") {
		t.Errorf("Output should contain provided data: %s", output)
	}
}

func TestMiyaRender_AutoEscape(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("{{ content }}"), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	manifest := &report.Manifest{
		Title:       "T",
		Author:      "A",
		Version:     "1",
		Description: "D",
	}
	r := report.NewReportFile(zfs, manifest)

	// With AutoEscape enabled
	mr := NewMiyaRender(r, WithAutoEscape(true))
	output, _ := mr.Render(map[string]interface{}{
		"content": "<script>alert('xss')</script>",
	})

	if strings.Contains(output, "<script>") {
		t.Error("AutoEscape should escape HTML")
	}
}

func TestMiyaRender_TemplateNotFound(t *testing.T) {
	r := createTestReportForRender()
	mr := NewMiyaRender(r)

	_, err := mr.RenderTemplate("nonexistent.html", nil)
	if err == nil {
		t.Error("RenderTemplate should fail for non-existent template")
	}
}

func TestMiyaRender_DoesNotMutateCallerData(t *testing.T) {
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("{{ name }} {{ extra }}"), false)
	_ = zfs.Add("data.json", []byte(`{"extra":"fromjson"}`), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)

	r := report.NewReportFile(zfs, &report.Manifest{Title: "T", Author: "A", Version: "1", Description: "D"})
	mr := NewMiyaRender(r)

	data := map[string]interface{}{"name": "n"}
	output, err := mr.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(output, "fromjson") {
		t.Errorf("output should contain data.json value: %s", output)
	}
	if _, ok := data["extra"]; ok || len(data) != 1 {
		t.Errorf("caller's data was modified: %v", data)
	}
}

func newDataJSONRender(t *testing.T, dataJSON string) *MiyaRender {
	t.Helper()
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte("{{ name }} - {{ extra }}"), false)
	_ = zfs.Add("data.json", []byte(dataJSON), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)
	manifest := &report.Manifest{Title: "T", Author: "A", Version: "1", Description: "D"}
	return NewMiyaRender(report.NewReportFile(zfs, manifest))
}

func TestMiyaRender_InvalidDataJSONFails(t *testing.T) {
	mr := newDataJSONRender(t, `{"name": `)
	_, err := mr.Render(nil)
	if err == nil || !strings.Contains(err.Error(), "invalid data.json") {
		t.Fatalf("expected invalid data.json error, got %v", err)
	}
}

func TestMiyaRender_EmptyDataJSONAllowed(t *testing.T) {
	for _, content := range []string{"", "  \n", "null"} {
		mr := newDataJSONRender(t, content)
		output, err := mr.Render(map[string]interface{}{"name": "n", "extra": "e"})
		if err != nil {
			t.Fatalf("data.json %q: Render failed: %v", content, err)
		}
		if !strings.Contains(output, "n - e") {
			t.Errorf("data.json %q: unexpected output %q", content, output)
		}
	}
}

func TestMiyaRender_DataJSONWithBOM(t *testing.T) {
	mr := newDataJSONRender(t, "\xef\xbb\xbf{\"name\":\"bom\",\"extra\":\"x\"}")
	output, err := mr.Render(nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(output, "bom") {
		t.Errorf("output should contain BOM data: %q", output)
	}
}

// Also pins that filterError is checked before the single-use claim.
func TestMiyaRender_FilterErrorSurfaced(t *testing.T) {
	mr := NewMiyaRender(createTestReportForRender())
	mr.filterError = errors.New("boom")
	if _, err := mr.Render(map[string]interface{}{"name": "x"}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Render err = %v, want boom", err)
	}
	if _, err := mr.RenderString("{{ 1 }}", nil); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("RenderString err = %v, want boom", err)
	}
}

func TestMiyaRender_SecondRenderFails(t *testing.T) {
	mr := NewMiyaRender(createTestReportForRender())
	out, err := mr.Render(map[string]interface{}{"name": "First"})
	if err != nil || out != "<html><body>First</body></html>" {
		t.Fatalf("first Render = %q, %v", out, err)
	}
	out, err = mr.Render(map[string]interface{}{"name": "Second"})
	if out != "" || !errors.Is(err, ErrRendererUsed) {
		t.Errorf("second Render = %q, %v; want empty and ErrRendererUsed", out, err)
	}
}

func TestMiyaRender_RenderStringAfterRenderFails(t *testing.T) {
	mr := NewMiyaRender(createTestReportForRender())
	if _, err := mr.Render(map[string]interface{}{"name": "x"}); err != nil {
		t.Fatalf("first Render: %v", err)
	}
	if _, err := mr.RenderString("{{ 1 }}", nil); !errors.Is(err, ErrRendererUsed) {
		t.Errorf("RenderString err = %v, want ErrRendererUsed", err)
	}
}

func TestMiyaRender_RenderTemplateAfterRenderStringFails(t *testing.T) {
	mr := NewMiyaRender(createTestReportForRender())
	if _, err := mr.RenderString("x", nil); err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if _, err := mr.RenderTemplate("index.html", map[string]interface{}{"name": "a"}); !errors.Is(err, ErrRendererUsed) {
		t.Errorf("RenderTemplate err = %v, want ErrRendererUsed", err)
	}
}

func TestMiyaRender_FailedRenderStillConsumes(t *testing.T) {
	mr := NewMiyaRender(createTestReportForRender(), WithStrictParams(true))
	if _, err := mr.Render(nil); err == nil {
		t.Fatal("expected missing-parameter error")
	}
	if _, err := mr.Render(map[string]interface{}{"name": "x"}); !errors.Is(err, ErrRendererUsed) {
		t.Errorf("retry err = %v, want ErrRendererUsed", err)
	}
}

func TestMiyaRender_ConcurrentRendersOneWins(t *testing.T) {
	mr := NewMiyaRender(createTestReportForRender())
	const n = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins, used := 0, 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			out, err := mr.Render(map[string]interface{}{"name": "x"})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && out == "<html><body>x</body></html>":
				wins++
			case errors.Is(err, ErrRendererUsed):
				used++
			default:
				t.Errorf("unexpected result %q, %v", out, err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if wins != 1 || used != n-1 {
		t.Errorf("wins=%d used=%d, want 1 and %d", wins, used, n-1)
	}
}

func renderOnZip(t *testing.T, index, dataJSON string, data map[string]interface{}) string {
	t.Helper()
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte(index), false)
	if dataJSON != "" {
		_ = zfs.Add("data.json", []byte(dataJSON), false)
	}
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)
	manifest := &report.Manifest{Title: "T", Author: "A", Version: "1", Description: "D"}
	out, err := NewMiyaRender(report.NewReportFile(zfs, manifest)).Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	return out
}

func TestMiyaRender_DataJSONNumbersPlain(t *testing.T) {
	out := renderOnZip(t,
		`Invoice {{ inv }} year {{ year }} total {{ total }} pi {{ pi }}`,
		`{"inv": 10023, "year": 2024, "total": 1234.5, "pi": 3.14159}`, nil)
	want := "Invoice 10023 year 2024 total 1234.5 pi 3.14159"
	if out != want {
		t.Errorf("Output = %q, want %q", out, want)
	}
}

func TestMiyaRender_DataJSONComparisons(t *testing.T) {
	out := renderOnZip(t,
		`{% if zero > 0 %}T{% else %}F{% endif %}{% if (one - 1) > 0 %}T{% else %}F{% endif %}{% if zero == 0 %}T{% else %}F{% endif %}{% if 0 in [zero] %}T{% else %}F{% endif %}`,
		`{"zero": 0, "one": 1}`, nil)
	if out != "FFTT" {
		t.Errorf("Output = %q, want %q", out, "FFTT")
	}
}

func TestMiyaRender_GoTypedComparisons(t *testing.T) {
	out := renderOnZip(t,
		`{% if u > 0 %}T{% else %}F{% endif %}{% if f == 0 %}T{% else %}F{% endif %}`,
		"", map[string]interface{}{"u": uint(0), "f": float32(0)})
	if out != "FT" {
		t.Errorf("Output = %q, want %q", out, "FT")
	}
}

func TestMiyaRender_SelfIncludeErrors(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(64 << 20))
	zfs := fileutils.NewZipFs()
	_ = zfs.Add("index.html", []byte(`x{% include "index.html" %}`), false)
	_ = zfs.Add("manifest.json", []byte(`{"title":"T","author":"A","version":"1","description":"D"}`), false)
	manifest := &report.Manifest{Title: "T", Author: "A", Version: "1", Description: "D"}
	_, err := NewMiyaRender(report.NewReportFile(zfs, manifest)).Render(nil)
	if !errors.Is(err, miyaruntime.ErrMaxRecursion) {
		t.Errorf("err = %v, want ErrMaxRecursion", err)
	}
}
