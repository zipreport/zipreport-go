// Package processor provides PDF and MIME rendering backends.
package processor

import "github.com/zipreport/zipreport-go/pkg/report"

// Processor defines the interface for report processors.
type Processor interface {
	// Process processes a report job and returns the result.
	Process(job *report.ReportJob) *report.JobResult
}
