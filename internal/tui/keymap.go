package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	ToggleViewed   key.Binding
	Tab            key.Binding
	Quit           key.Binding
	HalfPageUp     key.Binding
	HalfPageDown   key.Binding
	Comment        key.Binding
	Reply          key.Binding
	Resolve        key.Binding
	NextThread     key.Binding
	PrevThread     key.Binding
	SubmitReview   key.Binding
	Submit         key.Binding
	Cancel         key.Binding
	NextUnviewed   key.Binding
	PrevUnviewed   key.Binding
	NextHunk       key.Binding
	PrevHunk       key.Binding
	ToggleDiffMode key.Binding
	OpenInBrowser  key.Binding
	Help           key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		ToggleViewed: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "toggle viewed"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "switch pane"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("C-u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "half page down"),
		),
		Comment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comment"),
		),
		Reply: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reply"),
		),
		Resolve: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "resolve/unresolve"),
		),
		NextThread: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next thread"),
		),
		PrevThread: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev thread"),
		),
		SubmitReview: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "submit review"),
		),
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("C-s", "submit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "cancel"),
		),
		NextUnviewed: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next unviewed"),
		),
		PrevUnviewed: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev unviewed"),
		),
		NextHunk: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "next hunk"),
		),
		PrevHunk: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "prev hunk"),
		),
		ToggleDiffMode: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle split"),
		),
		OpenInBrowser: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open in browser"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.ToggleViewed, k.Tab, k.Comment, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.HalfPageUp, k.HalfPageDown},
		{k.Tab, k.NextUnviewed, k.PrevUnviewed, k.ToggleViewed},
		{k.NextHunk, k.PrevHunk, k.NextThread, k.PrevThread, k.ToggleDiffMode},
		{k.Comment, k.Reply, k.Resolve, k.SubmitReview},
		{k.OpenInBrowser, k.Help, k.Quit},
	}
}
