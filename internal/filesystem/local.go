package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrSkipDir tells Walk to skip the current directory.
// It keeps the Domain layer decoupled from io/fs.
var ErrSkipDir = errors.New("skip this directory")

type LocalFS struct{}

// NewLocal creates a new instance of LocalFS.
func NewLocal() FS {
	return &LocalFS{}
}

func (f *LocalFS) Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fs: failed to read file %q: %w", path, err)
	}
	return data, nil
}

func (f *LocalFS) Write(path string, data []byte) error {
	dir := filepath.Dir(path)

	// Ensure parent directory exists
	if err := f.MkdirAll(dir); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, defaultFileMode); err != nil {
		return fmt.Errorf("fs: failed to write file %q: %w", path, err)
	}
	return nil
}

func (f *LocalFS) Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	// Return true if the file exists but has permission or other non-NotExist errors
	return !errors.Is(err, fs.ErrNotExist)
}

func (f *LocalFS) MkdirAll(path string) error {
	if err := os.MkdirAll(path, defaultDirMode); err != nil {
		return fmt.Errorf("fs: failed to create directory %q: %w", path, err)
	}
	return nil
}

func (f *LocalFS) Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("fs: failed to remove %q: %w", path, err)
	}
	return nil
}

func (f *LocalFS) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("fs: failed to remove all %q: %w", path, err)
	}
	return nil
}

// Walk bridges the gap between filepath.WalkDir and our clean FS interface.
func (f *LocalFS) Walk(root string, fn WalkFunc) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		var isDir bool
		if d != nil {
			isDir = d.IsDir()
		}

		// Delegate error handling and skip logic to the caller
		cbErr := fn(path, isDir, err)

		// Map our custom ErrSkipDir to the standard fs.SkipDir
		if errors.Is(cbErr, ErrSkipDir) {
			return fs.SkipDir
		}

		return cbErr
	})
}

func (f *LocalFS) ReadDir(name string) ([]Entry, error) {
	osEntries, err := os.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("fs: failed to read directory %q: %w", name, err)
	}

	entries := make([]Entry, 0, len(osEntries))
	for _, e := range osEntries {
		entries = append(entries, Entry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
		})
	}
	return entries, nil
}

func (f *LocalFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (f *LocalFS) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}
