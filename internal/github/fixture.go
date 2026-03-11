package github

import (
	"encoding/json"
	"fmt"
	"os"
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
