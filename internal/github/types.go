package github

type ViewedState string

const (
	ViewedStateViewed   ViewedState = "VIEWED"
	ViewedStateUnviewed ViewedState = "UNVIEWED"
)

type DiffSide string

const (
	DiffSideLeft  DiffSide = "LEFT"
	DiffSideRight DiffSide = "RIGHT"
)

type PRFile struct {
	Path              string
	Additions         int
	Deletions         int
	ViewerViewedState ViewedState
	Patch             string
}

type PullRequest struct {
	ID           string
	Title        string
	Number       int
	Additions    int
	Deletions    int
	ChangedFiles int
	Files        []PRFile
}

type ReviewComment struct {
	ID        string
	Body      string
	Author    string
	CreatedAt string
}

type ReviewThread struct {
	ID         string
	IsResolved bool
	Path       string
	Line       int
	DiffSide   DiffSide
	Comments   []ReviewComment
}

// PRListItem はPR一覧表示用の軽量な型
type PRListItem struct {
	Number    int
	Title     string
	Author    string
	UpdatedAt string
	IsDraft   bool
}

type ReviewEvent string

const (
	ReviewEventApprove        ReviewEvent = "APPROVE"
	ReviewEventRequestChanges ReviewEvent = "REQUEST_CHANGES"
	ReviewEventComment        ReviewEvent = "COMMENT"
)
