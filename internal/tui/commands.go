package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

func (m Model) fetchPRCmd() tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		pr, err := m.client.FetchPR(m.prNumber)
		return PRFetchedMsg{PR: pr, Err: err}
	}
}

func (m Model) fetchDiffsCmd() tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		patches, err := m.client.FetchDiffs(m.prNumber)
		return DiffFetchedMsg{Patches: patches, Err: err}
	}
}

func (m Model) fetchThreadsCmd() tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		threads, err := m.client.FetchReviewThreads(m.prNumber)
		return ThreadsFetchedMsg{Threads: threads, Err: err}
	}
}

func (m Model) toggleViewedCmd() tea.Cmd {
	f := m.fileList.SelectedFile()
	if f == nil {
		return nil
	}
	path := f.Path
	currentState := f.ViewerViewedState
	prID := m.pr.ID

	return func() tea.Msg {
		var newState gh.ViewedState
		var err error
		if currentState == gh.ViewedStateViewed {
			err = m.client.UnmarkFileAsViewed(prID, path)
			newState = gh.ViewedStateUnviewed
		} else {
			err = m.client.MarkFileAsViewed(prID, path)
			newState = gh.ViewedStateViewed
		}
		return ViewedToggledMsg{Path: path, NewState: newState, Err: err}
	}
}

func (m Model) addCommentCmd(body string) tea.Cmd {
	dl := m.diffView.CursorLine()
	if dl == nil {
		return nil
	}
	f := m.fileList.SelectedFile()
	if f == nil {
		return nil
	}
	path := f.Path
	prID := m.pr.ID

	var side gh.DiffSide
	var line int
	if dl.Type == diff.LineRemoved {
		side = gh.DiffSideLeft
		line = dl.OldLineNum
	} else {
		side = gh.DiffSideRight
		line = dl.NewLineNum
	}

	return func() tea.Msg {
		err := m.client.AddComment(prID, path, body, side, line)
		return CommentAddedMsg{Err: err}
	}
}

func (m Model) replyToThreadCmd(threadID, body string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.ReplyToThread(threadID, body)
		return ThreadRepliedMsg{Err: err}
	}
}

func (m Model) toggleResolveCmd(t *gh.ReviewThread) tea.Cmd {
	threadID := t.ID
	isResolved := t.IsResolved
	return func() tea.Msg {
		var err error
		if isResolved {
			err = m.client.UnresolveThread(threadID)
		} else {
			err = m.client.ResolveThread(threadID)
		}
		return ThreadResolvedMsg{
			ThreadID:   threadID,
			IsResolved: !isResolved,
			Err:        err,
		}
	}
}

func (m Model) submitReviewCmd(event gh.ReviewEvent, body string) tea.Cmd {
	prID := m.pr.ID
	return func() tea.Msg {
		err := m.client.SubmitReview(prID, event, body)
		return ReviewSubmittedMsg{Event: event, Err: err}
	}
}

func (m Model) refreshDataCmd() tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		pr, err := m.client.FetchPR(m.prNumber)
		if err != nil {
			return DataRefreshedMsg{Err: err}
		}
		patches, err := m.client.FetchDiffs(m.prNumber)
		if err != nil {
			return DataRefreshedMsg{Err: err}
		}
		threads, err := m.client.FetchReviewThreads(m.prNumber)
		if err != nil {
			return DataRefreshedMsg{Err: err}
		}
		return DataRefreshedMsg{PR: pr, Patches: patches, Threads: threads}
	}
}

// --- Data update helpers ---

func (m *Model) updateLayout() {
	leftWidth := m.leftPaneWidth()
	contentHeight := m.contentHeight()
	m.fileList.SetSize(leftWidth-2, contentHeight)
	rightWidth := m.rightPaneWidth()
	m.diffView.SetSize(rightWidth-4, contentHeight-1) // -1 for file path line
	m.textInput.SetWidth(m.width - 4)
	if m.state == stateReady {
		m.updateDiffView()
	}
}

func (m *Model) updateDiffView() {
	f := m.fileList.SelectedFile()
	if f == nil {
		m.diffView.SetContent(nil, nil)
		return
	}
	patch := m.patches[f.Path]
	lines := diff.Parse(patch)
	fileThreads := m.threadsForFile(f.Path)
	m.diffView.SetContent(lines, fileThreads)
}

func (m *Model) threadsForFile(path string) []gh.ReviewThread {
	var result []gh.ReviewThread
	for _, t := range m.threads {
		if t.Path == path {
			result = append(result, t)
		}
	}
	return result
}

func (m *Model) findThreadAtCursor() *gh.ReviewThread {
	dl := m.diffView.CursorLine()
	if dl == nil {
		return nil
	}
	f := m.fileList.SelectedFile()
	if f == nil {
		return nil
	}
	for i, t := range m.diffView.threads {
		if matchesThread(*dl, t) {
			m.diffView.threadCursor = i
			return &m.diffView.threads[i]
		}
	}
	return nil
}

func (m Model) unresolvedThreadCount() int {
	count := 0
	for _, t := range m.threads {
		if !t.IsResolved {
			count++
		}
	}
	return count
}

// --- Layout helpers ---

func (m Model) leftPaneWidth() int {
	w := m.width * 30 / 100
	if w < 20 {
		w = 20
	}
	return w
}

func (m Model) rightPaneWidth() int {
	return m.width - m.leftPaneWidth()
}

func (m Model) contentHeight() int {
	overhead := 4 // header(1) + border top(1) + border bottom(1) + status(1)
	switch m.mode {
	case modeComment, modeReply:
		overhead = 7 // header(1) + border(2) + input label(1) + textarea(3)
	case modeReview:
		overhead = 15 // header(1) + border(2) + review modal(~12)
	}
	h := m.height - overhead
	if h < 5 {
		h = 5
	}
	return h
}
