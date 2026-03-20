package git

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTechDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func setupTechDirWithDirs(t *testing.T, files map[string]string, dirs []string) string {
	t.Helper()
	dir := setupTechDir(t, files)
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDetectTechStack_Go(t *testing.T) {
	dir := setupTechDirWithDirs(t, map[string]string{
		"go.mod":         "module example.com/app\n\ngo 1.22",
		"go.sum":         "",
		"Makefile":       "",
		".golangci.yml":  "",
	}, []string{"cmd", "internal"})

	ts := DetectTechStack(dir)

	if len(ts.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(ts.Languages))
	}
	if ts.Languages[0].Name != "Go" {
		t.Errorf("expected Go, got %s", ts.Languages[0].Name)
	}
	if ts.Languages[0].Version != "1.22" {
		t.Errorf("expected version 1.22, got %s", ts.Languages[0].Version)
	}
	if !ts.Languages[0].Primary {
		t.Error("expected Go to be primary")
	}
	if !contains(ts.TestRunners, "go test") {
		t.Error("expected 'go test' in test runners")
	}
	if !contains(ts.Linters, "golangci-lint") {
		t.Error("expected golangci-lint in linters")
	}
	if !contains(ts.BuildTools, "Make") {
		t.Error("expected Make in build tools")
	}
	if !contains(ts.PackageManagers, "Go Modules") {
		t.Error("expected Go Modules in package managers")
	}
	if ts.DirectoryLayout != "go-standard" {
		t.Errorf("expected go-standard layout, got %s", ts.DirectoryLayout)
	}
}

func TestDetectTechStack_TypeScript(t *testing.T) {
	dir := setupTechDirWithDirs(t, map[string]string{
		"package.json":    `{"name": "app"}`,
		"tsconfig.json":   "{}",
		"next.config.js":  "",
		"jest.config.ts":  "",
		"eslint.config.js": "",
		"pnpm-lock.yaml":  "",
	}, []string{"src", "public"})

	ts := DetectTechStack(dir)

	if len(ts.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d: %+v", len(ts.Languages), ts.Languages)
	}
	if ts.Languages[0].Name != "TypeScript" {
		t.Errorf("expected TypeScript, got %s", ts.Languages[0].Name)
	}
	if !contains(ts.Frameworks, "Next.js") {
		t.Errorf("expected Next.js framework, got %v", ts.Frameworks)
	}
	if !contains(ts.TestRunners, "Jest") {
		t.Errorf("expected Jest, got %v", ts.TestRunners)
	}
	if !contains(ts.Linters, "ESLint") {
		t.Errorf("expected ESLint, got %v", ts.Linters)
	}
	if !contains(ts.PackageManagers, "pnpm") {
		t.Errorf("expected pnpm, got %v", ts.PackageManagers)
	}
	if ts.DirectoryLayout != "frontend-spa" {
		t.Errorf("expected frontend-spa layout, got %s", ts.DirectoryLayout)
	}
}

func TestDetectTechStack_Python(t *testing.T) {
	dir := setupTechDirWithDirs(t, map[string]string{
		"pyproject.toml": `[project]\npython_requires = ">=3.11"`,
		"manage.py":      "",
		".flake8":        "",
		"poetry.lock":    "",
	}, []string{"src", "tests", "docs"})

	ts := DetectTechStack(dir)

	if len(ts.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(ts.Languages))
	}
	if ts.Languages[0].Name != "Python" {
		t.Errorf("expected Python, got %s", ts.Languages[0].Name)
	}
	if !contains(ts.Frameworks, "Django") {
		t.Errorf("expected Django, got %v", ts.Frameworks)
	}
	if !contains(ts.Linters, "Flake8") {
		t.Errorf("expected Flake8, got %v", ts.Linters)
	}
	if !contains(ts.PackageManagers, "Poetry") {
		t.Errorf("expected Poetry, got %v", ts.PackageManagers)
	}
	if ts.DirectoryLayout != "python-standard" {
		t.Errorf("expected python-standard layout, got %s", ts.DirectoryLayout)
	}
}

func TestDetectTechStack_Rust(t *testing.T) {
	dir := setupTechDirWithDirs(t, map[string]string{
		"Cargo.toml": "",
		"Cargo.lock": "",
	}, []string{"src"})

	ts := DetectTechStack(dir)

	if len(ts.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(ts.Languages))
	}
	if ts.Languages[0].Name != "Rust" {
		t.Errorf("expected Rust, got %s", ts.Languages[0].Name)
	}
	if !contains(ts.PackageManagers, "Cargo") {
		t.Errorf("expected Cargo, got %v", ts.PackageManagers)
	}
}

func TestDetectTechStack_MultiLanguage(t *testing.T) {
	dir := setupTechDir(t, map[string]string{
		"go.mod":       "module app\n\ngo 1.22",
		"package.json": `{"name": "frontend"}`,
	})

	ts := DetectTechStack(dir)

	if len(ts.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(ts.Languages))
	}
	// Go should be primary (detected first)
	if ts.Languages[0].Name != "Go" || !ts.Languages[0].Primary {
		t.Errorf("expected Go as primary, got %+v", ts.Languages[0])
	}
	if ts.Languages[1].Primary {
		t.Error("second language should not be primary")
	}
}

func TestDetectTechStack_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ts := DetectTechStack(dir)

	if len(ts.Languages) != 0 {
		t.Errorf("expected 0 languages, got %d", len(ts.Languages))
	}
	if ts.DirectoryLayout != "flat" {
		t.Errorf("expected flat layout, got %s", ts.DirectoryLayout)
	}
}

func TestPrimaryLanguage(t *testing.T) {
	ts := TechStack{
		Languages: []Language{
			{Name: "Go", Primary: false},
			{Name: "Python", Primary: true},
		},
	}
	if ts.PrimaryLanguage() != "Python" {
		t.Errorf("expected Python, got %s", ts.PrimaryLanguage())
	}
}

func TestPrimaryLanguage_NoPrimary(t *testing.T) {
	ts := TechStack{
		Languages: []Language{
			{Name: "Go"},
		},
	}
	if ts.PrimaryLanguage() != "Go" {
		t.Errorf("expected Go as fallback, got %s", ts.PrimaryLanguage())
	}
}

func TestPrimaryLanguage_Empty(t *testing.T) {
	ts := TechStack{}
	if ts.PrimaryLanguage() != "" {
		t.Errorf("expected empty string, got %s", ts.PrimaryLanguage())
	}
}

func TestDetectLayout_JavaStandard(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.MkdirAll(filepath.Join(dir, "test"), 0o755)

	layout := detectLayout(dir)
	if layout != "java-standard" {
		t.Errorf("expected java-standard, got %s", layout)
	}
}

func TestDetectLayout_RailsLike(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)

	layout := detectLayout(dir)
	if layout != "rails-like" {
		t.Errorf("expected rails-like, got %s", layout)
	}
}

func TestDetectLayout_SrcBased(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)

	layout := detectLayout(dir)
	if layout != "src-based" {
		t.Errorf("expected src-based, got %s", layout)
	}
}

func TestDetectLayout_LibBased(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)

	layout := detectLayout(dir)
	if layout != "lib-based" {
		t.Errorf("expected lib-based, got %s", layout)
	}
}

func TestDetectTechStack_Docker(t *testing.T) {
	dir := setupTechDir(t, map[string]string{
		"Dockerfile":          "",
		"docker-compose.yml":  "",
	})

	ts := DetectTechStack(dir)
	if !contains(ts.BuildTools, "Docker") {
		t.Errorf("expected Docker, got %v", ts.BuildTools)
	}
	if !contains(ts.BuildTools, "Docker Compose") {
		t.Errorf("expected Docker Compose, got %v", ts.BuildTools)
	}
}

func TestDetectTechStack_Playwright(t *testing.T) {
	dir := setupTechDir(t, map[string]string{
		"package.json":          `{}`,
		"playwright.config.ts":  "",
	})

	ts := DetectTechStack(dir)
	if !contains(ts.TestRunners, "Playwright") {
		t.Errorf("expected Playwright, got %v", ts.TestRunners)
	}
}

func TestDetectTechStack_Vitest(t *testing.T) {
	dir := setupTechDir(t, map[string]string{
		"package.json":      `{}`,
		"vitest.config.ts":  "",
	})

	ts := DetectTechStack(dir)
	if !contains(ts.TestRunners, "Vitest") {
		t.Errorf("expected Vitest, got %v", ts.TestRunners)
	}
}

func TestDetectTechStack_GoReleaser(t *testing.T) {
	dir := setupTechDir(t, map[string]string{
		"go.mod":            "module app\n\ngo 1.22",
		".goreleaser.yml":   "",
	})

	ts := DetectTechStack(dir)
	if !contains(ts.BuildTools, "GoReleaser") {
		t.Errorf("expected GoReleaser, got %v", ts.BuildTools)
	}
}

func TestExtractLinePrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("go 1.22\nother line"), 0o644)

	version := extractLinePrefix(path, "go ")
	if version != "1.22" {
		t.Errorf("expected 1.22, got %s", version)
	}
}

func TestExtractLinePrefix_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("no match here"), 0o644)

	result := extractLinePrefix(path, "go ")
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestExtractLinePrefix_FileNotExist(t *testing.T) {
	result := extractLinePrefix("/nonexistent/file", "prefix")
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
