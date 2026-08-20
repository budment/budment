package endpoint

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"

	filesystem "github.com/vunas/blaster/internal/filesystem"
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
	if len(parts) < 2 {
		return nil
	}

	protocol := parts[0]
	fileName := parts[len(parts)-1]

	dir := "/"
	if len(parts) > 2 {
		dir = "/" + strings.Join(parts[1:len(parts)-1], "/")
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

	ep := &Endpoint{Protocol: protocol, Method: method, Path: dir}
	r.walkYamlMap(raw, []string{}, ep)
	return ep
}

func (r *Reader) normalizeTargetID(raw string, currentProtocol string) string {
	if raw == "?" || raw == "ignore" || raw == "identity" || raw == "" {
		return raw
	}

	parts := strings.SplitN(raw, ":", 2)
	if len(parts) > 1 {
		prefix := parts[0]
		switch prefix {
		case "rest", "grpc", "graphql", "kafka", "websocket", "webhook":
			return raw
		}
	}

	return currentProtocol + ":" + raw
}

func (r *Reader) walkYamlMap(node map[string]any, currentPath []string, ep *Endpoint) {
	for key, val := range node {
		if len(currentPath) == 0 && (key == "protocol" || key == "method" || key == "path") {
			continue
		}

		newPath := append(append([]string(nil), currentPath...), key)

		switch v := val.(type) {
		case map[string]any:
			if mapTarget, ok := v["map"].(string); ok {
				r.processLeafNode(newPath, mapTarget, ep)
			} else {
				r.walkYamlMap(v, newPath, ep)
			}
		case string:
			r.processLeafNode(newPath, v, ep)
		}
	}
}

// Determines whether the node is an Identity (Producer) or Relative (Consumer).
func (r *Reader) processLeafNode(path []string, value string, ep *Endpoint) {
	if len(path) < 2 {
		return
	}

	rootCategory := path[0]
	name := path[len(path)-1]
	nodePath := path[1:]

	switch rootCategory {
	case "responses":
		ident := Identity{NodePath: nodePath, Name: name}
		if value == "?" {
			ident.Status = StatusPending
		} else if value == "ignore" {
			ident.Status = StatusIgnored
		} else if value != "identity" { // Contains a Branch mapping
			ident.Status = StatusResolved
			ident.TargetID = r.normalizeTargetID(value, ep.Protocol)
		} else {
			//'identity' -> Root
			ident.Status = StatusResolved
		}
		ep.Identities = append(ep.Identities, ident)

	case "request":
		rel := Relative{NodePath: nodePath, Name: name}
		switch value {
		case "?":
			rel.Status = StatusPending
		case "ignore":
			rel.Status = StatusIgnored
		default:
			rel.Status = StatusResolved
			rel.TargetID = r.normalizeTargetID(value, ep.Protocol)
		}
		ep.Relatives = append(ep.Relatives, rel)
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

		dirPath := w.fs.Join(w.BaseDir, ep.Protocol, cleanPath)
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

// (w *Writer) WriteAll, writeEndpoint, cleanOrphans...
