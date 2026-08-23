package runtime

// Scope manages execution context and variables for a single VU iteration.
type Scope interface {
	Set(key string, val any)
	Get(key string) (any, bool)
	Delete(key string)
}

// SharedState handles cross-VU state sharing and data distribution.
type SharedState interface {
	Set(key string, val any)
	Get(key string) (any, bool)
	Push(queueName string, val any) bool
	Pop(queueName string) any
	StoreDistribution(key string, items []any, fallback any)
}

// MetricsSink receives metrics, logs, and lifecycle events from workers and hooks.
type MetricsSink interface {
	Log(workerID int, nodeID string, level string, msg string)
	RecordEvent(workerID int, nodeID string, eventType string, reason string)
	Tag(workerID int, key string, value string)
	RecordCustom(workerID int, mType string, name string, val float64)
}
