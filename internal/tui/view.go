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
