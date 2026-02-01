package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
)

var (
	ErrManifestNotFound = errors.New("manifest.json not found")
	ErrIndexNotFound    = errors.New("index.html not found")
	ErrInvalidManifest  = errors.New("invalid manifest")
	ErrMissingField     = errors.New("missing required manifest field")
)

// Load loads a ReportFile from a path.
// It auto-detects whether the path is a directory or a .zpt file.
func Load(path string) (*ReportFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return LoadDir(path)
	}

	return LoadFile(path)
}

// LoadFile loads a ReportFile from a .zpt file.
func LoadFile(filename string) (*ReportFile, error) {
	zfs, err := fileutils.NewZipFsFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open zpt file: %w", err)
	}

	return LoadZipFs(zfs)
}

// LoadBytes loads a ReportFile from ZIP bytes.
func LoadBytes(data []byte) (*ReportFile, error) {
	zfs, err := fileutils.NewZipFsFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse zpt data: %w", err)
	}

	return LoadZipFs(zfs)
}

// LoadZipFs loads a ReportFile from an existing ZipFs.
func LoadZipFs(zfs *fileutils.ZipFs) (*ReportFile, error) {
	// Validate required files
	if !zfs.Exists(FileManifest) {
		return nil, ErrManifestNotFound
	}

	if !zfs.Exists(FileIndex) {
		return nil, ErrIndexNotFound
	}

	// Parse manifest
	manifestData, err := zfs.GetBytes(FileManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	manifest, err := parseManifest(manifestData)
	if err != nil {
		return nil, err
	}

	return NewReportFile(zfs, manifest), nil
}

// LoadDir loads a ReportFile from a directory.
func LoadDir(dirPath string) (*ReportFile, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}

	dfs, err := fileutils.NewDiskFs(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory: %w", err)
	}

	// Validate required files
	if !dfs.Exists(FileManifest) {
		return nil, ErrManifestNotFound
	}

	if !dfs.Exists(FileIndex) {
		return nil, ErrIndexNotFound
	}

	// Parse manifest
	manifestData, err := dfs.GetBytes(FileManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	manifest, err := parseManifest(manifestData)
	if err != nil {
		return nil, err
	}

	return NewReportFile(dfs, manifest), nil
}

// parseManifest parses and validates a manifest.
func parseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}

	// Validate required fields
	if manifest.Title == "" {
		return nil, fmt.Errorf("%w: title", ErrMissingField)
	}
	if manifest.Author == "" {
		return nil, fmt.Errorf("%w: author", ErrMissingField)
	}
	if manifest.Version == "" {
		return nil, fmt.Errorf("%w: version", ErrMissingField)
	}
	if manifest.Description == "" {
		return nil, fmt.Errorf("%w: description", ErrMissingField)
	}
	if manifest.Params == nil {
		manifest.Params = []string{}
	}

	return &manifest, nil
}

// ValidateManifest validates a manifest without loading the full report.
func ValidateManifest(data []byte) error {
	_, err := parseManifest(data)
	return err
}
