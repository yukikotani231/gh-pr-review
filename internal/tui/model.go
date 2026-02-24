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
)

type Model struct {
	state    state
	err      error
	client   *gh.Client
	pr       *gh.PullRequest
	prNumber int
	patches  map[string]string
	threads  []gh.ReviewThread

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
}

func NewModel(client *gh.Client, prNumber int) Model {
	h := help.New()
	h.ShowAll = false

	ta := textarea.New()
	ta.Placeholder = "Write a comment..."
	ta.ShowLineNumbers = false
	ta.SetHeight(3)

	return Model{
		state:     stateLoading,
		client:    client,
		prNumber:  prNumber,
		help:      h,
		keyMap:    DefaultKeyMap(),
		focus:     leftPane,
		diffView:  NewDiffViewModel(),
		textInput: ta,
		mode:      modeNormal,
	}
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
			m.patches = msg.Patches
		}
		if m.threadsFetched {
			m.state = stateReady
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

	case DataRefreshedMsg:
		if msg.Err != nil {
			m.statusMsg = errorStyle.Render(fmt.Sprintf("Refresh error: %v", msg.Err))
			return m, nil
		}
		m.pr = msg.PR
		m.patches = msg.Patches
		m.threads = msg.Threads
		m.fileList.SetFiles(msg.PR.Files)
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
	}

	switch {
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
	}

	if m.focus == leftPane {
		return m.handleLeftPaneKey(msg)
	}
	return m.handleRightPaneKey(msg)
}

func (m *Model) handleLeftPaneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Up):
		m.fileList.MoveUp()
		m.updateDiffView()
	case key.Matches(msg, m.keyMap.Down):
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

	switch msg.String() {
	case "up", "k":
		if m.reviewCursor > 0 {
			m.reviewCursor--
		}
		return m, nil
	case "down", "j":
		if m.reviewCursor < 2 {
			m.reviewCursor++
		}
		return m, nil
	case "tab":
		if m.textInput.Focused() {
			m.textInput.Blur()
		} else {
			m.textInput.Focus()
		}
		return m, nil
	}

	if m.textInput.Focused() {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}
