package fileutils

import (
	"path"
	"strings"
	"sync"
)

// PathCache is a trie-based structure for efficient path lookups.
// It accelerates file/directory existence checks and listing operations.
type PathCache struct {
	mu   sync.RWMutex
	root *pathNode
}

type pathNode struct {
	children map[string]*pathNode
	isFile   bool
	isDir    bool
}

// NewPathCache creates a new PathCache.
func NewPathCache() *PathCache {
	return &PathCache{
		root: &pathNode{
			children: make(map[string]*pathNode),
			isDir:    true,
		},
	}
}

// normalizePath cleans and normalizes a path.
func normalizePath(p string) string {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	return p
}

// splitPath splits a path into its components.
func splitPath(p string) []string {
	p = normalizePath(p)
	if p == "" || p == "." {
		return nil
	}
	return strings.Split(p, "/")
}

// Add adds a path to the cache.
func (pc *PathCache) Add(p string, isDir bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	parts := splitPath(p)
	if len(parts) == 0 {
		return
	}

	node := pc.root
	for i, part := range parts {
		if node.children == nil {
			node.children = make(map[string]*pathNode)
		}

		child, exists := node.children[part]
		if !exists {
			child = &pathNode{
				children: make(map[string]*pathNode),
			}
			node.children[part] = child
		}

		// Mark intermediate directories
		if i < len(parts)-1 {
			child.isDir = true
		}

		node = child
	}

	if isDir {
		node.isDir = true
	} else {
		node.isFile = true
	}
}

// Remove removes a path from the cache.
func (pc *PathCache) Remove(p string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	parts := splitPath(p)
	if len(parts) == 0 {
		return
	}

	// Find parent and remove the target
	var parent *pathNode
	var targetKey string
	node := pc.root

	for i, part := range parts {
		if node.children == nil {
			return
		}
		child, exists := node.children[part]
		if !exists {
			return
		}
		if i == len(parts)-1 {
			parent = node
			targetKey = part
		}
		node = child
	}

	if parent != nil {
		delete(parent.children, targetKey)
	}
}

// Exists checks if a path exists.
func (pc *PathCache) Exists(p string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	node := pc.getNode(p)
	return node != nil && (node.isFile || node.isDir)
}

// IsFile checks if a path is a file.
func (pc *PathCache) IsFile(p string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	node := pc.getNode(p)
	return node != nil && node.isFile
}

// IsDir checks if a path is a directory.
func (pc *PathCache) IsDir(p string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	node := pc.getNode(p)
	return node != nil && node.isDir
}

// getNode retrieves the node for a path (must hold at least read lock).
func (pc *PathCache) getNode(p string) *pathNode {
	parts := splitPath(p)
	if len(parts) == 0 {
		return pc.root
	}

	node := pc.root
	for _, part := range parts {
		if node.children == nil {
			return nil
		}
		child, exists := node.children[part]
		if !exists {
			return nil
		}
		node = child
	}
	return node
}

// List returns all entries in a directory.
func (pc *PathCache) List(p string) []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	node := pc.getNode(p)
	if node == nil || node.children == nil {
		return nil
	}

	result := make([]string, 0, len(node.children))
	for name := range node.children {
		result = append(result, name)
	}
	return result
}

// ListFiles returns only files in a directory.
func (pc *PathCache) ListFiles(p string) []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	node := pc.getNode(p)
	if node == nil || node.children == nil {
		return nil
	}

	var result []string
	for name, child := range node.children {
		if child.isFile {
			result = append(result, name)
		}
	}
	return result
}

// ListDirs returns only directories in a directory.
func (pc *PathCache) ListDirs(p string) []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	node := pc.getNode(p)
	if node == nil || node.children == nil {
		return nil
	}

	var result []string
	for name, child := range node.children {
		if child.isDir {
			result = append(result, name)
		}
	}
	return result
}

// Clear removes all entries from the cache.
func (pc *PathCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.root = &pathNode{
		children: make(map[string]*pathNode),
		isDir:    true,
	}
}
