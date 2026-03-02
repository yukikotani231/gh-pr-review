package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

func TestFileSwitch_SavesScrollUnderPreviousFile(t *testing.T) {
	m := NewModel(nil, 42)
	m.state = stateReady
	m.width = 120
	m.height = 40
	m.fileList.SetFiles([]gh.PRFile{
		{Path: "a.go", ViewerViewedState: gh.ViewedStateUnviewed},
		{Path: "b.go", ViewerViewedState: gh.ViewedStateUnviewed},
	})
	m.diffResult = &gh.DiffResult{
		Patches: map[string]string{
			"a.go": "@@ -1,2 +1,2 @@\n line1\n+line2",
			"b.go": "@@ -1,2 +1,2 @@\n line1\n+line2",
		},
	}
	m.updateLayout()
	m.updateDiffView() // a.go

	// Simulate user scrolling on a.go.
	lines := diff.Parse(m.diffResult.Patches["a.go"])
	m.diffView.SetContent(lines, nil)
	m.diffView.cursor = 1
	m.diffView.scrollY = 1

	_, _ = m.handleLeftPaneKey(tea.KeyMsg{Type: tea.KeyDown})

	posA, okA := m.scrollCache["a.go"]
	if !okA {
		t.Fatal("expected scroll cache for a.go")
	}
	if posA.cursor != 1 || posA.scrollY != 1 {
		t.Fatalf("unexpected cached position for a.go: %+v", posA)
	}
	if _, okB := m.scrollCache["b.go"]; okB {
		t.Fatal("did not expect b.go to be cached when switching from a.go")
	}
}
