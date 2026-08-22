package metrics

import (
	"sync"
)

type CustomMetricType string

const (
	TypeCounter CustomMetricType = "counter"
	TypeTrend   CustomMetricType = "trend"
	TypeGauge   CustomMetricType = "gauge"
)

type CustomMetric struct {
	Type  CustomMetricType
	Count int64
	Sum   float64
	Min   float64
	Max   float64
	Last  float64
}

type CustomMetricsStore struct {
	mu      sync.RWMutex
	metrics map[string]*CustomMetric
}

func NewCustomMetricsStore() *CustomMetricsStore {
	return &CustomMetricsStore{
		metrics: make(map[string]*CustomMetric),
	}
}

func (s *CustomMetricsStore) Record(mType CustomMetricType, name string, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, exists := s.metrics[name]
	if !exists {
		m = &CustomMetric{Type: mType, Min: val, Max: val}
		s.metrics[name] = m
	}

	m.Count++
	m.Sum += val
	m.Last = val

	if mType == TypeTrend {
		if val < m.Min {
			m.Min = val
		}
		if val > m.Max {
			m.Max = val
		}
	}
}

func (s *CustomMetricsStore) GetAll() map[string]CustomMetric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]CustomMetric)
	for k, v := range s.metrics {
		res[k] = *v
	}
	return res
}
