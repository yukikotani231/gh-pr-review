package github

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// graphQLDoer はGraphQL APIの呼び出しに必要なメソッドを定義する
type graphQLDoer interface {
	Do(query string, variables map[string]interface{}, response interface{}) error
}

// restGetter はREST APIの呼び出しに必要なメソッドを定義する
type restGetter interface {
	Get(path string, response interface{}) error
}

type Client struct {
	gql     graphQLDoer
	rest    restGetter
	owner   string
	repo    string
	fixture *FixtureData
}

func NewClient(owner, repo string) (*Client, error) {
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("GraphQLクライアントの初期化に失敗: %w", err)
	}
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("RESTクライアントの初期化に失敗: %w", err)
	}
	return &Client{gql: gql, rest: rest, owner: owner, repo: repo}, nil
}

func (c *Client) FetchPR(number int) (*PullRequest, error) {
	if c.fixture != nil {
		if number != 0 && number != c.fixture.PullRequest.Number {
			return nil, fmt.Errorf("fixture PR #%d is not available", number)
		}
		pr := c.fixture.PullRequest
		pr.Files = append([]PRFile(nil), pr.Files...)
		return &pr, nil
	}

	var allFiles []PRFile
	var cursor *string
	var pr *PullRequest

	for {
		variables := map[string]interface{}{
			"owner":  c.owner,
			"repo":   c.repo,
			"number": number,
		}
		if cursor != nil {
			variables["after"] = *cursor
		}

		var resp struct {
			Repository struct {
				PullRequest struct {
					ID           string
					Title        string
					Additions    int
					Deletions    int
					ChangedFiles int
					Files        struct {
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
						Nodes []struct {
							Path              string
							Additions         int
							Deletions         int
							ViewerViewedState string
						}
					}
				}
			}
		}

		err := c.gql.Do(prFilesQuery, variables, &resp)
		if err != nil {
			return nil, fmt.Errorf("PR情報の取得に失敗: %w", err)
		}

		prData := resp.Repository.PullRequest
		for _, f := range prData.Files.Nodes {
			allFiles = append(allFiles, PRFile{
				Path:              f.Path,
				Additions:         f.Additions,
				Deletions:         f.Deletions,
				ViewerViewedState: ViewedState(f.ViewerViewedState),
			})
		}

		if pr == nil {
			pr = &PullRequest{
				ID:           prData.ID,
				Title:        prData.Title,
				Number:       number,
				Additions:    prData.Additions,
				Deletions:    prData.Deletions,
				ChangedFiles: prData.ChangedFiles,
			}
		}

		if !prData.Files.PageInfo.HasNextPage {
			break
		}
		endCursor := prData.Files.PageInfo.EndCursor
		cursor = &endCursor
	}

	pr.Files = allFiles
	return pr, nil
}

func (c *Client) Owner() string { return c.owner }
func (c *Client) Repo() string  { return c.repo }

func (c *Client) FetchDiffs(number int) (*DiffResult, error) {
	if c.fixture != nil {
		if number != 0 && number != c.fixture.PullRequest.Number {
			return nil, fmt.Errorf("fixture PR #%d is not available", number)
		}
		result := &DiffResult{
			Patches:           make(map[string]string, len(c.fixture.DiffResult.Patches)),
			FileStatuses:      make(map[string]FileStatus, len(c.fixture.DiffResult.FileStatuses)),
			PreviousFilenames: make(map[string]string, len(c.fixture.DiffResult.PreviousFilenames)),
		}
		for path, patch := range c.fixture.DiffResult.Patches {
			result.Patches[path] = patch
		}
		for path, status := range c.fixture.DiffResult.FileStatuses {
			result.FileStatuses[path] = status
		}
		for path, prev := range c.fixture.DiffResult.PreviousFilenames {
			result.PreviousFilenames[path] = prev
		}
		return result, nil
	}

	result := &DiffResult{
		Patches:           make(map[string]string),
		FileStatuses:      make(map[string]FileStatus),
		PreviousFilenames: make(map[string]string),
	}
	page := 1

	for {
		var files []struct {
			Filename         string `json:"filename"`
			Patch            string `json:"patch"`
			Status           string `json:"status"`
			PreviousFilename string `json:"previous_filename"`
		}

		path := fmt.Sprintf("repos/%s/%s/pulls/%d/files?per_page=100&page=%d",
			c.owner, c.repo, number, page)
		err := c.rest.Get(path, &files)
		if err != nil {
			return nil, fmt.Errorf("diff情報の取得に失敗: %w", err)
		}

		if len(files) == 0 {
			break
		}

		for _, f := range files {
			result.Patches[f.Filename] = f.Patch
			result.FileStatuses[f.Filename] = FileStatus(f.Status)
			if f.PreviousFilename != "" {
				result.PreviousFilenames[f.Filename] = f.PreviousFilename
			}
		}

		page++
	}

	return result, nil
}

func (c *Client) MarkFileAsViewed(pullRequestID, path string) error {
	if c.fixture != nil {
		return c.fixtureSetViewedState(path, ViewedStateViewed)
	}
	variables := map[string]interface{}{
		"pullRequestId": pullRequestID,
		"path":          path,
	}
	var resp interface{}
	return c.gql.Do(markFileAsViewedMutation, variables, &resp)
}

func (c *Client) UnmarkFileAsViewed(pullRequestID, path string) error {
	if c.fixture != nil {
		return c.fixtureSetViewedState(path, ViewedStateUnviewed)
	}
	variables := map[string]interface{}{
		"pullRequestId": pullRequestID,
		"path":          path,
	}
	var resp interface{}
	return c.gql.Do(unmarkFileAsViewedMutation, variables, &resp)
}

func (c *Client) FetchReviewThreads(number int) ([]ReviewThread, error) {
	if c.fixture != nil {
		if number != 0 && number != c.fixture.PullRequest.Number {
			return nil, fmt.Errorf("fixture PR #%d is not available", number)
		}
		threads := append([]ReviewThread(nil), c.fixture.Threads...)
		for i := range threads {
			threads[i].Comments = append([]ReviewComment(nil), threads[i].Comments...)
		}
		return threads, nil
	}

	var allThreads []ReviewThread
	var cursor *string

	for {
		variables := map[string]interface{}{
			"owner":  c.owner,
			"repo":   c.repo,
			"number": number,
		}
		if cursor != nil {
			variables["after"] = *cursor
		}

		var resp struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
						Nodes []struct {
							ID         string
							IsResolved bool
							Path       string
							Line       int
							DiffSide   string
							Comments   struct {
								Nodes []struct {
									ID        string
									Body      string
									Author    struct{ Login string }
									CreatedAt string
								}
							}
						}
					}
				}
			}
		}

		err := c.gql.Do(reviewThreadsQuery, variables, &resp)
		if err != nil {
			return nil, fmt.Errorf("レビュースレッドの取得に失敗: %w", err)
		}

		for _, t := range resp.Repository.PullRequest.ReviewThreads.Nodes {
			thread := ReviewThread{
				ID:         t.ID,
				IsResolved: t.IsResolved,
				Path:       t.Path,
				Line:       t.Line,
				DiffSide:   DiffSide(t.DiffSide),
			}
			for _, comment := range t.Comments.Nodes {
				thread.Comments = append(thread.Comments, ReviewComment{
					ID:        comment.ID,
					Body:      comment.Body,
					Author:    comment.Author.Login,
					CreatedAt: comment.CreatedAt,
				})
			}
			allThreads = append(allThreads, thread)
		}

		if !resp.Repository.PullRequest.ReviewThreads.PageInfo.HasNextPage {
			break
		}
		endCursor := resp.Repository.PullRequest.ReviewThreads.PageInfo.EndCursor
		cursor = &endCursor
	}

	pendingThreads, err := c.fetchPendingReviewThreads(number)
	if err != nil {
		return nil, err
	}
	allThreads = append(allThreads, pendingThreads...)

	return allThreads, nil
}

func (c *Client) fetchPendingReviewThreads(number int) ([]ReviewThread, error) {
	variables := map[string]interface{}{
		"owner":  c.owner,
		"repo":   c.repo,
		"number": number,
	}

	var resp struct {
		Repository struct {
			PullRequest struct {
				Reviews struct {
					Nodes []struct {
						ID       string
						Comments struct {
							Nodes []struct {
								ID        string
								Body      string
								Path      string
								Line      int
								DiffSide  string
								CreatedAt string
								Author    struct{ Login string }
								ReplyTo   *struct{ ID string }
							}
						}
					}
				} `json:"reviews"`
			}
		}
	}

	if err := c.gql.Do(pendingReviewCommentsQuery, variables, &resp); err != nil {
		return nil, fmt.Errorf("Pending review コメントの取得に失敗: %w", err)
	}

	var pendingThreads []ReviewThread
	threadIndexByRootComment := map[string]int{}

	for _, review := range resp.Repository.PullRequest.Reviews.Nodes {
		for _, comment := range review.Comments.Nodes {
			if comment.Path == "" || comment.Line == 0 {
				continue
			}

			if comment.ReplyTo == nil {
				threadIndexByRootComment[comment.ID] = len(pendingThreads)
				pendingThreads = append(pendingThreads, ReviewThread{
					ID:        "pending-review:" + review.ID + ":" + comment.ID,
					IsPending: true,
					Path:      comment.Path,
					Line:      comment.Line,
					DiffSide:  DiffSide(comment.DiffSide),
					Comments: []ReviewComment{{
						ID:        comment.ID,
						Body:      comment.Body,
						Author:    comment.Author.Login,
						CreatedAt: comment.CreatedAt,
					}},
				})
				continue
			}

			threadIdx, ok := threadIndexByRootComment[comment.ReplyTo.ID]
			if !ok {
				threadIndexByRootComment[comment.ID] = len(pendingThreads)
				pendingThreads = append(pendingThreads, ReviewThread{
					ID:        "pending-review:" + review.ID + ":" + comment.ID,
					IsPending: true,
					Path:      comment.Path,
					Line:      comment.Line,
					DiffSide:  DiffSide(comment.DiffSide),
				})
				threadIdx = len(pendingThreads) - 1
			}
			pendingThreads[threadIdx].Comments = append(pendingThreads[threadIdx].Comments, ReviewComment{
				ID:        comment.ID,
				Body:      comment.Body,
				Author:    comment.Author.Login,
				CreatedAt: comment.CreatedAt,
			})
		}
	}

	return pendingThreads, nil
}

func (c *Client) AddComment(pullRequestID, path, body string, side DiffSide, line int) error {
	if c.fixture != nil {
		now := time.Now().UTC().Format(time.RFC3339)
		threadID := c.fixtureNextID("fixture-thread")
		commentID := c.fixtureNextID("fixture-comment")
		c.fixture.Threads = append(c.fixture.Threads, ReviewThread{
			ID:         threadID,
			IsResolved: false,
			IsPending:  true,
			Path:       path,
			Line:       line,
			DiffSide:   side,
			Comments: []ReviewComment{{
				ID:        commentID,
				Body:      body,
				Author:    "you",
				CreatedAt: now,
			}},
		})
		return nil
	}
	variables := map[string]interface{}{
		"pullRequestId": pullRequestID,
		"body":          body,
		"path":          path,
		"line":          line,
		"side":          string(side),
	}
	var resp interface{}
	return c.gql.Do(addReviewCommentMutation, variables, &resp)
}

func (c *Client) ReplyToThread(threadID, body string) error {
	if c.fixture != nil {
		for i := range c.fixture.Threads {
			if c.fixture.Threads[i].ID != threadID {
				continue
			}
			c.fixture.Threads[i].IsPending = true
			c.fixture.Threads[i].Comments = append(c.fixture.Threads[i].Comments, ReviewComment{
				ID:        c.fixtureNextID("fixture-comment"),
				Body:      body,
				Author:    "you",
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			})
			return nil
		}
		return fmt.Errorf("fixture thread not found: %s", threadID)
	}
	variables := map[string]interface{}{
		"threadId": threadID,
		"body":     body,
	}
	var resp interface{}
	return c.gql.Do(replyToThreadMutation, variables, &resp)
}

func (c *Client) ResolveThread(threadID string) error {
	if c.fixture != nil {
		return c.fixtureSetThreadResolved(threadID, true)
	}
	variables := map[string]interface{}{
		"threadId": threadID,
	}
	var resp interface{}
	return c.gql.Do(resolveThreadMutation, variables, &resp)
}

func (c *Client) UnresolveThread(threadID string) error {
	if c.fixture != nil {
		return c.fixtureSetThreadResolved(threadID, false)
	}
	variables := map[string]interface{}{
		"threadId": threadID,
	}
	var resp interface{}
	return c.gql.Do(unresolveThreadMutation, variables, &resp)
}

func (c *Client) FetchOpenPRs() ([]PRListItem, error) {
	if c.fixture != nil {
		if len(c.fixture.OpenPRs) > 0 {
			return append([]PRListItem(nil), c.fixture.OpenPRs...), nil
		}
		return []PRListItem{{
			Number:    c.fixture.PullRequest.Number,
			Title:     c.fixture.PullRequest.Title,
			Author:    "fixture",
			UpdatedAt: "2026-03-11T00:00:00Z",
		}}, nil
	}

	var allPRs []PRListItem
	var cursor *string

	for {
		variables := map[string]interface{}{
			"owner": c.owner,
			"repo":  c.repo,
		}
		if cursor != nil {
			variables["after"] = *cursor
		}

		var resp struct {
			Repository struct {
				PullRequests struct {
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
					Nodes []struct {
						Number    int
						Title     string
						IsDraft   bool
						UpdatedAt string
						Author    struct{ Login string }
					}
				}
			}
		}

		err := c.gql.Do(openPRsQuery, variables, &resp)
		if err != nil {
			return nil, fmt.Errorf("オープンPR一覧の取得に失敗: %w", err)
		}

		for _, pr := range resp.Repository.PullRequests.Nodes {
			allPRs = append(allPRs, PRListItem{
				Number:    pr.Number,
				Title:     pr.Title,
				Author:    pr.Author.Login,
				UpdatedAt: pr.UpdatedAt,
				IsDraft:   pr.IsDraft,
			})
		}

		if !resp.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		endCursor := resp.Repository.PullRequests.PageInfo.EndCursor
		cursor = &endCursor
	}

	return allPRs, nil
}

func (c *Client) SubmitReview(pullRequestID string, event ReviewEvent, body string) error {
	if c.fixture != nil {
		return fmt.Errorf("fixture mode is read-only")
	}
	variables := map[string]interface{}{
		"pullRequestId": pullRequestID,
		"event":         string(event),
	}
	if body != "" {
		variables["body"] = body
	}
	var resp interface{}
	return c.gql.Do(submitReviewMutation, variables, &resp)
}
