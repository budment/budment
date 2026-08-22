package metrics

import (
	"fmt"
	"strconv"
	"strings"
)

type AssertionManager struct {
	Thresholds map[string]string
}

func NewAssertionManager(thresholds map[string]string) *AssertionManager {
	if thresholds == nil {
		thresholds = make(map[string]string)
	}
	return &AssertionManager{
		Thresholds: thresholds,
	}
}

func (a *AssertionManager) EvaluateThresholds(metrics *EngineMetrics) error {
	total, _, _, netFail, logicFail, _, _, _ := metrics.Snapshot()

	if total == 0 {
		return fmt.Errorf("no requests were executed")
	}

	fail := netFail + logicFail
	failRate := float64(fail) / float64(total) * 100

	for key, condition := range a.Thresholds {
		if key == "fail_rate" || key == "error_rate" {
			condition = strings.TrimSpace(condition)

			if strings.HasPrefix(condition, "<") {
				valStr := strings.TrimSpace(strings.TrimPrefix(condition, "<"))
				if maxLimit, err := strconv.ParseFloat(valStr, 64); err == nil {
					if failRate > maxLimit {
						return fmt.Errorf("threshold broken: %s is %.2f%% (allowed: %s)", key, failRate, condition)
					}
				}
			} else if strings.HasPrefix(condition, ">") {
				valStr := strings.TrimSpace(strings.TrimPrefix(condition, ">"))
				if minLimit, err := strconv.ParseFloat(valStr, 64); err == nil {
					if failRate < minLimit {
						return fmt.Errorf("threshold broken: %s is %.2f%% (requires: %s)", key, failRate, condition)
					}
				}
			}
		}
	}

	if logicFail > 0 {
		return fmt.Errorf("%d logic assertions (expect or abort) failed", logicFail)
	}

	return nil
}
