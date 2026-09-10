package cli

import (
	"context"
	"testing"

	"github.com/budment/budment/internal/metrics"
	"github.com/budment/budment/internal/tui"
)

func TestCLIMetricsSink_Log_WithoutDashboard(t *testing.T) {
	agg := metrics.NewAggregator(100, nil)
	sink := &cliMetricsSink{agg: agg, dashboard: nil}

	// Ensure log calls across levels do not panic when dashboard is nil
	sink.Log(1, "node_setup", "INFO", "Informational status message")
	sink.Log(1, "node_http", "WARN", "Warning threshold approached")
	sink.Log(1, "node_http", "ERROR", "Critical response violation")
	sink.Log(1, "node_http", "SYS_ERR", "Dial tcp timeout occurred")
}

func TestCLIMetricsSink_Log_WithDashboardBuffer(t *testing.T) {
	agg := metrics.NewAggregator(100, nil)
	dash := &tui.LiveDashboard{
		LogChan: make(chan string, 2),
	}
	sink := &cliMetricsSink{agg: agg, dashboard: dash}

	// SYS_ERR must be suppressed when dashboard is active to prevent UI clutter
	sink.Log(2, "node_err", "SYS_ERR", "Internal system failure")
	select {
	case msg := <-dash.LogChan:
		t.Fatalf("Expected SYS_ERR to be suppressed in TUI dashboard, but got: %s", msg)
	default:
	}

	// INFO must be pushed directly into LogChan
	sink.Log(2, "node_ok", "INFO", "Dashboard visible entry")
	select {
	case <-dash.LogChan:
		// Successfully received log
	default:
		t.Fatal("Expected log entry in dashboard LogChan, but channel was empty")
	}
}

func TestCLIMetricsSink_RecordEvent_LogicFailForwarding(t *testing.T) {
	agg := metrics.NewAggregator(100, nil)
	ctx, cancel := context.WithCancel(context.Background())
	aggDone := make(chan struct{})
	go agg.Run(ctx, aggDone)

	sink := &cliMetricsSink{agg: agg, dashboard: nil}

	// Record a business logic FAIL event
	sink.RecordEvent(3, "auth_node", "FAIL", "Invalid JWT payload")

	// Stop Aggregator to flush the event queue into EngineMetrics
	cancel()
	<-aggDone

	_, _, _, _, logicFail, _, _, _ := agg.Metrics.Snapshot()
	if logicFail != 1 {
		t.Errorf("Expected logicFail count 1, got %d", logicFail)
	}

	topErrors := agg.Metrics.GetTopErrors()
	expectedErr := "[FAIL] Invalid JWT payload"
	if count := topErrors[expectedErr]; count != 1 {
		t.Errorf("Expected error '%s' with count 1, got %d", expectedErr, count)
	}
}

func TestCLIMetricsSink_RecordCustom(t *testing.T) {
	agg := metrics.NewAggregator(100, nil)
	ctx, cancel := context.WithCancel(context.Background())
	aggDone := make(chan struct{})
	go agg.Run(ctx, aggDone)

	sink := &cliMetricsSink{agg: agg, dashboard: nil}
	sink.RecordCustom(5, "counter", "orders_placed", 1.0)

	cancel()
	<-aggDone

	val, ok := agg.Metrics.Custom.Get("orders_placed")
	if !ok {
		t.Fatal("Expected custom metric 'orders_placed' to be recorded in store")
	}
	if val != 1.0 {
		t.Errorf("Custom metric mismatch: expected 1.0, got %f", val)
	}
}
