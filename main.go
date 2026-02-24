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
		fmt.Fprintf(os.Stderr, "Usage: gh pr-review <PR number>\n")
		os.Exit(1)
	}

	prNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: PR number must be an integer: %v\n", err)
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

	model := tui.NewModel(client, prNumber)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
