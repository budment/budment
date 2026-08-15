package filesystem

const (
	defaultFileMode = 0644
	defaultDirMode  = 0755
)

type Entry struct {
	Name  string
	IsDir bool
}

type WalkFunc func(path string, isDir bool, err error) error

// FS isolates the Domain layer from the underlying OS filesystem.
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
