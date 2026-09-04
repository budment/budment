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

	totalFail := netFail + logicFail
	failRate := (float64(totalFail) / float64(totalReqs)) * 100.0

	for rawMetric, rawCondition := range a.Thresholds {
		metricName := strings.ToLower(strings.TrimSpace(rawMetric))
		condStr := strings.TrimSpace(rawCondition)

		isRateMetric := metricName == "fail_rate" || metricName == "error_rate" || metricName == "http_req_failed"
		isDurationMetric := strings.Contains(metricName, "p") || strings.Contains(metricName, "latency") ||
			strings.Contains(metricName, "duration") || metricName == "min" || metricName == "max"

		op, targetVal, err := parseConditionWithContext(condStr, isRateMetric, isDurationMetric)
		if err != nil {
			failMsg := fmt.Sprintf("invalid threshold format for '%s': %s", rawMetric, rawCondition)
			a.failures = append(a.failures, failMsg)
			continue
		}

		var actualVal float64
		switch metricName {
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
			if customVal, ok := m.Custom.Get(metricName); ok {
				actualVal = customVal
			} else {
				continue
			}
		}

		passed := evaluateOperator(actualVal, op, targetVal)
		res := ThresholdResult{
			Metric:   rawMetric,
			Criteria: rawCondition,
			Actual:   actualVal,
			Passed:   passed,
		}

		if !passed {
			res.Reason = fmt.Sprintf("breached: actual %.2f does not satisfy %s", actualVal, rawCondition)
			a.failures = append(a.failures, fmt.Sprintf("[%s] %s", rawMetric, res.Reason))
		}
		a.results = append(a.results, res)
	}

	if len(a.failures) > 0 {
		return fmt.Errorf("threshold check failed with %d violations", len(a.failures))
	}
	return nil
}

func parseConditionWithContext(cond string, isRateMetric bool, isDurationMetric bool) (op string, val float64, err error) {
	cond = strings.TrimSpace(cond)
	operators := []string{"<=", ">=", "!=", "==", "<", ">"}

	for _, o := range operators {
		if strings.HasPrefix(cond, o) {
			op = o
			valStr := strings.TrimSpace(strings.TrimPrefix(cond, o))

			if isRateMetric {
				if strings.HasSuffix(valStr, "%") {
					valStr = strings.TrimSuffix(valStr, "%")
					val, err = strconv.ParseFloat(strings.TrimSpace(valStr), 64)
					return op, val, err
				}
				val, err = strconv.ParseFloat(valStr, 64)
				if err == nil && val <= 1.0 && val > 0 {
					val = val * 100.0
				}
				return op, val, err
			}

			multiplier := 1.0
			if isDurationMetric {
				if strings.HasSuffix(valStr, "ms") {
					valStr = strings.TrimSuffix(valStr, "ms")
				} else if strings.HasSuffix(valStr, "s") {
					valStr = strings.TrimSuffix(valStr, "s")
					multiplier = 1000.0
				} else if strings.HasSuffix(valStr, "m") {
					valStr = strings.TrimSuffix(valStr, "m")
					multiplier = 60000.0
				}
			}

			val, err = strconv.ParseFloat(strings.TrimSpace(valStr), 64)
			val = val * multiplier
			return op, val, err
		}
	}
	return "", 0, fmt.Errorf("unknown operator in '%s'", cond)
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
