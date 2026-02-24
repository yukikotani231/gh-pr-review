# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o gh-pr-review .

# Run all tests with race detector
go test -v -race -count=1 ./...

# Run tests for a single package
go test -v ./internal/diff/
go test -v ./internal/tui/
go test -v ./internal/github/

# Run a single test
go test -v -run TestParse_SingleHunk ./internal/diff/

# Vet
go vet ./...

# Install locally as gh extension
gh extension install .

# Uninstall
gh extension remove pr-review
```

## Architecture

This is a GitHub CLI extension (`gh pr-review <PR number>`) built with Go and the Bubble Tea TUI framework. It runs inside a git repository and auto-detects the GitHub owner/repo.

### Package structure

- **`main.go`** — Entry point. Parses args, detects repo via `go-gh`, initializes API client, starts Bubble Tea program.
- **`internal/github/`** — GitHub API layer. `Client` wraps both GraphQL (`go-gh` `api.GraphQLClient`) and REST (`api.RESTClient`). All queries/mutations are raw strings in `queries.go`. Types are in `types.go`. Pagination is handled internally (100 items per page).
- **`internal/diff/`** — Pure diff parsing. `Parse()` converts a unified diff patch string into `[]DiffLine` with tracked old/new line numbers. `RenderLine()` produces colored terminal output.
- **`internal/tui/`** — Bubble Tea TUI. The main `Model` in `model.go` orchestrates everything. `FileListModel` (left pane) and `DiffViewModel` (right pane) are sub-components. Input modes: NORMAL → COMMENT/REPLY/REVIEW.

### Data flow

1. `Init()` fires `fetchPRCmd` (GraphQL)
2. On `PRFetchedMsg`, fires `fetchDiffsCmd` (REST) and `fetchThreadsCmd` (GraphQL) concurrently via `tea.Batch`
3. Both use `bool` flags (`patchesFetched`/`threadsFetched`) to track completion — **not nil checks** (a nil slice from zero results vs "not yet fetched" must be distinguished)
4. On ready, `DiffViewModel` renders parsed diff lines with inline review threads mapped by line number

### Key bindings are defined in `keymap.go`, styles in `styles.go`, async message types in `messages.go`.

## Conventions

- GraphQL queries use `client.gql.Do()` with raw query strings, not the shurcooL struct-tag approach
- API error messages are in Japanese (matches the user's locale)
- `DiffViewModel.buildDisplayRows()` maps diff lines + thread comments into flat `displayRow` slices for rendering; thread comments have `diffLineIdx = -1`
- `matchesThread()` uses `DiffSide` (LEFT/RIGHT) to match threads to old vs new line numbers
