package filesystem

import (
	"os"
	"time"
)

// FileSystem provides a minimal abstraction over filesystem operations
type FileSystem interface {
	ReadFile(name string) ([]byte, error)
	Stat(name string) (os.FileInfo, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// MemoryFileSystem placeholder for testing
type MemoryFileSystem struct {
	files map[string][]byte
}

// NewMemoryFileSystem creates a new in-memory filesystem for testing
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		files: make(map[string][]byte),
	}
}

// WriteFile stores data in memory
func (fs *MemoryFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	fs.files[name] = make([]byte, len(data))
	copy(fs.files[name], data)
	return nil
}

// ReadFile retrieves data from memory
func (fs *MemoryFileSystem) ReadFile(name string) ([]byte, error) {
	data, exists := fs.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// Stat returns file info for in-memory files
func (fs *MemoryFileSystem) Stat(name string) (os.FileInfo, error) {
	_, exists := fs.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}
	// Return a minimal FileInfo implementation for testing
	return &memoryFileInfo{name: name, size: int64(len(fs.files[name]))}, nil
}

// memoryFileInfo implements os.FileInfo for in-memory files
type memoryFileInfo struct {
	name string
	size int64
}

func (fi *memoryFileInfo) Name() string       { return fi.name }
func (fi *memoryFileInfo) Size() int64        { return fi.size }
func (fi *memoryFileInfo) Mode() os.FileMode  { return 0o644 }
func (fi *memoryFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *memoryFileInfo) IsDir() bool        { return false }
func (fi *memoryFileInfo) Sys() interface{}   { return nil }

// MkdirAll creates directories recursively in memory (simple implementation for testing)
func (fs *MemoryFileSystem) MkdirAll(path string, perm os.FileMode) error {
	// For in-memory filesystem, we just need to track that the directory exists
	// This allows WriteFile to work in any "created" directory
	return nil
}

// OSFileSystem implements FileSystem using real os operations
type OSFileSystem struct{}

// NewOSFileSystem creates a new OS-backed filesystem for production
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// WriteFile writes data to the real filesystem
func (fs *OSFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm) //nolint:wrapcheck // direct os wrapper
}

// ReadFile reads data from the real filesystem
func (fs *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:wrapcheck,gosec // direct os wrapper, controlled by caller
}

// Stat returns file info from the real filesystem
func (fs *OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name) //nolint:wrapcheck,gosec // direct os wrapper, controlled by caller
}

// MkdirAll creates directories recursively using the real filesystem
func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm) //nolint:wrapcheck,gosec // direct os wrapper, controlled by caller
}
