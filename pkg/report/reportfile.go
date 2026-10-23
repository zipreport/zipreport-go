package report

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

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

// Clone returns a copy of the report whose writes (such as rendering output and
// generated images) go to a private in-memory layer. Reads fall through to the
// original file system, which is never modified.
func (r *ReportFile) Clone() *ReportFile {
	manifest := *r.manifest
	manifest.Params = slices.Clone(r.manifest.Params)
	return NewReportFile(fileutils.NewOverlayFs(r.fs), &manifest)
}

// snapshot materialises fs into a new ZipFs holding every file a renderer needs.
func snapshot(fs fileutils.FsInterface, manifest *Manifest) (*fileutils.ZipFs, error) {
	switch src := fs.(type) {
	case *fileutils.ZipFs:
		return src.Clone(), nil
	case *fileutils.DiskFs:
		return src.Snapshot()
	case *fileutils.OverlayFs:
		zfs, err := snapshot(src.Base(), manifest)
		if err != nil {
			return nil, err
		}
		overlay := src.Overlay()
		for _, name := range overlay.Files() {
			content, err := overlay.GetBytes(name)
			if err != nil {
				return nil, err
			}
			if err := zfs.Add(name, content, true); err != nil {
				return nil, err
			}
		}
		return zfs, nil
	default:
		return snapshotGeneric(fs, manifest)
	}
}

// maxCopyDepth bounds folder nesting when copying a custom file system.
const maxCopyDepth = 32

// snapshotGeneric copies a custom FsInterface by listing it, then adds any of the
// report's known files that listing did not provide.
func snapshotGeneric(fs fileutils.FsInterface, manifest *Manifest) (*fileutils.ZipFs, error) {
	zfs := fileutils.NewZipFs()
	for _, root := range []string{"", "."} {
		if _, err := fs.List(root); err == nil {
			if err := copyTree(fs, zfs, root, "", 0); err != nil {
				return nil, err
			}
			break
		}
	}

	// Always include the files a report is known to need, even when listing is unsupported or incomplete
	known := []string{FileManifest, FileIndex, FileReport, FileData}
	if manifest.Report != "" {
		known = append(known, manifest.Report)
	}
	for _, name := range known {
		if zfs.Exists(name) || !fs.Exists(name) {
			continue
		}
		content, err := fs.GetBytes(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", name, err)
		}
		if err := zfs.Add(name, content, true); err != nil {
			return nil, err
		}
	}
	return zfs, nil
}

// copyTree copies every non-hidden readable file under src into zfs under dst.
// Entries that cannot be listed or read are skipped, as the previous export did.
func copyTree(fs fileutils.FsInterface, zfs *fileutils.ZipFs, src, dst string, depth int) error {
	if depth > maxCopyDepth {
		return fmt.Errorf("folder nesting deeper than %d at %q", maxCopyDepth, dst)
	}
	names, err := fs.List(src)
	if err != nil {
		return nil
	}
	for _, name := range names {
		name = path.Base(name) // tolerate implementations that return full paths
		if strings.HasPrefix(name, ".") {
			continue
		}
		srcPath := path.Join(src, name)
		dstPath := path.Join(dst, name)
		if fs.IsDir(srcPath) {
			if err := copyTree(fs, zfs, srcPath, dstPath, depth+1); err != nil {
				return err
			}
			continue
		}
		content, err := fs.GetBytes(srcPath)
		if err != nil {
			continue
		}
		if err := zfs.Add(dstPath, content, true); err != nil {
			return err
		}
	}
	return nil
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
	if zfs, ok := r.fs.(*fileutils.ZipFs); ok {
		return zfs.Save()
	}

	zfs, err := snapshot(r.fs, r.manifest)
	if err != nil {
		return nil, err
	}
	if err := r.addManifestIfMissing(zfs); err != nil {
		return nil, err
	}
	if err := r.checkRequired(zfs); err != nil {
		return nil, err
	}
	return zfs.Save()
}

// checkRequired fails when the template every render needs is missing from the package.
// manifest.Report is the render output, so it is not required before rendering.
func (r *ReportFile) checkRequired(zfs *fileutils.ZipFs) error {
	if !zfs.Exists(FileIndex) || zfs.IsDir(FileIndex) {
		return fmt.Errorf("%s is missing or unreadable", FileIndex)
	}
	return nil
}

// addManifestIfMissing writes the in-memory manifest when the package has none.
func (r *ReportFile) addManifestIfMissing(zfs *fileutils.ZipFs) error {
	if zfs.Exists(FileManifest) {
		return nil
	}
	data, err := json.MarshalIndent(r.manifest, "", "  ")
	if err != nil {
		return err
	}
	return zfs.Add(FileManifest, data, true)
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
