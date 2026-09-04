package metrics

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
)

type ThresholdResult struct {
	Metric   string
	Criteria string
	Actual   float64
	Passed   bool
	Reason   string
}

type AssertionManager struct {
	Thresholds map[string]string
	failures   []string
	results    []ThresholdResult
	evaluated  bool
}

func NewAssertionManager(thresholds map[string]string) *AssertionManager {
	if thresholds == nil {
		thresholds = make(map[string]string)
	}
	return &AssertionManager{
		Thresholds: thresholds,
		failures:   make([]string, 0),
		results:    make([]ThresholdResult, 0),
	}
}

func (a *AssertionManager) HasFailures() bool {
	return len(a.failures) > 0
}

func (a *AssertionManager) GetFailures() []string {
	return a.failures
}

func (a *AssertionManager) GetResults() []ThresholdResult {
	return a.results
}

func (a *AssertionManager) EvaluateThresholds(m *EngineMetrics) error {
	a.failures = a.failures[:0]
	a.results = a.results[:0]
	a.evaluated = true

	totalReqs, _, _, netFail, logicFail, _, _, _ := m.Snapshot()

	if totalReqs == 0 {
		errStr := "no requests were executed during the test"
		a.failures = append(a.failures, errStr)
		return fmt.Errorf("%s", errStr)
	}

	failRate := float64(netFail+logicFail) / float64(totalReqs)

	for rawMetric, rawCondition := range a.Thresholds {
		metricName := strings.ToLower(strings.TrimSpace(rawMetric))
		condStr := strings.TrimSpace(rawCondition)

		subMetric, op, targetVal, err := parseCondition(condStr)
		if err != nil {
			failMsg := fmt.Sprintf("[%s] invalid threshold format '%s': %v", rawMetric, rawCondition, err)
			a.failures = append(a.failures, failMsg)
			continue
		}

		effectiveMetric := metricName
		if subMetric != "" && subMetric != "rate" && subMetric != "value" {
			effectiveMetric = subMetric
		}

		var actualVal float64
		var found bool = true

		switch effectiveMetric {
		case "fail_rate", "error_rate", "http_req_failed":
			actualVal = failRate

		case "p50", "latency_p50":
			actualVal = float64(m.RequestDuration.Percentile(50))
		case "p90", "latency_p90":
			actualVal = float64(m.RequestDuration.Percentile(90))
		case "p95", "latency_p95", "http_req_duration":
			actualVal = float64(m.RequestDuration.Percentile(95))
		case "p99", "latency_p99":
			actualVal = float64(m.RequestDuration.Percentile(99))
		case "max", "latency_max":
			actualVal = float64(m.RequestDuration.Max())
		case "min", "latency_min":
			actualVal = float64(m.RequestDuration.Min())

		case "iterations":
			actualVal = float64(atomic.LoadInt64(&m.TotalIterations))
		case "requests":
			actualVal = float64(totalReqs)

		default:
			if customVal, ok := m.Custom.Get(effectiveMetric); ok {
				actualVal = customVal
			} else {
				found = false
			}
		}

		if !found {
			failMsg := fmt.Sprintf("[%s] metric not found or unsupported", rawMetric)
			a.failures = append(a.failures, failMsg)
			a.results = append(a.results, ThresholdResult{
				Metric:   rawMetric,
				Criteria: rawCondition,
				Actual:   0,
				Passed:   false,
				Reason:   "metric not found",
			})
			continue
		}

		passed := evaluateOperator(actualVal, op, targetVal)
		res := ThresholdResult{
			Metric:   rawMetric,
			Criteria: rawCondition,
			Actual:   actualVal,
			Passed:   passed,
		}

		if !passed {
			if metricName == "http_req_failed" || metricName == "fail_rate" {
				res.Reason = fmt.Sprintf("breached: actual %.4f (%.2f%%) does not satisfy %s", actualVal, actualVal*100, rawCondition)
			} else {
				res.Reason = fmt.Sprintf("breached: actual %.2f does not satisfy %s", actualVal, rawCondition)
			}
			a.failures = append(a.failures, fmt.Sprintf("[%s] %s", rawMetric, res.Reason))
		}
		a.results = append(a.results, res)
	}

	if len(a.failures) > 0 {
		return fmt.Errorf("threshold check failed with %d violations", len(a.failures))
	}
	return nil
}

func parseCondition(cond string) (subMetric string, op string, val float64, err error) {
	cond = strings.TrimSpace(cond)
	operators := []string{"<=", ">=", "==", "!=", "<", ">"}
	opIdx := -1
	for _, o := range operators {
		if idx := strings.Index(cond, o); idx != -1 {
			if opIdx == -1 || idx < opIdx {
				opIdx = idx
				op = o
			}
		}
	}

	if opIdx == -1 {
		return "", "", 0, fmt.Errorf("missing comparison operator (expected <, <=, >, >=, ==, !=)")
	}

	subMetric = strings.ToLower(strings.TrimSpace(cond[:opIdx]))
	valStr := strings.TrimSpace(cond[opIdx+len(op):])

	if strings.HasSuffix(valStr, "%") {
		numStr := strings.TrimSpace(strings.TrimSuffix(valStr, "%"))
		v, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return "", "", 0, fmt.Errorf("invalid percentage value '%s'", valStr)
		}
		return subMetric, op, v / 100.0, nil
	}

	multiplier := 1.0
	if strings.HasSuffix(valStr, "ms") {
		valStr = strings.TrimSuffix(valStr, "ms")
	} else if strings.HasSuffix(valStr, "s") {
		valStr = strings.TrimSuffix(valStr, "s")
		multiplier = 1000.0
	} else if strings.HasSuffix(valStr, "m") {
		valStr = strings.TrimSuffix(valStr, "m")
		multiplier = 60000.0
	}

	val, err = strconv.ParseFloat(strings.TrimSpace(valStr), 64)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid numeric value '%s'", valStr)
	}
	return subMetric, op, val * multiplier, nil
}

func evaluateOperator(actual float64, op string, target float64) bool {
	switch op {
	case "<":
		return actual < target
	case "<=":
		return actual <= target
	case ">":
		return actual > target
	case ">=":
		return actual >= target
	case "==":
		return actual == target
	case "!=":
		return actual != target
	default:
		return false
	}
}
