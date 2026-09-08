package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestPlanCmd_MissingScript(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"plan", "non_existent_file.ts"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	quietMode = true
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when targeting a non-existent script file, but got nil")
	}
}

func TestPlanCmd_FlagOverrides(t *testing.T) {
	// Create a minimal dummy script file
	tmpFile, err := os.CreateTemp("", "dummy_script_*.ts")
	if err != nil {
		t.Fatalf("Failed to create temporary script file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, _ = tmpFile.WriteString(`export default [];`)
	_ = tmpFile.Close()

	cmd := rootCmd
	cmd.SetArgs([]string{
		"plan", tmpFile.Name(),
		"--vus", "25",
		"--duration", "45s",
		"--json",
	})

	quietMode = true
	jsonMode = true
	defer func() {
		quietMode = false
		jsonMode = false
	}()

	_ = cmd.Execute()
	if cliVUs != 25 {
		t.Errorf("Expected cliVUs override to be 25, got %d", cliVUs)
	}
	if cliDuration != "45s" {
		t.Errorf("Expected cliDuration override to be '45s', got '%s'", cliDuration)
	}
}
