package diff

import (
	"testing"
)

func TestParse_EmptyPatch(t *testing.T) {
	result := Parse("")
	if result != nil {
		t.Errorf("expected nil for empty patch, got %d lines", len(result))
	}
}

func TestParse_SingleHunk(t *testing.T) {
	patch := `@@ -10,6 +10,8 @@
 context line
-removed line
+added line
+another added line
 more context`

	lines := Parse(patch)

	expected := []struct {
		lineType   LineType
		content    string
		oldLineNum int
		newLineNum int
	}{
		{LineHunkHeader, "@@ -10,6 +10,8 @@", 0, 0},
		{LineContext, " context line", 10, 10},
		{LineRemoved, "-removed line", 11, 0},
		{LineAdded, "+added line", 0, 11},
		{LineAdded, "+another added line", 0, 12},
		{LineContext, " more context", 12, 13},
	}

	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d", len(expected), len(lines))
	}

	for i, exp := range expected {
		got := lines[i]
		if got.Type != exp.lineType {
			t.Errorf("line %d: expected type %d, got %d", i, exp.lineType, got.Type)
		}
		if got.Content != exp.content {
			t.Errorf("line %d: expected content %q, got %q", i, exp.content, got.Content)
		}
		if got.OldLineNum != exp.oldLineNum {
			t.Errorf("line %d: expected oldLineNum %d, got %d", i, exp.oldLineNum, got.OldLineNum)
		}
		if got.NewLineNum != exp.newLineNum {
			t.Errorf("line %d: expected newLineNum %d, got %d", i, exp.newLineNum, got.NewLineNum)
		}
	}
}

func TestParse_MultipleHunks(t *testing.T) {
	patch := `@@ -1,3 +1,3 @@
 line1
-old
+new
@@ -20,3 +20,4 @@
 context
+inserted
 end`

	lines := Parse(patch)

	// First hunk
	assertLine(t, lines, 0, LineHunkHeader, 0, 0)
	assertLine(t, lines, 1, LineContext, 1, 1)
	assertLine(t, lines, 2, LineRemoved, 2, 0)
	assertLine(t, lines, 3, LineAdded, 0, 2)

	// Second hunk
	assertLine(t, lines, 4, LineHunkHeader, 0, 0)
	assertLine(t, lines, 5, LineContext, 20, 20)
	assertLine(t, lines, 6, LineAdded, 0, 21)
	assertLine(t, lines, 7, LineContext, 21, 22)
}

func TestParse_ConsecutiveAdditions(t *testing.T) {
	patch := `@@ -5,2 +5,5 @@
 existing
+new1
+new2
+new3
 existing2`

	lines := Parse(patch)

	assertLine(t, lines, 0, LineHunkHeader, 0, 0)
	assertLine(t, lines, 1, LineContext, 5, 5)
	assertLine(t, lines, 2, LineAdded, 0, 6)
	assertLine(t, lines, 3, LineAdded, 0, 7)
	assertLine(t, lines, 4, LineAdded, 0, 8)
	assertLine(t, lines, 5, LineContext, 6, 9)
}

func TestParse_ConsecutiveDeletions(t *testing.T) {
	patch := `@@ -10,5 +10,2 @@
 keep
-del1
-del2
-del3
 keep2`

	lines := Parse(patch)

	assertLine(t, lines, 1, LineContext, 10, 10)
	assertLine(t, lines, 2, LineRemoved, 11, 0)
	assertLine(t, lines, 3, LineRemoved, 12, 0)
	assertLine(t, lines, 4, LineRemoved, 13, 0)
	assertLine(t, lines, 5, LineContext, 14, 11)
}

func TestParse_NoNewlineAtEndOfFile(t *testing.T) {
	patch := `@@ -1,2 +1,2 @@
-old
+new
\ No newline at end of file`

	lines := Parse(patch)

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if lines[3].Content != `\ No newline at end of file` {
		t.Errorf("expected no newline marker, got %q", lines[3].Content)
	}
	if lines[3].Type != LineContext {
		t.Errorf("expected LineContext for no-newline marker, got %d", lines[3].Type)
	}
	// No newline marker should not have line numbers
	if lines[3].OldLineNum != 0 || lines[3].NewLineNum != 0 {
		t.Errorf("no-newline marker should have 0 line numbers, got old=%d new=%d",
			lines[3].OldLineNum, lines[3].NewLineNum)
	}
}

func TestParse_NewFile(t *testing.T) {
	patch := `@@ -0,0 +1,3 @@
+line1
+line2
+line3`

	lines := Parse(patch)

	assertLine(t, lines, 0, LineHunkHeader, 0, 0)
	assertLine(t, lines, 1, LineAdded, 0, 1)
	assertLine(t, lines, 2, LineAdded, 0, 2)
	assertLine(t, lines, 3, LineAdded, 0, 3)
}

func TestParse_DeletedFile(t *testing.T) {
	patch := `@@ -1,3 +0,0 @@
-line1
-line2
-line3`

	lines := Parse(patch)

	assertLine(t, lines, 0, LineHunkHeader, 0, 0)
	assertLine(t, lines, 1, LineRemoved, 1, 0)
	assertLine(t, lines, 2, LineRemoved, 2, 0)
	assertLine(t, lines, 3, LineRemoved, 3, 0)
}

func TestParse_HunkHeaderWithFunctionName(t *testing.T) {
	patch := `@@ -10,6 +10,7 @@ func main() {
 existing`

	lines := Parse(patch)

	assertLine(t, lines, 0, LineHunkHeader, 0, 0)
	if lines[0].Content != "@@ -10,6 +10,7 @@ func main() {" {
		t.Errorf("expected full hunk header content, got %q", lines[0].Content)
	}
	assertLine(t, lines, 1, LineContext, 10, 10)
}

func TestParse_HunkHeaderNoComma(t *testing.T) {
	// Single-line hunks omit the count: @@ -1 +1 @@
	patch := `@@ -1 +1 @@
-old
+new`

	lines := Parse(patch)

	assertLine(t, lines, 0, LineHunkHeader, 0, 0)
	assertLine(t, lines, 1, LineRemoved, 1, 0)
	assertLine(t, lines, 2, LineAdded, 0, 1)
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		input    string
		oldStart int
		newStart int
	}{
		{"@@ -10,6 +10,8 @@", 10, 10},
		{"@@ -1,3 +1,3 @@", 1, 1},
		{"@@ -0,0 +1,5 @@", 1, 1}, // -0 falls back to 1
		{"@@ -100,20 +105,25 @@ func foo()", 100, 105},
		{"@@ -1 +1 @@", 1, 1},
		{"invalid", 1, 1},
	}

	for _, tt := range tests {
		old, new := parseHunkHeader(tt.input)
		if old != tt.oldStart || new != tt.newStart {
			t.Errorf("parseHunkHeader(%q) = (%d, %d), want (%d, %d)",
				tt.input, old, new, tt.oldStart, tt.newStart)
		}
	}
}

func TestRenderLine_Types(t *testing.T) {
	// Just verify no panics and non-empty output for each type
	types := []struct {
		dl DiffLine
	}{
		{DiffLine{Content: " context", Type: LineContext, OldLineNum: 1, NewLineNum: 1}},
		{DiffLine{Content: "+added", Type: LineAdded, NewLineNum: 5}},
		{DiffLine{Content: "-removed", Type: LineRemoved, OldLineNum: 3}},
		{DiffLine{Content: "@@ -1,3 +1,3 @@", Type: LineHunkHeader}},
	}

	for _, tt := range types {
		result := RenderLine(tt.dl, 80, false)
		if result == "" {
			t.Errorf("RenderLine(%v) returned empty string", tt.dl)
		}

		highlighted := RenderLine(tt.dl, 80, true)
		if highlighted == "" {
			t.Errorf("RenderLine(%v, highlighted) returned empty string", tt.dl)
		}
	}
}

func TestRenderLine_Truncation(t *testing.T) {
	longContent := "+This is a very long line that should be truncated because it exceeds the maximum width"
	dl := DiffLine{Content: longContent, Type: LineAdded, NewLineNum: 1}

	result := RenderLine(dl, 30, false)
	if result == "" {
		t.Error("RenderLine returned empty for long content")
	}
}

func assertLine(t *testing.T, lines []DiffLine, idx int, expectedType LineType, expectedOld, expectedNew int) {
	t.Helper()
	if idx >= len(lines) {
		t.Fatalf("index %d out of range (len=%d)", idx, len(lines))
	}
	got := lines[idx]
	if got.Type != expectedType {
		t.Errorf("line %d: type = %d, want %d", idx, got.Type, expectedType)
	}
	if got.OldLineNum != expectedOld {
		t.Errorf("line %d: oldLineNum = %d, want %d", idx, got.OldLineNum, expectedOld)
	}
	if got.NewLineNum != expectedNew {
		t.Errorf("line %d: newLineNum = %d, want %d", idx, got.NewLineNum, expectedNew)
	}
}
