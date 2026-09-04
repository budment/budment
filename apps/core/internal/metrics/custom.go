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
	Type      CustomMetricType `json:"type"`
	Count     int64            `json:"count"`
	Sum       float64          `json:"sum"`
	Min       float64          `json:"min"`
	Max       float64          `json:"max"`
	Last      float64          `json:"last"`
	Histogram *AtomicHistogram `json:"-"`
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
		m = &CustomMetric{
			Type: mType,
			Min:  val,
			Max:  val,
		}
		if mType == TypeTrend {
			m.Histogram = NewAtomicHistogram()
		}
		s.metrics[name] = m
	}

	m.Count++
	m.Sum += val
	m.Last = val

	if val < m.Min {
		m.Min = val
	}
	if val > m.Max {
		m.Max = val
	}

	if mType == TypeTrend && m.Histogram != nil {
		m.Histogram.Record(int64(val))
	}
}

func (s *CustomMetricsStore) Get(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, exists := s.metrics[name]
	if !exists {
		return 0, false
	}

	switch m.Type {
	case TypeCounter:
		return m.Sum, true
	case TypeGauge:
		return m.Last, true
	case TypeTrend:
		if m.Count > 0 {
			return m.Sum / float64(m.Count), true
		}
		return 0, true
	default:
		return m.Last, true
	}
}

func (s *CustomMetricsStore) GetPercentile(name string, p float64) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, exists := s.metrics[name]
	if !exists || m.Type != TypeTrend || m.Histogram == nil {
		return 0, false
	}
	return m.Histogram.Percentile(p), true
}

func (s *CustomMetricsStore) GetAll() map[string]CustomMetric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]CustomMetric, len(s.metrics))
	for k, v := range s.metrics {
		res[k] = *v
	}
	return res
}
