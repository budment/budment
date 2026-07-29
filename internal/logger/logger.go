package logger

import (
	"io"
	"log/slog"
	"os"
)

type Config struct {
	JSON    bool // JSON enables JSON formatting for log output
	Verbose bool // Verbose enables debug-level logging.
	Quiet   bool
	Out     io.Writer // Out specifies the writer to write logs to.
}

// New creates and configures a new *slog.Logger instance based on the provided Config.
func New(cfg Config) *slog.Logger {
	if cfg.Quiet {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	out := cfg.Out
	if out == nil {
		out = os.Stderr
	}

	// Determine the log level based on the Verbose flag.
	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Initialize the appropriate handler based on the JSON flag.
	var handler slog.Handler
	if cfg.JSON {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	return slog.New(handler)
}
