package template

import (
	"fmt"

	"github.com/zipreport/zipreport-go/pkg/report"
)

// ReportFileLoader implements miya's Loader interface for ReportFile.
// It loads templates from the report's file system.
type ReportFileLoader struct {
	report *report.ReportFile
}

// NewReportFileLoader creates a new ReportFileLoader.
func NewReportFileLoader(r *report.ReportFile) *ReportFileLoader {
	return &ReportFileLoader{report: r}
}

// GetSource returns the template source for a given name.
// This implements miya's Loader interface.
func (l *ReportFileLoader) GetSource(name string) (string, error) {
	content, err := l.report.GetBytes(name)
	if err != nil {
		return "", fmt.Errorf("template not found: %s", name)
	}
	return string(content), nil
}

// Exists checks if a template exists.
func (l *ReportFileLoader) Exists(name string) bool {
	return l.report.Exists(name)
}

// IsCached returns whether the template is cached.
// ReportFileLoader doesn't cache, so always returns false.
func (l *ReportFileLoader) IsCached(name string) bool {
	return false
}

// ListTemplates satisfies miya's Loader interface. It reports which of index.html, base.html and layout.html exist; it is not a full listing.
func (l *ReportFileLoader) ListTemplates() ([]string, error) {
	// Return common template files that might exist
	var templates []string
	commonNames := []string{"index.html", "base.html", "layout.html"}
	for _, name := range commonNames {
		if l.Exists(name) {
			templates = append(templates, name)
		}
	}
	return templates, nil
}
