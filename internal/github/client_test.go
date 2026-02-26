package github

import (
	"strings"
	"testing"
)

func TestViewedState_Constants(t *testing.T) {
	if ViewedStateViewed != "VIEWED" {
		t.Errorf("ViewedStateViewed = %q, want %q", ViewedStateViewed, "VIEWED")
	}
	if ViewedStateUnviewed != "UNVIEWED" {
		t.Errorf("ViewedStateUnviewed = %q, want %q", ViewedStateUnviewed, "UNVIEWED")
	}
}

func TestDiffSide_Constants(t *testing.T) {
	if DiffSideLeft != "LEFT" {
		t.Errorf("DiffSideLeft = %q, want %q", DiffSideLeft, "LEFT")
	}
	if DiffSideRight != "RIGHT" {
		t.Errorf("DiffSideRight = %q, want %q", DiffSideRight, "RIGHT")
	}
}

func TestReviewEvent_Constants(t *testing.T) {
	if ReviewEventApprove != "APPROVE" {
		t.Errorf("ReviewEventApprove = %q, want %q", ReviewEventApprove, "APPROVE")
	}
	if ReviewEventRequestChanges != "REQUEST_CHANGES" {
		t.Errorf("ReviewEventRequestChanges = %q, want %q", ReviewEventRequestChanges, "REQUEST_CHANGES")
	}
	if ReviewEventComment != "COMMENT" {
		t.Errorf("ReviewEventComment = %q, want %q", ReviewEventComment, "COMMENT")
	}
}

func TestPullRequest_Fields(t *testing.T) {
	pr := PullRequest{
		ID:           "PR_123",
		Title:        "Test PR",
		Number:       42,
		Additions:    10,
		Deletions:    5,
		ChangedFiles: 3,
		Files: []PRFile{
			{Path: "a.go", Additions: 5, Deletions: 2, ViewerViewedState: ViewedStateViewed},
			{Path: "b.go", Additions: 5, Deletions: 3, ViewerViewedState: ViewedStateUnviewed},
		},
	}

	if pr.ID != "PR_123" {
		t.Errorf("ID = %q", pr.ID)
	}
	if len(pr.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(pr.Files))
	}
	if pr.Files[0].ViewerViewedState != ViewedStateViewed {
		t.Error("first file should be viewed")
	}
}

func TestReviewThread_Fields(t *testing.T) {
	thread := ReviewThread{
		ID:         "RT_1",
		IsResolved: false,
		Path:       "main.go",
		Line:       10,
		DiffSide:   DiffSideRight,
		Comments: []ReviewComment{
			{ID: "C_1", Body: "Fix this", Author: "alice", CreatedAt: "2026-01-01T00:00:00Z"},
			{ID: "C_2", Body: "Done", Author: "bob", CreatedAt: "2026-01-01T01:00:00Z"},
		},
	}

	if thread.IsResolved {
		t.Error("thread should not be resolved")
	}
	if len(thread.Comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(thread.Comments))
	}
	if thread.Comments[0].Author != "alice" {
		t.Errorf("first comment author = %q, want alice", thread.Comments[0].Author)
	}
}

func TestFileStatus_Constants(t *testing.T) {
	tests := []struct {
		status FileStatus
		want   string
	}{
		{FileStatusAdded, "added"},
		{FileStatusModified, "modified"},
		{FileStatusRemoved, "removed"},
		{FileStatusRenamed, "renamed"},
		{FileStatusCopied, "copied"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("FileStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestDiffResult_Fields(t *testing.T) {
	result := DiffResult{
		Patches:           map[string]string{"a.go": "+line"},
		FileStatuses:      map[string]FileStatus{"a.go": FileStatusModified},
		PreviousFilenames: map[string]string{"b.go": "old_b.go"},
	}
	if result.Patches["a.go"] != "+line" {
		t.Error("unexpected patch")
	}
	if result.FileStatuses["a.go"] != FileStatusModified {
		t.Error("unexpected status")
	}
	if result.PreviousFilenames["b.go"] != "old_b.go" {
		t.Error("unexpected previous filename")
	}
}

func TestGraphQLQueryStrings_NonEmpty(t *testing.T) {
	queries := map[string]string{
		"prFilesQuery":              prFilesQuery,
		"markFileAsViewedMutation":  markFileAsViewedMutation,
		"unmarkFileAsViewedMutation": unmarkFileAsViewedMutation,
		"reviewThreadsQuery":        reviewThreadsQuery,
		"addReviewCommentMutation":  addReviewCommentMutation,
		"replyToThreadMutation":     replyToThreadMutation,
		"resolveThreadMutation":     resolveThreadMutation,
		"unresolveThreadMutation":   unresolveThreadMutation,
		"submitReviewMutation":      submitReviewMutation,
	}

	for name, q := range queries {
		if q == "" {
			t.Errorf("query %s is empty", name)
		}
	}
}

func TestGraphQLQueryStrings_ContainExpectedFields(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			"prFilesQuery",
			prFilesQuery,
			[]string{"viewerViewedState", "pullRequest", "files", "pageInfo", "hasNextPage"},
		},
		{
			"reviewThreadsQuery",
			reviewThreadsQuery,
			[]string{"reviewThreads", "isResolved", "diffSide", "comments"},
		},
		{
			"markFileAsViewedMutation",
			markFileAsViewedMutation,
			[]string{"markFileAsViewed", "pullRequestId", "path"},
		},
		{
			"addReviewCommentMutation",
			addReviewCommentMutation,
			[]string{"addPullRequestReview", "threads", "body", "path", "line", "side"},
		},
		{
			"submitReviewMutation",
			submitReviewMutation,
			[]string{"addPullRequestReview", "event"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, field := range tt.expected {
				if !strings.Contains(tt.query, field) {
					t.Errorf("query %s should contain %q", tt.name, field)
				}
			}
		})
	}
}
