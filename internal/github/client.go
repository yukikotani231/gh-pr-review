package github

import (
	"fmt"

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
	gql   graphQLDoer
	rest  restGetter
	owner string
	repo  string
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

func (c *Client) FetchDiffs(number int) (map[string]string, error) {
	patches := make(map[string]string)
	page := 1

	for {
		var files []struct {
			Filename string `json:"filename"`
			Patch    string `json:"patch"`
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
			patches[f.Filename] = f.Patch
		}

		page++
	}

	return patches, nil
}

func (c *Client) MarkFileAsViewed(pullRequestID, path string) error {
	variables := map[string]interface{}{
		"pullRequestId": pullRequestID,
		"path":          path,
	}
	var resp interface{}
	return c.gql.Do(markFileAsViewedMutation, variables, &resp)
}

func (c *Client) UnmarkFileAsViewed(pullRequestID, path string) error {
	variables := map[string]interface{}{
		"pullRequestId": pullRequestID,
		"path":          path,
	}
	var resp interface{}
	return c.gql.Do(unmarkFileAsViewedMutation, variables, &resp)
}

func (c *Client) FetchReviewThreads(number int) ([]ReviewThread, error) {
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

	return allThreads, nil
}

func (c *Client) AddComment(pullRequestID, path, body string, side DiffSide, line int) error {
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
	variables := map[string]interface{}{
		"threadId": threadID,
		"body":     body,
	}
	var resp interface{}
	return c.gql.Do(replyToThreadMutation, variables, &resp)
}

func (c *Client) ResolveThread(threadID string) error {
	variables := map[string]interface{}{
		"threadId": threadID,
	}
	var resp interface{}
	return c.gql.Do(resolveThreadMutation, variables, &resp)
}

func (c *Client) UnresolveThread(threadID string) error {
	variables := map[string]interface{}{
		"threadId": threadID,
	}
	var resp interface{}
	return c.gql.Do(unresolveThreadMutation, variables, &resp)
}

func (c *Client) FetchOpenPRs() ([]PRListItem, error) {
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
