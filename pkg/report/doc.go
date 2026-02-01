// Package report provides types and functions for managing ZipReport report files.
//
// A report consists of a manifest (manifest.json), an index template (index.html),
// and optional assets (CSS, JS, images). Reports can be loaded from:
//   - .zpt files (ZIP archives)
//   - Directories containing template files
//
// Loading a report:
//
//	// Auto-detect source type
//	report, err := report.Load("path/to/report.zpt")
//	report, err := report.Load("path/to/template/dir")
//
//	// Explicit loading
//	report, err := report.LoadFile("report.zpt")
//	report, err := report.LoadDir("template/dir")
//
// Creating a rendering job:
//
//	job := report.NewReportJob(report)
//	job.SetPageSize(report.PageA4)
//	job.SetMargins(report.MarginStandard)
//	job.SetLandscape(false)
//	job.UseJSEvent(true)
//
// Building a .zpt file from a directory:
//
//	result := report.Build("source/dir", "output.zpt", nil)
//	if !result.Success() {
//	    log.Fatal(result.Error())
//	}
package report
