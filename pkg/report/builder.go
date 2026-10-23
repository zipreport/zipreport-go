package report

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zipreport/zipreport-go/pkg/fileutils"
)

// BuildResult represents the result of a build operation.
type BuildResult struct {
	success bool
	errors  []string
}

// Success returns whether the build was successful.
func (r *BuildResult) Success() bool {
	return r.success
}

// Errors returns the list of errors encountered during build.
func (r *BuildResult) Errors() []string {
	return r.errors
}

// Error returns a combined error string.
func (r *BuildResult) Error() string {
	return strings.Join(r.errors, "; ")
}

// BuildOptions configures the build process.
type BuildOptions struct {
	FollowSymlinks bool
	ExcludeHidden  bool
}

// DefaultBuildOptions returns default build options.
func DefaultBuildOptions() *BuildOptions {
	return &BuildOptions{
		FollowSymlinks: false,
		ExcludeHidden:  true,
	}
}

// Build builds a .zpt file from a source directory.
func Build(srcDir, destFile string, opts *BuildOptions) *BuildResult {
	if opts == nil {
		opts = DefaultBuildOptions()
	}

	result := &BuildResult{success: true}

	// Validate source directory
	srcInfo, err := os.Stat(srcDir)
	if err != nil {
		result.success = false
		result.errors = append(result.errors, fmt.Sprintf("source not found: %v", err))
		return result
	}

	if !srcInfo.IsDir() {
		result.success = false
		result.errors = append(result.errors, "source is not a directory")
		return result
	}

	// Build to ZipFs
	zfs, buildResult := BuildZipFs(srcDir, opts)
	if !buildResult.Success() {
		return buildResult
	}

	// Save to file
	data, err := zfs.Save()
	if err != nil {
		result.success = false
		result.errors = append(result.errors, fmt.Sprintf("failed to create zip: %v", err))
		return result
	}

	if err := os.WriteFile(destFile, data, 0644); err != nil {
		result.success = false
		result.errors = append(result.errors, fmt.Sprintf("failed to write file: %v", err))
		return result
	}

	return result
}

// BuildZipFs builds a ZipFs from a source directory.
func BuildZipFs(srcDir string, opts *BuildOptions) (*fileutils.ZipFs, *BuildResult) {
	if opts == nil {
		opts = DefaultBuildOptions()
	}

	result := &BuildResult{success: true}
	zfs := fileutils.NewZipFs()

	absPath, err := filepath.Abs(srcDir)
	if err != nil {
		result.success = false
		result.errors = append(result.errors, fmt.Sprintf("invalid path: %v", err))
		return nil, result
	}

	// Walk the real folder even when srcDir is a symlink (WalkDir doesn't follow a symlinked root)
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	}

	// Validate required files exist
	manifestPath := filepath.Join(absPath, FileManifest)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		result.success = false
		result.errors = append(result.errors, "manifest.json not found")
	}

	indexPath := filepath.Join(absPath, FileIndex)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		result.success = false
		result.errors = append(result.errors, "index.html not found")
	}

	// Validate manifest
	if result.success {
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			result.success = false
			result.errors = append(result.errors, fmt.Sprintf("failed to read manifest: %v", err))
		} else {
			if err := ValidateManifest(manifestData); err != nil {
				result.success = false
				result.errors = append(result.errors, fmt.Sprintf("invalid manifest: %v", err))
			}
		}
	}

	if !result.success {
		return nil, result
	}

	// Walk directory and add files
	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files if configured
		if opts.ExcludeHidden && path != absPath && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Handle symlinks
		if d.Type()&os.ModeSymlink != 0 {
			if !opts.FollowSymlinks {
				return nil
			}
			// Resolve symlink
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				result.errors = append(result.errors, fmt.Sprintf("failed to resolve symlink %s: %v", path, err))
				return nil
			}
			info, err := os.Stat(resolved)
			if err != nil {
				return nil
			}
			if info.IsDir() {
				linkRel, _ := filepath.Rel(absPath, path)
				result.errors = append(result.errors, fmt.Sprintf("symlinked directories are not supported: %s (copy the folder, or build without -s to skip it)", filepath.ToSlash(linkRel)))
				return nil
			}
			if resolved != absPath && !strings.HasPrefix(resolved, strings.TrimSuffix(absPath, string(filepath.Separator))+string(filepath.Separator)) {
				linkRel, _ := filepath.Rel(absPath, path)
				result.errors = append(result.errors, fmt.Sprintf("symlink points outside the template folder: %s", filepath.ToSlash(linkRel)))
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil // sockets, devices, FIFOs are never template files
			}
		}

		// Skip directories (they'll be created implicitly)
		if d.IsDir() {
			return nil
		}

		// Skip sockets, devices and FIFOs: reading a FIFO would block forever
		if d.Type()&os.ModeSymlink == 0 && !d.Type().IsRegular() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			return err
		}

		// Normalize path separators
		relPath = filepath.ToSlash(relPath)

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("failed to read %s: %v", relPath, err))
			return nil
		}

		// Add to zip
		if err := zfs.Add(relPath, content, true); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("failed to add %s: %v", relPath, err))
		}

		return nil
	})

	if err != nil {
		result.success = false
		result.errors = append(result.errors, fmt.Sprintf("walk error: %v", err))
		return nil, result
	}

	// The required files must be in the package (e.g. not skipped as symlinks)
	for _, required := range []string{FileManifest, FileIndex} {
		if !zfs.Exists(required) || zfs.IsDir(required) {
			result.errors = append(result.errors, fmt.Sprintf("%s was not included in the build (is it a symlink? use -s (FollowSymlinks))", required))
		}
	}

	// A build that skipped any file is not a successful build
	if len(result.errors) > 0 {
		result.success = false
		return nil, result
	}

	return zfs, result
}

// BuildFromReportFile creates a new ZipFs from an existing ReportFile.
// This is useful for converting a directory-based report to a .zpt file.
func BuildFromReportFile(report *ReportFile) (*fileutils.ZipFs, error) {
	zfs, err := snapshot(report.fs, report.manifest)
	if err != nil {
		return nil, err
	}
	if err := report.addManifestIfMissing(zfs); err != nil {
		return nil, err
	}
	if err := report.checkRequired(zfs); err != nil {
		return nil, err
	}
	return zfs, nil
}
