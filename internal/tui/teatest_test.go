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

func testPRList() []gh.PRListItem {
	return []gh.PRListItem{
		{Number: 42, Title: "Fix authentication bug", Author: "alice", UpdatedAt: "2026-02-26T11:30:00Z"},
		{Number: 99, Title: "Add new feature", Author: "bob", UpdatedAt: "2026-02-25T12:00:00Z", IsDraft: true},
		{Number: 101, Title: "Update dependencies", Author: "charlie", UpdatedAt: "2026-02-26T08:00:00Z"},
	}
}

func TestTea_Selector_ShowItems(t *testing.T) {
	m := NewSelectorModel(nil) // client=nil, Init returns nil cmd
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Inject pre-loaded PR data
	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	// Wait for items to appear in output
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42")) && bytes.Contains(out, []byte("#99"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTea_Selector_SelectWithEnter(t *testing.T) {
	m := NewSelectorModel(nil)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Load items
	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42"))
	}, teatest.WithDuration(3*time.Second))

	// Press Enter to select the first item
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

	// Load items
	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42"))
	}, teatest.WithDuration(3*time.Second))

	// Press q to quit
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

	// Load items
	tm.Send(openPRsFetchedMsg{PRs: testPRList()})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#42"))
	}, teatest.WithDuration(3*time.Second))

	// Navigate down to second item, then select
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

	// Send error
	tm.Send(openPRsFetchedMsg{Err: fmt.Errorf("API rate limit exceeded")})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("エラー"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
