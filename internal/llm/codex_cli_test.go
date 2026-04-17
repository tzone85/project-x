package llm

import (
	"errors"
	"testing"
)

func TestNewCodexCLIClient(t *testing.T) {
	client := NewCodexCLIClient()
	if client.cliPath != "codex" {
		t.Errorf("cliPath = %q, want %q", client.cliPath, "codex")
	}
}

func TestNewCodexCLIClientWithPath(t *testing.T) {
	client := NewCodexCLIClientWithPath("/usr/local/bin/codex")
	if client.cliPath != "/usr/local/bin/codex" {
		t.Errorf("cliPath = %q, want %q", client.cliPath, "/usr/local/bin/codex")
	}
}

func TestCodexCLIClient_BuildArgs(t *testing.T) {
	client := NewCodexCLIClient()

	tests := []struct {
		name       string
		req        CompletionRequest
		outputPath string
		wantModel  bool
	}{
		{"no model", CompletionRequest{}, "/tmp/out.txt", false},
		{"with model", CompletionRequest{Model: "gpt-4o"}, "/tmp/out.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := client.buildArgs(tt.req, tt.outputPath)

			// Should always have: exec --skip-git-repo-check --sandbox read-only --color never --output-last-message <path> [-]
			if args[0] != "exec" {
				t.Errorf("first arg = %q, want %q", args[0], "exec")
			}

			// Last arg should be "-" (stdin prompt).
			if args[len(args)-1] != "-" {
				t.Errorf("last arg = %q, want %q", args[len(args)-1], "-")
			}

			hasModel := false
			for i, arg := range args {
				if arg == "--model" && i+1 < len(args) {
					hasModel = true
					if args[i+1] != tt.req.Model {
						t.Errorf("model arg = %q, want %q", args[i+1], tt.req.Model)
					}
				}
			}
			if hasModel != tt.wantModel {
				t.Errorf("hasModel = %v, want %v", hasModel, tt.wantModel)
			}
		})
	}
}

func TestClassifyCodexCLIError(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantCode  int
		wantRetry bool
	}{
		{"quota exceeded", "Exceeded your current quota, check your plan and billing", 429, false},
		{"billing issue", "Please check your billing settings", 429, false},
		{"auth error", "Authentication failed: not logged in", 401, false},
		{"unauthorized", "Unauthorized: invalid API key", 401, false},
		{"rate limit", "Rate limit exceeded, too many requests", 429, true},
		{"too many requests", "Too many requests, please wait", 429, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyCodexCLIError(errors.New("exit 1"), []byte(tt.output))
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected APIError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tt.wantCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantCode)
			}
			if apiErr.Retryable != tt.wantRetry {
				t.Errorf("Retryable = %v, want %v", apiErr.Retryable, tt.wantRetry)
			}
		})
	}
}

func TestClassifyCodexCLIError_UnknownError(t *testing.T) {
	err := classifyCodexCLIError(errors.New("exit 1"), []byte("some random failure"))
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Error("expected non-APIError for unknown failure")
	}
	if err == nil {
		t.Error("expected non-nil error")
	}
}

func TestContainsAny_LLM(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		parts []string
		want  bool
	}{
		{"match first", "hello world", []string{"hello"}, true},
		{"match second", "hello world", []string{"foo", "world"}, true},
		{"no match", "hello", []string{"foo", "bar"}, false},
		{"empty", "", []string{"x"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAny(tt.s, tt.parts...)
			if got != tt.want {
				t.Errorf("containsAny(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
