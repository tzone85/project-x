package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Setup initializes the global slog logger with the given level and optional log file.
// It writes JSON to both stderr and the log file (if logDir is non-empty).
// Returns a cleanup function to close the log file.
func Setup(level string, logDir string) (func(), error) {
	logLevel := parseLevel(level)

	writers := []io.Writer{os.Stderr}

	var logFile *os.File
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return func() {}, fmt.Errorf("create log dir: %w", err)
		}
		path := filepath.Join(logDir, "px.log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return func() {}, fmt.Errorf("open log file: %w", err)
		}
		writers = append(writers, f)
		logFile = f
	}

	w := io.MultiWriter(writers...)
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(handler))

	cleanup := func() {
		if logFile != nil {
			logFile.Close()
		}
	}
	return cleanup, nil
}

// ForComponent returns a child logger tagged with the given component name.
func ForComponent(component string) *slog.Logger {
	return slog.Default().With("component", component)
}

// WithStory returns a child logger tagged with both component and story ID.
func WithStory(component, storyID string) *slog.Logger {
	return slog.Default().With("component", component, "story_id", storyID)
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
