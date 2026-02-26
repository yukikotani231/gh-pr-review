package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

type FileListModel struct {
	files  []gh.PRFile
	cursor int
	offset int
	height int
	width  int
}

func (m *FileListModel) SetFiles(files []gh.PRFile) {
	m.files = files
	m.cursor = 0
	m.offset = 0
}

func (m *FileListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *FileListModel) SelectedFile() *gh.PRFile {
	if len(m.files) == 0 {
		return nil
	}
	return &m.files[m.cursor]
}

func (m *FileListModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
	}
}

func (m *FileListModel) MoveDown() {
	if m.cursor < len(m.files)-1 {
		m.cursor++
		if m.cursor >= m.offset+m.height {
			m.offset = m.cursor - m.height + 1
		}
	}
}

func (m *FileListModel) MoveToNextUnviewed() {
	for i := m.cursor + 1; i < len(m.files); i++ {
		if m.files[i].ViewerViewedState != gh.ViewedStateViewed {
			m.cursor = i
			m.adjustOffset()
			return
		}
	}
	for i := 0; i < m.cursor; i++ {
		if m.files[i].ViewerViewedState != gh.ViewedStateViewed {
			m.cursor = i
			m.adjustOffset()
			return
		}
	}
}

func (m *FileListModel) adjustOffset() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
}

func (m *FileListModel) UpdateViewedState(path string, state gh.ViewedState) {
	for i := range m.files {
		if m.files[i].Path == path {
			m.files[i].ViewerViewedState = state
			return
		}
	}
}

func (m *FileListModel) ViewedCount() int {
	count := 0
	for _, f := range m.files {
		if f.ViewerViewedState == gh.ViewedStateViewed {
			count++
		}
	}
	return count
}

func (m *FileListModel) View() string {
	if len(m.files) == 0 {
		return "No files"
	}

	var sb strings.Builder
	visibleEnd := m.offset + m.height
	if visibleEnd > len(m.files) {
		visibleEnd = len(m.files)
	}

	for i := m.offset; i < visibleEnd; i++ {
		f := m.files[i]

		check := uncheckStyle.Render("[ ]")
		if f.ViewerViewedState == gh.ViewedStateViewed {
			check = checkStyle.Render("[✓]")
		}

		name := filepath.Base(f.Path)
		addStr := fmt.Sprintf("+%d", f.Additions)
		delStr := fmt.Sprintf(" -%d", f.Deletions)
		statWidth := len(addStr) + len(delStr)

		// layout: "[✓] " (4) + name (maxNameLen) + " " (1) + stat (statWidth)
		maxNameLen := m.width - 5 - statWidth
		if maxNameLen < 10 {
			maxNameLen = 10
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-1] + "…"
		}

		stat := fmt.Sprintf("%s%s",
			addStyle.Render(addStr),
			delStyle.Render(delStr),
		)

		line := fmt.Sprintf("%s %-*s %s", check, maxNameLen, name, stat)

		if i == m.cursor {
			line = selectedStyle.Render(line)
		} else if f.ViewerViewedState == gh.ViewedStateViewed {
			line = viewedStyle.Render(line)
		}

		sb.WriteString(line)
		if i < visibleEnd-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
