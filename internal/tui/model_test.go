package tui

import "testing"

func TestModel_UpdateDiffModeStatusClearsNarrowWarningAfterResize(t *testing.T) {
	m := NewModel(nil, 42)
	m.statusMsg = splitTooNarrowMsg
	m.diffView.SetMode("split")
	m.diffView.SetSize(100, 20)

	m.updateDiffModeStatus()

	if m.statusMsg != "" {
		t.Fatalf("statusMsg = %q, want empty", m.statusMsg)
	}
}
