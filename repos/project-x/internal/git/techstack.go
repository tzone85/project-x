package git

import (
	"context"
	"os"
	"path/filepath"
)

// TechStack describes the detected technology stack of a project.
type TechStack struct {
	Language   string `json:"language"`
	Framework  string `json:"framework"`
	TestRunner string `json:"test_runner"`
	Linter     string `json:"linter"`
	BuildTool  string `json:"build_tool"`
	DirLayout  string `json:"dir_layout"`
}

// indicator maps a file marker to what it indicates.
type indicator struct {
	file    string
	field   string // which TechStack field to set
	value   string
}

// detectionRules defines file-based tech stack detection.
// Order matters: first match wins per field.
var detectionRules = []indicator{
	// Language detection
	{file: "go.mod", field: "language", value: "Go"},
	{file: "package.json", field: "language", value: "JavaScript/TypeScript"},
	{file: "Cargo.toml", field: "language", value: "Rust"},
	{file: "pyproject.toml", field: "language", value: "Python"},
	{file: "requirements.txt", field: "language", value: "Python"},
	{file: "pom.xml", field: "language", value: "Java"},
	{file: "build.gradle", field: "language", value: "Java"},

	// Framework detection
	{file: "next.config.js", field: "framework", value: "Next.js"},
	{file: "next.config.mjs", field: "framework", value: "Next.js"},
	{file: "nuxt.config.ts", field: "framework", value: "Nuxt"},
	{file: "angular.json", field: "framework", value: "Angular"},
	{file: "svelte.config.js", field: "framework", value: "Svelte"},

	// Test runner detection
	{file: "jest.config.js", field: "test_runner", value: "Jest"},
	{file: "jest.config.ts", field: "test_runner", value: "Jest"},
	{file: "vitest.config.ts", field: "test_runner", value: "Vitest"},
	{file: "pytest.ini", field: "test_runner", value: "pytest"},
	{file: "setup.cfg", field: "test_runner", value: "pytest"},

	// Linter detection
	{file: ".eslintrc.json", field: "linter", value: "ESLint"},
	{file: ".eslintrc.js", field: "linter", value: "ESLint"},
	{file: "eslint.config.js", field: "linter", value: "ESLint"},
	{file: ".golangci.yml", field: "linter", value: "golangci-lint"},
	{file: ".golangci.yaml", field: "linter", value: "golangci-lint"},
	{file: "ruff.toml", field: "linter", value: "Ruff"},

	// Build tool detection
	{file: "Makefile", field: "build_tool", value: "Make"},
	{file: "Taskfile.yml", field: "build_tool", value: "Task"},
	{file: "webpack.config.js", field: "build_tool", value: "Webpack"},
	{file: "vite.config.ts", field: "build_tool", value: "Vite"},
	{file: "turbo.json", field: "build_tool", value: "Turborepo"},
}

// DetectTechStack scans the project directory for known files and returns
// the detected technology stack. It does not shell out — it uses file existence
// checks only, making it fast and safe.
func DetectTechStack(_ context.Context, dir string) TechStack {
	var ts TechStack
	set := make(map[string]bool)

	for _, rule := range detectionRules {
		if set[rule.field] {
			continue // first match wins per field
		}
		if fileExists(filepath.Join(dir, rule.file)) {
			set[rule.field] = true
			switch rule.field {
			case "language":
				ts.Language = rule.value
			case "framework":
				ts.Framework = rule.value
			case "test_runner":
				ts.TestRunner = rule.value
			case "linter":
				ts.Linter = rule.value
			case "build_tool":
				ts.BuildTool = rule.value
			}
		}
	}

	// Go-specific: infer test runner
	if ts.Language == "Go" && ts.TestRunner == "" {
		ts.TestRunner = "go test"
	}

	// Detect directory layout
	ts.DirLayout = detectLayout(dir)

	return ts
}

func detectLayout(dir string) string {
	if dirExists(filepath.Join(dir, "internal")) && dirExists(filepath.Join(dir, "cmd")) {
		return "Go standard (cmd/ + internal/)"
	}
	if dirExists(filepath.Join(dir, "src")) && dirExists(filepath.Join(dir, "tests")) {
		return "src/ + tests/"
	}
	if dirExists(filepath.Join(dir, "src")) {
		return "src/"
	}
	if dirExists(filepath.Join(dir, "lib")) {
		return "lib/"
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
