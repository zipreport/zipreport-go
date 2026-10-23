package fileutils

import (
	"errors"
	"io"
)

// ErrReadOnly is returned when removing a file that belongs to an overlay's base.
var ErrReadOnly = errors.New("file belongs to the read-only base")

// OverlayFs layers an in-memory ZipFs over a base file system. Reads check the
// overlay first and fall through to the base; every write goes to the overlay,
// so the base is never modified.
type OverlayFs struct {
	base    FsInterface
	overlay *ZipFs
}

// NewOverlayFs creates an OverlayFs over base.
func NewOverlayFs(base FsInterface) *OverlayFs {
	return &OverlayFs{base: base, overlay: NewZipFs()}
}

// Base returns the underlying read-only file system.
func (o *OverlayFs) Base() FsInterface {
	return o.base
}

// Overlay returns the in-memory layer holding all writes.
func (o *OverlayFs) Overlay() *ZipFs {
	return o.overlay
}

// isRoot reports whether name refers to the file system root ("", "." or "/").
func isRoot(name string) bool {
	n := normalizePath(name)
	return n == "" || n == "."
}

// inOverlay reports whether name is a file in the overlay.
func (o *OverlayFs) inOverlay(name string) bool {
	return o.overlay.Exists(name) && !o.overlay.IsDir(name)
}

// Get retrieves a file's content as a ReadCloser.
func (o *OverlayFs) Get(name string) (io.ReadCloser, error) {
	if o.inOverlay(name) {
		return o.overlay.Get(name)
	}
	return o.base.Get(name)
}

// GetBytes retrieves a file's content as bytes.
func (o *OverlayFs) GetBytes(name string) ([]byte, error) {
	if o.inOverlay(name) {
		return o.overlay.GetBytes(name)
	}
	return o.base.GetBytes(name)
}

// Add adds or updates a file in the overlay.
func (o *OverlayFs) Add(name string, content []byte, overwrite bool) error {
	if !overwrite && o.Exists(name) && !o.IsDir(name) {
		return ErrFileExists
	}
	return o.overlay.Add(name, content, true)
}

// Mkdir creates a directory in the overlay.
func (o *OverlayFs) Mkdir(name string) error {
	return o.overlay.Mkdir(name)
}

// Exists checks if a path exists in the overlay or the base.
func (o *OverlayFs) Exists(name string) bool {
	if isRoot(name) {
		return o.base.Exists(name)
	}
	return o.inOverlay(name) || o.overlay.IsDir(name) || o.base.Exists(name)
}

// IsDir checks if a path is a directory in the overlay or the base.
func (o *OverlayFs) IsDir(name string) bool {
	if isRoot(name) {
		return o.base.IsDir(name)
	}
	return o.overlay.IsDir(name) || o.base.IsDir(name)
}

// List returns all entries in a directory of the overlay and the base.
func (o *OverlayFs) List(p string) ([]string, error) {
	return o.merge(p, o.base.List, o.overlay.List)
}

// ListFiles returns only files in a directory of the overlay and the base.
func (o *OverlayFs) ListFiles(p string) ([]string, error) {
	return o.merge(p, o.base.ListFiles, o.overlay.ListFiles)
}

// ListDirs returns only directories in a directory of the overlay and the base.
func (o *OverlayFs) ListDirs(p string) ([]string, error) {
	return o.merge(p, o.base.ListDirs, o.overlay.ListDirs)
}

// merge combines listings from both layers without duplicates.
func (o *OverlayFs) merge(p string, base, overlay func(string) ([]string, error)) ([]string, error) {
	baseNames, baseErr := base(p)
	overlayNames, overlayErr := overlay(p)
	// The overlay's root always lists, so a base error at the root must not be hidden
	if baseErr != nil && (overlayErr != nil || isRoot(p)) {
		return nil, baseErr
	}
	seen := make(map[string]bool, len(baseNames)+len(overlayNames))
	result := make([]string, 0, len(baseNames)+len(overlayNames))
	for _, n := range append(baseNames, overlayNames...) {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	return result, nil
}

// Remove removes a file from the overlay; base files cannot be removed.
func (o *OverlayFs) Remove(name string) error {
	if o.inOverlay(name) {
		return o.overlay.Remove(name)
	}
	if o.base.Exists(name) {
		return ErrReadOnly
	}
	return ErrFileNotFound
}
