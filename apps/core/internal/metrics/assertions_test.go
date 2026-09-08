package metrics

import (
	"testing"
)

func TestAssertions_ParseCondition_Variants(t *testing.T) {
	tests := []struct {
		input       string
		expectedSub string
		expectedOp  string
		expectedVal float64
		expectErr   bool
	}{
		{"p95 < 200ms", "p95", "<", 200.0, false},
		{"rate <= 0.01", "rate", "<=", 0.01, false},
		{"fail_rate < 5%", "fail_rate", "<", 0.05, false},
		{"latency_max <= 2s", "latency_max", "<=", 2000.0, false},
		{"requests >= 1000", "requests", ">=", 1000.0, false},
		{"invalid_no_operator", "", "", 0, true},
	}

	for _, tc := range tests {
		sub, op, val, err := parseCondition(tc.input)
		if tc.expectErr {
			if err == nil {
				t.Errorf("[%s] expected parse error, got nil", tc.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("[%s] unexpected parse error: %v", tc.input, err)
			continue
		}
		if sub != tc.expectedSub || op != tc.expectedOp || val != tc.expectedVal {
			t.Errorf("[%s] parse mismatch: got (%s, %s, %f), expected (%s, %s, %f)",
				tc.input, sub, op, val, tc.expectedSub, tc.expectedOp, tc.expectedVal)
		}
	}
}

func TestAssertions_EvaluateThresholds_PassAndBreach(t *testing.T) {
	m := NewEngineMetrics()

	// Simulate test execution: 90 successes, 10 failures (10% failure rate)
	for i := 0; i < 90; i++ {
		m.RecordRequest("n1", "GET", "api", true, 50_000, 10_000, 0, 0, 100, 200, 200) // 50ms
	}
	for i := 0; i < 10; i++ {
		m.RecordRequest("n1", "GET", "api", false, 150_000, 20_000, 0, 0, 100, 0, 500) // 150ms
	}

	// Case 1: Threshold breached (expected failure rate < 5%, actual 10%)
	thresholdsBreached := map[string]string{
		"http_req_failed": "< 5%",
	}
	mgr1 := NewAssertionManager(thresholdsBreached)
	err := mgr1.EvaluateThresholds(m)
	if err == nil {
		t.Fatal("expected threshold evaluation to fail due to 10% error rate, got nil")
	}
	if !mgr1.HasFailures() || len(mgr1.GetFailures()) != 1 {
		t.Fatalf("expected 1 recorded failure, got %v", mgr1.GetFailures())
	}

	// Case 2: Thresholds satisfied (expected failure rate <= 15% and requests >= 100)
	thresholdsPassed := map[string]string{
		"http_req_failed": "<= 15%",
		"requests":        ">= 100",
	}
	mgr2 := NewAssertionManager(thresholdsPassed)
	err = mgr2.EvaluateThresholds(m)
	if err != nil {
		t.Fatalf("expected all thresholds to pass, got error: %v", err)
	}
	if mgr2.HasFailures() {
		t.Fatalf("unexpected failures recorded: %v", mgr2.GetFailures())
	}
}
