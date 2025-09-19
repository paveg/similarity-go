# Repository Guidelines

## Project Structure & Module Organization

- Core CLI lives in `cmd/similarity-go` and orchestrates repository scans.
- Domain logic sits under `internal/` (notably `ast`, `similarity`, `config`, `worker`); keep new internals private here.
- Shared helpers reside in `pkg/` (e.g., `pkg/math-util`, `pkg/types`), fixtures in `testdata/`, documentation in `docs/`, and built binaries in `bin/`.
- Maintain clear separation: cross-package utilities go in `pkg/`, CLI-facing assets in `cmd/`, and benchmarks under `weight-benchmark/`.

## Build, Test, and Development Commands

- `make build` compiles the CLI to `bin/similarity-go`; run before publishing binaries.
- `make test` triggers the race-enabled suite; use `go test ./...` for faster iterations.
- `make lint` wraps `golangci-lint` with the repo’s `.golangci.yml`; install golangci-lint v2.x locally.
- `make quality` chains format, vet, lint, and coverage reports—this matches the CI expectations.

## Coding Style & Naming Conventions

- Target Go 1.25; always format via `goimports`/`golines` (invoked through `make lint` or your editor).
- Stick to Go defaults: tabs for indentation, exported identifiers in `CamelCase`, unexported in `camelCase`.
- Keep packages cohesive and filenames descriptive (`worker_pool.go`), and prefer small, focused public APIs in `pkg/`.

## Testing Guidelines

- Co-locate tests as `*_test.go`; table-driven tests with `t.Run` are encouraged for coverage clarity.
- Store fixtures and golden files in `testdata/`; avoid mutating committed samples.
- Check coverage with `go test -cover ./...`; maintain the project’s ~80% baseline via `make test-coverage`.
- Concurrency or worker changes must pass `go test -race ./...` to keep the pipeline race-free.

## Commit & Pull Request Guidelines

- Use Conventional Commit verbs (`feat:`, `fix:`, `refactor:`), optionally prefixed with the emoji style seen in history (e.g., `:recycle:`).
- Write focused commits with explanatory bodies when behavior shifts or thresholds change.
- Pull requests should include change summaries, validation commands, linked issues, and screenshots/logs for CLI output adjustments.
- Rebase onto `main`, ensure CI is green, and request review only after lint/test targets pass locally.

## Configuration Tips

- Default thresholds and weights live in `.similarity-config.yaml`; update docs in `docs/` when introducing new keys.
- Keep sensitive values out of version control; rely on environment variables or ignored `.env` files for local secrets.
