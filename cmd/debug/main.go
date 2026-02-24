package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"

	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run ./cmd/debug <PR number>\n")
		os.Exit(1)
	}

	prNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: PR number must be an integer: %v\n", err)
		os.Exit(1)
	}

	// Step 1: Repository detection
	fmt.Println("=== Step 1: Repository Detection ===")
	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		fmt.Fprintf(os.Stderr, "  -> Make sure you run this from inside a git repository\n")
		os.Exit(1)
	}
	fmt.Printf("OK: %s/%s\n\n", repo.Owner, repo.Name)

	// Step 2: API Client
	fmt.Println("=== Step 2: API Client Init ===")
	client, err := gh.NewClient(repo.Owner, repo.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")
	fmt.Println()

	// Step 3: Fetch PR
	fmt.Printf("=== Step 3: Fetch PR #%d ===\n", prNumber)
	pr, err := client.FetchPR(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %s\n", pr.Title)
	fmt.Printf("  ID: %s\n", pr.ID)
	fmt.Printf("  +%d -%d, %d files\n", pr.Additions, pr.Deletions, pr.ChangedFiles)
	fmt.Println()

	// Step 4: Files & Viewed State
	fmt.Println("=== Step 4: Files & Viewed State ===")
	for _, f := range pr.Files {
		check := "[ ]"
		if f.ViewerViewedState == gh.ViewedStateViewed {
			check = "[x]"
		}
		fmt.Printf("  %s %s (+%d -%d)\n", check, f.Path, f.Additions, f.Deletions)
	}
	fmt.Println()

	// Step 5: Fetch Diffs
	fmt.Println("=== Step 5: Fetch Diffs (REST API) ===")
	patches, err := client.FetchDiffs(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %d files with patches\n", len(patches))
	for path, patch := range patches {
		lines := strings.Count(patch, "\n")
		fmt.Printf("  %s (%d lines)\n", path, lines)
	}
	fmt.Println()

	// Step 6: Diff Parser
	fmt.Println("=== Step 6: Diff Parser ===")
	if len(pr.Files) > 0 {
		first := pr.Files[0]
		patch := patches[first.Path]
		diffLines := diff.Parse(patch)
		fmt.Printf("OK: %s -> %d DiffLines\n", first.Path, len(diffLines))
		limit := 10
		if len(diffLines) < limit {
			limit = len(diffLines)
		}
		for i := 0; i < limit; i++ {
			dl := diffLines[i]
			typeName := []string{"CTX", "ADD", "DEL", "HDR"}[dl.Type]
			fmt.Printf("  [%s] old:%d new:%d %s\n", typeName, dl.OldLineNum, dl.NewLineNum, truncate(dl.Content, 60))
		}
		if len(diffLines) > limit {
			fmt.Printf("  ... (%d more lines)\n", len(diffLines)-limit)
		}
	}
	fmt.Println()

	// Step 7: Fetch Review Threads
	fmt.Println("=== Step 7: Fetch Review Threads ===")
	threads, err := client.FetchReviewThreads(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %d threads\n", len(threads))
	for _, t := range threads {
		resolved := "unresolved"
		if t.IsResolved {
			resolved = "resolved"
		}
		fmt.Printf("  [%s] %s:%d (%s side, %d comments)\n",
			resolved, t.Path, t.Line, t.DiffSide, len(t.Comments))
		for _, c := range t.Comments {
			body := truncate(strings.ReplaceAll(c.Body, "\n", " "), 50)
			fmt.Printf("    @%s: %s\n", c.Author, body)
		}
	}
	fmt.Println()

	fmt.Println("=== All checks passed! ===")
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}
