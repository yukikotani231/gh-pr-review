package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	unfocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	viewedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	checkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2"))

	uncheckStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	addStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2"))

	delStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))

	inputLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6"))

	reviewModalStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2)

	reviewOptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15"))

	reviewSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("2"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).Bold(true)

	modifiedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3"))

	renamedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("4"))

	copiedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5"))

	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2)
)
