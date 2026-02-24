package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	threads  []gh.ReviewThread // all threads for this PR

	fileList FileListModel
	diffView DiffViewModel
	help     help.Model
	keyMap   KeyMap

	width  int
	height int
	focus  pane

	// Data fetch tracking
	patchesFetched bool
	threadsFetched bool

	// Input mode state
	mode          inputMode
	textInput     textarea.Model
	replyThreadID string // thread ID when replying

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
		state:    stateLoading,
		client:   client,
		prNumber: prNumber,
		help:     h,
		keyMap:   DefaultKeyMap(),
		focus:    leftPane,
		diffView: NewDiffViewModel(),
		textInput: ta,
		mode:     modeNormal,
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
		// Re-sync file list viewed states
		m.fileList.SetFiles(msg.PR.Files)
		m.updateDiffView()
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input modes first
	switch m.mode {
	case modeComment, modeReply:
		return m.handleInputKey(msg)
	case modeReview:
		return m.handleReviewKey(msg)
	}

	// Normal mode
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

	// Pane-specific keys
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
			// Try to find a thread at the current cursor line
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
		// modeReply
		return m, m.replyToThreadCmd(m.replyThreadID, body)
	}

	// Forward to textarea
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
		// Switch focus to/from textarea
		if m.textInput.Focused() {
			m.textInput.Blur()
		} else {
			m.textInput.Focus()
		}
		return m, nil
	}

	// If textarea is focused, forward input
	if m.textInput.Focused() {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the entire UI
func (m Model) View() string {
	if m.state == stateLoading {
		return fmt.Sprintf("\n  Loading PR #%d...\n", m.prNumber)
	}
	if m.state == stateError {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err)
	}

	header := m.renderHeader()
	content := m.renderContent()

	var bottom string
	switch m.mode {
	case modeComment, modeReply:
		bottom = m.renderInputArea()
	case modeReview:
		bottom = m.renderReviewModal()
	default:
		bottom = m.renderStatusBar()
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, bottom)
}

func (m Model) renderHeader() string {
	progress := fmt.Sprintf("%d/%d viewed",
		m.fileList.ViewedCount(), len(m.pr.Files))

	threadCount := m.unresolvedThreadCount()
	var threadInfo string
	if threadCount > 0 {
		threadInfo = fmt.Sprintf(", %d unresolved", threadCount)
	}

	title := fmt.Sprintf(" PR #%d: %s  (+%d -%d, %d files, %s%s)",
		m.pr.Number, m.pr.Title,
		m.pr.Additions, m.pr.Deletions,
		m.pr.ChangedFiles, progress, threadInfo)

	return headerStyle.Width(m.width).Render(title)
}

func (m Model) renderContent() string {
	leftWidth := m.leftPaneWidth()
	rightWidth := m.rightPaneWidth()
	contentHeight := m.contentHeight()

	var leftBorder, rightBorder lipgloss.Style
	if m.focus == leftPane {
		leftBorder = focusedBorderStyle.Width(leftWidth - 2).Height(contentHeight)
		rightBorder = unfocusedBorderStyle.Width(rightWidth - 2).Height(contentHeight)
	} else {
		leftBorder = unfocusedBorderStyle.Width(leftWidth - 2).Height(contentHeight)
		rightBorder = focusedBorderStyle.Width(rightWidth - 2).Height(contentHeight)
	}

	left := leftBorder.Render(m.fileList.View())

	// Show file path above diff
	var diffContent string
	if f := m.fileList.SelectedFile(); f != nil {
		pathLine := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render(f.Path)
		diffContent = pathLine + "\n" + m.diffView.View()
	} else {
		diffContent = m.diffView.View()
	}
	right := rightBorder.Render(diffContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) renderStatusBar() string {
	var helpBindings []key.Binding
	if m.focus == leftPane {
		helpBindings = []key.Binding{m.keyMap.Up, m.keyMap.Down, m.keyMap.ToggleViewed, m.keyMap.Tab, m.keyMap.SubmitReview, m.keyMap.Quit}
	} else {
		helpBindings = []key.Binding{m.keyMap.Up, m.keyMap.Down, m.keyMap.Comment, m.keyMap.Reply, m.keyMap.Resolve, m.keyMap.NextThread, m.keyMap.ToggleViewed, m.keyMap.SubmitReview, m.keyMap.Tab, m.keyMap.Quit}
	}
	helpView := m.help.ShortHelpView(helpBindings)

	status := helpView
	if m.statusMsg != "" {
		status = m.statusMsg + "  " + helpView
	}

	if f := m.fileList.SelectedFile(); f != nil {
		fileInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(f.Path)
		gap := strings.Repeat(" ", max(1, m.width-lipgloss.Width(status)-lipgloss.Width(fileInfo)-2))
		status = fileInfo + gap + status
	}

	return statusBarStyle.Width(m.width).Render(status)
}

func (m Model) renderInputArea() string {
	var label string
	if m.mode == modeComment {
		label = inputLabelStyle.Render(" New comment (Ctrl+s: submit, Esc: cancel)")
	} else {
		label = inputLabelStyle.Render(" Reply (Ctrl+s: submit, Esc: cancel)")
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		label,
		m.textInput.View(),
	)
}

func (m Model) renderReviewModal() string {
	events := []struct {
		label string
		event gh.ReviewEvent
	}{
		{"Approve", gh.ReviewEventApprove},
		{"Request Changes", gh.ReviewEventRequestChanges},
		{"Comment", gh.ReviewEventComment},
	}

	var sb strings.Builder
	sb.WriteString(inputLabelStyle.Render("Submit Review") + "\n\n")

	for i, e := range events {
		prefix := "  "
		style := reviewOptionStyle
		if i == m.reviewCursor {
			prefix = "> "
			style = reviewSelectedStyle
		}
		sb.WriteString(style.Render(fmt.Sprintf("%s%s", prefix, e.label)))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.textInput.View())
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		"  ↑↓: select  Tab: edit body  Ctrl+s: submit  Esc: cancel"))

	return reviewModalStyle.Width(m.width - 4).Render(sb.String())
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

// --- Commands ---

func (m Model) fetchPRCmd() tea.Cmd {
	return func() tea.Msg {
		pr, err := m.client.FetchPR(m.prNumber)
		return PRFetchedMsg{PR: pr, Err: err}
	}
}

func (m Model) fetchDiffsCmd() tea.Cmd {
	return func() tea.Msg {
		patches, err := m.client.FetchDiffs(m.prNumber)
		return DiffFetchedMsg{Patches: patches, Err: err}
	}
}

func (m Model) fetchThreadsCmd() tea.Cmd {
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

	var side string
	var line int
	if dl.Type == diff.LineRemoved {
		side = "LEFT"
		line = dl.OldLineNum
	} else {
		side = "RIGHT"
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
	overhead := 3 // header(1) + status(1) + margin(1)
	switch m.mode {
	case modeComment, modeReply:
		overhead = 6 // header(1) + input area(~5)
	case modeReview:
		overhead = 14 // header(1) + review modal(~13)
	}
	h := m.height - overhead
	if h < 5 {
		h = 5
	}
	return h
}
