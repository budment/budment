package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPlan_ValidSingleScenario(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "valid_scenario.js")

	jsContent := `
		export const options = {
			vus: 10,
			duration: "250ms",
			iterations: 50,
			order: 2
		};
		export default [
			sleep(0.01),
			log("Plan compilation step verified")
		];
	`
	if err := os.WriteFile(scriptPath, []byte(jsContent), 0o600); err != nil {
		t.Fatalf("failed to create temporary script file: %v", err)
	}

	planResult, err := BuildPlan(scriptPath)
	if err != nil {
		t.Fatalf("BuildPlan failed unexpectedly: %v", err)
	}

	if planResult == nil {
		t.Fatal("expected non-nil PlanResult")
	}

	if len(planResult.Scenarios) != 1 {
		t.Fatalf("compiled scenarios count mismatch: expected 1, got %d", len(planResult.Scenarios))
	}

	scn := planResult.Scenarios[0]
	if scn.Graph == nil {
		t.Fatal("expected non-nil planner.Graph in compiled scenario")
	}

	if planResult.ASTConfig.VUs == nil || *planResult.ASTConfig.VUs != 10 {
		t.Errorf("ASTConfig VUs mismatch: expected 10, got %v", planResult.ASTConfig.VUs)
	}

	if planResult.ASTConfig.Duration == nil || *planResult.ASTConfig.Duration != "250ms" {
		t.Errorf("ASTConfig Duration mismatch: expected '250ms', got %v", planResult.ASTConfig.Duration)
	}

	if planResult.ASTConfig.Iterations == nil || *planResult.ASTConfig.Iterations != 50 {
		t.Errorf("ASTConfig Iterations mismatch: expected 50, got %v", planResult.ASTConfig.Iterations)
	}

	if planResult.ASTConfig.Order == nil || *planResult.ASTConfig.Order != 2 {
		t.Errorf("ASTConfig Order mismatch: expected 2, got %v", planResult.ASTConfig.Order)
	}

	if len(planResult.RawASTJSON) == 0 {
		t.Error("expected RawASTJSON to contain serialized JSON data")
	}

	var rawVerify []map[string]any
	if err := json.Unmarshal(planResult.RawASTJSON, &rawVerify); err != nil {
		t.Errorf("RawASTJSON is not valid JSON array: %v", err)
	}

	if len(planResult.JSBundle) == 0 {
		t.Error("expected non-empty JSBundle from bundler")
	}
}

func TestBuildPlan_FileNotFound(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "ghost_script.ts")

	_, err := BuildPlan(nonExistentPath)
	if err == nil {
		t.Fatal("expected error when target script file does not exist, got nil")
	}
}

func TestBuildPlan_InvalidSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	brokenScript := filepath.Join(tmpDir, "syntax_error.ts")

	brokenContent := `
		export default [
			const broken = ;;;;
		];
	`
	if err := os.WriteFile(brokenScript, []byte(brokenContent), 0o600); err != nil {
		t.Fatalf("failed to write broken script: %v", err)
	}

	_, err := BuildPlan(brokenScript)
	if err == nil {
		t.Fatal("expected bundler syntax compilation error, got nil")
	}
}

func TestBuildPlan_MultiScenarioFallbackConfig(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "multi_scenario.js")

	jsContent := `
		export const options = {
			vus: 5,
			duration: "100ms"
		};
		export const scenario_a = [
			sleep(0.01)
		];
		export const scenario_b = [
			sleep(0.02)
		];
	`
	if err := os.WriteFile(scriptPath, []byte(jsContent), 0o600); err != nil {
		t.Fatalf("failed to write multi-scenario script: %v", err)
	}

	planResult, err := BuildPlan(scriptPath)
	if err != nil {
		t.Fatalf("BuildPlan failed for multi-scenario bundle: %v", err)
	}

	if len(planResult.Scenarios) < 2 {
		t.Fatalf("compiled scenarios count mismatch: expected at least 2, got %d", len(planResult.Scenarios))
	}

	if planResult.ASTConfig.VUs == nil || *planResult.ASTConfig.VUs != 5 {
		t.Errorf("fallback ASTConfig VUs mismatch: expected 5, got %v", planResult.ASTConfig.VUs)
	}
}
