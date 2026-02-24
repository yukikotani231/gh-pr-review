package main

import (
	"fmt"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cli/go-gh/v2/pkg/repository"

	gh "github.com/yukikotani231/gh-pr-review/internal/github"
	"github.com/yukikotani231/gh-pr-review/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gh pr-review <PR number> [--debug]\n")
		os.Exit(1)
	}

	// Parse args: support "gh pr-review 123" and "gh pr-review --debug 123"
	var prNumber int
	var debug bool
	for _, arg := range os.Args[1:] {
		if arg == "--debug" {
			debug = true
			continue
		}
		if n, err := strconv.Atoi(arg); err == nil {
			prNumber = n
		}
	}

	if prNumber == 0 {
		fmt.Fprintf(os.Stderr, "Error: PR number is required\n")
		fmt.Fprintf(os.Stderr, "Usage: gh pr-review <PR number> [--debug]\n")
		os.Exit(1)
	}

	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine repository: %v\n", err)
		fmt.Fprintf(os.Stderr, "  Make sure you run this from inside a git repository.\n")
		os.Exit(1)
	}

	client, err := gh.NewClient(repo.Owner, repo.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if debug {
		runDebug(client, repo.Owner, repo.Name, prNumber)
		return
	}

	model := tui.NewModel(client, prNumber)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runDebug(client *gh.Client, owner, repo string, prNumber int) {
	fmt.Printf("repo: %s/%s\n", owner, repo)
	fmt.Printf("PR: #%d\n\n", prNumber)

	fmt.Println("Fetching PR...")
	pr, err := client.FetchPR(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL FetchPR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %s (+%d -%d, %d files)\n\n", pr.Title, pr.Additions, pr.Deletions, pr.ChangedFiles)

	for _, f := range pr.Files {
		mark := "[ ]"
		if f.ViewerViewedState == gh.ViewedStateViewed {
			mark = "[x]"
		}
		fmt.Printf("  %s %s +%d -%d\n", mark, f.Path, f.Additions, f.Deletions)
	}

	fmt.Println("\nFetching diffs...")
	patches, err := client.FetchDiffs(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL FetchDiffs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %d patches\n\n", len(patches))

	fmt.Println("Fetching review threads...")
	threads, err := client.FetchReviewThreads(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL FetchReviewThreads: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %d threads\n", len(threads))
	for _, t := range threads {
		status := "open"
		if t.IsResolved {
			status = "resolved"
		}
		fmt.Printf("  [%s] %s:%d (%d comments)\n", status, t.Path, t.Line, len(t.Comments))
	}

	fmt.Println("\nAll checks passed!")
}
