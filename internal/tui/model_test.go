package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

func TestModel_UpdateDiffModeStatusClearsNarrowWarningAfterResize(t *testing.T) {
	m := NewModel(nil, 42)
	m.statusMsg = splitTooNarrowMsg
	m.diffView.SetMode("split")
	m.diffView.SetSize(100, 20)

	m.updateDiffModeStatus()

	if m.statusMsg != "" {
		t.Fatalf("statusMsg = %q, want empty", m.statusMsg)
	}
}

func TestModel_HandleRightPaneKey_ResolvePendingThreadShowsStatus(t *testing.T) {
	m := NewModel(nil, 1)
	m.state = stateReady
	m.focus = rightPane
	m.threads = []gh.ReviewThread{
		{
			ID:        "pending-1",
			IsPending: true,
			Path:      "main.go",
			Line:      1,
			DiffSide:  gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{ID: "c1", Body: "pending", Author: "alice", CreatedAt: "2026-01-01T00:00:00Z"},
			},
		},
	}
	m.diffView.SetSize(80, 20)
	m.diffView.SetContent([]diff.DiffLine{
		{Type: diff.LineHunkHeader, Content: "@@ -1 +1 @@"},
		{Type: diff.LineAdded, Content: "+line", NewLineNum: 1},
	}, m.threads)
	m.diffView.threadCursor = 0

	_, cmd := m.handleRightPaneKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if cmd != nil {
		t.Fatal("expected no resolve command for pending thread")
	}
	if m.statusMsg != "Pending review threads cannot be resolved" {
		t.Fatalf("statusMsg = %q", m.statusMsg)
	}
}
