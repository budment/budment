package exporters

import "github.com/vunas/blaster/internal/metrics"

type Reporter interface {
	Export(engine *metrics.EngineMetrics, assert *metrics.AssertionManager) error
}
