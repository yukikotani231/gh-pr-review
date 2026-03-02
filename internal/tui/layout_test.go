package tui

import "testing"

func TestLayout_NarrowTerminal_ClampSizes(t *testing.T) {
	m := NewModel(nil, 1)
	m.width = 10
	m.height = 6
	m.updateLayout()

	if got := m.leftPaneWidth(); got < 1 || got >= m.width {
		t.Fatalf("leftPaneWidth out of range: %d (total=%d)", got, m.width)
	}
	if got := m.rightPaneWidth(); got < 1 {
		t.Fatalf("rightPaneWidth should be >=1, got %d", got)
	}
	if m.fileList.width < 1 {
		t.Fatalf("fileList width should be >=1, got %d", m.fileList.width)
	}
	if m.diffView.width < 1 {
		t.Fatalf("diffView width should be >=1, got %d", m.diffView.width)
	}
	if m.diffView.height < 1 {
		t.Fatalf("diffView height should be >=1, got %d", m.diffView.height)
	}
}
