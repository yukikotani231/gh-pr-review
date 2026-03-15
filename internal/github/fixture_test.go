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
		"threads": [{"id":"T1","is_resolved":true,"path":"main.go","line":1,"diff_side":"RIGHT","comments":[]}]
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
	if fixture.PullRequest.ChangedFiles != 1 {
		t.Fatalf("PullRequest.ChangedFiles = %d, want 1", fixture.PullRequest.ChangedFiles)
	}
	if fixture.DiffResult.Patches["main.go"] == "" {
		t.Fatal("expected patch for main.go")
	}
	if len(fixture.Threads) != 1 || !fixture.Threads[0].IsResolved {
		t.Fatalf("expected resolved thread to be decoded, got %+v", fixture.Threads)
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

func TestFixtureClientMutatesFixtureStateLocally(t *testing.T) {
	fixture := &FixtureData{
		Owner: "demo",
		Repo:  "sample",
		PullRequest: PullRequest{
			ID:     "PR_fixture",
			Title:  "Fixture PR",
			Number: 42,
			Files: []PRFile{
				{Path: "main.go", ViewerViewedState: ViewedStateUnviewed},
			},
		},
		DiffResult: DiffResult{
			Patches:           map[string]string{"main.go": "@@ -1 +1 @@\n-old\n+new"},
			FileStatuses:      map[string]FileStatus{"main.go": FileStatusModified},
			PreviousFilenames: map[string]string{},
		},
		Threads: []ReviewThread{
			{
				ID:         "thread-1",
				Path:       "main.go",
				Line:       1,
				DiffSide:   DiffSideRight,
				IsResolved: false,
				Comments: []ReviewComment{
					{ID: "comment-1", Body: "existing", Author: "alice", CreatedAt: "2026-03-15T00:00:00Z"},
				},
			},
		},
	}

	client := NewFixtureClient(fixture)

	if err := client.MarkFileAsViewed("PR_fixture", "main.go"); err != nil {
		t.Fatalf("MarkFileAsViewed: %v", err)
	}
	if fixture.PullRequest.Files[0].ViewerViewedState != ViewedStateViewed {
		t.Fatalf("ViewerViewedState = %q, want VIEWED", fixture.PullRequest.Files[0].ViewerViewedState)
	}

	if err := client.AddComment("PR_fixture", "main.go", "new top-level comment", DiffSideRight, 2); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if len(fixture.Threads) != 2 {
		t.Fatalf("len(Threads) = %d, want 2", len(fixture.Threads))
	}

	if err := client.ReplyToThread("thread-1", "reply body"); err != nil {
		t.Fatalf("ReplyToThread: %v", err)
	}
	if got := fixture.Threads[0].Comments[len(fixture.Threads[0].Comments)-1].Body; got != "reply body" {
		t.Fatalf("reply body = %q, want reply body", got)
	}

	if err := client.ResolveThread("thread-1"); err != nil {
		t.Fatalf("ResolveThread: %v", err)
	}
	if !fixture.Threads[0].IsResolved {
		t.Fatal("expected thread-1 to be resolved")
	}

	if err := client.UnresolveThread("thread-1"); err != nil {
		t.Fatalf("UnresolveThread: %v", err)
	}
	if fixture.Threads[0].IsResolved {
		t.Fatal("expected thread-1 to be unresolved")
	}
}
