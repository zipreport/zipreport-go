package report

import (
	"encoding/json"
	"errors"
	"testing"
)

// On Windows, filepath.IsLocal is the only check that rejects colons (alternate data streams) and reserved device names.
func TestParseManifest_ReportRejectsWindowsOnlyNames(t *testing.T) {
	for _, r := range []string{"report.html:stream", "NUL", "sub/con"} {
		b, _ := json.Marshal(map[string]any{"title": "T", "author": "A", "version": "1", "description": "D", "report": r})
		if _, err := parseManifest(b); !errors.Is(err, ErrInvalidManifest) {
			t.Errorf("report %q: err = %v, want ErrInvalidManifest", r, err)
		}
	}
}
