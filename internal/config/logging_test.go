package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"unknown", slog.LevelInfo}, // defaults to info
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LogLevelFromString(tt.input)
			if got != tt.expected {
				t.Errorf("LogLevelFromString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSetupLogging(t *testing.T) {
	dir := t.TempDir()

	cleanup, err := SetupLogging("info", dir)
	if err != nil {
		t.Fatalf("SetupLogging failed: %v", err)
	}
	defer cleanup()

	// Verify log directory was created
	logDir := filepath.Join(dir, "logs")
	info, err := os.Stat(logDir)
	if err != nil {
		t.Fatalf("log directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("logs path is not a directory")
	}

	// Write a log entry and verify it appears in the file
	slog.Info("test message", "key", "value")

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("reading log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no log files created")
	}

	logContent, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	// Verify JSON format
	var logEntry map[string]any
	if err := json.Unmarshal(logContent, &logEntry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\ncontent: %s", err, logContent)
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg='test message', got %v", logEntry["msg"])
	}
}

func TestSetupLoggingInvalidDir(t *testing.T) {
	_, err := SetupLogging("info", "/nonexistent/readonly/path")
	if err == nil {
		t.Fatal("expected error for invalid log directory")
	}
}

func TestComponentLogger(t *testing.T) {
	dir := t.TempDir()
	cleanup, err := SetupLogging("debug", dir)
	if err != nil {
		t.Fatalf("SetupLogging failed: %v", err)
	}
	defer cleanup()

	logger := ComponentLogger("pipeline")
	logger.Info("stage complete", "stage", "review")

	// Read the log file and check for component tag
	logDir := filepath.Join(dir, "logs")
	entries, err := os.ReadDir(logDir)
	if err != nil || len(entries) == 0 {
		t.Fatal("no log files found")
	}

	logContent, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	// Log file may have multiple lines; check the last one
	lines := splitNonEmpty(string(logContent))
	lastLine := lines[len(lines)-1]

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(lastLine), &logEntry); err != nil {
		t.Fatalf("log line is not valid JSON: %v", err)
	}

	if logEntry["component"] != "pipeline" {
		t.Errorf("expected component=pipeline, got %v", logEntry["component"])
	}
}

func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range splitLines(s) {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
