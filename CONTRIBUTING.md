# Contributing to px-dispatch

Thanks for your interest in making `px-dispatch` better.

## Code of Conduct

This project follows the [Contributor Covenant 2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). By participating you agree to uphold it. Report unacceptable behavior privately via GitHub Security Advisories (see [Security](#security) below) or to the repo owner directly.

## Development setup

1. Clone the repo
2. Install Go 1.22+ — no C toolchain required (pure-Go SQLite)
3. Install runtime tools: `tmux`, `git`, `gh`
4. Fetch deps: `go mod download`
5. Run tests: `make test`
6. Build: `make build`

## Code style

- Go formatting via `gofmt` and `go vet` is enforced by CI.
- `golangci-lint` is **blocking** in CI — run `make lint` locally before pushing.
- Files: keep cohesive, ~200–400 lines typical, 800 max.
- Functions: <50 lines, single-purpose.
- Immutable patterns: return new objects, do not mutate inputs.
- Interfaces at package boundaries; avoid leaking concrete types.
- Comments explain *why*, not *what*. Skip comments that restate the next line.

## Testing

- TDD where possible: write a failing test, then the implementation.
- `go test ./... -race` must pass.
- Total coverage gate is **75 %** in CI — drops fail the build.
- Most packages already sit above 80 %; keep new code in that range.

## Pull requests

1. Create a feature branch off `main`.
2. Use conventional-commit subjects: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`, `perf:`, `ci:`. GoReleaser uses these to group the auto-generated changelog at release time.
3. Run `make lint && make test` locally.
4. Open the PR with a clear "what + why" description; link to issues where relevant.
5. CI must be green (test + lint + govulncheck) before merge.

## Security

Please **do not** open public GitHub issues for vulnerabilities. Use the **Security** tab on this repository → *Report a vulnerability* (private advisories). Maintainers will respond within 7 days. Coordinated disclosure is expected; please give us time to ship a fix before going public.

## Releases

Releases are cut by tagging:

```
git tag v0.1.0
git push origin v0.1.0
```

The `release.yml` workflow then runs GoReleaser, which:

- Cross-compiles darwin/linux/windows × amd64/arm64.
- Publishes a GitHub Release with checksums and an SBOM.
- Updates the [`tzone85/homebrew-tap`](https://github.com/tzone85/homebrew-tap) formula.
- Builds and pushes a multi-arch image to `ghcr.io/tzone85/px-dispatch`.

The repo must have the `HOMEBREW_TAP_GITHUB_TOKEN` secret set for the tap step to succeed.

## Architecture

See [`docs/architecture.md`](docs/architecture.md) for the deep dive and [`docs/onboarding.md`](docs/onboarding.md) for first-PR onboarding.
