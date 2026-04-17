package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetup_StderrOnly(t *testing.T) {
	cleanup, err := Setup("info", "")
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer cleanup()

	// Should not panic when logging.
	slog.Info("test message from stderr-only setup")
}

func TestSetup_WithLogDir(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	cleanup, err := Setup("debug", logDir)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Log a message to force write.
	slog.Info("test message with log file")

	cleanup()

	// Verify log file was created.
	logFile := filepath.Join(logDir, "px.log")
	info, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("log file is empty, expected content")
	}
}

func TestSetup_InvalidLogDir(t *testing.T) {
	// Create a file where the log dir should be, making MkdirAll fail.
	dir := t.TempDir()
	blockingFile := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	invalidDir := filepath.Join(blockingFile, "logs")
	_, err := Setup("info", invalidDir)
	if err == nil {
		t.Fatal("expected error for invalid log dir")
	}
}

func TestForComponent(t *testing.T) {
	logger := ForComponent("test-component")
	if logger == nil {
		t.Fatal("ForComponent returned nil")
	}
}

func TestWithStory(t *testing.T) {
	logger := WithStory("pipeline", "STR-001")
	if logger == nil {
		t.Fatal("WithStory returned nil")
	}
}
