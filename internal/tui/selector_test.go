package tui

import (
	"strings"
	"testing"
)

func TestPRItem_Title(t *testing.T) {
	item := prItem{
		number: 42,
		title:  "Fix authentication bug",
		author: "alice",
	}
	got := item.Title()
	if got != "#42 Fix authentication bug" {
		t.Errorf("Title() = %q, want %q", got, "#42 Fix authentication bug")
	}
}

func TestPRItem_Title_Draft(t *testing.T) {
	item := prItem{
		number:  99,
		title:   "WIP feature",
		author:  "bob",
		isDraft: true,
	}
	got := item.Title()
	if !strings.Contains(got, "[draft]") {
		t.Errorf("Title() = %q, should contain [draft]", got)
	}
	if !strings.Contains(got, "#99") {
		t.Errorf("Title() = %q, should contain #99", got)
	}
}

func TestPRItem_Description(t *testing.T) {
	item := prItem{
		number:    42,
		title:     "Fix bug",
		author:    "alice",
		updatedAt: "invalid-time",
	}
	got := item.Description()
	if !strings.Contains(got, "@alice") {
		t.Errorf("Description() = %q, should contain @alice", got)
	}
}

func TestPRItem_FilterValue(t *testing.T) {
	item := prItem{
		number: 42,
		title:  "Fix bug",
		author: "alice",
	}
	got := item.FilterValue()
	if !strings.Contains(got, "42") {
		t.Errorf("FilterValue() = %q, should contain PR number", got)
	}
	if !strings.Contains(got, "Fix bug") {
		t.Errorf("FilterValue() = %q, should contain title", got)
	}
	if !strings.Contains(got, "alice") {
		t.Errorf("FilterValue() = %q, should contain author", got)
	}
}

func TestSelectorModel_InitialState(t *testing.T) {
	m := SelectorModel{}
	if m.Selected() != 0 {
		t.Errorf("Selected() = %d, want 0", m.Selected())
	}
	if m.Quitting() {
		t.Error("Quitting() should be false initially")
	}
}
