package config

import (
	"fmt"
	"os"

	"github.com/budment/budment/internal/filesystem"
	"gopkg.in/yaml.v3"
)

type Loader struct {
	fs       filesystem.FS
	filePath string
}

func NewLoader(path string, fs filesystem.FS) *Loader {
	if path == "" {
		path = "budment.yaml"
	}
	if fs == nil {
		fs = filesystem.NewLocal()
	}
	return &Loader{fs: fs, filePath: path}
}

func (l *Loader) Load() (EngineConfig, error) {
	cfg := DefaultEngineConfig()

	if !l.fs.Exists(l.filePath) {
		if l.filePath == "budment.yaml" {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config file not found: %s", l.filePath)
	}

	data, err := l.fs.Read(l.filePath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	expandedData := []byte(os.ExpandEnv(string(data)))

	if err := yaml.Unmarshal(expandedData, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}
