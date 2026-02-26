package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

// shouldUpdate checks the -update flag (registered by teatest or ourselves).
func shouldUpdate() bool {
	f := flag.CommandLine.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

func assertGolden(t *testing.T, name, actual string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if shouldUpdate() {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file not found: %s (run with -update to create)", path)
	}
	if actual != string(expected) {
		t.Errorf("View() output differs from golden file %s\n--- got ---\n%s\n--- want ---\n%s",
			path, actual, string(expected))
	}
}

// fixedTime returns a fixed time for deterministic test output.
func fixedTime() time.Time {
	return time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)
}

func setupTimeNow(t *testing.T) {
	t.Helper()
	orig := timeNow
	timeNow = fixedTime
	t.Cleanup(func() { timeNow = orig })
}

// --- FileList golden tests ---

func TestGolden_FileList_Basic(t *testing.T) {
	m := FileListModel{}
	m.SetSize(60, 10)
	m.SetFiles(newTestFiles())

	assertGolden(t, "filelist_basic", m.View())
}

func TestGolden_FileList_CursorMoved(t *testing.T) {
	m := FileListModel{}
	m.SetSize(60, 10)
	m.SetFiles(newTestFiles())
	m.MoveDown()
	m.MoveDown()

	assertGolden(t, "filelist_cursor_moved", m.View())
}

// --- DiffView golden tests ---

func TestGolden_DiffView_Basic(t *testing.T) {
	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), nil)

	assertGolden(t, "diffview_basic", m.View())
}

func TestGolden_DiffView_WithThreads(t *testing.T) {
	setupTimeNow(t)

	threads := []gh.ReviewThread{
		{
			ID: "thread1", IsResolved: false, Path: "test.go",
			Line: 2, DiffSide: gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{ID: "c1", Body: "Fix this", Author: "alice", CreatedAt: "2026-02-26T11:00:00Z"},
			},
		},
		{
			ID: "thread2", IsResolved: true, Path: "test.go",
			Line: 5, DiffSide: gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{ID: "c2", Body: "LGTM", Author: "bob", CreatedAt: "2026-02-26T10:00:00Z"},
				{ID: "c3", Body: "Thanks!", Author: "alice", CreatedAt: "2026-02-26T10:30:00Z"},
			},
		},
	}

	m := NewDiffViewModel()
	m.SetSize(80, 30)
	m.SetContent(testDiffLines(), threads)

	assertGolden(t, "diffview_with_threads", m.View())
}

func TestGolden_DiffView_CursorOnThread(t *testing.T) {
	setupTimeNow(t)

	threads := []gh.ReviewThread{
		{
			ID: "thread1", IsResolved: false, Path: "test.go",
			Line: 2, DiffSide: gh.DiffSideRight,
			Comments: []gh.ReviewComment{
				{ID: "c1", Body: "Fix this", Author: "alice", CreatedAt: "2026-02-26T11:00:00Z"},
			},
		},
	}

	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(testDiffLines(), threads)
	m.NextThread() // threadCursor=0, highlights thread1

	assertGolden(t, "diffview_cursor_on_thread", m.View())
}

// --- Selector golden tests ---

func TestGolden_Selector_PRItems(t *testing.T) {
	setupTimeNow(t)

	items := []prItem{
		{number: 42, title: "Fix authentication bug", author: "alice", updatedAt: "2026-02-26T11:30:00Z", isDraft: false},
		{number: 99, title: "WIP: Add new feature", author: "bob", updatedAt: "2026-02-25T12:00:00Z", isDraft: true},
		{number: 101, title: "Update dependencies", author: "charlie", updatedAt: "2026-02-26T08:00:00Z", isDraft: false},
	}

	var sb strings.Builder
	for _, item := range items {
		fmt.Fprintf(&sb, "Title:       %s\n", item.Title())
		fmt.Fprintf(&sb, "Description: %s\n", item.Description())
		fmt.Fprintf(&sb, "FilterValue: %s\n", item.FilterValue())
		sb.WriteString("---\n")
	}

	assertGolden(t, "selector_pr_items", sb.String())
}

// --- Multi-hunk diff golden test ---

func TestGolden_DiffView_MultiHunk(t *testing.T) {
	lines := diff.Parse(`@@ -1,4 +1,5 @@
 package main

+import "fmt"
 func main() {
 	println("hello")
@@ -10,3 +11,4 @@
 func helper() {
 	return
+	// TODO: implement
 }`)

	m := NewDiffViewModel()
	m.SetSize(80, 20)
	m.SetContent(lines, nil)

	assertGolden(t, "diffview_multihunk", m.View())
}
