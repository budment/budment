package logger

import (
	"io"
	"log/slog"
	"os"
)

// Logger defines the logging interface used across Blaster.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	// With returns a logger containing additional context.
	With(args ...any) Logger
}

// Config controls logger behavior.
type Config struct {
	Verbose bool
	Quiet   bool
	JSON    bool
	Writer  io.Writer
}

type logger struct {
	slog *slog.Logger
}

// New creates a new logger.
func New(cfg Config) Logger {
	writer := cfg.Writer
	if writer == nil {
		// CLI tools should log to stderr.
		writer = os.Stderr
	}

	if cfg.Quiet {
		writer = io.Discard
	}

	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.JSON {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	return &logger{
		slog: slog.New(handler),
	}
}

func (l *logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

func (l *logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

func (l *logger) With(args ...any) Logger {
	return &logger{
		slog: l.slog.With(args...),
	}
}
