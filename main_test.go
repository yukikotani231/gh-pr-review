package main

import "testing"

func TestParsePRTarget_Number(t *testing.T) {
	n, owner, repo, isURL, err := parsePRTarget("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 123 {
		t.Fatalf("number = %d, want 123", n)
	}
	if owner != "" || repo != "" {
		t.Fatalf("owner/repo should be empty for numeric target, got %q/%q", owner, repo)
	}
	if isURL {
		t.Fatal("isURL should be false for numeric target")
	}
}

func TestParsePRTarget_URL(t *testing.T) {
	n, owner, repo, isURL, err := parsePRTarget("https://github.com/cli/cli/pull/999#discussion_r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 999 {
		t.Fatalf("number = %d, want 999", n)
	}
	if owner != "cli" || repo != "cli" {
		t.Fatalf("owner/repo = %q/%q, want cli/cli", owner, repo)
	}
	if !isURL {
		t.Fatal("isURL should be true for URL target")
	}
}

func TestParsePRTarget_EnterpriseURL(t *testing.T) {
	n, owner, repo, isURL, err := parsePRTarget("https://github.example.com/cli/cli/pull/999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 999 {
		t.Fatalf("number = %d, want 999", n)
	}
	if owner != "cli" || repo != "cli" {
		t.Fatalf("owner/repo = %q/%q, want cli/cli", owner, repo)
	}
	if !isURL {
		t.Fatal("isURL should be true for URL target")
	}
}

func TestParsePRTarget_Invalid(t *testing.T) {
	_, _, _, _, err := parsePRTarget("https://github.com/cli/cli/issues/1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRepoOverride(t *testing.T) {
	owner, repo, err := parseRepoOverride("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "owner" || repo != "repo" {
		t.Fatalf("owner/repo = %q/%q, want owner/repo", owner, repo)
	}
}

func TestParseRepoOverride_Invalid(t *testing.T) {
	_, _, err := parseRepoOverride("owner")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSameRepo_CaseInsensitive(t *testing.T) {
	if !sameRepo("Owner", "Repo", "owner", "repo") {
		t.Fatal("expected sameRepo to treat owner/repo as case-insensitive")
	}
}

func TestParseCLIArgs_Basic(t *testing.T) {
	opts, err := parseCLIArgs([]string{"-R", "owner/repo", "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.repoOverride != "owner/repo" {
		t.Fatalf("repoOverride = %q, want owner/repo", opts.repoOverride)
	}
	if opts.targetArg != "42" {
		t.Fatalf("targetArg = %q, want 42", opts.targetArg)
	}
	if opts.showHelp {
		t.Fatal("showHelp should be false")
	}
}

func TestParseCLIArgs_Help(t *testing.T) {
	opts, err := parseCLIArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.showHelp {
		t.Fatal("showHelp should be true")
	}
}

func TestParseCLIArgs_Version(t *testing.T) {
	opts, err := parseCLIArgs([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.showVersion {
		t.Fatal("showVersion should be true")
	}
}

func TestParseCLIArgs_UnknownOption(t *testing.T) {
	_, err := parseCLIArgs([]string{"--unknown"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCLIArgs_TooManyArgs(t *testing.T) {
	_, err := parseCLIArgs([]string{"1", "2"})
	if err == nil {
		t.Fatal("expected error")
	}
}
