package endpoint

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vunas/blaster/internal/filesystem"
)

// Reader loads existing Endpoint configurations from storage.
type Reader struct {
	BaseDir string
	fs      filesystem.FS
}

func NewReader(baseDir string, fs filesystem.FS) *Reader {
	return &Reader{BaseDir: baseDir, fs: fs}
}

// ReadAll parses YAML files recursively into Endpoint models.
func (r *Reader) ReadAll() []*Endpoint {
	var endpoints []*Endpoint
	if !r.fs.Exists(r.BaseDir) {
		return endpoints
	}

	r.fs.Walk(r.BaseDir, func(path string, isDir bool, err error) error {
		if isDir || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		if ep := r.parseFile(path); ep != nil {
			endpoints = append(endpoints, ep)
		}
		return nil
	})
	return endpoints
}

func (r *Reader) parseFile(fullPath string) *Endpoint {
	relPath, err := r.fs.Rel(r.BaseDir, fullPath)
	if err != nil {
		return nil
	}

	parts := strings.Split(strings.ReplaceAll(relPath, "\\", "/"), "/")
	fileName := parts[len(parts)-1]
	dir := "/"
	if len(parts) > 1 {
		dir = "/" + strings.Join(parts[:len(parts)-1], "/")
	}

	method := strings.ToUpper(strings.TrimSuffix(fileName, ".yaml"))
	data, err := r.fs.Read(fullPath)
	if err != nil {
		return nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}

	ep := &Endpoint{Method: method, Path: dir}
	r.walkYamlMap(raw, []string{}, ep)
	return ep
}

func (r *Reader) walkYamlMap(node map[string]any, currentPath []string, ep *Endpoint) {
	for key, val := range node {
		newPath := append(append([]string(nil), currentPath...), key)
		switch v := val.(type) {
		case map[string]any:
			r.walkYamlMap(v, newPath, ep)
		case string:
			if len(newPath) > 0 && newPath[0] == "response" {
				ident := Identity{NodePath: newPath[1:], Name: key}
				if v == "?" {
					ident.Status = StatusPending
				} else if v == "ignore" {
					ident.Status = StatusIgnored
				} else if v != "identity" { // Contains a Branch mapping
					ident.Status = StatusResolved
					ident.TargetID = v
				} else {
					ident.Status = StatusResolved
				}
				ep.Identities = append(ep.Identities, ident)
			} else {
				rel := Relative{NodePath: newPath, Name: key}
				switch v {
				case "?":
					rel.Status = StatusPending
				case "ignore":
					rel.Status = StatusIgnored
				default:
					rel.Status = StatusResolved
					rel.TargetID = v
				}
				ep.Relatives = append(ep.Relatives, rel)
			}
		}
	}
}

// Writer handles idempotent output of Endpoints and Garbage Collection.
type Writer struct {
	BaseDir string
	fs      filesystem.FS
}

func NewWriter(baseDir string, fs filesystem.FS) *Writer {
	return &Writer{BaseDir: baseDir, fs: fs}
}

func (w *Writer) WriteAll(endpoints []*Endpoint) error {
	if err := w.fs.MkdirAll(w.BaseDir); err != nil {
		return err
	}

	activePaths := make(map[string]bool)
	for _, ep := range endpoints {
		cleanPath := strings.TrimPrefix(ep.Path, "/")
		methodFile := strings.ToLower(ep.Method) + ".yaml"
		dirPath := w.fs.Join(w.BaseDir, cleanPath)
		fullPath := w.fs.Join(dirPath, methodFile)

		activePaths[fullPath] = true

		if err := w.writeEndpoint(ep, dirPath, fullPath); err != nil {
			return err
		}
	}

	w.cleanOrphans(activePaths)
	return nil
}

func (w *Writer) writeEndpoint(ep *Endpoint, dirPath, fullPath string) error {
	fileExists := w.fs.Exists(fullPath)
	hasData := len(ep.Identities) > 0 || len(ep.Relatives) > 0

	if !hasData && !fileExists {
		return nil
	}

	var existingData []byte
	if fileExists {
		existingData, _ = w.fs.Read(fullPath)
	}

	newData := MarshalAST(ep, existingData)
	if len(newData) == 0 {
		newData = []byte("{}")
	}

	if err := w.fs.MkdirAll(dirPath); err != nil {
		return err
	}

	// Skip write if exactly identical (saves disk IO & modification time)
	if len(existingData) > 0 && bytes.Equal(newData, bytes.TrimSpace(existingData)) {
		return nil
	}

	return w.fs.Write(fullPath, newData)
}

func (w *Writer) cleanOrphans(activePaths map[string]bool) {
	w.fs.Walk(w.BaseDir, func(path string, isDir bool, err error) error {
		if !isDir && strings.HasSuffix(path, ".yaml") && !activePaths[path] {
			w.fs.Remove(path)
		}
		return nil
	})
	w.removeEmptyDirs(w.BaseDir)
}

func (w *Writer) removeEmptyDirs(dir string) bool {
	entries, err := w.fs.ReadDir(dir)
	if err != nil {
		return false
	}
	isEmpty := true
	for _, entry := range entries {
		if entry.IsDir {
			childDir := w.fs.Join(dir, entry.Name)
			if !w.removeEmptyDirs(childDir) {
				isEmpty = false
			}
		} else {
			isEmpty = false
		}
	}
	if isEmpty && dir != w.BaseDir {
		w.fs.Remove(dir)
	}
	return isEmpty
}
