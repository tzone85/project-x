package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner abstracts os/exec for testing.
type CommandRunner interface {
	Run(dir string, name string, args ...string) (string, error)
}

// ExecRunner runs commands via os/exec.
type ExecRunner struct{}

// Run executes a command in the given directory and returns combined output.
func (r ExecRunner) Run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// RecordedCommand captures a command invocation for test assertions.
type RecordedCommand struct {
	Dir  string
	Name string
	Args []string
}

// mockResponse holds a configured output/error pair.
type mockResponse struct {
	output string
	err    error
}

// MockRunner records commands and returns pre-configured responses.
type MockRunner struct {
	Commands  []RecordedCommand
	responses []mockResponse
	callIndex int
}

// NewMockRunner creates a new MockRunner ready for use.
func NewMockRunner() *MockRunner {
	return &MockRunner{}
}

// AddResponse queues an output/error pair to be returned on the next call.
func (m *MockRunner) AddResponse(output string, err error) {
	m.responses = append(m.responses, mockResponse{output: output, err: err})
}

// Run records the command and returns the next configured response.
func (m *MockRunner) Run(dir, name string, args ...string) (string, error) {
	m.Commands = append(m.Commands, RecordedCommand{
		Dir:  dir,
		Name: name,
		Args: append([]string{}, args...),
	})

	if m.callIndex >= len(m.responses) {
		panic(fmt.Sprintf("MockRunner: no response configured for call %d (%s %s)", m.callIndex, name, strings.Join(args, " ")))
	}

	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp.output, resp.err
}

// FetchBranch fetches a specific branch from origin.
func FetchBranch(runner CommandRunner, repoDir, branch string) error {
	_, err := runner.Run(repoDir, "git", "fetch", "origin", branch)
	return err
}

// RebaseOnto rebases the current branch onto the given upstream ref.
func RebaseOnto(runner CommandRunner, worktreeDir, upstream string) error {
	_, err := runner.Run(worktreeDir, "git", "rebase", upstream)
	return err
}

// Diff returns the diff between HEAD and the merge-base with origin/main.
func Diff(runner CommandRunner, worktreeDir string) (string, error) {
	base, err := MergeBase(runner, worktreeDir, "HEAD", "origin/main")
	if err != nil {
		return "", fmt.Errorf("finding merge-base: %w", err)
	}

	output, err := runner.Run(worktreeDir, "git", "diff", base)
	if err != nil {
		return "", fmt.Errorf("running diff: %w", err)
	}
	return output, nil
}

// MergeBase returns the best common ancestor between two refs.
func MergeBase(runner CommandRunner, worktreeDir, ref1, ref2 string) (string, error) {
	output, err := runner.Run(worktreeDir, "git", "merge-base", ref1, ref2)
	if err != nil {
		return "", err
	}
	return output, nil
}

// DeleteRemoteBranch deletes a branch from the origin remote.
func DeleteRemoteBranch(runner CommandRunner, repoDir, branch string) error {
	_, err := runner.Run(repoDir, "git", "push", "origin", "--delete", branch)
	return err
}

// DiffNameOnly returns a list of file names changed relative to the given base commit.
func DiffNameOnly(runner CommandRunner, worktreeDir, base string) ([]string, error) {
	output, err := runner.Run(worktreeDir, "git", "diff", "--name-only", base)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}
