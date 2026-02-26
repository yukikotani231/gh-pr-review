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
	// シェル補完モード
	if len(os.Args) >= 2 && os.Args[1] == "--__complete" {
		runCompletion()
		return
	}

	var prNumber int

	if len(os.Args) >= 2 {
		n, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: PR number must be an integer: %v\n", err)
			os.Exit(1)
		}
		prNumber = n
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

	// PR番号が未指定の場合、選択UIを表示
	if prNumber == 0 {
		prNumber, err = runSelector(client)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if prNumber == 0 {
			os.Exit(0)
		}
	}

	model := tui.NewModel(client, prNumber)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSelector(client *gh.Client) (int, error) {
	selector := tui.NewSelectorModel(client)
	p := tea.NewProgram(selector, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return 0, err
	}
	m := finalModel.(tui.SelectorModel)
	if m.Quitting() {
		return 0, nil
	}
	return m.Selected(), nil
}

func runCompletion() {
	repo, err := repository.Current()
	if err != nil {
		return
	}

	client, err := gh.NewClient(repo.Owner, repo.Name)
	if err != nil {
		return
	}

	prs, err := client.FetchOpenPRs()
	if err != nil {
		return
	}

	for _, pr := range prs {
		_, _ = fmt.Fprintf(os.Stdout, "%d\t#%d: %s (@%s)\n", pr.Number, pr.Number, pr.Title, pr.Author)
	}
}
