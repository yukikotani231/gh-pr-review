package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

type pane int

const (
	leftPane pane = iota
	rightPane
)

type inputMode int

const (
	modeNormal  inputMode = iota
	modeComment           // Adding a new comment on a diff line
	modeReply             // Replying to an existing thread
	modeReview            // Review submission dialog
	modeHelp              // Full keybinding help overlay
)

type Model struct {
	state      state
	err        error
	client     *gh.Client
	pr         *gh.PullRequest
	prNumber   int
	diffResult *gh.DiffResult
	threads    []gh.ReviewThread

	fileList FileListModel
	diffView DiffViewModel
	help     help.Model
	keyMap   KeyMap

	width  int
	height int
	focus  pane

	// Data fetch tracking (use bool flags, not nil checks, to distinguish
	// "not yet fetched" from "fetched but zero results")
	patchesFetched bool
	threadsFetched bool

	// Input mode state
	mode          inputMode
	textInput     textarea.Model
	replyThreadID string

	// Review submission state
	reviewCursor int // 0=approve, 1=request changes, 2=comment

	statusMsg string

	// Scroll position cache per file
	scrollCache map[string]scrollPosition

	onDiffModeChange func(string)
}

type scrollPosition struct {
	cursor       int
	scrollY      int
	threadCursor int
}

type ModelOption func(*Model)

func WithInitialDiffMode(mode string) ModelOption {
	return func(m *Model) {
		m.diffView.SetMode(mode)
	}
}

func WithDiffModeChangeHandler(fn func(string)) ModelOption {
	return func(m *Model) {
		m.onDiffModeChange = fn
	}
}

func NewModel(client *gh.Client, prNumber int, opts ...ModelOption) Model {
	h := help.New()
	h.ShowAll = false

	ta := textarea.New()
	ta.Placeholder = "Write a comment..."
	ta.ShowLineNumbers = false
	ta.SetHeight(3)

	model := Model{
		state:       stateLoading,
		client:      client,
		prNumber:    prNumber,
		help:        h,
		keyMap:      DefaultKeyMap(),
		focus:       leftPane,
		diffView:    NewDiffViewModel(),
		textInput:   ta,
		mode:        modeNormal,
		scrollCache: make(map[string]scrollPosition),
	}
	for _, opt := range opts {
		opt(&model)
	}
	return model
}

func (m Model) Init() tea.Cmd {
	return m.fetchPRCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case PRFetchedMsg:
		if msg.Err != nil {
			m.state = stateError
			m.err = msg.Err
			return m, nil
		}
		m.pr = msg.PR
		m.fileList.SetFiles(msg.PR.Files)
		return m, tea.Batch(m.fetchDiffsCmd(), m.fetchThreadsCmd())

	case DiffFetchedMsg:
		m.patchesFetched = true
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Diff fetch error: %v", msg.Err))
		} else {
			m.diffResult = msg.Result
		}
		if m.threadsFetched {
			m.state = stateReady
			m.fileList.MergeStatuses(m.diffResult)
			m.updateDiffView()
		}
		return m, nil

	case ThreadsFetchedMsg:
		m.threadsFetched = true
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Thread fetch error: %v", msg.Err))
		} else {
			m.threads = msg.Threads
		}
		if m.patchesFetched {
			m.state = stateReady
			m.fileList.MergeStatuses(m.diffResult)
			m.updateDiffView()
		}
		return m, nil

	case ViewedToggledMsg:
		m.statusMsg = ""
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Error: %v", msg.Err))
			return m, nil
		}
		m.fileList.UpdateViewedState(msg.Path, msg.NewState)
		if msg.NewState == gh.ViewedStateViewed {
			if f := m.fileList.SelectedFile(); f != nil {
				m.saveScrollPositionForPath(f.Path)
			}
			m.fileList.MoveToNextUnviewed()
		}
		m.updateDiffView()
		return m, nil

	case CommentAddedMsg:
		m.mode = modeNormal
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Comment error: %v", msg.Err))
			return m, nil
		}
		m.statusMsg = "Comment added"
		return m, m.refreshDataCmd()

	case ThreadRepliedMsg:
		m.mode = modeNormal
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Reply error: %v", msg.Err))
			return m, nil
		}
		m.statusMsg = "Reply added"
		return m, m.refreshDataCmd()

	case ThreadResolvedMsg:
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Resolve error: %v", msg.Err))
			return m, nil
		}
		if msg.IsResolved {
			m.statusMsg = "Thread resolved"
		} else {
			m.statusMsg = "Thread unresolved"
		}
		return m, m.refreshDataCmd()

	case ReviewSubmittedMsg:
		m.mode = modeNormal
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Review error: %v", msg.Err))
			return m, nil
		}
		m.statusMsg = fmt.Sprintf("Review submitted: %s", msg.Event)
		return m, nil

	case openedInBrowserMsg:
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Browser error: %v", msg.Err))
		} else {
			m.statusMsg = "Opened in browser"
		}
		return m, nil

	case DataRefreshedMsg:
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Refresh error: %v", msg.Err))
			return m, nil
		}
		m.pr = msg.PR
		m.diffResult = msg.Result
		m.threads = msg.Threads
		m.fileList.SetFiles(msg.PR.Files)
		m.fileList.MergeStatuses(m.diffResult)
		m.updateDiffView()
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeComment, modeReply:
		return m.handleInputKey(msg)
	case modeReview:
		return m.handleReviewKey(msg)
	case modeHelp:
		return m.handleHelpKey(msg)
	}

	switch {
	case key.Matches(msg, m.keyMap.Help):
		m.mode = modeHelp
		return m, nil

	case key.Matches(msg, m.keyMap.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keyMap.Tab):
		if m.focus == leftPane {
			m.focus = rightPane
		} else {
			m.focus = leftPane
		}
		return m, nil

	case key.Matches(msg, m.keyMap.ToggleViewed):
		if m.state != stateReady {
			return m, nil
		}
		return m, m.toggleViewedCmd()

	case key.Matches(msg, m.keyMap.SubmitReview):
		if m.state != stateReady {
			return m, nil
		}
		m.mode = modeReview
		m.reviewCursor = 0
		m.textInput.Reset()
		m.textInput.Placeholder = "Review body (optional)..."
		m.textInput.Focus()
		return m, nil

	case key.Matches(msg, m.keyMap.NextUnviewed):
		if m.state != stateReady {
			return m, nil
		}
		if f := m.fileList.SelectedFile(); f != nil {
			m.saveScrollPositionForPath(f.Path)
		}
		if !m.fileList.MoveToNextUnviewed() {
			m.statusMsg = "All files viewed"
		}
		m.updateDiffView()
		return m, nil

	case key.Matches(msg, m.keyMap.PrevUnviewed):
		if m.state != stateReady {
			return m, nil
		}
		if f := m.fileList.SelectedFile(); f != nil {
			m.saveScrollPositionForPath(f.Path)
		}
		if !m.fileList.MoveToPrevUnviewed() {
			m.statusMsg = "All files viewed"
		}
		m.updateDiffView()
		return m, nil

	case key.Matches(msg, m.keyMap.OpenInBrowser):
		if m.state != stateReady {
			return m, nil
		}
		return m, m.openInBrowserCmd()
	}

	if m.focus == leftPane {
		return m.handleLeftPaneKey(msg)
	}
	return m.handleRightPaneKey(msg)
}

func (m *Model) handleLeftPaneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Up):
		if f := m.fileList.SelectedFile(); f != nil {
			m.saveScrollPositionForPath(f.Path)
		}
		m.fileList.MoveUp()
		m.updateDiffView()
	case key.Matches(msg, m.keyMap.Down):
		if f := m.fileList.SelectedFile(); f != nil {
			m.saveScrollPositionForPath(f.Path)
		}
		m.fileList.MoveDown()
		m.updateDiffView()
	}
	return m, nil
}

func (m *Model) handleRightPaneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state != stateReady {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keyMap.Up):
		m.diffView.MoveUp()
	case key.Matches(msg, m.keyMap.Down):
		m.diffView.MoveDown()
	case key.Matches(msg, m.keyMap.HalfPageUp):
		m.diffView.HalfPageUp()
	case key.Matches(msg, m.keyMap.HalfPageDown):
		m.diffView.HalfPageDown()

	case key.Matches(msg, m.keyMap.Comment):
		dl := m.diffView.CursorLine()
		if dl == nil || dl.Type == diff.LineHunkHeader {
			return m, nil
		}
		m.mode = modeComment
		m.textInput.Reset()
		m.textInput.Placeholder = "Write a comment..."
		m.textInput.Focus()

	case key.Matches(msg, m.keyMap.Reply):
		t := m.diffView.CursorThread()
		if t == nil {
			t = m.findThreadAtCursor()
		}
		if t == nil {
			m.statusMsg = "No thread to reply to. Use 'n' to navigate to a thread."
			return m, nil
		}
		m.mode = modeReply
		m.replyThreadID = t.ID
		m.textInput.Reset()
		m.textInput.Placeholder = "Write a reply..."
		m.textInput.Focus()

	case key.Matches(msg, m.keyMap.Resolve):
		t := m.diffView.CursorThread()
		if t == nil {
			t = m.findThreadAtCursor()
		}
		if t == nil {
			m.statusMsg = "No thread to resolve. Use 'n' to navigate to a thread."
			return m, nil
		}
		return m, m.toggleResolveCmd(t)

	case key.Matches(msg, m.keyMap.NextThread):
		m.diffView.NextThread()
	case key.Matches(msg, m.keyMap.PrevThread):
		m.diffView.PrevThread()

	case key.Matches(msg, m.keyMap.NextHunk):
		m.diffView.NextHunk()
	case key.Matches(msg, m.keyMap.PrevHunk):
		m.diffView.PrevHunk()
	case key.Matches(msg, m.keyMap.ToggleDiffMode):
		m.diffView.ToggleMode()
		if m.onDiffModeChange != nil {
			m.onDiffModeChange(m.diffView.ModeString())
		}
		if m.diffView.Mode() == diffModeSplit && !m.diffView.CanRenderSplit() {
			m.statusMsg = "Split diff requires a wider pane"
		} else {
			m.statusMsg = ""
		}
	}

	return m, nil
}

func (m *Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Help), key.Matches(msg, m.keyMap.Cancel),
		key.Matches(msg, m.keyMap.Quit):
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Cancel):
		m.mode = modeNormal
		m.textInput.Blur()
		return m, nil

	case key.Matches(msg, m.keyMap.Submit):
		body := strings.TrimSpace(m.textInput.Value())
		if body == "" {
			return m, nil
		}
		m.textInput.Blur()

		if m.mode == modeComment {
			return m, m.addCommentCmd(body)
		}
		return m, m.replyToThreadCmd(m.replyThreadID, body)
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) handleReviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Cancel and Submit always take priority
	switch {
	case key.Matches(msg, m.keyMap.Cancel):
		m.mode = modeNormal
		m.textInput.Blur()
		return m, nil

	case key.Matches(msg, m.keyMap.Submit):
		events := []gh.ReviewEvent{gh.ReviewEventApprove, gh.ReviewEventRequestChanges, gh.ReviewEventComment}
		event := events[m.reviewCursor]
		body := strings.TrimSpace(m.textInput.Value())
		m.textInput.Blur()
		return m, m.submitReviewCmd(event, body)
	}

	// Tab switches focus between review options and textarea
	if msg.String() == "tab" {
		if m.textInput.Focused() {
			m.textInput.Blur()
		} else {
			m.textInput.Focus()
		}
		return m, nil
	}

	// When textarea is focused, forward all other input to it
	if m.textInput.Focused() {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Review option navigation (only when textarea is NOT focused)
	switch msg.String() {
	case "up", "k":
		if m.reviewCursor > 0 {
			m.reviewCursor--
		}
	case "down", "j":
		if m.reviewCursor < 2 {
			m.reviewCursor++
		}
	}

	return m, nil
}
