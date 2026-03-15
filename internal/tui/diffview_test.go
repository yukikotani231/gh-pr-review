package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

func testDiffLines() []diff.DiffLine {
	return diff.Parse(`@@ -1,5 +1,6 @@
 line1
-old line2
+new line2
+added line3
 line4
 line5`)
}

func testDiffLinesMultipleChanges() []diff.DiffLine {
	return diff.Parse(`@@ -1,6 +1,6 @@
 line1
-old line2
-old line3
+new line2
+new line3
 line4
 line5`)
}

func testThreads() []gh.ReviewThread {
	return []gh.ReviewThread{
		{
			ID:         "thread1",
			IsResolved: false,
			Path:       "test.go",
			Line:       2, // matches new line2 (NewLineNum=2)
			DiffSide:   gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{ID: "c1", Body: "Fix this", Author: "alice", CreatedAt: "2026-02-24T10:00:00Z"},
			},
		},
		{
			ID:         "thread2",
			IsResolved: true,
			Path:       "test.go",
			Line:       5, // matches line5 (NewLineNum=5)
			DiffSide:   gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{ID: "c2", Body: "LGTM", Author: "bob", CreatedAt: "2026-02-24T09:00:00Z"},
				{ID: "c3", Body: "Thanks!", Author: "alice", CreatedAt: "2026-02-24T09:30:00Z"},
			},
		},
	}
}

func TestDiffView_SetContent(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)

	lines := testDiffLines()
	m.SetContent(lines, nil)

	if m.cursor != 0 {
		t.Errorf("cursor should be 0, got %d", m.cursor)
	}
	if len(m.diffLines) != len(lines) {
		t.Errorf("expected %d lines, got %d", len(lines), len(m.diffLines))
	}
}

func TestDiffView_CursorLine(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	cl := m.CursorLine()
	if cl == nil {
		t.Fatal("CursorLine returned nil")
	}
	if cl.Type != diff.LineHunkHeader {
		t.Errorf("expected hunk header at cursor 0, got type %d", cl.Type)
	}
}

func TestDiffView_CursorLine_Empty(t *testing.T) {
	m := NewDiffViewModel()
	if m.CursorLine() != nil {
		t.Error("CursorLine should return nil for empty diff")
	}
}

func TestDiffView_MoveDown(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	m.MoveDown()
	if m.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", m.cursor)
	}

	cl := m.CursorLine()
	if cl.Content != " line1" {
		t.Errorf("expected ' line1', got %q", cl.Content)
	}
}

func TestDiffView_MoveUp(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	// Can't move up from 0
	m.MoveUp()
	if m.cursor != 0 {
		t.Errorf("cursor should stay 0, got %d", m.cursor)
	}

	m.MoveDown()
	m.MoveDown()
	m.MoveUp()
	if m.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", m.cursor)
	}
}

func TestDiffView_MoveDownBoundary(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	lines := testDiffLines()
	m.SetContent(lines, nil)

	for i := 0; i < 100; i++ {
		m.MoveDown()
	}
	if m.cursor != len(lines)-1 {
		t.Errorf("cursor should be %d, got %d", len(lines)-1, m.cursor)
	}
}

func TestDiffView_HalfPageDown(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 4)
	m.SetContent(testDiffLines(), nil)

	m.HalfPageDown() // moves by 2
	if m.cursor != 2 {
		t.Errorf("cursor should be 2, got %d", m.cursor)
	}
}

func TestDiffView_HalfPageUp(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 4)
	m.SetContent(testDiffLines(), nil)

	m.cursor = 4
	m.HalfPageUp() // moves by 2
	if m.cursor != 2 {
		t.Errorf("cursor should be 2, got %d", m.cursor)
	}
}

func TestDiffView_HalfPageDown_Empty(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 10)
	m.SetContent(nil, nil)

	// Should not panic
	m.HalfPageDown()
	if m.cursor != 0 {
		t.Errorf("cursor should be 0 on empty, got %d", m.cursor)
	}
}

func TestDiffView_WithThreads(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(testDiffLines(), testThreads())

	// Display rows should be more than diff lines due to inline comments
	m.buildDisplayRows()
	if len(m.displayRows) <= len(m.diffLines) {
		t.Errorf("display rows (%d) should be more than diff lines (%d) when threads exist",
			len(m.displayRows), len(m.diffLines))
	}
}

func TestDiffView_NextThread(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(testDiffLines(), testThreads())

	m.NextThread()
	if m.threadCursor != 0 {
		t.Errorf("threadCursor should be 0, got %d", m.threadCursor)
	}

	ct := m.CursorThread()
	if ct == nil {
		t.Fatal("CursorThread returned nil")
	}
	if ct.ID != "thread1" {
		t.Errorf("expected thread1, got %s", ct.ID)
	}

	m.NextThread()
	if m.threadCursor != 1 {
		t.Errorf("threadCursor should be 1, got %d", m.threadCursor)
	}

	// Wrap around
	m.NextThread()
	if m.threadCursor != 0 {
		t.Errorf("threadCursor should wrap to 0, got %d", m.threadCursor)
	}
}

func TestDiffView_PrevThread(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(testDiffLines(), testThreads())

	// First PrevThread should wrap to last
	m.PrevThread()
	if m.threadCursor != 1 {
		t.Errorf("threadCursor should wrap to 1, got %d", m.threadCursor)
	}

	m.PrevThread()
	if m.threadCursor != 0 {
		t.Errorf("threadCursor should be 0, got %d", m.threadCursor)
	}
}

func TestDiffView_NextThread_NoThreads(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	// Should not panic
	m.NextThread()
	m.PrevThread()

	if m.CursorThread() != nil {
		t.Error("CursorThread should be nil when no threads")
	}
}

func TestDiffView_CursorThread_OutOfRange(t *testing.T) {
	m := NewDiffViewModel()
	m.threadCursor = 5
	if m.CursorThread() != nil {
		t.Error("CursorThread should return nil for out-of-range index")
	}
}

func TestDiffView_MoveResetsThreadCursor(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(testDiffLines(), testThreads())

	m.NextThread()
	if m.threadCursor != 0 {
		t.Fatal("setup failed")
	}

	m.MoveDown()
	if m.threadCursor != -1 {
		t.Errorf("MoveDown should reset threadCursor to -1, got %d", m.threadCursor)
	}

	m.NextThread()
	m.MoveUp()
	if m.threadCursor != -1 {
		t.Errorf("MoveUp should reset threadCursor to -1, got %d", m.threadCursor)
	}

	m.NextThread()
	m.HalfPageDown()
	if m.threadCursor != -1 {
		t.Errorf("HalfPageDown should reset threadCursor to -1, got %d", m.threadCursor)
	}
}

func TestDiffView_ScrollDown(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 3) // Small viewport
	m.SetContent(testDiffLines(), nil)

	m.ScrollDown()
	if m.scrollY != 1 {
		t.Errorf("scrollY should be 1, got %d", m.scrollY)
	}
}

func TestDiffView_ScrollUp(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 3)
	m.SetContent(testDiffLines(), nil)

	// Can't scroll up from 0
	m.ScrollUp()
	if m.scrollY != 0 {
		t.Errorf("scrollY should stay 0, got %d", m.scrollY)
	}

	m.ScrollDown()
	m.ScrollDown()
	m.ScrollUp()
	if m.scrollY != 1 {
		t.Errorf("scrollY should be 1, got %d", m.scrollY)
	}
}

func TestDiffView_View_Empty(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(nil, nil)

	view := m.View()
	if view != "(no diff available)" {
		t.Errorf("expected no diff message, got %q", view)
	}
}

func TestDiffView_View_NonEmpty(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	view := m.View()
	if view == "" || view == "(no diff available)" {
		t.Errorf("expected non-empty view, got %q", view)
	}
}

func TestDiffView_ToggleMode_Split(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	m.ToggleMode()

	if m.Mode() != diffModeSplit {
		t.Fatalf("mode = %v, want split", m.Mode())
	}
	if !m.CanRenderSplit() {
		t.Fatal("expected split mode to be renderable at width 80")
	}
	if got := m.ModeString(); got != "split" {
		t.Fatalf("ModeString() = %q, want split", got)
	}

	view := m.View()
	if !strings.Contains(view, "old line2") || !strings.Contains(view, "new line2") {
		t.Fatalf("split view should contain both sides of changed line, got %q", view)
	}
	if !strings.Contains(view, "-    2 old line2") {
		t.Fatalf("split view should show removed marker column, got %q", view)
	}
	if !strings.Contains(view, "+    2 new line2") {
		t.Fatalf("split view should show added marker column, got %q", view)
	}
	if strings.Contains(view, "-old line2") || strings.Contains(view, "+new line2") {
		t.Fatalf("split view should drop inline diff prefixes inside cells, got %q", view)
	}
}

func TestDiffView_SetMode(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	m.SetMode("split")
	if got := m.ModeString(); got != "split" {
		t.Fatalf("ModeString() = %q, want split", got)
	}

	m.SetMode("unified")
	if got := m.ModeString(); got != "unified" {
		t.Fatalf("ModeString() = %q, want unified", got)
	}
}

func TestDiffView_SplitFallbackOnNarrowWidth(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(32, 20)
	m.SetContent(testDiffLines(), nil)

	m.ToggleMode()

	if m.Mode() != diffModeSplit {
		t.Fatalf("mode = %v, want split preference", m.Mode())
	}
	if m.CanRenderSplit() {
		t.Fatal("expected split mode to fall back at narrow width")
	}
	if got := m.ModeLabel(); got != "[split->unified]" {
		t.Fatalf("ModeLabel() = %q, want fallback label", got)
	}
}

func TestDiffView_SplitBecomesAvailableAtReducedThreshold(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(minSplitDiffWidth, 20)
	m.SetContent(testDiffLines(), nil)

	m.ToggleMode()

	if !m.CanRenderSplit() {
		t.Fatalf("expected split to be available at width %d", minSplitDiffWidth)
	}
	if got := m.ModeLabel(); got != "[split]" {
		t.Fatalf("ModeLabel() = %q, want [split]", got)
	}
}

func TestDiffView_SplitHunkHeaderKeepsHeaderText(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	m.ToggleMode()
	view := m.View()
	if !strings.Contains(view, "@@ -1,5 +1,6 @@") {
		t.Fatalf("split hunk header missing from view: %q", view)
	}
}

func TestDiffView_SplitAlignsRemovedAndAddedRows(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	m.ToggleMode()
	m.buildDisplayRows()

	if got := m.lineToFirstRow[2]; got != m.lineToFirstRow[3] {
		t.Fatalf("removed and added line should share a split row, got %d and %d", m.lineToFirstRow[2], m.lineToFirstRow[3])
	}
}

func TestDiffView_SplitAlignsMultipleRemovedAndAddedRows(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLinesMultipleChanges(), nil)

	m.ToggleMode()
	m.buildDisplayRows()

	if got := m.lineToFirstRow[2]; got != m.lineToFirstRow[4] {
		t.Fatalf("first removed/added pair should share a split row, got %d and %d", m.lineToFirstRow[2], m.lineToFirstRow[4])
	}
	if got := m.lineToFirstRow[3]; got != m.lineToFirstRow[5] {
		t.Fatalf("second removed/added pair should share a split row, got %d and %d", m.lineToFirstRow[3], m.lineToFirstRow[5])
	}
}

func TestDiffView_ToggleMode_ClampsScroll(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 3)
	m.SetContent(testDiffLines(), nil)
	m.scrollY = len(m.displayRows) - 1

	m.ToggleMode()

	maxScroll := len(m.displayRows) - m.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY > maxScroll {
		t.Fatalf("scrollY = %d, want <= %d after mode toggle", m.scrollY, maxScroll)
	}
}

func TestMatchesThread(t *testing.T) {
	tests := []struct {
		name     string
		dl       diff.DiffLine
		thread   gh.ReviewThread
		expected bool
	}{
		{
			name:     "RIGHT side matches NewLineNum",
			dl:       diff.DiffLine{NewLineNum: 10, OldLineNum: 8},
			thread:   gh.ReviewThread{Line: 10, DiffSide: gh.DiffSideRight},
			expected: true,
		},
		{
			name:     "RIGHT side no match",
			dl:       diff.DiffLine{NewLineNum: 10, OldLineNum: 8},
			thread:   gh.ReviewThread{Line: 8, DiffSide: gh.DiffSideRight},
			expected: false,
		},
		{
			name:     "LEFT side matches OldLineNum",
			dl:       diff.DiffLine{NewLineNum: 10, OldLineNum: 8},
			thread:   gh.ReviewThread{Line: 8, DiffSide: gh.DiffSideLeft},
			expected: true,
		},
		{
			name:     "LEFT side no match",
			dl:       diff.DiffLine{NewLineNum: 10, OldLineNum: 8},
			thread:   gh.ReviewThread{Line: 10, DiffSide: gh.DiffSideLeft},
			expected: false,
		},
		{
			name:     "added line with RIGHT thread",
			dl:       diff.DiffLine{NewLineNum: 5, OldLineNum: 0, Type: diff.LineAdded},
			thread:   gh.ReviewThread{Line: 5, DiffSide: gh.DiffSideRight},
			expected: true,
		},
		{
			name:     "removed line with LEFT thread",
			dl:       diff.DiffLine{NewLineNum: 0, OldLineNum: 3, Type: diff.LineRemoved},
			thread:   gh.ReviewThread{Line: 3, DiffSide: gh.DiffSideLeft},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesThread(tt.dl, tt.thread)
			if result != tt.expected {
				t.Errorf("matchesThread() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func testMultiHunkDiffLines() []diff.DiffLine {
	return diff.Parse(`@@ -1,3 +1,3 @@
 line1
-old2
+new2
 line3
@@ -10,3 +10,4 @@
 line10
 line11
+added12
 line12
@@ -20,2 +21,2 @@
 line20
-old21
+new21`)
}

func TestDiffView_NextHunk(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(testMultiHunkDiffLines(), nil)

	// Start at hunk header 0, next should go to second hunk
	m.NextHunk()
	if m.diffLines[m.cursor].Type != diff.LineHunkHeader {
		t.Errorf("cursor should be on hunk header, got type %d", m.diffLines[m.cursor].Type)
	}
	// Should be on second hunk header (not the first one at 0)
	if m.cursor == 0 {
		t.Error("cursor should have moved past first hunk header")
	}

	// Move to third hunk
	secondPos := m.cursor
	m.NextHunk()
	if m.cursor <= secondPos {
		t.Errorf("cursor should have moved past second hunk")
	}
	if m.diffLines[m.cursor].Type != diff.LineHunkHeader {
		t.Errorf("cursor should be on hunk header")
	}

	// No more hunks, cursor should stay
	thirdPos := m.cursor
	m.NextHunk()
	if m.cursor != thirdPos {
		t.Errorf("cursor should stay when no more hunks")
	}
}

func TestDiffView_PrevHunk(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(testMultiHunkDiffLines(), nil)

	// Move to last line
	m.cursor = len(m.diffLines) - 1
	m.PrevHunk()
	if m.diffLines[m.cursor].Type != diff.LineHunkHeader {
		t.Errorf("cursor should be on hunk header")
	}

	// Move to previous hunk
	m.PrevHunk()
	if m.diffLines[m.cursor].Type != diff.LineHunkHeader {
		t.Errorf("cursor should be on hunk header")
	}

	// Move to first hunk
	m.PrevHunk()
	if m.cursor != 0 {
		t.Errorf("cursor should be on first hunk header at 0, got %d", m.cursor)
	}

	// No more hunks before, stay
	m.PrevHunk()
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0")
	}
}

func TestDiffView_HunkPosition(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(testMultiHunkDiffLines(), nil)

	current, total := m.HunkPosition()
	if total != 3 {
		t.Errorf("expected 3 total hunks, got %d", total)
	}
	if current != 1 {
		t.Errorf("expected current=1, got %d", current)
	}

	// Move past all hunks
	m.cursor = len(m.diffLines) - 1
	current, total = m.HunkPosition()
	if current != 3 {
		t.Errorf("expected current=3 at end, got %d", current)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
}

func TestDiffView_HunkPosition_NoHunks(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 40)
	m.SetContent(nil, nil)

	current, total := m.HunkPosition()
	if current != 0 || total != 0 {
		t.Errorf("expected (0, 0) for empty diff, got (%d, %d)", current, total)
	}
}

func TestFormatTime(t *testing.T) {
	// Invalid time returns the input string
	result := formatTime("not-a-time")
	if result != "not-a-time" {
		t.Errorf("expected input string for invalid time, got %q", result)
	}
}

func TestDiffView_ThreadRows_NoWrapOverflow(t *testing.T) {
	lines := diff.Parse(`@@ -1,2 +1,2 @@
+short
 line`)
	threads := []gh.ReviewThread{
		{
			ID:         "t1",
			IsResolved: false,
			Path:       "a.go",
			Line:       1,
			DiffSide:   gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{
					ID:        "c1",
					Author:    "alice",
					CreatedAt: "2026-02-24T10:00:00Z",
					Body:      "これはとても長いコメント本文で、表示幅を超えるケースを再現するための文字列です",
				},
			},
		},
	}

	m := NewDiffViewModel()
	m.SetSize(20, 8)
	m.SetContent(lines, threads)
	m.buildDisplayRows()

	for i, row := range m.displayRows {
		if strings.Contains(row.text, "\n") {
			t.Fatalf("row %d unexpectedly contains newline: %q", i, row.text)
		}
		if w := lipgloss.Width(row.text); w > m.width {
			t.Fatalf("row %d width overflow: got %d > %d, row=%q", i, w, m.width, row.text)
		}
	}
}

func TestDiffView_WideSplitFixtureDoesNotDuplicateUnchangedReplacementLine(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "fixtures", "wide-split.json")
	fixture, err := gh.LoadFixtureData(fixturePath)
	if err != nil {
		t.Fatalf("LoadFixtureData: %v", err)
	}

	patch := fixture.DiffResult.Patches["internal/tui/split_layout.go"]
	lines := diff.Parse(patch)

	m := NewDiffViewModel()
	m.SetSize(120, 40)
	m.SetContent(lines, nil)
	m.SetMode("split")
	m.buildDisplayRows()

	count := 0
	for _, row := range m.displayRows {
		if strings.Contains(row.text, "return renderUnified(width)") {
			count++
		}
	}

	if count != 1 {
		t.Fatalf("renderUnified line count = %d, want 1", count)
	}
}

func TestDiffView_SplitDoesNotDuplicateAddedOnlyBlock(t *testing.T) {
	lines := diff.Parse(`@@ -1,1 +1,3 @@
+first added
+second added
 line`)

	m := NewDiffViewModel()
	m.SetSize(120, 20)
	m.SetContent(lines, nil)
	m.SetMode("split")
	m.buildDisplayRows()

	count := 0
	for _, row := range m.displayRows {
		if strings.Contains(row.text, "first added") {
			count++
		}
	}

	if count != 1 {
		t.Fatalf("first added row count = %d, want 1", count)
	}
}

func TestDiffView_ViewDoesNotAccumulateRowsAcrossRenders(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "fixtures", "wide-split.json")
	fixture, err := gh.LoadFixtureData(fixturePath)
	if err != nil {
		t.Fatalf("LoadFixtureData: %v", err)
	}

	patch := fixture.DiffResult.Patches["internal/tui/split_layout.go"]
	lines := diff.Parse(patch)

	m := NewDiffViewModel()
	m.SetSize(120, 40)
	m.SetContent(lines, nil)
	m.SetMode("split")

	first := m.View()
	second := m.View()

	if strings.Count(first, "keep an extra trailing line") != 1 {
		t.Fatalf("first render duplicated trailing line: %q", first)
	}
	if strings.Count(second, "keep an extra trailing line") != 1 {
		t.Fatalf("second render duplicated trailing line: %q", second)
	}
}
