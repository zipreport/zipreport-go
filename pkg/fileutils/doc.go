// Package fileutils provides file system abstractions for ZIP and disk operations.
//
// The package defines the FsInterface interface which is implemented by:
//   - ZipFs: In-memory ZIP file operations
//   - DiskFs: Disk-based file operations with path traversal protection
//
// PathCache provides a trie-based structure for efficient path lookups,
// used internally by ZipFs for fast file existence checks and listing operations.
//
// Example usage with ZipFs:
//
//	// Create a new empty ZipFs
//	zfs := fileutils.NewZipFs()
//	zfs.Add("index.html", []byte("<html>...</html>"), false)
//
//	// Load from existing ZIP file
//	zfs, err := fileutils.NewZipFsFromFile("report.zpt")
//
//	// Save to bytes
//	data, err := zfs.Save()
//
// Example usage with DiskFs:
//
//	// Create a DiskFs rooted at a directory
//	dfs, err := fileutils.NewDiskFs("/path/to/template")
//
//	// Read a file
//	content, err := dfs.GetBytes("index.html")
package fileutils
