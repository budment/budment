package exporters

import (
	"strings"

	"github.com/budment/budment/internal/metrics"
)

type Reporter interface {
	Export(engine *metrics.EngineMetrics, assert *metrics.AssertionManager) error
}

func extractPath(rawURL string) string {
	path := rawURL
	if idx := strings.Index(path, "://"); idx != -1 {
		if slashIdx := strings.Index(path[idx+3:], "/"); slashIdx != -1 {
			path = path[idx+3+slashIdx:]
		} else {
			path = "/"
		}
	}
	return path
}
