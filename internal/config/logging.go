package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogLevelFromString converts a string log level to slog.Level.
func LogLevelFromString(level string) slog.Level {
	switch strings.ToLower(level) {
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

// SetupLogging configures the global slog logger with JSON output to both
// stderr and a daily-rotated log file at <dataDir>/logs/px.log.
// Returns a cleanup function that closes the log file.
func SetupLogging(level string, dataDir string) (func(), error) {
	slogLevel := LogLevelFromString(level)

	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	logFileName := fmt.Sprintf("px-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, logFileName)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	writer := io.MultiWriter(os.Stderr, logFile)

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: slogLevel,
	})

	slog.SetDefault(slog.New(handler))

	cleanup := func() {
		logFile.Close()
	}

	return cleanup, nil
}

// ComponentLogger returns a logger tagged with the given component name.
func ComponentLogger(component string) *slog.Logger {
	return slog.Default().With("component", component)
}
