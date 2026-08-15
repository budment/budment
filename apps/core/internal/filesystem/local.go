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

// LocalFS is the standard OS-backed implementation of FS.
type LocalFS struct{}

func NewLocal() FS {
	return &LocalFS{}
}

func (f *LocalFS) Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fs read failed [%s]: %w", path, err)
	}
	return data, nil
}

func (f *LocalFS) Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := f.MkdirAll(dir); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, defaultFileMode); err != nil {
		return fmt.Errorf("fs write failed [%s]: %w", path, err)
	}
	return nil
}

func (f *LocalFS) Exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func (f *LocalFS) MkdirAll(path string) error {
	if err := os.MkdirAll(path, defaultDirMode); err != nil {
		return fmt.Errorf("fs mkdir failed [%s]: %w", path, err)
	}
	return nil
}

func (f *LocalFS) Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("fs remove failed [%s]: %w", path, err)
	}
	return nil
}

func (f *LocalFS) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("fs remove all failed [%s]: %w", path, err)
	}
	return nil
}

func (f *LocalFS) Walk(root string, fn WalkFunc) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		isDir := false
		if d != nil {
			isDir = d.IsDir()
		}

		cbErr := fn(path, isDir, err)
		if errors.Is(cbErr, ErrSkipDir) {
			return fs.SkipDir
		}
		return cbErr
	})
}

func (f *LocalFS) ReadDir(name string) ([]Entry, error) {
	osEntries, err := os.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("fs readdir failed [%s]: %w", name, err)
	}

	entries := make([]Entry, 0, len(osEntries))
	for _, e := range osEntries {
		entries = append(entries, Entry{Name: e.Name(), IsDir: e.IsDir()})
	}
	return entries, nil
}

func (f *LocalFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (f *LocalFS) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}
