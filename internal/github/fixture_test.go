package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFixtureData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "basic.json")
	content := `{
		"owner": "demo",
		"repo": "sample",
		"pull_request": {
			"id": "PR_fixture",
			"title": "Fixture PR",
			"number": 42,
			"additions": 3,
			"deletions": 1,
			"changed_files": 1,
			"files": [{"path": "main.go", "additions": 3, "deletions": 1, "viewer_viewed_state": "UNVIEWED"}]
		},
		"diff_result": {
			"patches": {"main.go": "@@ -1 +1 @@\n-old\n+new"},
			"file_statuses": {"main.go": "modified"},
			"previous_filenames": {}
		},
		"threads": []
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fixture, err := LoadFixtureData(path)
	if err != nil {
		t.Fatalf("LoadFixtureData: %v", err)
	}
	if fixture.PullRequest.Number != 42 {
		t.Fatalf("PullRequest.Number = %d, want 42", fixture.PullRequest.Number)
	}
	if fixture.DiffResult.Patches["main.go"] == "" {
		t.Fatal("expected patch for main.go")
	}
}

func TestFixtureClientFetchesFixtureData(t *testing.T) {
	fixture := &FixtureData{
		Owner: "demo",
		Repo:  "sample",
		PullRequest: PullRequest{
			ID:           "PR_fixture",
			Title:        "Fixture PR",
			Number:       42,
			Additions:    3,
			Deletions:    1,
			ChangedFiles: 1,
			Files: []PRFile{
				{Path: "main.go", Additions: 3, Deletions: 1, ViewerViewedState: ViewedStateUnviewed},
			},
		},
		DiffResult: DiffResult{
			Patches:           map[string]string{"main.go": "@@ -1 +1 @@\n-old\n+new"},
			FileStatuses:      map[string]FileStatus{"main.go": FileStatusModified},
			PreviousFilenames: map[string]string{},
		},
		Threads: []ReviewThread{
			{ID: "T1", Path: "main.go", Line: 1, DiffSide: DiffSideRight},
		},
	}

	client := NewFixtureClient(fixture)

	pr, err := client.FetchPR(42)
	if err != nil {
		t.Fatalf("FetchPR: %v", err)
	}
	if pr.Title != "Fixture PR" {
		t.Fatalf("pr.Title = %q, want Fixture PR", pr.Title)
	}

	diff, err := client.FetchDiffs(42)
	if err != nil {
		t.Fatalf("FetchDiffs: %v", err)
	}
	if diff.FileStatuses["main.go"] != FileStatusModified {
		t.Fatalf("status = %q, want modified", diff.FileStatuses["main.go"])
	}

	threads, err := client.FetchReviewThreads(42)
	if err != nil {
		t.Fatalf("FetchReviewThreads: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("len(threads) = %d, want 1", len(threads))
	}
}

func TestFixtureClientIsReadOnly(t *testing.T) {
	client := NewFixtureClient(&FixtureData{
		Owner: "demo",
		Repo:  "sample",
		PullRequest: PullRequest{
			ID:     "PR_fixture",
			Title:  "Fixture PR",
			Number: 42,
		},
		DiffResult: DiffResult{
			Patches:           map[string]string{},
			FileStatuses:      map[string]FileStatus{},
			PreviousFilenames: map[string]string{},
		},
	})

	err := client.SubmitReview("PR_fixture", ReviewEventComment, "")
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("SubmitReview error = %v, want read-only error", err)
	}
}
