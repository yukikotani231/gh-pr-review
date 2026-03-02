package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderStatusBar_NoWrapAcrossWidths(t *testing.T) {
	m := NewModel(nil, 42)
	m.focus = rightPane
	m.fileList.SetFiles(newTestFiles())
	m.diffView.SetSize(30, 8)
	m.diffView.SetContent(testDiffLines(), nil)

	for _, w := range []int{16, 20, 24, 30, 40, 60} {
		m.width = w
		out := m.renderStatusBar()

		if strings.Contains(out, "\n") {
			t.Fatalf("status bar should stay single-line at width=%d, got %q", w, out)
		}
		if got := lipgloss.Width(out); got > w {
			t.Fatalf("status bar overflow at width=%d: got %d", w, got)
		}
	}
}
