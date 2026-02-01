// Package fileutils provides file system abstractions for ZIP and disk operations.
package fileutils

import "io"

// FsInterface defines the contract for file system operations.
// It provides a unified interface for both ZIP-based and disk-based file systems.
type FsInterface interface {
	// Get retrieves a file's content as a ReadCloser.
	Get(name string) (io.ReadCloser, error)

	// GetBytes retrieves a file's content as bytes.
	GetBytes(name string) ([]byte, error)

	// Add adds or updates a file in the file system.
	Add(name string, content []byte, overwrite bool) error

	// Mkdir creates a directory.
	Mkdir(name string) error

	// Exists checks if a path exists.
	Exists(name string) bool

	// IsDir checks if a path is a directory.
	IsDir(name string) bool

	// List returns all entries (files and directories) in a path.
	List(path string) ([]string, error)

	// ListFiles returns only files in a path.
	ListFiles(path string) ([]string, error)

	// ListDirs returns only directories in a path.
	ListDirs(path string) ([]string, error)

	// Remove removes a file or directory.
	Remove(name string) error
}
