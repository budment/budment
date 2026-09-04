package metrics

import (
	"context"
	"sync/atomic"
	"time"
)

type MetricEvent struct {
	WorkerID    int
	NodeID      string
	ErrorMsg    string
	IsLogicFail bool
	IsCustom    bool
	CustomType  CustomMetricType
	CustomName  string
	CustomVal   float64
}

type Aggregator struct {
	eventChan     chan MetricEvent
	Metrics       *EngineMetrics
	DroppedEvents int64 // Measures events dropped under overload.
}

func NewAggregator(bufferSize int) *Aggregator {
	if bufferSize <= 0 {
		bufferSize = 100_000
	}
	return &Aggregator{
		eventChan: make(chan MetricEvent, bufferSize),
		Metrics:   NewEngineMetrics(),
	}
}

func (a *Aggregator) PushEvent(e MetricEvent) {
	select {
	case a.eventChan <- e:
	default:
		atomic.AddInt64(&a.DroppedEvents, 1)
	}
}

func (a *Aggregator) processEvent(event MetricEvent) {
	if event.IsCustom {
		a.Metrics.Custom.Record(event.CustomType, event.CustomName, event.CustomVal)
	} else {
		if event.ErrorMsg != "" {
			a.Metrics.RecordError(event.ErrorMsg)
		}
		if event.IsLogicFail {
			a.Metrics.RecordLogicFail()
		}
	}
}

func (a *Aggregator) Run(ctx context.Context, done chan struct{}) {
	defer close(done) // Signal completion to the Director

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastTotalRequests int64 = 0

	for {
		select {
		case <-ctx.Done():
			// Drain any remaining events from the channel
			for {
				select {
				case event := <-a.eventChan:
					a.processEvent(event)
				default:
					return // Safe exit when channel is empty
				}
			}

		case <-ticker.C:
			currentTotal := atomic.LoadInt64(&a.Metrics.TotalRequests)
			currentVUs := atomic.LoadInt64(&a.Metrics.ActiveVUs)

			reqsThisSecond := currentTotal - lastTotalRequests
			lastTotalRequests = currentTotal

			a.Metrics.RecordTick(float64(reqsThisSecond), currentVUs)

		case event := <-a.eventChan:
			a.processEvent(event)
		}
	}
}
