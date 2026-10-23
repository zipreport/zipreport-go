package fileutils

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DiskFs implements FsInterface for disk-based file operations.
// It includes path traversal protection for security.
type DiskFs struct {
	basePath string
}

// NewDiskFs creates a new DiskFs rooted at the given path.
func NewDiskFs(basePath string) (*DiskFs, error) {
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, ErrNotADirectory
	}

	return &DiskFs{basePath: absPath}, nil
}

// buildPath constructs a safe path within the base directory.
// It prevents path traversal attacks.
func (dfs *DiskFs) buildPath(name string) (string, error) {
	// Clean the path to remove any . or .. components
	cleaned := filepath.Clean(name)

	// Join with base path
	fullPath := filepath.Join(dfs.basePath, cleaned)

	// Ensure the resulting path is within the base directory
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	// Reject anything that resolves outside the base directory
	base := strings.TrimSuffix(dfs.basePath, string(filepath.Separator))
	if absPath != dfs.basePath && !strings.HasPrefix(absPath, base+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}

	// Reject paths that leave the base directory through a symlink
	if err := dfs.checkResolved(absPath); err != nil {
		return "", err
	}

	return absPath, nil
}

// IsHiddenPath reports whether any segment of name starts with a dot.
// Both "/" and "\\" count as separators, so Windows-style names cannot bypass it.
func IsHiddenPath(name string) bool {
	for _, seg := range strings.FieldsFunc(name, func(r rune) bool { return r == '/' || r == '\\' }) {
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

// checkResolved fails when absPath, or its deepest existing ancestor, resolves
// outside the base directory once symlinks are followed. A dangling symlink is
// rejected because its target cannot be checked.
func (dfs *DiskFs) checkResolved(absPath string) error {
	base, err := filepath.EvalSymlinks(dfs.basePath)
	if err != nil {
		return err
	}
	prefix := strings.TrimSuffix(base, string(filepath.Separator)) + string(filepath.Separator)
	for p := absPath; ; {
		resolved, err := filepath.EvalSymlinks(p)
		if err == nil {
			if resolved != base && !strings.HasPrefix(resolved, prefix) {
				return ErrOutsideFolder
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if info, lerr := os.Lstat(p); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			return ErrOutsideFolder
		}
		parent := filepath.Dir(p)
		if parent == p {
			return nil
		}
		p = parent
	}
}

// Get retrieves a file's content as a ReadCloser.
func (dfs *DiskFs) Get(name string) (io.ReadCloser, error) {
	fullPath, err := dfs.buildPath(name)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	return file, nil
}

// GetBytes retrieves a file's content as bytes.
func (dfs *DiskFs) GetBytes(name string) ([]byte, error) {
	fullPath, err := dfs.buildPath(name)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	return content, nil
}

// Add adds or updates a file on disk.
func (dfs *DiskFs) Add(name string, content []byte, overwrite bool) error {
	fullPath, err := dfs.buildPath(name)
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); err == nil && !overwrite {
		return ErrFileExists
	}

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, content, 0644)
}

// Mkdir creates a directory.
func (dfs *DiskFs) Mkdir(name string) error {
	fullPath, err := dfs.buildPath(name)
	if err != nil {
		return err
	}

	return os.MkdirAll(fullPath, 0755)
}

// Exists checks if a path exists.
func (dfs *DiskFs) Exists(name string) bool {
	fullPath, err := dfs.buildPath(name)
	if err != nil {
		return false
	}

	_, err = os.Stat(fullPath)
	return err == nil
}

// IsDir checks if a path is a directory.
func (dfs *DiskFs) IsDir(name string) bool {
	fullPath, err := dfs.buildPath(name)
	if err != nil {
		return false
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// List returns all entries in a directory.
func (dfs *DiskFs) List(p string) ([]string, error) {
	fullPath, err := dfs.buildPath(p)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.Name())
	}

	return result, nil
}

// ListFiles returns only files in a directory.
func (dfs *DiskFs) ListFiles(p string) ([]string, error) {
	fullPath, err := dfs.buildPath(p)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			result = append(result, entry.Name())
		}
	}

	return result, nil
}

// ListDirs returns only directories in a directory.
func (dfs *DiskFs) ListDirs(p string) ([]string, error) {
	fullPath, err := dfs.buildPath(p)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, entry.Name())
		}
	}

	return result, nil
}

// Remove removes a file or directory.
func (dfs *DiskFs) Remove(name string) error {
	fullPath, err := dfs.buildPath(name)
	if err != nil {
		return err
	}

	return os.RemoveAll(fullPath)
}

// BasePath returns the base path of the DiskFs.
func (dfs *DiskFs) BasePath() string {
	return dfs.basePath
}

// maxSnapshotDepth bounds folder nesting in Snapshot.
const maxSnapshotDepth = 32

// Snapshot copies the folder into a new in-memory ZipFs, as sent to a renderer.
// Hidden entries, root-level _gen_* files (generated images left by older versions),
// non-regular files (sockets, FIFOs, devices) and broken symlinks are skipped.
// Symlinks are followed only when they resolve inside the folder.
func (dfs *DiskFs) Snapshot() (*ZipFs, error) {
	root, err := filepath.EvalSymlinks(dfs.basePath)
	if err != nil {
		return nil, err
	}
	zfs := NewZipFs()
	if err := snapshotDir(zfs, root, root, "", map[string]bool{root: true}, 0); err != nil {
		return nil, err
	}
	return zfs, nil
}

// snapshotDir copies dir (a real path inside root) into zfs under rel.
// ancestors holds the real paths of the folders being walked, to stop symlink loops.
func snapshotDir(zfs *ZipFs, root, dir, rel string, ancestors map[string]bool, depth int) error {
	if depth > maxSnapshotDepth {
		return fmt.Errorf("folder nesting deeper than %d at %q", maxSnapshotDepth, rel)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if depth > 0 && errors.Is(err, fs.ErrPermission) {
			return nil // unreadable subfolder: skipped, as renders never needed it
		}
		return err
	}
	prefix := strings.TrimSuffix(root, string(filepath.Separator)) + string(filepath.Separator)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || (rel == "" && strings.HasPrefix(name, "_gen_")) {
			continue
		}
		real, err := filepath.EvalSymlinks(filepath.Join(dir, name))
		if err != nil {
			continue // broken symlink
		}
		if real != root && !strings.HasPrefix(real, prefix) {
			continue // symlink leaving the folder
		}
		info, err := os.Stat(real)
		if err != nil {
			continue
		}
		relName := path.Join(rel, name)
		switch {
		case info.IsDir():
			if ancestors[real] {
				continue // symlink loop
			}
			ancestors[real] = true
			err := snapshotDir(zfs, root, real, relName, ancestors, depth+1)
			delete(ancestors, real)
			if err != nil {
				return err
			}
		case info.Mode().IsRegular():
			content, err := os.ReadFile(real)
			if errors.Is(err, fs.ErrPermission) {
				continue // unreadable file: skipped; Save checks the required files afterwards
			}
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", relName, err)
			}
			if err := zfs.Add(relName, content, true); err != nil {
				if errors.Is(err, ErrUnportableName) {
					return fmt.Errorf("%s: %w (rename it)", relName, err)
				}
				return err
			}
		}
	}
	return nil
}

// writeFileImpl is the actual implementation used by ZipFs.SaveToFile
func writeFileImpl(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}
