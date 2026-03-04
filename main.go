package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cli/go-gh/v2/pkg/repository"

	gh "github.com/yukikotani231/gh-pr-review/internal/github"
	"github.com/yukikotani231/gh-pr-review/internal/tui"
)

const version = "v0.1.0"

type cliOptions struct {
	repoOverride string
	targetArg    string
	showHelp     bool
	showVersion  bool
}

func main() {
	// シェル補完モード
	if len(os.Args) >= 2 && os.Args[1] == "--__complete" {
		runCompletion()
		return
	}

	opts, err := parseCLIArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printUsage(os.Stderr)
		os.Exit(1)
	}

	if opts.showHelp {
		printUsage(os.Stdout)
		return
	}
	if opts.showVersion {
		fmt.Fprintf(os.Stdout, "gh pr-review %s\n", version)
		return
	}

	var (
		prNumber  int
		urlOwner  string
		urlRepo   string
		hasURLArg bool
	)
	if opts.targetArg != "" {
		var parseErr error
		prNumber, urlOwner, urlRepo, hasURLArg, parseErr = parsePRTarget(opts.targetArg)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
			os.Exit(1)
		}
	}

	var owner, repoName string
	if hasURLArg {
		owner = urlOwner
		repoName = urlRepo
	}

	if opts.repoOverride != "" {
		overrideOwner, overrideRepo, err := parseRepoOverride(opts.repoOverride)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if hasURLArg && (overrideOwner != owner || overrideRepo != repoName) {
			fmt.Fprintf(
				os.Stderr,
				"Error: --repo (%s/%s) does not match PR URL repository (%s/%s)\n",
				overrideOwner, overrideRepo, owner, repoName,
			)
			os.Exit(1)
		}
		owner = overrideOwner
		repoName = overrideRepo
	}

	if owner == "" || repoName == "" {
		repo, err := repository.Current()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not determine repository: %v\n", err)
			fmt.Fprintf(os.Stderr, "  Make sure you run this from inside a git repository.\n")
			os.Exit(1)
		}
		owner = repo.Owner
		repoName = repo.Name
	}

	client, err := gh.NewClient(owner, repoName)
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

func parseCLIArgs(args []string) (cliOptions, error) {
	var opts cliOptions
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--help" || arg == "-h":
			opts.showHelp = true
		case arg == "--version":
			opts.showVersion = true
		case arg == "--repo" || arg == "-R":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%s requires value in OWNER/REPO format", arg)
			}
			i++
			opts.repoOverride = args[i]
		case strings.HasPrefix(arg, "--repo="):
			opts.repoOverride = strings.TrimPrefix(arg, "--repo=")
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown option: %s", arg)
		default:
			positional = append(positional, arg)
		}
	}

	if len(positional) > 1 {
		return opts, fmt.Errorf("too many arguments: %s", strings.Join(positional, " "))
	}
	if len(positional) == 1 {
		opts.targetArg = positional[0]
	}

	return opts, nil
}

func parsePRTarget(arg string) (number int, owner, repo string, isURL bool, err error) {
	if n, parseErr := strconv.Atoi(arg); parseErr == nil {
		if n <= 0 {
			return 0, "", "", false, fmt.Errorf("PR number must be greater than 0")
		}
		return n, "", "", false, nil
	}

	owner, repo, number, ok, urlErr := parsePullURL(arg)
	if !ok {
		return 0, "", "", false, fmt.Errorf("invalid PR target: %q (use PR number or GitHub PR URL)", arg)
	}
	if urlErr != nil {
		return 0, "", "", false, urlErr
	}
	return number, owner, repo, true, nil
}

func parsePullURL(raw string) (owner, repo string, number int, ok bool, err error) {
	u, parseErr := url.Parse(raw)
	if parseErr != nil || u.Scheme == "" || u.Host == "" {
		return "", "", 0, false, nil
	}

	if !strings.EqualFold(u.Host, "github.com") {
		return "", "", 0, true, fmt.Errorf("unsupported host in PR URL: %s", u.Host)
	}

	cleanPath := path.Clean(strings.Trim(u.Path, "/"))
	parts := strings.Split(cleanPath, "/")
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", 0, true, fmt.Errorf("invalid PR URL path: %s", u.Path)
	}

	n, convErr := strconv.Atoi(parts[3])
	if convErr != nil || n <= 0 {
		return "", "", 0, true, fmt.Errorf("invalid PR number in URL: %q", parts[3])
	}

	return parts[0], parts[1], n, true, nil
}

func parseRepoOverride(input string) (owner, repo string, err error) {
	parts := strings.Split(input, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("--repo must be in OWNER/REPO format: got %q", input)
	}
	return parts[0], parts[1], nil
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, "gh pr-review %s\n\n", version)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  gh pr-review [OPTIONS] [<PR number>|<PR URL>]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  -R, --repo OWNER/REPO  Select repository explicitly")
	_, _ = fmt.Fprintln(w, "  -h, --help             Show help")
	_, _ = fmt.Fprintln(w, "      --version          Show version")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  gh pr-review 123")
	_, _ = fmt.Fprintln(w, "  gh pr-review https://github.com/owner/repo/pull/123")
	_, _ = fmt.Fprintln(w, "  gh pr-review -R owner/repo 123")
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
