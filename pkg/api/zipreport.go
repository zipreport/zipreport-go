// Package api provides the high-level API for ZipReport.
package api

import (
	"github.com/zipreport/zipreport-go/pkg/processor"
	"github.com/zipreport/zipreport-go/pkg/report"
	"github.com/zipreport/zipreport-go/pkg/template"
)

// BaseReport provides common functionality for report generation.
type BaseReport struct {
	processor processor.Processor
}

// CreateJob creates a ReportJob from a ReportFile.
func (b *BaseReport) CreateJob(zpt *report.ReportFile) *report.ReportJob {
	return report.NewReportJob(zpt)
}

// Render renders a report job with the given data.
func (b *BaseReport) Render(job *report.ReportJob, data map[string]interface{}, wrapper template.EnvironmentWrapper) *report.JobResult {
	// Create renderer
	var opts []template.RenderOption
	if wrapper != nil {
		opts = append(opts, template.WithWrapper(wrapper))
	}
	renderer := template.NewMiyaRender(job.GetReport(), opts...)

	// Render template
	if _, err := renderer.Render(data); err != nil {
		return report.NewJobResultError("template rendering failed: " + err.Error())
	}

	// Process with backend
	return b.processor.Process(job)
}

// RenderDefaults renders a report with default job settings.
func (b *BaseReport) RenderDefaults(zpt *report.ReportFile, data map[string]interface{}, wrapper template.EnvironmentWrapper) *report.JobResult {
	return b.Render(b.CreateJob(zpt), data, wrapper)
}

// ZipReport provides PDF generation via zipreport-server.
type ZipReport struct {
	BaseReport
}

// NewZipReport creates a ZipReport using zipreport-server.
func NewZipReport(url, apiKey string, opts ...processor.ServerOption) *ZipReport {
	backend := processor.NewServerBackend(url, apiKey, opts...)
	return &ZipReport{
		BaseReport: BaseReport{
			processor: processor.NewZipReportProcessor(backend),
		},
	}
}

// NewZipReportWithBackend creates a ZipReport with a custom backend.
// This allows using alternative backends (e.g., LocalBackend in the future).
func NewZipReportWithBackend(backend processor.RenderBackend) *ZipReport {
	return &ZipReport{
		BaseReport: BaseReport{
			processor: processor.NewZipReportProcessor(backend),
		},
	}
}

// MIMEReport provides MIME email generation.
type MIMEReport struct {
	BaseReport
}

// NewMIMEReport creates a MIMEReport.
func NewMIMEReport(opts ...processor.MIMEOption) *MIMEReport {
	return &MIMEReport{
		BaseReport: BaseReport{
			processor: processor.NewMIMEProcessor(opts...),
		},
	}
}

// Quick helper functions for common operations

// RenderPDF renders a report to PDF using zipreport-server.
func RenderPDF(zpt *report.ReportFile, data map[string]interface{}, serverURL, apiKey string) *report.JobResult {
	client := NewZipReport(serverURL, apiKey)
	return client.RenderDefaults(zpt, data, nil)
}

// RenderMIME renders a report to a MIME email message.
func RenderMIME(zpt *report.ReportFile, data map[string]interface{}) *report.JobResult {
	client := NewMIMEReport()
	return client.RenderDefaults(zpt, data, nil)
}
