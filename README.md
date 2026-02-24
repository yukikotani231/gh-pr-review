# gh pr-review

A GitHub CLI extension for reviewing pull requests in your terminal. Browse files, view diffs, mark files as viewed, leave comments, and submit reviews — all without leaving the command line.

## Installation

```bash
gh extension install yukikotani231/gh-pr-review
```

## Usage

```bash
# Run from inside a git repo
gh pr-review <PR number>
```

## Key Bindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Tab` | Switch between file list and diff pane |
| `Ctrl+d` | Half page down (diff pane) |
| `Ctrl+u` | Half page up (diff pane) |
| `q` / `Ctrl+c` | Quit |

### Review Actions

| Key | Action |
|-----|--------|
| `v` | Toggle file as viewed/unviewed |
| `c` | Add a comment on the current line |
| `r` | Reply to a review thread |
| `R` | Resolve/unresolve a thread |
| `n` / `N` | Jump to next/previous review thread |
| `S` | Submit review (Approve / Request Changes / Comment) |

### Input Mode (comment/reply)

| Key | Action |
|-----|--------|
| `Ctrl+s` | Submit comment |
| `Esc` | Cancel |

## Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  PR #1234: Fix auth bug  (+45 -12, 8 files, 3/8 viewed)       │
├──────────────────────┬──────────────────────────────────────────┤
│ [ ] src/auth.go  +20 │  10   11  @@ -10,6 +10,8 @@             │
│ [✓] src/main.go   +5 │  10   11   func main() {                │
│ [ ] src/util.go  +10 │  11      -    oldCode()                  │
│                      │       12 +    newCode()                  │
│                      │       13 +    extraLine()                │
│                      │  12   14      existingCode()             │
├──────────────────────┴──────────────────────────────────────────┤
│  src/auth.go              ↑/k up  ↓/j down  v viewed  q quit   │
└─────────────────────────────────────────────────────────────────┘
```

## Requirements

- [GitHub CLI](https://cli.github.com/) (`gh`) v2.0+
- Authenticated via `gh auth login`
- Must be run from inside a git repository that has a GitHub remote

## Development

```bash
# Build
go build -o gh-pr-review .

# Test
go test -race ./...

# Lint
golangci-lint run ./...

# Install locally
gh extension install .

# Uninstall
gh extension remove pr-review
```

## License

MIT
