package report

import (
	"encoding/json"
	"io"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
)

// Manifest represents the report manifest (manifest.json).
type Manifest struct {
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Params      []string `json:"params"`
	UseJSEvent  bool     `json:"useJSEvent,omitempty"`
	Report      string   `json:"report,omitempty"`
}

// ReportFile represents a loaded report with its file system and manifest.
type ReportFile struct {
	fs       fileutils.FsInterface
	manifest *Manifest
}

// NewReportFile creates a new ReportFile from a file system and manifest.
func NewReportFile(fs fileutils.FsInterface, manifest *Manifest) *ReportFile {
	return &ReportFile{
		fs:       fs,
		manifest: manifest,
	}
}

// GetManifest returns the report manifest.
func (r *ReportFile) GetManifest() *Manifest {
	return r.manifest
}

// GetFs returns the underlying file system.
func (r *ReportFile) GetFs() fileutils.FsInterface {
	return r.fs
}

// GetParam retrieves a manifest field by name with a default value.
func (r *ReportFile) GetParam(name string, defaultVal interface{}) interface{} {
	switch name {
	case "title":
		if r.manifest.Title != "" {
			return r.manifest.Title
		}
	case "author":
		if r.manifest.Author != "" {
			return r.manifest.Author
		}
	case "version":
		if r.manifest.Version != "" {
			return r.manifest.Version
		}
	case "description":
		if r.manifest.Description != "" {
			return r.manifest.Description
		}
	case "params":
		if r.manifest.Params != nil {
			return r.manifest.Params
		}
	case "useJSEvent":
		return r.manifest.UseJSEvent
	case "report":
		if r.manifest.Report != "" {
			return r.manifest.Report
		}
	}
	return defaultVal
}

// Exists checks if a file exists in the report.
func (r *ReportFile) Exists(path string) bool {
	return r.fs.Exists(path)
}

// Get retrieves a file's content as a ReadCloser.
func (r *ReportFile) Get(name string) (io.ReadCloser, error) {
	return r.fs.Get(name)
}

// GetBytes retrieves a file's content as bytes.
func (r *ReportFile) GetBytes(name string) ([]byte, error) {
	return r.fs.GetBytes(name)
}

// Add adds or updates a file in the report.
func (r *ReportFile) Add(name string, content []byte, overwrite bool) error {
	return r.fs.Add(name, content, overwrite)
}

// Save exports the report as ZIP bytes.
func (r *ReportFile) Save() ([]byte, error) {
	// Type assertion to check if fs is a ZipFs
	if zfs, ok := r.fs.(*fileutils.ZipFs); ok {
		return zfs.Save()
	}

	// For DiskFs or other implementations, we need to create a new ZipFs
	// and copy all files into it
	return r.exportToZip()
}

// exportToZip creates a ZIP from any FsInterface implementation.
func (r *ReportFile) exportToZip() ([]byte, error) {
	zfs := fileutils.NewZipFs()

	// Copy manifest
	manifestData, err := json.MarshalIndent(r.manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := zfs.Add(FileManifest, manifestData, true); err != nil {
		return nil, err
	}

	// If the underlying fs is a DiskFs, we need to walk the directory
	// For simplicity, we'll copy known files and any that the manifest references
	filesToCopy := []string{FileIndex}
	if r.manifest.Report != "" && r.manifest.Report != FileReport {
		filesToCopy = append(filesToCopy, r.manifest.Report)
	}

	// Copy index.html
	for _, filename := range filesToCopy {
		if r.fs.Exists(filename) {
			content, err := r.fs.GetBytes(filename)
			if err != nil {
				continue
			}
			_ = zfs.Add(filename, content, true)
		}
	}

	// Copy report.html if it exists
	if r.fs.Exists(FileReport) {
		content, err := r.fs.GetBytes(FileReport)
		if err == nil {
			_ = zfs.Add(FileReport, content, true)
		}
	}

	// Copy data.json if it exists
	if r.fs.Exists(FileData) {
		content, err := r.fs.GetBytes(FileData)
		if err == nil {
			_ = zfs.Add(FileData, content, true)
		}
	}

	return zfs.Save()
}

// Title returns the report title.
func (r *ReportFile) Title() string {
	return r.manifest.Title
}

// Author returns the report author.
func (r *ReportFile) Author() string {
	return r.manifest.Author
}

// Version returns the report version.
func (r *ReportFile) Version() string {
	return r.manifest.Version
}

// Description returns the report description.
func (r *ReportFile) Description() string {
	return r.manifest.Description
}

// Params returns the expected template parameters.
func (r *ReportFile) Params() []string {
	return r.manifest.Params
}

// UseJSEvent returns whether the report uses JavaScript events.
func (r *ReportFile) UseJSEvent() bool {
	return r.manifest.UseJSEvent
}

// ReportFilename returns the output report filename (defaults to report.html).
func (r *ReportFile) ReportFilename() string {
	if r.manifest.Report != "" {
		return r.manifest.Report
	}
	return FileReport
}
