package git

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// DetectTechStack analyzes a repository directory and returns the detected tech stack.
func DetectTechStack(dir string) TechStack {
	ts := TechStack{}
	ts.Languages = detectLanguages(dir)
	ts.Frameworks = detectFrameworks(dir)
	ts.TestRunners = detectTestRunners(dir)
	ts.Linters = detectLinters(dir)
	ts.BuildTools = detectBuildTools(dir)
	ts.PackageManagers = detectPackageManagers(dir)
	ts.DirectoryLayout = detectLayout(dir)
	return ts
}

// PrimaryLanguage returns the primary language name or empty string.
func (ts TechStack) PrimaryLanguage() string {
	for _, lang := range ts.Languages {
		if lang.Primary {
			return lang.Name
		}
	}
	if len(ts.Languages) > 0 {
		return ts.Languages[0].Name
	}
	return ""
}

// languageIndicator maps a file to a language name with optional version extraction.
type languageIndicator struct {
	File       string
	Language   string
	VersionFn  func(dir string) string
}

var languageIndicators = []languageIndicator{
	{File: "go.mod", Language: "Go", VersionFn: goVersion},
	{File: "package.json", Language: "JavaScript/TypeScript", VersionFn: nodeVersion},
	{File: "Cargo.toml", Language: "Rust"},
	{File: "pyproject.toml", Language: "Python", VersionFn: pythonVersionFromPyproject},
	{File: "setup.py", Language: "Python"},
	{File: "requirements.txt", Language: "Python"},
	{File: "pom.xml", Language: "Java"},
	{File: "build.gradle", Language: "Java"},
	{File: "build.gradle.kts", Language: "Kotlin"},
	{File: "Gemfile", Language: "Ruby"},
	{File: "mix.exs", Language: "Elixir"},
	{File: "composer.json", Language: "PHP"},
	{File: "Package.swift", Language: "Swift"},
	{File: "*.csproj", Language: "C#"},
}

func detectLanguages(dir string) []Language {
	var langs []Language
	seen := map[string]bool{}

	for _, ind := range languageIndicators {
		if seen[ind.Language] {
			continue
		}
		var found bool
		if strings.Contains(ind.File, "*") {
			matches, _ := filepath.Glob(filepath.Join(dir, ind.File))
			found = len(matches) > 0
		} else {
			found = fileExists(filepath.Join(dir, ind.File))
		}
		if !found {
			continue
		}
		seen[ind.Language] = true
		lang := Language{
			Name:    ind.Language,
			Primary: len(langs) == 0,
		}
		if ind.VersionFn != nil {
			lang.Version = ind.VersionFn(dir)
		}
		langs = append(langs, lang)
	}

	// Check for TypeScript refinement
	if seen["JavaScript/TypeScript"] && fileExists(filepath.Join(dir, "tsconfig.json")) {
		for i := range langs {
			if langs[i].Name == "JavaScript/TypeScript" {
				langs[i].Name = "TypeScript"
			}
		}
	}

	return langs
}

type fileIndicator struct {
	File string
	Name string
}

var frameworkIndicators = []fileIndicator{
	{File: "next.config.js", Name: "Next.js"},
	{File: "next.config.ts", Name: "Next.js"},
	{File: "next.config.mjs", Name: "Next.js"},
	{File: "nuxt.config.ts", Name: "Nuxt"},
	{File: "angular.json", Name: "Angular"},
	{File: "svelte.config.js", Name: "SvelteKit"},
	{File: "astro.config.mjs", Name: "Astro"},
	{File: "vite.config.ts", Name: "Vite"},
	{File: "vite.config.js", Name: "Vite"},
	{File: "remix.config.js", Name: "Remix"},
	{File: "gatsby-config.js", Name: "Gatsby"},
	{File: "manage.py", Name: "Django"},
	{File: "Procfile", Name: "Heroku"},
}

func detectFrameworks(dir string) []string {
	return detectByFiles(dir, frameworkIndicators)
}

var testRunnerIndicators = []fileIndicator{
	{File: "jest.config.js", Name: "Jest"},
	{File: "jest.config.ts", Name: "Jest"},
	{File: "vitest.config.ts", Name: "Vitest"},
	{File: "vitest.config.js", Name: "Vitest"},
	{File: "cypress.config.ts", Name: "Cypress"},
	{File: "cypress.config.js", Name: "Cypress"},
	{File: "playwright.config.ts", Name: "Playwright"},
	{File: ".pytest.ini", Name: "pytest"},
	{File: "pytest.ini", Name: "pytest"},
	{File: "setup.cfg", Name: "pytest"},
	{File: "phpunit.xml", Name: "PHPUnit"},
}

func detectTestRunners(dir string) []string {
	runners := detectByFiles(dir, testRunnerIndicators)

	// Go projects use built-in testing
	if fileExists(filepath.Join(dir, "go.mod")) {
		runners = append(runners, "go test")
	}

	return runners
}

var linterIndicators = []fileIndicator{
	{File: ".eslintrc.js", Name: "ESLint"},
	{File: ".eslintrc.json", Name: "ESLint"},
	{File: ".eslintrc.yml", Name: "ESLint"},
	{File: "eslint.config.js", Name: "ESLint"},
	{File: "eslint.config.mjs", Name: "ESLint"},
	{File: ".golangci.yml", Name: "golangci-lint"},
	{File: ".golangci.yaml", Name: "golangci-lint"},
	{File: ".flake8", Name: "Flake8"},
	{File: ".pylintrc", Name: "Pylint"},
	{File: "ruff.toml", Name: "Ruff"},
	{File: ".prettierrc", Name: "Prettier"},
	{File: ".prettierrc.js", Name: "Prettier"},
	{File: "biome.json", Name: "Biome"},
	{File: ".rubocop.yml", Name: "RuboCop"},
	{File: "clippy.toml", Name: "Clippy"},
}

func detectLinters(dir string) []string {
	return detectByFiles(dir, linterIndicators)
}

var buildToolIndicators = []fileIndicator{
	{File: "Makefile", Name: "Make"},
	{File: "CMakeLists.txt", Name: "CMake"},
	{File: "Taskfile.yml", Name: "Task"},
	{File: "justfile", Name: "Just"},
	{File: "Dockerfile", Name: "Docker"},
	{File: "docker-compose.yml", Name: "Docker Compose"},
	{File: "docker-compose.yaml", Name: "Docker Compose"},
	{File: "Earthfile", Name: "Earthly"},
	{File: "Tiltfile", Name: "Tilt"},
	{File: ".goreleaser.yml", Name: "GoReleaser"},
	{File: ".goreleaser.yaml", Name: "GoReleaser"},
	{File: "webpack.config.js", Name: "Webpack"},
	{File: "turbo.json", Name: "Turborepo"},
}

func detectBuildTools(dir string) []string {
	return detectByFiles(dir, buildToolIndicators)
}

var packageManagerIndicators = []fileIndicator{
	{File: "pnpm-lock.yaml", Name: "pnpm"},
	{File: "yarn.lock", Name: "Yarn"},
	{File: "package-lock.json", Name: "npm"},
	{File: "bun.lockb", Name: "Bun"},
	{File: "Pipfile.lock", Name: "Pipenv"},
	{File: "poetry.lock", Name: "Poetry"},
	{File: "uv.lock", Name: "uv"},
	{File: "Cargo.lock", Name: "Cargo"},
	{File: "go.sum", Name: "Go Modules"},
	{File: "Gemfile.lock", Name: "Bundler"},
}

func detectPackageManagers(dir string) []string {
	return detectByFiles(dir, packageManagerIndicators)
}

func detectByFiles(dir string, indicators []fileIndicator) []string {
	seen := map[string]bool{}
	var results []string
	for _, ind := range indicators {
		if seen[ind.Name] {
			continue
		}
		if fileExists(filepath.Join(dir, ind.File)) {
			seen[ind.Name] = true
			results = append(results, ind.Name)
		}
	}
	return results
}

// detectLayout identifies common directory layout patterns.
func detectLayout(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "unknown"
	}

	dirs := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs[e.Name()] = true
		}
	}

	switch {
	case dirs["cmd"] && dirs["internal"]:
		return "go-standard"
	case dirs["src"] && dirs["tests"] && dirs["docs"]:
		return "python-standard"
	case dirs["src"] && (dirs["public"] || dirs["static"]):
		return "frontend-spa"
	case dirs["app"] && dirs["lib"]:
		return "rails-like"
	case dirs["src"] && dirs["test"]:
		return "java-standard"
	case dirs["src"]:
		return "src-based"
	case dirs["lib"]:
		return "lib-based"
	default:
		return "flat"
	}
}

// goVersion extracts the Go version from go.mod.
func goVersion(dir string) string {
	return extractLinePrefix(filepath.Join(dir, "go.mod"), "go ")
}

// nodeVersion extracts the Node.js engine version from package.json.
func nodeVersion(dir string) string {
	return extractLinePrefix(filepath.Join(dir, "package.json"), `"node":`)
}

// pythonVersionFromPyproject extracts the Python version from pyproject.toml.
func pythonVersionFromPyproject(dir string) string {
	return extractLinePrefix(filepath.Join(dir, "pyproject.toml"), "python_requires")
}

// extractLinePrefix scans a file for a line starting with prefix and returns the value.
func extractLinePrefix(path, prefix string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimPrefix(line, prefix)
			val = strings.Trim(val, ` ="',`)
			return val
		}
	}
	return ""
}
