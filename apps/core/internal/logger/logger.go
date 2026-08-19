package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Gray   = "\033[90m"
)

type Config struct {
	JSON    bool // JSON enables JSON formatting for log output
	Verbose bool // Verbose enables debug-level logging.
	Quiet   bool
	Out     io.Writer // Out specifies the writer to write logs to.
}

type PrettyHandler struct {
	slog.Handler
	out io.Writer
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()
	color := Reset

	switch r.Level {
	case slog.LevelDebug:
		color = Gray
	case slog.LevelInfo:
		color = Blue
	case slog.LevelWarn:
		color = Yellow
	case slog.LevelError:
		color = Red
	}

	timeStr := r.Time.Format("15:04:05.000")

	var sb strings.Builder
	sb.Grow(128)

	// Format: [15:04:05] INFO  | Message
	sb.WriteString(Gray)
	sb.WriteString("[")
	sb.WriteString(timeStr)
	sb.WriteString("] ")
	sb.WriteString(color)
	sb.WriteString(fmt.Sprintf("%-5s", level))
	sb.WriteString(Reset)
	sb.WriteString(" | ")
	sb.WriteString(r.Message)
	sb.WriteString("\n")

	io.WriteString(h.out, sb.String())

	return nil
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

	opts := &slog.HandlerOptions{Level: level}

	if cfg.JSON {
		return slog.New(slog.NewJSONHandler(out, opts))
	}

	return slog.New(&PrettyHandler{
		Handler: slog.NewTextHandler(out, opts),
		out:     out,
	})
}
