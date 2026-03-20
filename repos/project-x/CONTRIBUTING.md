# Contributing to Project X

## Development Setup

1. Install Go 1.22+
2. Install tmux 3.0+
3. Clone the repository:
   ```bash
   git clone https://github.com/tzone85/project-x.git
   cd project-x
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```
5. Run tests:
   ```bash
   go test -race ./...
   ```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Run `golangci-lint run` before submitting
- Keep files under 400 lines (800 max)
- Write table-driven tests

## PR Process

1. Fork the repo and create a feature branch
2. Write tests first (TDD)
3. Ensure `go test -race ./...` passes
4. Ensure 80%+ test coverage
5. Run `golangci-lint run`
6. Submit PR with clear description

## Commit Messages

Use conventional commits:
```
feat: add new runtime plugin
fix: correct budget calculation
refactor: extract pricing table
test: add circuit breaker tests
docs: update config reference
```

## DCO Sign-Off

All commits must be signed off:
```bash
git commit -s -m "feat: your change"
```

This certifies you wrote the code or have the right to submit it under the Apache 2.0 license.

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
