package tui

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

// --- Test data helpers ---

func testPRList() []gh.PRListItem {
	return []gh.PRListItem{
		{Number: 42, Title: "Fix authentication bug", Author: "alice", UpdatedAt: "2026-02-26T11:30:00Z"},
		{Number: 99, Title: "Add new feature", Author: "bob", UpdatedAt: "2026-02-25T12:00:00Z", IsDraft: true},
		{Number: 101, Title: "Update dependencies", Author: "charlie", UpdatedAt: "2026-02-26T08:00:00Z"},
	}
}

func mockPR() *gh.PullRequest {
	return &gh.PullRequest{
		ID:           "PR_123",
		Title:        "Fix authentication bug",
		Number:       42,
		Additions:    15,
		Deletions:    3,
		ChangedFiles: 2,
		Files: []gh.PRFile{
			{Path: "src/auth.go", Additions: 10, Deletions: 2, ViewerViewedState: gh.ViewedStateUnviewed},
			{Path: "src/auth_test.go", Additions: 5, Deletions: 1, ViewerViewedState: gh.ViewedStateViewed},
		},
	}
}

func mockPatches() map[string]string {
	return map[string]string{
		"src/auth.go": `@@ -1,5 +1,6 @@
 package auth

+import "errors"
 func Login(user, pass string) error {
 	return nil
 }`,
		"src/auth_test.go": `@@ -1,3 +1,4 @@
 package auth

+func TestLogin(t *testing.T) {}
 // existing test`,
	}
}

func mockThreads() []gh.ReviewThread {
	return []gh.ReviewThread{
		{
			ID: "RT_1", IsResolved: false, Path: "src/auth.go",
			Line: 3, DiffSide: gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{ID: "C_1", Body: "Should validate input", Author: "reviewer", CreatedAt: "2026-02-26T10:00:00Z"},
			},
		},
	}
}

// sendReadyState injects the messages needed to bring Model to stateReady.
func sendReadyState(tm *teatest.TestModel) {
	tm.Send(PRFetchedMsg{PR: mockPR()})
	tm.Send(DiffFetchedMsg{Patches: mockPatches()})
	tm.Send(ThreadsFetchedMsg{Threads: mockThreads()})
}

func waitForReady(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Fix authentication bug"))
	}, teatest.WithDuration(3*time.Second))
}

// --- Selector tests ---

func TestTea_Selector_ShowItems(t *testing.T) {
	m := NewSelectorModel(nil)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42")) && bytes.Contains(out, []byte("#99"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Selector_SelectWithEnter(t *testing.T) {
	m := NewSelectorModel(nil)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	finalModel := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	sm := finalModel.(SelectorModel)

	if sm.Selected() != 42 {
		t.Errorf("Selected() = %d, want 42", sm.Selected())
	}
	if sm.Quitting() {
		t.Error("Quitting() should be false when selecting with Enter")
	}
}

func TestTea_Selector_QuitWithQ(t *testing.T) {
	m := NewSelectorModel(nil)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	finalModel := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	sm := finalModel.(SelectorModel)

	if sm.Selected() != 0 {
		t.Errorf("Selected() = %d, want 0", sm.Selected())
	}
	if !sm.Quitting() {
		t.Error("Quitting() should be true after pressing q")
	}
}

func TestTea_Selector_NavigateDown(t *testing.T) {
	m := NewSelectorModel(nil)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	finalModel := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	sm := finalModel.(SelectorModel)

	if sm.Selected() != 99 {
		t.Errorf("Selected() = %d, want 99 (second item)", sm.Selected())
	}
}

func TestTea_Selector_ErrorDisplay(t *testing.T) {
	m := NewSelectorModel(nil)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(openPRsFetchedMsg{Err: fmt.Errorf("API rate limit exceeded")})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("エラー"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// --- Model tests ---

func TestTea_Model_DataLoading(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)

	// Verify PR title and file list are visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Fix authentication bug")) &&
			bytes.Contains(out, []byte("auth.go"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Model_TabSwitchPane(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)
	waitForReady(t, tm)

	// Switch to right pane
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})

	// Verify diff content is shown (right pane has diff)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("auth.go"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Model_FileNavigation(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)
	waitForReady(t, tm)

	// Move down to second file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Verify second file appears in diff area
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("auth_test.go"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Model_CommentMode(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)
	waitForReady(t, tm)

	// Switch to right pane, move down from hunk header, then press c
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // move to diff line
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Verify comment input area appears
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("New comment"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Model_CommentCancel(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)
	waitForReady(t, tm)

	// Enter comment mode
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("New comment"))
	}, teatest.WithDuration(3*time.Second))

	// Cancel with Esc
	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})

	// Verify back to normal mode (status bar visible again)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// After cancel, the "New comment" label should disappear
		return !bytes.Contains(out, []byte("New comment"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Model_ReviewMode(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)
	waitForReady(t, tm)

	// Press S to open review modal
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})

	// Verify review modal appears
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Submit Review")) || bytes.Contains(out, []byte("Approve"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Model_ReviewCancel(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)
	waitForReady(t, tm)

	// Open review then cancel
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Approve"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return !bytes.Contains(out, []byte("Approve"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Model_Quit(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendReadyState(tm)
	waitForReady(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	// If we reach here without timeout, quit worked
}

func TestTea_Model_ErrorState(t *testing.T) {
	m := NewModel(nil, 42)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Send error
	tm.Send(PRFetchedMsg{Err: fmt.Errorf("not found")})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Error")) || bytes.Contains(out, []byte("not found"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
