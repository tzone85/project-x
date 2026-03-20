package git

import (
	"context"
	"fmt"
	"strings"
)

// MockRunner records calls and returns predefined responses for testing.
type MockRunner struct {
	Calls    []MockCall
	Stubs    map[string]MockResponse
	FallBack *ExecRunner
}

// MockCall records a single command invocation.
type MockCall struct {
	Dir     string
	Command string
	Args    []string
}

// MockResponse is a predefined response for a command pattern.
type MockResponse struct {
	Output string
	Err    error
}

// NewMockRunner creates a mock runner with the given stubs.
func NewMockRunner() *MockRunner {
	return &MockRunner{
		Stubs: make(map[string]MockResponse),
	}
}

// Stub registers a response for a command pattern (command + args joined by space).
func (m *MockRunner) Stub(pattern string, output string, err error) {
	m.Stubs[pattern] = MockResponse{Output: output, Err: err}
}

// Run records the call and returns the stubbed response.
func (m *MockRunner) Run(_ context.Context, dir, command string, args ...string) (string, error) {
	call := MockCall{Dir: dir, Command: command, Args: args}
	m.Calls = append(m.Calls, call)

	key := command + " " + strings.Join(args, " ")

	// Try exact match first
	if resp, ok := m.Stubs[key]; ok {
		return resp.Output, resp.Err
	}

	// Try prefix match
	for pattern, resp := range m.Stubs {
		if strings.HasPrefix(key, pattern) {
			return resp.Output, resp.Err
		}
	}

	if m.FallBack != nil {
		return m.FallBack.Run(context.Background(), dir, command, args...)
	}

	return "", &CommandError{
		Command: key,
		Err:     fmt.Errorf("no stub for: %s", key),
	}
}

// Called returns true if a command pattern was called.
func (m *MockRunner) Called(pattern string) bool {
	for _, c := range m.Calls {
		key := c.Command + " " + strings.Join(c.Args, " ")
		if strings.Contains(key, pattern) {
			return true
		}
	}
	return false
}

// CallCount returns the number of times a pattern was called.
func (m *MockRunner) CallCount(pattern string) int {
	count := 0
	for _, c := range m.Calls {
		key := c.Command + " " + strings.Join(c.Args, " ")
		if strings.Contains(key, pattern) {
			count++
		}
	}
	return count
}
