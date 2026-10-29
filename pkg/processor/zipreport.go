package processor

import (
	"context"

	"github.com/zipreport/zipreport-go/pkg/report"
)

// ZipReportProcessor orchestrates PDF rendering using a RenderBackend.
type ZipReportProcessor struct {
	backend RenderBackend
}

// NewZipReportProcessor creates a new ZipReportProcessor with the given backend.
func NewZipReportProcessor(backend RenderBackend) *ZipReportProcessor {
	return &ZipReportProcessor{backend: backend}
}

// Process processes a report job and returns the result.
func (p *ZipReportProcessor) Process(job *report.ReportJob) *report.JobResult {
	return p.ProcessContext(context.Background(), job)
}

// ProcessContext is Process with a context, passed to the backend when it implements ContextRenderBackend.
func (p *ZipReportProcessor) ProcessContext(ctx context.Context, job *report.ReportJob) *report.JobResult {
	// Prepare ZPT data
	zptData, err := job.GetReport().Save()
	if err != nil {
		return report.NewJobResultError("failed to prepare report: " + err.Error())
	}

	// Get job options
	options := job.GetOptions()

	// Delegate to backend
	var pdfData []byte
	if cb, ok := p.backend.(ContextRenderBackend); ok {
		pdfData, err = cb.RenderContext(ctx, zptData, options)
	} else {
		pdfData, err = p.backend.Render(zptData, options)
	}
	if err != nil {
		return report.NewJobResultError("rendering failed: " + err.Error())
	}

	return report.NewJobResult(pdfData)
}

// Backend returns the underlying RenderBackend.
func (p *ZipReportProcessor) Backend() RenderBackend {
	return p.backend
}
