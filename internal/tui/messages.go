package tui

import gh "github.com/yukikotani231/gh-pr-review/internal/github"

type PRFetchedMsg struct {
	PR  *gh.PullRequest
	Err error
}

type DiffFetchedMsg struct {
	Result *gh.DiffResult
	Err    error
}

type ThreadsFetchedMsg struct {
	Threads []gh.ReviewThread
	Err     error
}

type ViewedToggledMsg struct {
	Path     string
	NewState gh.ViewedState
	Err      error
}

type CommentAddedMsg struct {
	Err error
}

type ThreadRepliedMsg struct {
	Err error
}

type ThreadResolvedMsg struct {
	ThreadID   string
	IsResolved bool
	Err        error
}

type ReviewSubmittedMsg struct {
	Event gh.ReviewEvent
	Err   error
}

type openedInBrowserMsg struct {
	Err error
}

type DataRefreshedMsg struct {
	PR      *gh.PullRequest
	Result  *gh.DiffResult
	Threads []gh.ReviewThread
	Err     error
}
