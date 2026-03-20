package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanTechStack_GoProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "go" {
		t.Errorf("expected language 'go', got %q", ts.Language)
	}
	if ts.TestRunner != "go test" {
		t.Errorf("expected test runner 'go test', got %q", ts.TestRunner)
	}
	if ts.Linter != "golangci-lint" {
		t.Errorf("expected linter 'golangci-lint', got %q", ts.Linter)
	}
	if ts.BuildTool != "go build" {
		t.Errorf("expected build tool 'go build', got %q", ts.BuildTool)
	}
	if ts.PackageManager != "go modules" {
		t.Errorf("expected package manager 'go modules', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_PythonRequirements(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==2.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "python" {
		t.Errorf("expected language 'python', got %q", ts.Language)
	}
	if ts.TestRunner != "pytest" {
		t.Errorf("expected test runner 'pytest', got %q", ts.TestRunner)
	}
	if ts.PackageManager != "pip" {
		t.Errorf("expected package manager 'pip', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_PythonPyproject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"foo\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "python" {
		t.Errorf("expected language 'python', got %q", ts.Language)
	}
	if ts.TestRunner != "pytest" {
		t.Errorf("expected test runner 'pytest', got %q", ts.TestRunner)
	}
}

func TestScanTechStack_JavaScriptProject(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := `{
  "name": "my-app",
  "scripts": {
    "test": "jest",
    "lint": "eslint ."
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "javascript" {
		t.Errorf("expected language 'javascript', got %q", ts.Language)
	}
	if ts.TestRunner != "jest" {
		t.Errorf("expected test runner 'jest', got %q", ts.TestRunner)
	}
	if ts.Linter != "eslint" {
		t.Errorf("expected linter 'eslint', got %q", ts.Linter)
	}
	if ts.PackageManager != "npm" {
		t.Errorf("expected package manager 'npm', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_TypeScriptProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "typescript" {
		t.Errorf("expected language 'typescript', got %q", ts.Language)
	}
}

func TestScanTechStack_RustProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"foo\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "rust" {
		t.Errorf("expected language 'rust', got %q", ts.Language)
	}
	if ts.TestRunner != "cargo test" {
		t.Errorf("expected test runner 'cargo test', got %q", ts.TestRunner)
	}
	if ts.BuildTool != "cargo" {
		t.Errorf("expected build tool 'cargo', got %q", ts.BuildTool)
	}
	if ts.PackageManager != "cargo" {
		t.Errorf("expected package manager 'cargo', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_JavaMaven(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project></project>"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "java" {
		t.Errorf("expected language 'java', got %q", ts.Language)
	}
	if ts.BuildTool != "maven" {
		t.Errorf("expected build tool 'maven', got %q", ts.BuildTool)
	}
	if ts.PackageManager != "maven" {
		t.Errorf("expected package manager 'maven', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_JavaGradle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("apply plugin: 'java'"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "java" {
		t.Errorf("expected language 'java', got %q", ts.Language)
	}
	if ts.BuildTool != "gradle" {
		t.Errorf("expected build tool 'gradle', got %q", ts.BuildTool)
	}
	if ts.PackageManager != "gradle" {
		t.Errorf("expected package manager 'gradle', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_NextJSFramework(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports = {}"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Framework != "nextjs" {
		t.Errorf("expected framework 'nextjs', got %q", ts.Framework)
	}
}

func TestScanTechStack_AngularFramework(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "angular.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Framework != "angular" {
		t.Errorf("expected framework 'angular', got %q", ts.Framework)
	}
}

func TestScanTechStack_RubyGemfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\ngem 'rails'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.Language != "ruby" {
		t.Errorf("expected language 'ruby', got %q", ts.Language)
	}
	if ts.Framework != "rails" {
		t.Errorf("expected framework 'rails', got %q", ts.Framework)
	}
	if ts.PackageManager != "bundler" {
		t.Errorf("expected package manager 'bundler', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	ts := ScanTechStack(dir)

	if ts.Language != "" {
		t.Errorf("expected empty language, got %q", ts.Language)
	}
	if ts.Framework != "" {
		t.Errorf("expected empty framework, got %q", ts.Framework)
	}
	if ts.TestRunner != "" {
		t.Errorf("expected empty test runner, got %q", ts.TestRunner)
	}
	if ts.Linter != "" {
		t.Errorf("expected empty linter, got %q", ts.Linter)
	}
	if ts.BuildTool != "" {
		t.Errorf("expected empty build tool, got %q", ts.BuildTool)
	}
	if ts.PackageManager != "" {
		t.Errorf("expected empty package manager, got %q", ts.PackageManager)
	}
}

func TestScanTechStack_YarnLockManager(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.PackageManager != "yarn" {
		t.Errorf("expected package manager 'yarn', got %q", ts.PackageManager)
	}
}

func TestScanTechStack_PnpmLockManager(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	ts := ScanTechStack(dir)

	if ts.PackageManager != "pnpm" {
		t.Errorf("expected package manager 'pnpm', got %q", ts.PackageManager)
	}
}
