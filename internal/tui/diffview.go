package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/yukikotani231/gh-pr-review/internal/diff"
	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

var (
	commentBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	commentAuthorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("3")).Bold(true)

	commentBodyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	resolvedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).Bold(true)

	unresolvedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))

	threadHighlightStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236"))
)

type displayRow struct {
	text        string
	diffLineIdx int // -1 for comment/thread rows
	threadIdx   int // index into threads for this file, -1 if not a thread row
}

type DiffViewModel struct {
	diffLines []diff.DiffLine
	threads   []gh.ReviewThread // threads for the current file

	cursor  int // index into diffLines
	scrollY int // index into displayRows for scroll offset
	height  int
	width   int

	displayRows   []displayRow
	lineToFirstRow map[int]int // diffLine index -> first displayRow index
	threadCursor   int         // which thread the cursor is on (-1 for none)
}

func NewDiffViewModel() DiffViewModel {
	return DiffViewModel{
		threadCursor: -1,
	}
}

func (m *DiffViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *DiffViewModel) SetContent(lines []diff.DiffLine, threads []gh.ReviewThread) {
	m.diffLines = lines
	m.threads = threads
	m.cursor = 0
	m.scrollY = 0
	m.threadCursor = -1
	m.buildDisplayRows()
}

func (m *DiffViewModel) CursorLine() *diff.DiffLine {
	if m.cursor < 0 || m.cursor >= len(m.diffLines) {
		return nil
	}
	return &m.diffLines[m.cursor]
}

func (m *DiffViewModel) CursorThread() *gh.ReviewThread {
	if m.threadCursor < 0 || m.threadCursor >= len(m.threads) {
		return nil
	}
	return &m.threads[m.threadCursor]
}

func (m *DiffViewModel) MoveUp() {
	m.threadCursor = -1
	if m.cursor > 0 {
		m.cursor--
		m.ensureVisible()
	}
}

func (m *DiffViewModel) MoveDown() {
	m.threadCursor = -1
	if m.cursor < len(m.diffLines)-1 {
		m.cursor++
		m.ensureVisible()
	}
}

func (m *DiffViewModel) HalfPageUp() {
	m.threadCursor = -1
	m.cursor -= m.height / 2
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
}

func (m *DiffViewModel) HalfPageDown() {
	m.threadCursor = -1
	m.cursor += m.height / 2
	if m.cursor >= len(m.diffLines) {
		m.cursor = len(m.diffLines) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
}

// NextThread moves cursor to the next thread
func (m *DiffViewModel) NextThread() {
	if len(m.threads) == 0 {
		return
	}
	m.threadCursor++
	if m.threadCursor >= len(m.threads) {
		m.threadCursor = 0
	}
	// Move diff cursor to the thread's line
	t := m.threads[m.threadCursor]
	for i, dl := range m.diffLines {
		if matchesThread(dl, t) {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

// PrevThread moves cursor to the previous thread
func (m *DiffViewModel) PrevThread() {
	if len(m.threads) == 0 {
		return
	}
	m.threadCursor--
	if m.threadCursor < 0 {
		m.threadCursor = len(m.threads) - 1
	}
	t := m.threads[m.threadCursor]
	for i, dl := range m.diffLines {
		if matchesThread(dl, t) {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

func matchesThread(dl diff.DiffLine, t gh.ReviewThread) bool {
	if t.DiffSide == gh.DiffSideLeft {
		return dl.OldLineNum == t.Line
	}
	return dl.NewLineNum == t.Line
}

func (m *DiffViewModel) ensureVisible() {
	row, ok := m.lineToFirstRow[m.cursor]
	if !ok {
		return
	}
	if row < m.scrollY {
		m.scrollY = row
	} else if row >= m.scrollY+m.height {
		m.scrollY = row - m.height + 1
	}
}

func (m *DiffViewModel) ScrollUp() {
	if m.scrollY > 0 {
		m.scrollY--
	}
}

func (m *DiffViewModel) ScrollDown() {
	maxScroll := len(m.displayRows) - m.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY < maxScroll {
		m.scrollY++
	}
}

func (m *DiffViewModel) buildDisplayRows() {
	m.displayRows = nil
	m.lineToFirstRow = make(map[int]int)

	// Build thread lookup: which threads attach to which diff line index
	threadsByLine := m.buildThreadLookup()

	for i, dl := range m.diffLines {
		m.lineToFirstRow[i] = len(m.displayRows)
		rendered := diff.RenderLine(dl, m.width, i == m.cursor)
		m.displayRows = append(m.displayRows, displayRow{
			text:        rendered,
			diffLineIdx: i,
			threadIdx:   -1,
		})

		// Add comment threads after this line
		if threads, ok := threadsByLine[i]; ok {
			for _, tidx := range threads {
				t := m.threads[tidx]
				commentRows := m.renderThread(t, tidx)
				m.displayRows = append(m.displayRows, commentRows...)
			}
		}
	}
}

func (m *DiffViewModel) buildThreadLookup() map[int][]int {
	lookup := make(map[int][]int)
	for tidx, t := range m.threads {
		for i, dl := range m.diffLines {
			if matchesThread(dl, t) {
				lookup[i] = append(lookup[i], tidx)
				break
			}
		}
	}
	return lookup
}

func (m *DiffViewModel) renderThread(t gh.ReviewThread, tidx int) []displayRow {
	var rows []displayRow
	indent := "     "
	border := commentBorderStyle.Render

	isHighlighted := m.threadCursor == tidx

	// Thread status
	var statusLabel string
	if t.IsResolved {
		statusLabel = resolvedStyle.Render(" [resolved]")
	} else {
		statusLabel = unresolvedStyle.Render(" [unresolved]")
	}

	topBorder := border(indent + "┌──") + statusLabel
	if isHighlighted {
		topBorder = threadHighlightStyle.Render(topBorder)
	}
	rows = append(rows, displayRow{text: topBorder, diffLineIdx: -1, threadIdx: tidx})

	for i, c := range t.Comments {
		var prefix string
		if i > 0 {
			prefix = border(indent + "├── ")
		} else {
			prefix = border(indent + "│ ")
		}

		timeStr := formatTime(c.CreatedAt)
		authorLine := prefix + commentAuthorStyle.Render("@"+c.Author) + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(timeStr)
		if isHighlighted {
			authorLine = threadHighlightStyle.Render(authorLine)
		}
		rows = append(rows, displayRow{text: authorLine, diffLineIdx: -1, threadIdx: tidx})

		// Body lines
		bodyLines := strings.Split(strings.TrimRight(c.Body, "\n"), "\n")
		for _, bl := range bodyLines {
			bodyRow := border(indent+"│ ") + commentBodyStyle.Render(bl)
			if isHighlighted {
				bodyRow = threadHighlightStyle.Render(bodyRow)
			}
			rows = append(rows, displayRow{text: bodyRow, diffLineIdx: -1, threadIdx: tidx})
		}
	}

	bottomBorder := border(indent + "└──")
	if isHighlighted {
		bottomBorder = threadHighlightStyle.Render(bottomBorder)
	}
	rows = append(rows, displayRow{text: bottomBorder, diffLineIdx: -1, threadIdx: tidx})

	return rows
}

func (m *DiffViewModel) View() string {
	if len(m.diffLines) == 0 {
		return "(no diff available)"
	}

	// Rebuild display rows to reflect current cursor position
	m.buildDisplayRows()

	var sb strings.Builder
	end := m.scrollY + m.height
	if end > len(m.displayRows) {
		end = len(m.displayRows)
	}

	for i := m.scrollY; i < end; i++ {
		sb.WriteString(m.displayRows[i].text)
		if i < end-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func formatTime(isoTime string) string {
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
