package github

import (
	"encoding/json"
	"errors"
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

func TestPRListItem_Fields(t *testing.T) {
	item := PRListItem{
		Number:    42,
		Title:     "Fix bug",
		Author:    "alice",
		UpdatedAt: "2026-01-01T00:00:00Z",
		IsDraft:   true,
	}
	if item.Number != 42 {
		t.Errorf("Number = %d, want 42", item.Number)
	}
	if item.Author != "alice" {
		t.Errorf("Author = %q, want alice", item.Author)
	}
	if !item.IsDraft {
		t.Error("IsDraft should be true")
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
		"prFilesQuery":               prFilesQuery,
		"markFileAsViewedMutation":   markFileAsViewedMutation,
		"unmarkFileAsViewedMutation": unmarkFileAsViewedMutation,
		"reviewThreadsQuery":         reviewThreadsQuery,
		"addReviewCommentMutation":   addReviewCommentMutation,
		"replyToThreadMutation":      replyToThreadMutation,
		"resolveThreadMutation":      resolveThreadMutation,
		"unresolveThreadMutation":    unresolveThreadMutation,
		"submitReviewMutation":       submitReviewMutation,
		"openPRsQuery":               openPRsQuery,
	}

	for name, q := range queries {
		if q == "" {
			t.Errorf("query %s is empty", name)
		}
	}
}

// --- Mock types ---

type mockDoCall struct {
	Query     string
	Variables map[string]interface{}
}

type mockGraphQL struct {
	DoFunc  func(query string, variables map[string]interface{}, response interface{}) error
	DoCalls []mockDoCall
}

func (m *mockGraphQL) Do(query string, variables map[string]interface{}, response interface{}) error {
	m.DoCalls = append(m.DoCalls, mockDoCall{Query: query, Variables: variables})
	return m.DoFunc(query, variables, response)
}

type mockREST struct {
	GetFunc  func(path string, response interface{}) error
	GetCalls []string
}

func (m *mockREST) Get(path string, response interface{}) error {
	m.GetCalls = append(m.GetCalls, path)
	return m.GetFunc(path, response)
}

func newTestClient(gql *mockGraphQL, rest *mockREST) *Client {
	return &Client{gql: gql, rest: rest, owner: "testowner", repo: "testrepo"}
}

func respondJSON(response interface{}, data string) error {
	return json.Unmarshal([]byte(data), response)
}

// --- FetchPR tests ---

func TestFetchPR_SinglePage(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			return respondJSON(response, `{
				"Repository": {
					"PullRequest": {
						"ID": "PR_1",
						"Title": "Test PR",
						"Additions": 10,
						"Deletions": 5,
						"ChangedFiles": 2,
						"Files": {
							"PageInfo": {"HasNextPage": false, "EndCursor": ""},
							"Nodes": [
								{"Path": "a.go", "Additions": 6, "Deletions": 3, "ViewerViewedState": "VIEWED"},
								{"Path": "b.go", "Additions": 4, "Deletions": 2, "ViewerViewedState": "UNVIEWED"}
							]
						}
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	pr, err := c.FetchPR(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.ID != "PR_1" {
		t.Errorf("ID = %q, want PR_1", pr.ID)
	}
	if pr.Title != "Test PR" {
		t.Errorf("Title = %q, want Test PR", pr.Title)
	}
	if pr.Number != 42 {
		t.Errorf("Number = %d, want 42", pr.Number)
	}
	if pr.Additions != 10 {
		t.Errorf("Additions = %d, want 10", pr.Additions)
	}
	if pr.Deletions != 5 {
		t.Errorf("Deletions = %d, want 5", pr.Deletions)
	}
	if pr.ChangedFiles != 2 {
		t.Errorf("ChangedFiles = %d, want 2", pr.ChangedFiles)
	}
	if len(pr.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(pr.Files))
	}
	if pr.Files[0].Path != "a.go" {
		t.Errorf("Files[0].Path = %q, want a.go", pr.Files[0].Path)
	}
	if pr.Files[0].ViewerViewedState != ViewedStateViewed {
		t.Errorf("Files[0].ViewerViewedState = %q, want VIEWED", pr.Files[0].ViewerViewedState)
	}
	if pr.Files[1].ViewerViewedState != ViewedStateUnviewed {
		t.Errorf("Files[1].ViewerViewedState = %q, want UNVIEWED", pr.Files[1].ViewerViewedState)
	}
	if len(gql.DoCalls) != 1 {
		t.Errorf("expected 1 Do call, got %d", len(gql.DoCalls))
	}
}

func TestFetchPR_MultiPage(t *testing.T) {
	callCount := 0
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			callCount++
			if callCount == 1 {
				return respondJSON(response, `{
					"Repository": {
						"PullRequest": {
							"ID": "PR_2", "Title": "Multi", "Additions": 20, "Deletions": 10, "ChangedFiles": 3,
							"Files": {
								"PageInfo": {"HasNextPage": true, "EndCursor": "cursor1"},
								"Nodes": [{"Path": "a.go", "Additions": 10, "Deletions": 5, "ViewerViewedState": "VIEWED"}]
							}
						}
					}
				}`)
			}
			return respondJSON(response, `{
				"Repository": {
					"PullRequest": {
						"ID": "PR_2", "Title": "Multi", "Additions": 20, "Deletions": 10, "ChangedFiles": 3,
						"Files": {
							"PageInfo": {"HasNextPage": false, "EndCursor": ""},
							"Nodes": [
								{"Path": "b.go", "Additions": 5, "Deletions": 3, "ViewerViewedState": "UNVIEWED"},
								{"Path": "c.go", "Additions": 5, "Deletions": 2, "ViewerViewedState": "UNVIEWED"}
							]
						}
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	pr, err := c.FetchPR(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pr.Files) != 3 {
		t.Fatalf("len(Files) = %d, want 3", len(pr.Files))
	}
	if pr.Files[0].Path != "a.go" || pr.Files[1].Path != "b.go" || pr.Files[2].Path != "c.go" {
		t.Errorf("unexpected file paths: %v", pr.Files)
	}
	if len(gql.DoCalls) != 2 {
		t.Fatalf("expected 2 Do calls, got %d", len(gql.DoCalls))
	}
	after, ok := gql.DoCalls[1].Variables["after"]
	if !ok || after != "cursor1" {
		t.Errorf("second call after = %v, want cursor1", after)
	}
}

func TestFetchPR_Error(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			return errors.New("network error")
		},
	}
	c := newTestClient(gql, nil)

	_, err := c.FetchPR(1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "PR情報の取得に失敗") {
		t.Errorf("error = %q, want containing 'PR情報の取得に失敗'", err.Error())
	}
}

// --- FetchDiffs tests ---

func TestFetchDiffs_SinglePage(t *testing.T) {
	rest := &mockREST{
		GetFunc: func(path string, response interface{}) error {
			if strings.Contains(path, "&page=1") {
				return respondJSON(response, `[
					{"filename": "a.go", "patch": "@@ -1,3 +1,4 @@\n+added"},
					{"filename": "b.go", "patch": "@@ -1 +1 @@\n-old\n+new"}
				]`)
			}
			return respondJSON(response, `[]`)
		},
	}
	c := newTestClient(nil, rest)

	result, err := c.FetchDiffs(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patches) != 2 {
		t.Fatalf("len(result.Patches) = %d, want 2", len(result.Patches))
	}
	if !strings.Contains(result.Patches["a.go"], "+added") {
		t.Errorf("result.Patches[a.go] = %q, want containing +added", result.Patches["a.go"])
	}
	if !strings.Contains(result.Patches["b.go"], "+new") {
		t.Errorf("result.Patches[b.go] = %q, want containing +new", result.Patches["b.go"])
	}
}

func TestFetchDiffs_MultiPage(t *testing.T) {
	rest := &mockREST{
		GetFunc: func(path string, response interface{}) error {
			if strings.Contains(path, "&page=1") {
				return respondJSON(response, `[{"filename": "a.go", "patch": "p1"}]`)
			}
			if strings.Contains(path, "&page=2") {
				return respondJSON(response, `[{"filename": "b.go", "patch": "p2"}]`)
			}
			return respondJSON(response, `[]`)
		},
	}
	c := newTestClient(nil, rest)

	result, err := c.FetchDiffs(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patches) != 2 {
		t.Fatalf("len(result.Patches) = %d, want 2", len(result.Patches))
	}
	if result.Patches["a.go"] != "p1" || result.Patches["b.go"] != "p2" {
		t.Errorf("unexpected patches: %v", result.Patches)
	}
	if len(rest.GetCalls) != 3 {
		t.Errorf("expected 3 Get calls, got %d", len(rest.GetCalls))
	}
}

func TestFetchDiffs_Empty(t *testing.T) {
	rest := &mockREST{
		GetFunc: func(path string, response interface{}) error {
			return respondJSON(response, `[]`)
		},
	}
	c := newTestClient(nil, rest)

	result, err := c.FetchDiffs(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patches) != 0 {
		t.Errorf("len(result.Patches) = %d, want 0", len(result.Patches))
	}
}

func TestFetchDiffs_Error(t *testing.T) {
	rest := &mockREST{
		GetFunc: func(path string, response interface{}) error {
			return errors.New("api error")
		},
	}
	c := newTestClient(nil, rest)

	_, err := c.FetchDiffs(1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "diff情報の取得に失敗") {
		t.Errorf("error = %q, want containing 'diff情報の取得に失敗'", err.Error())
	}
}

// --- FetchReviewThreads tests ---

func TestFetchReviewThreads_SinglePage(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			if strings.Contains(query, "reviews(first: 100, states: PENDING)") {
				return respondJSON(response, `{
					"Repository": {
						"PullRequest": {
							"Reviews": {
								"Nodes": []
							}
						}
					}
				}`)
			}
			return respondJSON(response, `{
				"Repository": {
					"PullRequest": {
						"ReviewThreads": {
							"PageInfo": {"HasNextPage": false, "EndCursor": ""},
							"Nodes": [{
								"ID": "RT_1",
								"IsResolved": false,
								"Path": "main.go",
								"Line": 10,
								"DiffSide": "RIGHT",
								"Comments": {
									"Nodes": [
										{"ID": "C_1", "Body": "Fix this", "Author": {"Login": "alice"}, "CreatedAt": "2026-01-01T00:00:00Z"},
										{"ID": "C_2", "Body": "Done", "Author": {"Login": "bob"}, "CreatedAt": "2026-01-01T01:00:00Z"}
									]
								}
							}]
						}
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	threads, err := c.FetchReviewThreads(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("len(threads) = %d, want 1", len(threads))
	}
	th := threads[0]
	if th.ID != "RT_1" {
		t.Errorf("ID = %q, want RT_1", th.ID)
	}
	if th.IsResolved {
		t.Error("expected unresolved")
	}
	if th.Path != "main.go" {
		t.Errorf("Path = %q, want main.go", th.Path)
	}
	if th.Line != 10 {
		t.Errorf("Line = %d, want 10", th.Line)
	}
	if th.DiffSide != DiffSideRight {
		t.Errorf("DiffSide = %q, want RIGHT", th.DiffSide)
	}
	if len(th.Comments) != 2 {
		t.Fatalf("len(Comments) = %d, want 2", len(th.Comments))
	}
	if th.Comments[0].Author != "alice" {
		t.Errorf("Comments[0].Author = %q, want alice", th.Comments[0].Author)
	}
	if th.Comments[1].Body != "Done" {
		t.Errorf("Comments[1].Body = %q, want Done", th.Comments[1].Body)
	}
}

func TestFetchReviewThreads_MultiPage(t *testing.T) {
	callCount := 0
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			if strings.Contains(query, "reviews(first: 100, states: PENDING)") {
				return respondJSON(response, `{
					"Repository": {
						"PullRequest": {
							"Reviews": {
								"Nodes": []
							}
						}
					}
				}`)
			}
			callCount++
			if callCount == 1 {
				return respondJSON(response, `{
					"Repository": {
						"PullRequest": {
							"ReviewThreads": {
								"PageInfo": {"HasNextPage": true, "EndCursor": "tc1"},
								"Nodes": [{"ID": "RT_1", "IsResolved": false, "Path": "a.go", "Line": 1, "DiffSide": "LEFT", "Comments": {"Nodes": []}}]
							}
						}
					}
				}`)
			}
			return respondJSON(response, `{
				"Repository": {
					"PullRequest": {
						"ReviewThreads": {
							"PageInfo": {"HasNextPage": false, "EndCursor": ""},
							"Nodes": [{"ID": "RT_2", "IsResolved": true, "Path": "b.go", "Line": 5, "DiffSide": "RIGHT", "Comments": {"Nodes": []}}]
						}
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	threads, err := c.FetchReviewThreads(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("len(threads) = %d, want 2", len(threads))
	}
	if threads[0].ID != "RT_1" || threads[1].ID != "RT_2" {
		t.Errorf("unexpected thread IDs: %q, %q", threads[0].ID, threads[1].ID)
	}
	after, ok := gql.DoCalls[1].Variables["after"]
	if !ok || after != "tc1" {
		t.Errorf("second call after = %v, want tc1", after)
	}
}

func TestFetchReviewThreads_Empty(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			if strings.Contains(query, "reviews(first: 100, states: PENDING)") {
				return respondJSON(response, `{
					"Repository": {
						"PullRequest": {
							"Reviews": {
								"Nodes": []
							}
						}
					}
				}`)
			}
			return respondJSON(response, `{
				"Repository": {
					"PullRequest": {
						"ReviewThreads": {
							"PageInfo": {"HasNextPage": false, "EndCursor": ""},
							"Nodes": []
						}
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	threads, err := c.FetchReviewThreads(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if threads != nil {
		t.Errorf("expected nil, got %v", threads)
	}
}

func TestFetchReviewThreads_Error(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			return errors.New("gql error")
		},
	}
	c := newTestClient(gql, nil)

	_, err := c.FetchReviewThreads(1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "レビュースレッドの取得に失敗") {
		t.Errorf("error = %q, want containing 'レビュースレッドの取得に失敗'", err.Error())
	}
}

func TestFetchReviewThreads_IncludesPendingReviewComments(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			if strings.Contains(query, "reviewThreads(first: 100") {
				return respondJSON(response, `{
					"Repository": {
						"PullRequest": {
							"ReviewThreads": {
								"PageInfo": {"HasNextPage": false, "EndCursor": ""},
								"Nodes": []
							}
						}
					}
				}`)
			}
			return respondJSON(response, `{
				"Repository": {
					"PullRequest": {
						"Reviews": {
							"Nodes": [{
								"ID": "PRR_1",
								"Comments": {
									"Nodes": [
										{"ID": "C_1", "Body": "pending root", "Path": "main.go", "Line": 12, "DiffSide": "RIGHT", "CreatedAt": "2026-01-01T00:00:00Z", "Author": {"Login": "alice"}, "ReplyTo": null},
										{"ID": "C_2", "Body": "pending reply", "Path": "main.go", "Line": 12, "DiffSide": "RIGHT", "CreatedAt": "2026-01-01T01:00:00Z", "Author": {"Login": "bob"}, "ReplyTo": {"ID": "C_1"}}
									]
								}
							}]
						}
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	threads, err := c.FetchReviewThreads(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("len(threads) = %d, want 1", len(threads))
	}
	if !threads[0].IsPending {
		t.Fatal("expected pending thread")
	}
	if threads[0].Path != "main.go" || threads[0].Line != 12 || threads[0].DiffSide != DiffSideRight {
		t.Fatalf("unexpected pending thread location: %+v", threads[0])
	}
	if len(threads[0].Comments) != 2 {
		t.Fatalf("len(Comments) = %d, want 2", len(threads[0].Comments))
	}
	if threads[0].Comments[1].Body != "pending reply" {
		t.Fatalf("reply body = %q, want pending reply", threads[0].Comments[1].Body)
	}
}

// --- FetchOpenPRs tests ---

func TestFetchOpenPRs_SinglePage(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			return respondJSON(response, `{
				"Repository": {
					"PullRequests": {
						"PageInfo": {"HasNextPage": false, "EndCursor": ""},
						"Nodes": [
							{"Number": 1, "Title": "PR One", "IsDraft": false, "UpdatedAt": "2026-01-01T00:00:00Z", "Author": {"Login": "alice"}},
							{"Number": 2, "Title": "PR Two", "IsDraft": true, "UpdatedAt": "2026-01-02T00:00:00Z", "Author": {"Login": "bob"}}
						]
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	prs, err := c.FetchOpenPRs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("len(prs) = %d, want 2", len(prs))
	}
	if prs[0].Number != 1 || prs[0].Title != "PR One" || prs[0].Author != "alice" || prs[0].IsDraft {
		t.Errorf("unexpected prs[0]: %+v", prs[0])
	}
	if prs[1].Number != 2 || !prs[1].IsDraft || prs[1].Author != "bob" {
		t.Errorf("unexpected prs[1]: %+v", prs[1])
	}
}

func TestFetchOpenPRs_MultiPage(t *testing.T) {
	callCount := 0
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			callCount++
			if callCount == 1 {
				return respondJSON(response, `{
					"Repository": {
						"PullRequests": {
							"PageInfo": {"HasNextPage": true, "EndCursor": "pc1"},
							"Nodes": [{"Number": 1, "Title": "A", "IsDraft": false, "UpdatedAt": "t1", "Author": {"Login": "x"}}]
						}
					}
				}`)
			}
			return respondJSON(response, `{
				"Repository": {
					"PullRequests": {
						"PageInfo": {"HasNextPage": false, "EndCursor": ""},
						"Nodes": [{"Number": 2, "Title": "B", "IsDraft": true, "UpdatedAt": "t2", "Author": {"Login": "y"}}]
					}
				}
			}`)
		},
	}
	c := newTestClient(gql, nil)

	prs, err := c.FetchOpenPRs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("len(prs) = %d, want 2", len(prs))
	}
	after, ok := gql.DoCalls[1].Variables["after"]
	if !ok || after != "pc1" {
		t.Errorf("second call after = %v, want pc1", after)
	}
}

func TestFetchOpenPRs_Error(t *testing.T) {
	gql := &mockGraphQL{
		DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
			return errors.New("gql error")
		},
	}
	c := newTestClient(gql, nil)

	_, err := c.FetchOpenPRs()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "オープンPR一覧の取得に失敗") {
		t.Errorf("error = %q, want containing 'オープンPR一覧の取得に失敗'", err.Error())
	}
}

// --- Mutation tests ---

func TestMarkFileAsViewed(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.MarkFileAsViewed("PR_1", "a.go")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gql.DoCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(gql.DoCalls))
		}
		call := gql.DoCalls[0]
		if call.Variables["pullRequestId"] != "PR_1" {
			t.Errorf("pullRequestId = %v, want PR_1", call.Variables["pullRequestId"])
		}
		if call.Variables["path"] != "a.go" {
			t.Errorf("path = %v, want a.go", call.Variables["path"])
		}
		if call.Query != markFileAsViewedMutation {
			t.Error("unexpected query")
		}
	})

	t.Run("error", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return errors.New("fail")
			},
		}
		c := newTestClient(gql, nil)

		err := c.MarkFileAsViewed("PR_1", "a.go")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestUnmarkFileAsViewed(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.UnmarkFileAsViewed("PR_2", "b.go")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		call := gql.DoCalls[0]
		if call.Variables["pullRequestId"] != "PR_2" {
			t.Errorf("pullRequestId = %v, want PR_2", call.Variables["pullRequestId"])
		}
		if call.Variables["path"] != "b.go" {
			t.Errorf("path = %v, want b.go", call.Variables["path"])
		}
		if call.Query != unmarkFileAsViewedMutation {
			t.Error("unexpected query")
		}
	})

	t.Run("error", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return errors.New("fail")
			},
		}
		c := newTestClient(gql, nil)

		err := c.UnmarkFileAsViewed("PR_2", "b.go")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAddComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.AddComment("PR_1", "main.go", "fix this", DiffSideRight, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		call := gql.DoCalls[0]
		if call.Variables["pullRequestId"] != "PR_1" {
			t.Errorf("pullRequestId = %v, want PR_1", call.Variables["pullRequestId"])
		}
		if call.Variables["body"] != "fix this" {
			t.Errorf("body = %v, want fix this", call.Variables["body"])
		}
		if call.Variables["path"] != "main.go" {
			t.Errorf("path = %v, want main.go", call.Variables["path"])
		}
		if call.Variables["line"] != 42 {
			t.Errorf("line = %v, want 42", call.Variables["line"])
		}
		if call.Variables["side"] != "RIGHT" {
			t.Errorf("side = %v, want RIGHT", call.Variables["side"])
		}
		if call.Query != addReviewCommentMutation {
			t.Error("unexpected query")
		}
	})

	t.Run("error", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return errors.New("fail")
			},
		}
		c := newTestClient(gql, nil)

		err := c.AddComment("PR_1", "main.go", "fix", DiffSideRight, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestReplyToThread(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.ReplyToThread("RT_1", "thanks")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		call := gql.DoCalls[0]
		if call.Variables["threadId"] != "RT_1" {
			t.Errorf("threadId = %v, want RT_1", call.Variables["threadId"])
		}
		if call.Variables["body"] != "thanks" {
			t.Errorf("body = %v, want thanks", call.Variables["body"])
		}
		if call.Query != replyToThreadMutation {
			t.Error("unexpected query")
		}
	})

	t.Run("error", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return errors.New("fail")
			},
		}
		c := newTestClient(gql, nil)

		err := c.ReplyToThread("RT_1", "x")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolveThread(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.ResolveThread("RT_1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		call := gql.DoCalls[0]
		if call.Variables["threadId"] != "RT_1" {
			t.Errorf("threadId = %v, want RT_1", call.Variables["threadId"])
		}
		if call.Query != resolveThreadMutation {
			t.Error("unexpected query")
		}
	})

	t.Run("error", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return errors.New("fail")
			},
		}
		c := newTestClient(gql, nil)

		err := c.ResolveThread("RT_1")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestUnresolveThread(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.UnresolveThread("RT_2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		call := gql.DoCalls[0]
		if call.Variables["threadId"] != "RT_2" {
			t.Errorf("threadId = %v, want RT_2", call.Variables["threadId"])
		}
		if call.Query != unresolveThreadMutation {
			t.Error("unexpected query")
		}
	})

	t.Run("error", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return errors.New("fail")
			},
		}
		c := newTestClient(gql, nil)

		err := c.UnresolveThread("RT_2")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSubmitReview(t *testing.T) {
	t.Run("success_with_body", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.SubmitReview("PR_1", ReviewEventApprove, "LGTM")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		call := gql.DoCalls[0]
		if call.Variables["pullRequestId"] != "PR_1" {
			t.Errorf("pullRequestId = %v, want PR_1", call.Variables["pullRequestId"])
		}
		if call.Variables["event"] != "APPROVE" {
			t.Errorf("event = %v, want APPROVE", call.Variables["event"])
		}
		if call.Variables["body"] != "LGTM" {
			t.Errorf("body = %v, want LGTM", call.Variables["body"])
		}
		if call.Query != submitReviewMutation {
			t.Error("unexpected query")
		}
	})

	t.Run("success_without_body", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return nil
			},
		}
		c := newTestClient(gql, nil)

		err := c.SubmitReview("PR_1", ReviewEventComment, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		call := gql.DoCalls[0]
		if _, ok := call.Variables["body"]; ok {
			t.Errorf("body should not be in variables when empty, got %v", call.Variables["body"])
		}
	})

	t.Run("error", func(t *testing.T) {
		gql := &mockGraphQL{
			DoFunc: func(query string, variables map[string]interface{}, response interface{}) error {
				return errors.New("fail")
			},
		}
		c := newTestClient(gql, nil)

		err := c.SubmitReview("PR_1", ReviewEventApprove, "")
		if err == nil {
			t.Fatal("expected error")
		}
	})
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
		{
			"openPRsQuery",
			openPRsQuery,
			[]string{"pullRequests", "OPEN", "number", "title", "isDraft", "author", "login", "updatedAt"},
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
