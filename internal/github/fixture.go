package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FixtureData struct {
	Owner       string         `json:"owner"`
	Repo        string         `json:"repo"`
	PullRequest PullRequest    `json:"pull_request"`
	DiffResult  DiffResult     `json:"diff_result"`
	Threads     []ReviewThread `json:"threads"`
	OpenPRs     []PRListItem   `json:"open_prs"`
}

func LoadFixtureData(path string) (*FixtureData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fixture read failed: %w", err)
	}

	var fixture FixtureData
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("fixture parse failed: %w", err)
	}
	if fixture.PullRequest.Number <= 0 {
		return nil, fmt.Errorf("fixture PR number must be greater than 0")
	}
	if fixture.PullRequest.Title == "" {
		return nil, fmt.Errorf("fixture PR title is required")
	}
	if fixture.Owner == "" {
		fixture.Owner = "fixture"
	}
	if fixture.Repo == "" {
		fixture.Repo = "demo"
	}
	if fixture.PullRequest.Files == nil {
		fixture.PullRequest.Files = []PRFile{}
	}
	if fixture.DiffResult.Patches == nil {
		fixture.DiffResult.Patches = map[string]string{}
	}
	if fixture.DiffResult.FileStatuses == nil {
		fixture.DiffResult.FileStatuses = map[string]FileStatus{}
	}
	if fixture.DiffResult.PreviousFilenames == nil {
		fixture.DiffResult.PreviousFilenames = map[string]string{}
	}

	return &fixture, nil
}

func NewFixtureClient(fixture *FixtureData) *Client {
	return &Client{
		owner:   fixture.Owner,
		repo:    fixture.Repo,
		fixture: fixture,
	}
}

func (c *Client) fixtureSetViewedState(path string, state ViewedState) error {
	for i := range c.fixture.PullRequest.Files {
		if c.fixture.PullRequest.Files[i].Path != path {
			continue
		}
		c.fixture.PullRequest.Files[i].ViewerViewedState = state
		return nil
	}
	return fmt.Errorf("fixture file not found: %s", path)
}

func (c *Client) fixtureSetThreadResolved(threadID string, resolved bool) error {
	for i := range c.fixture.Threads {
		if c.fixture.Threads[i].ID != threadID {
			continue
		}
		c.fixture.Threads[i].IsResolved = resolved
		return nil
	}
	return fmt.Errorf("fixture thread not found: %s", threadID)
}

func (c *Client) fixtureNextID(prefix string) string {
	maxSeen := 0
	for _, thread := range c.fixture.Threads {
		maxSeen = max(maxSeen, fixtureIDNumber(thread.ID, prefix))
		for _, comment := range thread.Comments {
			maxSeen = max(maxSeen, fixtureIDNumber(comment.ID, prefix))
		}
	}
	return fmt.Sprintf("%s-%d", prefix, maxSeen+1)
}

func fixtureIDNumber(id, prefix string) int {
	if !strings.HasPrefix(id, prefix+"-") {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, prefix+"-"))
	if err != nil {
		return 0
	}
	return n
}
