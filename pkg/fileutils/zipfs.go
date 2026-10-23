package fileutils

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"path"
)

var (
	ErrFileNotFound  = errors.New("file not found")
	ErrFileExists    = errors.New("file already exists")
	ErrNotADirectory = errors.New("not a directory")
	ErrNotAFile      = errors.New("not a file")
	ErrInvalidPath   = errors.New("invalid path")
)

// ZipFs implements FsInterface for in-memory ZIP operations.
type ZipFs struct {
	data  *bytes.Buffer
	files map[string][]byte
	cache *PathCache
}

// NewZipFs creates a new empty ZipFs.
func NewZipFs() *ZipFs {
	return &ZipFs{
		data:  &bytes.Buffer{},
		files: make(map[string][]byte),
		cache: NewPathCache(),
	}
}

// NewZipFsFromBytes creates a ZipFs from existing ZIP data.
func NewZipFsFromBytes(data []byte) (*ZipFs, error) {
	zfs := &ZipFs{
		data:  bytes.NewBuffer(data),
		files: make(map[string][]byte),
		cache: NewPathCache(),
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, f := range reader.File {
		name := normalizePath(f.Name)
		if f.FileInfo().IsDir() {
			zfs.cache.Add(name, true)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		zfs.files[name] = content
		zfs.cache.Add(name, false)
	}

	return zfs, nil
}

// NewZipFsFromFile creates a ZipFs from a ZIP file on disk.
func NewZipFsFromFile(filename string) (*ZipFs, error) {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	zfs := &ZipFs{
		data:  &bytes.Buffer{},
		files: make(map[string][]byte),
		cache: NewPathCache(),
	}

	for _, f := range reader.File {
		name := normalizePath(f.Name)
		if f.FileInfo().IsDir() {
			zfs.cache.Add(name, true)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		zfs.files[name] = content
		zfs.cache.Add(name, false)
	}

	return zfs, nil
}

// Get retrieves a file's content as a ReadCloser.
func (zfs *ZipFs) Get(name string) (io.ReadCloser, error) {
	name = normalizePath(name)
	content, exists := zfs.files[name]
	if !exists {
		return nil, ErrFileNotFound
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

// GetBytes retrieves a file's content as bytes.
func (zfs *ZipFs) GetBytes(name string) ([]byte, error) {
	name = normalizePath(name)
	content, exists := zfs.files[name]
	if !exists {
		return nil, ErrFileNotFound
	}
	// Return a copy to prevent modification
	result := make([]byte, len(content))
	copy(result, content)
	return result, nil
}

// Add adds or updates a file in the ZIP.
func (zfs *ZipFs) Add(name string, content []byte, overwrite bool) error {
	name = normalizePath(name)
	if name == "" {
		return ErrInvalidPath
	}

	_, exists := zfs.files[name]
	if exists && !overwrite {
		return ErrFileExists
	}

	// Store a copy of the content
	zfs.files[name] = make([]byte, len(content))
	copy(zfs.files[name], content)
	zfs.cache.Add(name, false)

	// Add parent directories to cache
	dir := path.Dir(name)
	for dir != "" && dir != "." {
		zfs.cache.Add(dir, true)
		dir = path.Dir(dir)
	}

	return nil
}

// Mkdir creates a directory.
func (zfs *ZipFs) Mkdir(name string) error {
	name = normalizePath(name)
	if name == "" {
		return nil
	}
	zfs.cache.Add(name, true)
	return nil
}

// Exists checks if a path exists.
func (zfs *ZipFs) Exists(name string) bool {
	name = normalizePath(name)
	if name == "" || name == "." {
		return true
	}
	return zfs.cache.Exists(name)
}

// IsDir checks if a path is a directory.
func (zfs *ZipFs) IsDir(name string) bool {
	name = normalizePath(name)
	if name == "" || name == "." {
		return true
	}
	return zfs.cache.IsDir(name)
}

// List returns all entries in a directory.
func (zfs *ZipFs) List(p string) ([]string, error) {
	p = normalizePath(p)
	if !zfs.IsDir(p) {
		return nil, ErrNotADirectory
	}
	return zfs.cache.List(p), nil
}

// ListFiles returns only files in a directory.
func (zfs *ZipFs) ListFiles(p string) ([]string, error) {
	p = normalizePath(p)
	if !zfs.IsDir(p) {
		return nil, ErrNotADirectory
	}
	return zfs.cache.ListFiles(p), nil
}

// ListDirs returns only directories in a directory.
func (zfs *ZipFs) ListDirs(p string) ([]string, error) {
	p = normalizePath(p)
	if !zfs.IsDir(p) {
		return nil, ErrNotADirectory
	}
	return zfs.cache.ListDirs(p), nil
}

// Remove removes a file.
func (zfs *ZipFs) Remove(name string) error {
	name = normalizePath(name)
	if _, exists := zfs.files[name]; !exists {
		return ErrFileNotFound
	}
	delete(zfs.files, name)
	zfs.cache.Remove(name)
	return nil
}

// Save serializes the ZipFs to ZIP format bytes.
func (zfs *ZipFs) Save() ([]byte, error) {
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)

	// Collect and sort directories
	dirs := make(map[string]bool)
	for name := range zfs.files {
		dir := path.Dir(name)
		for dir != "" && dir != "." {
			dirs[dir] = true
			dir = path.Dir(dir)
		}
	}

	// Write directories
	for dir := range dirs {
		_, err := writer.Create(dir + "/")
		if err != nil {
			return nil, err
		}
	}

	// Write files
	for name, content := range zfs.files {
		w, err := writer.Create(name)
		if err != nil {
			return nil, err
		}
		_, err = w.Write(content)
		if err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// SaveToFile saves the ZipFs to a file.
func (zfs *ZipFs) SaveToFile(filename string) error {
	data, err := zfs.Save()
	if err != nil {
		return err
	}

	return writeFile(filename, data)
}

// Clone returns an independent copy of the ZipFs. File contents are shared with
// the original, which is safe because stored content is never modified in place.
func (zfs *ZipFs) Clone() *ZipFs {
	c := NewZipFs()
	for name, content := range zfs.files {
		c.files[name] = content
		c.cache.Add(name, false)
	}
	return c
}

// Files returns a list of all file paths.
func (zfs *ZipFs) Files() []string {
	result := make([]string, 0, len(zfs.files))
	for name := range zfs.files {
		result = append(result, name)
	}
	return result
}

// writeFile is a helper to write data to a file (implemented separately to avoid import cycle).
func writeFile(filename string, data []byte) error {
	// This will be implemented using os.WriteFile
	// For now, we import it inline
	return writeFileImpl(filename, data)
}
