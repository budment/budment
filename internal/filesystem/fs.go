package filesystem

const (
	defaultFileMode = 0644
	defaultDirMode  = 0755
)

// Entry represents a directory entry.
type Entry struct {
	Name  string
	IsDir bool
}

// WalkFunc is the callback called for each file or directory visited by Walk.
// It includes an error parameter so the caller can handle permission or read errors.
type WalkFunc func(path string, isDir bool, err error) error

// FS defines the strict contract for all filesystem interactions.
// It is completely agnostic to the underlying storage mechanism.
type FS interface {
	Read(path string) ([]byte, error)
	Write(path string, data []byte) error
	Exists(path string) bool

	MkdirAll(path string) error
	ReadDir(name string) ([]Entry, error)

	Remove(path string) error
	RemoveAll(path string) error

	Walk(root string, fn WalkFunc) error

	Join(elem ...string) string
	Rel(basepath, targpath string) (string, error)
}
