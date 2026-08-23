package theme

import (
	"os"
	"strings"
)

var (
	// Indicates whether the current terminal supports ANSI color output.
	ColorEnabled = true
)

func init() {
	DetectEnvironment()
}

// Checks terminal capability flags and disables color support if unsupported.
func DetectEnvironment() {
	// Respect NO_COLOR standard (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		ColorEnabled = false
		return
	}

	// Disable color in non-interactive CI/CD pipelines
	if os.Getenv("CI") != "" || strings.ToLower(os.Getenv("CI")) == "true" {
		ColorEnabled = false
		return
	}

	// Dumb terminals do not support ANSI escape codes
	if os.Getenv("TERM") == "dumb" {
		ColorEnabled = false
		return
	}

	ColorEnabled = true
}
