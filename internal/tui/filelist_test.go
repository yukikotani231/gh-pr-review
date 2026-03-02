package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

func newTestFiles() []gh.PRFile {
	return []gh.PRFile{
		{Path: "src/main.go", Additions: 10, Deletions: 5, ViewerViewedState: gh.ViewedStateUnviewed},
		{Path: "src/auth.go", Additions: 20, Deletions: 3, ViewerViewedState: gh.ViewedStateViewed},
		{Path: "src/util.go", Additions: 5, Deletions: 0, ViewerViewedState: gh.ViewedStateUnviewed},
		{Path: "go.mod", Additions: 2, Deletions: 1, ViewerViewedState: gh.ViewedStateUnviewed},
		{Path: "go.sum", Additions: 50, Deletions: 10, ViewerViewedState: gh.ViewedStateViewed},
	}
}

func TestFileList_SetFiles(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(newTestFiles())

	if len(m.files) != 5 {
		t.Fatalf("expected 5 files, got %d", len(m.files))
	}
	if m.cursor != 0 {
		t.Errorf("cursor should be 0 after SetFiles, got %d", m.cursor)
	}
	if m.offset != 0 {
		t.Errorf("offset should be 0 after SetFiles, got %d", m.offset)
	}
}

func TestFileList_SelectedFile(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)

	// Empty list
	if f := m.SelectedFile(); f != nil {
		t.Error("SelectedFile should return nil for empty list")
	}

	// With files
	m.SetFiles(newTestFiles())
	f := m.SelectedFile()
	if f == nil {
		t.Fatal("SelectedFile returned nil")
	}
	if f.Path != "src/main.go" {
		t.Errorf("expected first file, got %s", f.Path)
	}
}

func TestFileList_MoveDown(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(newTestFiles())

	m.MoveDown()
	if m.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", m.cursor)
	}
	if m.SelectedFile().Path != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %s", m.SelectedFile().Path)
	}
}

func TestFileList_MoveUp(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(newTestFiles())

	// Can't move up from 0
	m.MoveUp()
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	// Move down then up
	m.MoveDown()
	m.MoveDown()
	m.MoveUp()
	if m.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", m.cursor)
	}
}

func TestFileList_MoveDownBoundary(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(newTestFiles())

	// Move to last file
	for i := 0; i < 10; i++ {
		m.MoveDown()
	}
	if m.cursor != 4 {
		t.Errorf("cursor should be 4 (last), got %d", m.cursor)
	}
}

func TestFileList_ScrollOffset(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 3) // Only 3 visible
	m.SetFiles(newTestFiles())

	// Move past visible area
	m.MoveDown() // cursor=1, visible 0-2
	m.MoveDown() // cursor=2, visible 0-2
	m.MoveDown() // cursor=3, should scroll: offset=1, visible 1-3
	if m.offset != 1 {
		t.Errorf("offset should be 1, got %d", m.offset)
	}

	// Move back up
	m.MoveUp() // cursor=2
	m.MoveUp() // cursor=1
	if m.offset != 1 {
		t.Errorf("offset should still be 1, got %d", m.offset)
	}
	m.MoveUp() // cursor=0, should scroll back: offset=0
	if m.offset != 0 {
		t.Errorf("offset should be 0, got %d", m.offset)
	}
}

func TestFileList_ViewedCount(t *testing.T) {
	m := FileListModel{}
	m.SetFiles(newTestFiles())

	if count := m.ViewedCount(); count != 2 {
		t.Errorf("expected 2 viewed, got %d", count)
	}
}

func TestFileList_UpdateViewedState(t *testing.T) {
	m := FileListModel{}
	m.SetFiles(newTestFiles())

	m.UpdateViewedState("src/main.go", gh.ViewedStateViewed)
	if m.files[0].ViewerViewedState != gh.ViewedStateViewed {
		t.Error("first file should be viewed")
	}
	if m.ViewedCount() != 3 {
		t.Errorf("expected 3 viewed, got %d", m.ViewedCount())
	}

	// Non-existent path should be no-op
	m.UpdateViewedState("nonexistent.go", gh.ViewedStateViewed)
	if m.ViewedCount() != 3 {
		t.Errorf("expected 3 viewed after no-op, got %d", m.ViewedCount())
	}
}

func TestFileList_MoveToNextUnviewed(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(newTestFiles())
	// files[0]=unviewed, [1]=viewed, [2]=unviewed, [3]=unviewed, [4]=viewed

	// From 0 (unviewed), next unviewed is 2 (skipping viewed 1)
	m.MoveToNextUnviewed()
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", m.cursor)
	}

	// From 2, next unviewed is 3
	m.MoveToNextUnviewed()
	if m.cursor != 3 {
		t.Errorf("expected cursor at 3, got %d", m.cursor)
	}

	// From 3, next unviewed wraps to 0
	m.MoveToNextUnviewed()
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 (wrap), got %d", m.cursor)
	}
}

func TestFileList_MoveToNextUnviewed_AllViewed(t *testing.T) {
	files := []gh.PRFile{
		{Path: "a.go", ViewerViewedState: gh.ViewedStateViewed},
		{Path: "b.go", ViewerViewedState: gh.ViewedStateViewed},
	}
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(files)

	m.MoveToNextUnviewed()
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0 when all viewed, got %d", m.cursor)
	}
}

func TestFileList_MoveToNextUnviewed_SingleFile(t *testing.T) {
	files := []gh.PRFile{
		{Path: "only.go", ViewerViewedState: gh.ViewedStateUnviewed},
	}
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(files)

	m.MoveToNextUnviewed()
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0 for single file, got %d", m.cursor)
	}
}

func TestFileList_MoveToPrevUnviewed(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(newTestFiles())
	// files[0]=unviewed, [1]=viewed, [2]=unviewed, [3]=unviewed, [4]=viewed

	// Start at 3, prev unviewed is 2
	m.cursor = 3
	if !m.MoveToPrevUnviewed() {
		t.Error("MoveToPrevUnviewed should return true")
	}
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", m.cursor)
	}

	// From 2, prev unviewed is 0 (skipping viewed 1)
	if !m.MoveToPrevUnviewed() {
		t.Error("MoveToPrevUnviewed should return true")
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}

	// From 0, prev unviewed wraps to 3
	if !m.MoveToPrevUnviewed() {
		t.Error("MoveToPrevUnviewed should return true")
	}
	if m.cursor != 3 {
		t.Errorf("expected cursor at 3 (wrap), got %d", m.cursor)
	}
}

func TestFileList_MoveToPrevUnviewed_AllViewed(t *testing.T) {
	files := []gh.PRFile{
		{Path: "a.go", ViewerViewedState: gh.ViewedStateViewed},
		{Path: "b.go", ViewerViewedState: gh.ViewedStateViewed},
	}
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(files)

	if m.MoveToPrevUnviewed() {
		t.Error("MoveToPrevUnviewed should return false when all viewed")
	}
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}
}

func TestFileList_MergeStatuses(t *testing.T) {
	m := FileListModel{}
	m.SetFiles(newTestFiles())

	result := &gh.DiffResult{
		Patches:           map[string]string{},
		FileStatuses:      map[string]gh.FileStatus{"src/main.go": gh.FileStatusModified, "src/auth.go": gh.FileStatusAdded},
		PreviousFilenames: map[string]string{"go.mod": "old_go.mod"},
	}
	m.MergeStatuses(result)

	if m.files[0].Status != gh.FileStatusModified {
		t.Errorf("expected Modified, got %q", m.files[0].Status)
	}
	if m.files[1].Status != gh.FileStatusAdded {
		t.Errorf("expected Added, got %q", m.files[1].Status)
	}
	if m.files[3].PreviousFilename != "old_go.mod" {
		t.Errorf("expected old_go.mod, got %q", m.files[3].PreviousFilename)
	}
}

func TestFileList_View_EmptyList(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)

	view := m.View()
	if view != "No files" {
		t.Errorf("expected 'No files', got %q", view)
	}
}

func TestFileList_View_NonEmpty(t *testing.T) {
	m := FileListModel{}
	m.SetSize(40, 10)
	m.SetFiles(newTestFiles())

	view := m.View()
	if view == "" {
		t.Error("View returned empty string")
	}
	if view == "No files" {
		t.Error("View returned 'No files' for non-empty list")
	}
}

func TestFileList_View_LargeStatNoOverflow(t *testing.T) {
	files := []gh.PRFile{
		{Path: "small.go", Additions: 1, Deletions: 1, ViewerViewedState: gh.ViewedStateUnviewed},
		{Path: "large.go", Additions: 10000, Deletions: 20000, ViewerViewedState: gh.ViewedStateUnviewed},
		{Path: "huge.go", Additions: 999999, Deletions: 999999, ViewerViewedState: gh.ViewedStateViewed},
	}

	for _, paneWidth := range []int{20, 25, 30, 40, 60} {
		m := FileListModel{}
		m.SetSize(paneWidth, 10)
		m.SetFiles(files)

		view := m.View()
		for i, line := range strings.Split(view, "\n") {
			w := lipgloss.Width(line)
			if w > paneWidth {
				t.Errorf("paneWidth=%d, line %d exceeds width: got %d\n  line: %q", paneWidth, i, w, line)
			}
		}
	}
}
