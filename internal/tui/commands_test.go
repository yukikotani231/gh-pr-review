package tui

import "testing"

func TestOpenURLCommand_NotEmptyPath(t *testing.T) {
	cmd := openURLCommand("https://example.com")
	if cmd == nil {
		t.Fatal("openURLCommand returned nil")
	}
	if cmd.Path == "" {
		t.Fatal("openURLCommand returned empty command path")
	}
}
