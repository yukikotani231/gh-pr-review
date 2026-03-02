# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go-based GitHub CLI extension (`gh pr-review`). The entrypoint is `main.go`.
- `internal/github/`: GitHub API client, GraphQL queries, and API-facing types.
- `internal/diff/`: diff parsing logic used by the TUI.
- `internal/tui/`: Bubble Tea models, view rendering, keymaps, and interaction flows.
- `.github/workflows/`: CI (`test.yml`) and release (`release.yml`) pipelines.
Keep new code in `internal/<domain>` packages and avoid exporting symbols unless needed across packages.

## Build, Test, and Development Commands
- `go build ./...`: compile all packages (quick sanity check).
- `go build -o gh-pr-review .`: build a local binary for manual testing.
- `go test -v -race -count=1 ./...`: run the full test suite with race detection (matches CI intent).
- `go vet ./...`: run static checks used in CI.
- `golangci-lint run ./...`: run configured linters from `.golangci.yml`.
- `gh extension install .`: install this extension from the local checkout.

## Coding Style & Naming Conventions
Follow idiomatic Go and always run `gofmt` (or editor format-on-save) before committing.
- Indentation: tabs (Go default), never manual alignment with spaces.
- Naming: package names are short and lowercase (`tui`, `diff`); exported identifiers use `CamelCase`; unexported use `camelCase`.
- Files: prefer descriptive names by responsibility (`client.go`, `parser.go`, `diffview.go`).
Linting focuses on `errcheck`, `staticcheck`, `govet`, `gocritic`, and related correctness linters.

## Testing Guidelines
Tests live next to implementation files as `*_test.go` (examples: `internal/diff/parser_test.go`, `internal/tui/diffview_test.go`).
- Prefer table-driven tests for parsers and command logic.
- Cover edge cases around diff hunks, line mapping, and narrow terminal widths.
- Run `go test -race ./...` locally before opening a PR.

## Commit & Pull Request Guidelines
Use Conventional Commit-style prefixes seen in history: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`, `ci:`.
Example: `fix: truncate file list lines in narrow panes`.

For pull requests:
- Write a clear summary of behavior changes and impacted modules.
- Link the related issue/PR number when applicable (`#123`).
- Include terminal screenshots or recordings for TUI-visible changes.
- Confirm `go build`, `go test -race -count=1 ./...`, and `go vet ./...` pass.
