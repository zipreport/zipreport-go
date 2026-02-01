package report

import "fmt"

// Margins represents custom page margins in millimeters.
type Margins struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}

// ReportJob represents a rendering job configuration.
type ReportJob struct {
	report        *ReportFile
	pageSize      PageSize
	margins       MarginStyle
	customMargins *Margins
	landscape     bool
	settlingTime  int  // milliseconds
	jsTimeout     int  // seconds
	jobTimeout    int  // seconds
	useJSEvent    bool
}

// NewReportJob creates a new ReportJob with default settings.
func NewReportJob(report *ReportFile) *ReportJob {
	job := &ReportJob{
		report:       report,
		pageSize:     DefaultPageSize,
		margins:      DefaultMarginStyle,
		landscape:    false,
		settlingTime: DefaultSettlingTime,
		jsTimeout:    DefaultJSTimeout,
		jobTimeout:   DefaultJobTimeout,
		useJSEvent:   false,
	}

	// Inherit useJSEvent from manifest if set
	if report.UseJSEvent() {
		job.useJSEvent = true
	}

	return job
}

// GetReport returns the underlying ReportFile.
func (j *ReportJob) GetReport() *ReportFile {
	return j.report
}

// SetPageSize sets the page size.
func (j *ReportJob) SetPageSize(size PageSize) *ReportJob {
	j.pageSize = size
	return j
}

// SetMargins sets the margin style.
func (j *ReportJob) SetMargins(style MarginStyle) *ReportJob {
	j.margins = style
	j.customMargins = nil
	return j
}

// SetCustomMargins sets custom margins in millimeters.
func (j *ReportJob) SetCustomMargins(margins *Margins) *ReportJob {
	j.customMargins = margins
	return j
}

// SetLandscape sets the page orientation.
func (j *ReportJob) SetLandscape(landscape bool) *ReportJob {
	j.landscape = landscape
	return j
}

// SetSettlingTime sets the settling time in milliseconds.
func (j *ReportJob) SetSettlingTime(ms int) *ReportJob {
	j.settlingTime = ms
	return j
}

// SetJSTimeout sets the JavaScript timeout in seconds.
func (j *ReportJob) SetJSTimeout(seconds int) *ReportJob {
	j.jsTimeout = seconds
	return j
}

// SetJobTimeout sets the total job timeout in seconds.
func (j *ReportJob) SetJobTimeout(seconds int) *ReportJob {
	j.jobTimeout = seconds
	return j
}

// UseJSEvent sets whether to wait for a JavaScript event.
func (j *ReportJob) UseJSEvent(use bool) *ReportJob {
	j.useJSEvent = use
	return j
}

// GetOptions returns the job configuration as a map.
// Option names match the zipreport-server API.
func (j *ReportJob) GetOptions() map[string]interface{} {
	options := map[string]interface{}{
		"page_size":     string(j.pageSize),
		"script":        j.report.ReportFilename(),
		"landscape":     j.landscape,
		"settling_time": j.settlingTime,
		"timeout_js":    j.jsTimeout,
		"timeout_job":   j.jobTimeout,
		"js_event":      j.useJSEvent,
	}

	if j.customMargins != nil {
		options["margins"] = "custom"
		options["margin_left"] = fmt.Sprintf("%.1f", j.customMargins.Left)
		options["margin_right"] = fmt.Sprintf("%.1f", j.customMargins.Right)
		options["margin_top"] = fmt.Sprintf("%.1f", j.customMargins.Top)
		options["margin_bottom"] = fmt.Sprintf("%.1f", j.customMargins.Bottom)
	} else {
		options["margins"] = string(j.margins)
		options["margin_left"] = "0.0"
		options["margin_right"] = "0.0"
		options["margin_top"] = "0.0"
		options["margin_bottom"] = "0.0"
	}

	return options
}

// PageSize returns the configured page size.
func (j *ReportJob) PageSize() PageSize {
	return j.pageSize
}

// Landscape returns whether landscape orientation is set.
func (j *ReportJob) Landscape() bool {
	return j.landscape
}

// JobResult represents the result of a rendering job.
type JobResult struct {
	Report  []byte
	Success bool
	Error   string
}

// NewJobResult creates a successful JobResult.
func NewJobResult(report []byte) *JobResult {
	return &JobResult{
		Report:  report,
		Success: true,
	}
}

// NewJobResultError creates a failed JobResult.
func NewJobResultError(err string) *JobResult {
	return &JobResult{
		Success: false,
		Error:   err,
	}
}

// NewJobResultFromError creates a failed JobResult from an error.
func NewJobResultFromError(err error) *JobResult {
	return &JobResult{
		Success: false,
		Error:   err.Error(),
	}
}
