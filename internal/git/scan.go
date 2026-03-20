package git

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TechStack describes the detected technology stack of a project.
type TechStack struct {
	Language       string `json:"language,omitempty"`
	Framework      string `json:"framework,omitempty"`
	TestRunner     string `json:"test_runner,omitempty"`
	Linter         string `json:"linter,omitempty"`
	BuildTool      string `json:"build_tool,omitempty"`
	PackageManager string `json:"package_manager,omitempty"`
}

// ScanTechStack detects the technology stack by examining marker files in the directory.
func ScanTechStack(dir string) TechStack {
	ts := TechStack{}

	switch {
	case fileExists(dir, "go.mod"):
		ts = scanGo(ts)
	case fileExists(dir, "package.json"):
		ts = scanJavaScript(dir, ts)
	case fileExists(dir, "requirements.txt"):
		ts = scanPython(ts, "pip")
	case fileExists(dir, "pyproject.toml"):
		ts = scanPython(ts, "pip")
	case fileExists(dir, "Cargo.toml"):
		ts = scanRust(ts)
	case fileExists(dir, "pom.xml"):
		ts = scanJava(ts, "maven")
	case fileExists(dir, "build.gradle"):
		ts = scanJava(ts, "gradle")
	case fileExists(dir, "Gemfile"):
		ts = scanRuby(dir, ts)
	}

	// Detect frameworks (may override or supplement language-specific detection)
	ts = detectFrameworks(dir, ts)

	return ts
}

func scanGo(ts TechStack) TechStack {
	return TechStack{
		Language:       "go",
		Framework:      ts.Framework,
		TestRunner:     "go test",
		Linter:         "golangci-lint",
		BuildTool:      "go build",
		PackageManager: "go modules",
	}
}

func scanJavaScript(dir string, ts TechStack) TechStack {
	result := TechStack{
		Language:       "javascript",
		Framework:      ts.Framework,
		PackageManager: detectJSPackageManager(dir),
	}

	// Check for TypeScript
	if fileExists(dir, "tsconfig.json") {
		result.Language = "typescript"
	}

	// Parse package.json for scripts
	result = parsePackageJSONScripts(dir, result)

	return result
}

func scanPython(ts TechStack, pkgManager string) TechStack {
	return TechStack{
		Language:       "python",
		Framework:      ts.Framework,
		TestRunner:     "pytest",
		Linter:         ts.Linter,
		BuildTool:      ts.BuildTool,
		PackageManager: pkgManager,
	}
}

func scanRust(ts TechStack) TechStack {
	return TechStack{
		Language:       "rust",
		Framework:      ts.Framework,
		TestRunner:     "cargo test",
		Linter:         ts.Linter,
		BuildTool:      "cargo",
		PackageManager: "cargo",
	}
}

func scanJava(ts TechStack, buildTool string) TechStack {
	return TechStack{
		Language:       "java",
		Framework:      ts.Framework,
		TestRunner:     ts.TestRunner,
		Linter:         ts.Linter,
		BuildTool:      buildTool,
		PackageManager: buildTool,
	}
}

func scanRuby(dir string, ts TechStack) TechStack {
	result := TechStack{
		Language:       "ruby",
		Framework:      ts.Framework,
		TestRunner:     ts.TestRunner,
		Linter:         ts.Linter,
		BuildTool:      ts.BuildTool,
		PackageManager: "bundler",
	}

	// Check for Rails in Gemfile
	content, err := os.ReadFile(filepath.Join(dir, "Gemfile"))
	if err == nil && strings.Contains(string(content), "rails") {
		result.Framework = "rails"
	}

	return result
}

func detectFrameworks(dir string, ts TechStack) TechStack {
	// Only check framework markers if not already set
	if ts.Framework != "" {
		return ts
	}

	switch {
	case fileExists(dir, "next.config.js") || fileExists(dir, "next.config.mjs") || fileExists(dir, "next.config.ts"):
		ts.Framework = "nextjs"
	case fileExists(dir, "angular.json"):
		ts.Framework = "angular"
	case fileExists(dir, "nuxt.config.js") || fileExists(dir, "nuxt.config.ts"):
		ts.Framework = "nuxt"
	case fileExists(dir, "svelte.config.js"):
		ts.Framework = "svelte"
	}

	return ts
}

func detectJSPackageManager(dir string) string {
	switch {
	case fileExists(dir, "pnpm-lock.yaml"):
		return "pnpm"
	case fileExists(dir, "yarn.lock"):
		return "yarn"
	default:
		return "npm"
	}
}

// packageJSONScripts is a minimal struct for parsing package.json scripts.
type packageJSONScripts struct {
	Scripts map[string]string `json:"scripts"`
}

func parsePackageJSONScripts(dir string, ts TechStack) TechStack {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ts
	}

	var pkg packageJSONScripts
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ts
	}

	if testScript, ok := pkg.Scripts["test"]; ok {
		ts.TestRunner = detectTestRunner(testScript)
	}

	if lintScript, ok := pkg.Scripts["lint"]; ok {
		ts.Linter = detectLinter(lintScript)
	}

	return ts
}

func detectTestRunner(script string) string {
	switch {
	case strings.Contains(script, "jest"):
		return "jest"
	case strings.Contains(script, "vitest"):
		return "vitest"
	case strings.Contains(script, "mocha"):
		return "mocha"
	case strings.Contains(script, "ava"):
		return "ava"
	default:
		return script
	}
}

func detectLinter(script string) string {
	switch {
	case strings.Contains(script, "eslint"):
		return "eslint"
	case strings.Contains(script, "biome"):
		return "biome"
	default:
		return script
	}
}

func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
