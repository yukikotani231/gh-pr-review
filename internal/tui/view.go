package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

// View renders the entire UI.
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

	base := lipgloss.JoinVertical(lipgloss.Left, header, content, bottom)

	if m.mode == modeHelp {
		return m.renderHelpOverlay()
	}

	return base
}

func (m Model) renderHeader() string {
	progress := fmt.Sprintf("%d/%d viewed",
		m.fileList.ViewedCount(), len(m.pr.Files))

	threadCount := m.unresolvedThreadCount()
	pendingCount := m.pendingThreadCount()
	var threadInfo string
	if threadCount > 0 {
		threadInfo = fmt.Sprintf(", %d unresolved", threadCount)
	}
	if pendingCount > 0 {
		threadInfo += fmt.Sprintf(", %d pending", pendingCount)
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

	var diffContent string
	if f := m.fileList.SelectedFile(); f != nil {
		pathLine := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			MaxWidth(max(1, m.diffView.width)).
			Render(f.Path)
		diffContent = pathLine + "\n" + m.diffView.View()
	} else {
		diffContent = m.diffView.View()
	}
	right := rightBorder.Render(diffContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) renderStatusBar() string {
	barWidth := max(1, m.width)
	contentWidth := max(1, barWidth-2) // account for statusBarStyle horizontal padding

	var helpBindings []key.Binding
	if m.focus == leftPane {
		helpBindings = []key.Binding{m.keyMap.Up, m.keyMap.Down, m.keyMap.ToggleViewed, m.keyMap.Tab, m.keyMap.SubmitReview, m.keyMap.Help, m.keyMap.Quit}
	} else {
		helpBindings = []key.Binding{m.keyMap.Up, m.keyMap.Down, m.keyMap.Comment, m.keyMap.Reply, m.keyMap.Resolve, m.keyMap.NextThread, m.keyMap.ToggleDiffMode, m.keyMap.ToggleViewed, m.keyMap.SubmitReview, m.keyMap.Help, m.keyMap.Tab, m.keyMap.Quit}
	}
	// Show hunk position when right pane is focused
	var hunkInfo string
	if m.focus == rightPane && len(m.diffView.diffLines) > 0 {
		current, total := m.diffView.HunkPosition()
		if total > 0 {
			hunkInfo = fmt.Sprintf("Hunk %d/%d %s", current, total, m.diffView.ModeLabel())
		}
	} else if m.focus == rightPane {
		hunkInfo = m.diffView.ModeLabel()
	}

	var tailPrefix string
	switch {
	case m.statusMsg != "" && hunkInfo != "":
		tailPrefix = m.statusMsg + "  " + hunkInfo
	case m.statusMsg != "":
		tailPrefix = m.statusMsg
	default:
		tailPrefix = hunkInfo
	}

	status := ""
	if f := m.fileList.SelectedFile(); f != nil {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(f.Path)
	}

	remaining := contentWidth - lipgloss.Width(status)
	if status != "" && remaining > 0 {
		status += " "
		remaining--
	}

	tail := ""
	if tailPrefix != "" {
		tail = truncateDisplay(tailPrefix, max(0, remaining))
	}
	remaining -= lipgloss.Width(tail)
	if tail != "" && remaining > 0 {
		tail += " "
		remaining--
	}

	helpView := m.shortHelpToFit(helpBindings, max(0, remaining))
	if tail != "" {
		status += tail
	}
	if helpView != "" {
		status += helpView
	}
	if status == "" {
		status = m.shortHelpToFit(helpBindings, contentWidth)
		if status == "" {
			status = m.statusMsg
		}
	}

	status = strings.ReplaceAll(status, "\n", " ")
	status = truncateDisplay(status, contentWidth)

	return statusBarStyle.MaxWidth(barWidth).Render(status)
}

func (m Model) shortHelpToFit(bindings []key.Binding, maxWidth int) string {
	if maxWidth <= 0 || len(bindings) == 0 {
		return ""
	}

	for n := len(bindings); n > 0; n-- {
		s := strings.ReplaceAll(m.help.ShortHelpView(bindings[:n]), "\n", " ")
		if lipgloss.Width(s) <= maxWidth {
			return s
		}
	}
	return ""
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

func (m Model) renderHelpOverlay() string {
	sections := []struct {
		title    string
		bindings []key.Binding
	}{
		{"Navigation", []key.Binding{m.keyMap.Up, m.keyMap.Down, m.keyMap.HalfPageUp, m.keyMap.HalfPageDown, m.keyMap.Tab}},
		{"File", []key.Binding{m.keyMap.NextUnviewed, m.keyMap.PrevUnviewed, m.keyMap.ToggleViewed}},
		{"Diff", []key.Binding{m.keyMap.NextHunk, m.keyMap.PrevHunk, m.keyMap.NextThread, m.keyMap.PrevThread, m.keyMap.ToggleDiffMode}},
		{"Actions", []key.Binding{m.keyMap.Comment, m.keyMap.Reply, m.keyMap.Resolve, m.keyMap.SubmitReview, m.keyMap.OpenInBrowser}},
		{"General", []key.Binding{m.keyMap.Help, m.keyMap.Quit}},
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	keyStyle := lipgloss.NewStyle().Bold(true).Width(12)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	for _, s := range sections {
		sb.WriteString(sectionStyle.Render(s.title))
		sb.WriteString("\n")
		for _, b := range s.bindings {
			h := b.Help()
			fmt.Fprintf(&sb, "  %s %s\n", keyStyle.Render(h.Key), h.Desc)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(hintStyle.Render("Press ? or Esc to close"))

	overlayWidth := 44
	if m.width-4 < overlayWidth {
		overlayWidth = m.width - 4
	}
	if overlayWidth < 1 {
		overlayWidth = 1
	}

	overlay := helpOverlayStyle.
		Width(overlayWidth).
		Render(sb.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
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

	return reviewModalStyle.Width(max(1, m.width-4)).Render(sb.String())
}
