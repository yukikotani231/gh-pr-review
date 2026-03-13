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

type FileStatus string

const (
	FileStatusAdded    FileStatus = "added"
	FileStatusModified FileStatus = "modified"
	FileStatusRemoved  FileStatus = "removed"
	FileStatusRenamed  FileStatus = "renamed"
	FileStatusCopied   FileStatus = "copied"
)

type PRFile struct {
	Path              string      `json:"path"`
	Additions         int         `json:"additions"`
	Deletions         int         `json:"deletions"`
	ViewerViewedState ViewedState `json:"viewer_viewed_state"`
	Patch             string      `json:"patch"`
	Status            FileStatus  `json:"status"`
	PreviousFilename  string      `json:"previous_filename"`
}

type DiffResult struct {
	Patches           map[string]string     `json:"patches"`
	FileStatuses      map[string]FileStatus `json:"file_statuses"`
	PreviousFilenames map[string]string     `json:"previous_filenames"`
}

type PullRequest struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Number       int      `json:"number"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	ChangedFiles int      `json:"changed_files"`
	Files        []PRFile `json:"files"`
}

type ReviewComment struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

type ReviewThread struct {
	ID         string          `json:"id"`
	IsResolved bool            `json:"is_resolved"`
	Path       string          `json:"path"`
	Line       int             `json:"line"`
	DiffSide   DiffSide        `json:"diff_side"`
	Comments   []ReviewComment `json:"comments"`
}

// PRListItem はPR一覧表示用の軽量な型
type PRListItem struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	UpdatedAt string `json:"updated_at"`
	IsDraft   bool   `json:"is_draft"`
}

type ReviewEvent string

const (
	ReviewEventApprove        ReviewEvent = "APPROVE"
	ReviewEventRequestChanges ReviewEvent = "REQUEST_CHANGES"
	ReviewEventComment        ReviewEvent = "COMMENT"
)
