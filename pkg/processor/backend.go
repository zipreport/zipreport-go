package processor

import "context"

// RenderBackend abstracts the PDF rendering mechanism.
// It allows the processor to work with both remote (zipreport-server)
// and future local implementations.
type RenderBackend interface {
	// Render takes a prepared ZPT (with report.html) and job options, returns PDF bytes.
	Render(zptData []byte, options map[string]interface{}) ([]byte, error)
}

// ContextRenderBackend is a RenderBackend whose renders can be cancelled.
// ZipReportProcessor.ProcessContext uses RenderContext when the backend implements it.
type ContextRenderBackend interface {
	RenderBackend
	RenderContext(ctx context.Context, zptData []byte, options map[string]interface{}) ([]byte, error)
}
