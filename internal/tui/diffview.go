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

	splitAddedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2"))

	splitRemovedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("1"))

	splitHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).Bold(true)

	splitLineNumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

type displayRow struct {
	text        string
	diffLineIdx int // -1 for comment/thread rows
	threadIdx   int // index into threads for this file, -1 if not a thread row
}

type diffMode int

const (
	diffModeUnified diffMode = iota
	diffModeSplit
)

const minSplitDiffWidth = 60

type DiffViewModel struct {
	diffLines []diff.DiffLine
	threads   []gh.ReviewThread // threads for the current file

	cursor  int // index into diffLines
	scrollY int // index into displayRows for scroll offset
	height  int
	width   int

	displayRows    []displayRow
	lineToFirstRow map[int]int // diffLine index -> first displayRow index
	threadCursor   int         // which thread the cursor is on (-1 for none)
	mode           diffMode
}

func NewDiffViewModel() DiffViewModel {
	return DiffViewModel{
		threadCursor: -1,
		mode:         diffModeUnified,
	}
}

func (m *DiffViewModel) SetSize(width, height int) {
	m.width = max(1, width)
	m.height = max(1, height)
}

func (m *DiffViewModel) SetContent(lines []diff.DiffLine, threads []gh.ReviewThread) {
	m.diffLines = lines
	m.threads = threads
	m.cursor = 0
	m.scrollY = 0
	m.threadCursor = -1
	m.buildDisplayRows()
}

func (m *DiffViewModel) ToggleMode() {
	if m.mode == diffModeUnified {
		m.mode = diffModeSplit
	} else {
		m.mode = diffModeUnified
	}
	m.buildDisplayRows()
}

func (m *DiffViewModel) SetMode(mode string) {
	switch mode {
	case "split":
		m.mode = diffModeSplit
	default:
		m.mode = diffModeUnified
	}
	m.buildDisplayRows()
}

func (m *DiffViewModel) Mode() diffMode {
	return m.mode
}

func (m *DiffViewModel) ModeString() string {
	if m.mode == diffModeSplit {
		return "split"
	}
	return "unified"
}

func (m *DiffViewModel) ModeLabel() string {
	if m.mode == diffModeSplit && m.CanRenderSplit() {
		return "[split]"
	}
	if m.mode == diffModeSplit {
		return "[split->unified]"
	}
	return "[unified]"
}

func (m *DiffViewModel) CanRenderSplit() bool {
	return m.width >= minSplitDiffWidth
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

// NextHunk moves cursor to the next hunk header.
func (m *DiffViewModel) NextHunk() {
	for i := m.cursor + 1; i < len(m.diffLines); i++ {
		if m.diffLines[i].Type == diff.LineHunkHeader {
			m.cursor = i
			m.threadCursor = -1
			m.ensureVisible()
			return
		}
	}
}

// PrevHunk moves cursor to the previous hunk header.
func (m *DiffViewModel) PrevHunk() {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.diffLines[i].Type == diff.LineHunkHeader {
			m.cursor = i
			m.threadCursor = -1
			m.ensureVisible()
			return
		}
	}
}

// HunkPosition returns (current, total) where current is 1-based index
// of the hunk containing the cursor line. Returns (0, total) if cursor
// is before the first hunk.
func (m *DiffViewModel) HunkPosition() (int, int) {
	current := 0
	total := 0
	for i, dl := range m.diffLines {
		if dl.Type == diff.LineHunkHeader {
			total++
			if i <= m.cursor {
				current = total
			}
		}
	}
	return current, total
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

	threadsByLine := m.buildThreadLookup()
	if m.mode == diffModeSplit && m.CanRenderSplit() {
		m.buildSplitDisplayRows(threadsByLine)
		return
	}

	for i, dl := range m.diffLines {
		m.lineToFirstRow[i] = len(m.displayRows)
		rendered := diff.RenderLine(dl, m.width, i == m.cursor)
		rendered = m.fitRow(rendered)
		m.displayRows = append(m.displayRows, displayRow{
			text:        rendered,
			diffLineIdx: i,
			threadIdx:   -1,
		})

		if threads, ok := threadsByLine[i]; ok {
			m.appendThreadRows(threads)
		}
	}
}

func (m *DiffViewModel) buildSplitDisplayRows(threadsByLine map[int][]int) {
	for i := 0; i < len(m.diffLines); i++ {
		dl := m.diffLines[i]
		rowIdx := len(m.displayRows)
		m.lineToFirstRow[i] = rowIdx

		if dl.Type == diff.LineHunkHeader {
			rendered := diff.RenderLine(dl, m.width, i == m.cursor)
			m.displayRows = append(m.displayRows, displayRow{
				text:        m.fitRow(rendered),
				diffLineIdx: i,
				threadIdx:   -1,
			})
			if threads, ok := threadsByLine[i]; ok {
				m.appendThreadRows(threads)
			}
			continue
		}

		leftIdx := i
		rightIdx := i
		leftLine := &m.diffLines[i]
		rightLine := &m.diffLines[i]

		switch dl.Type {
		case diff.LineRemoved:
			rightLine = nil
			rightIdx = -1
			if i+1 < len(m.diffLines) && m.diffLines[i+1].Type == diff.LineAdded {
				rightIdx = i + 1
				rightLine = &m.diffLines[rightIdx]
				m.lineToFirstRow[rightIdx] = rowIdx
				i++
			}
		case diff.LineAdded:
			leftLine = nil
			leftIdx = -1
		}

		rendered := m.renderSplitRow(leftLine, rightLine, leftIdx == m.cursor || rightIdx == m.cursor)
		m.displayRows = append(m.displayRows, displayRow{
			text:        rendered,
			diffLineIdx: max(leftIdx, rightIdx),
			threadIdx:   -1,
		})

		threadSet := orderedThreadIndexes(threadsByLine[leftIdx], threadsByLine[rightIdx])
		m.appendThreadRows(threadSet)
	}
}

func (m *DiffViewModel) appendThreadRows(threadIdxs []int) {
	for _, tidx := range threadIdxs {
		t := m.threads[tidx]
		commentRows := m.renderThread(t, tidx)
		m.displayRows = append(m.displayRows, commentRows...)
	}
}

func orderedThreadIndexes(left, right []int) []int {
	if len(left) == 0 {
		return append([]int(nil), right...)
	}
	result := append([]int(nil), left...)
	seen := make(map[int]struct{}, len(left))
	for _, idx := range left {
		seen[idx] = struct{}{}
	}
	for _, idx := range right {
		if _, ok := seen[idx]; ok {
			continue
		}
		result = append(result, idx)
	}
	return result
}

func (m *DiffViewModel) renderSplitRow(left, right *diff.DiffLine, highlighted bool) string {
	leftWidth := max(1, (m.width-3)/2)
	rightWidth := max(1, m.width-leftWidth-3)
	leftRendered := m.renderSplitCell(left, leftWidth, true)
	rightRendered := m.renderSplitCell(right, rightWidth, false)
	row := lipgloss.JoinHorizontal(lipgloss.Top, leftRendered, " | ", rightRendered)
	if highlighted {
		row = lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(row)
	}
	return m.fitRow(row)
}

func (m *DiffViewModel) renderSplitCell(dl *diff.DiffLine, width int, isLeft bool) string {
	if width < 1 {
		width = 1
	}
	if dl == nil {
		return lipgloss.NewStyle().Width(width).Render("")
	}
	if dl.Type == diff.LineHunkHeader {
		return lipgloss.NewStyle().Width(width).Render(splitHunkStyle.Render(dl.Content))
	}

	var lineNum string
	switch {
	case isLeft:
		if dl.OldLineNum > 0 {
			lineNum = fmt.Sprintf("%4d", dl.OldLineNum)
		} else {
			lineNum = "    "
		}
	case dl.NewLineNum > 0:
		lineNum = fmt.Sprintf("%4d", dl.NewLineNum)
	default:
		lineNum = "    "
	}

	contentWidth := max(1, width-5)
	content := truncateDisplay(dl.Content, contentWidth)
	contentStyle := lipgloss.NewStyle()
	switch dl.Type {
	case diff.LineAdded:
		contentStyle = splitAddedStyle
	case diff.LineRemoved:
		contentStyle = splitRemovedStyle
	}

	cell := splitLineNumStyle.Render(lineNum) + " " + contentStyle.Render(content)
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(cell)
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

	topBorder := border(indent+"┌──") + statusLabel
	if isHighlighted {
		topBorder = threadHighlightStyle.Render(topBorder)
	}
	topBorder = m.fitRow(topBorder)
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
		authorLine = m.fitRow(authorLine)
		rows = append(rows, displayRow{text: authorLine, diffLineIdx: -1, threadIdx: tidx})

		// Body lines
		bodyLines := strings.Split(strings.TrimRight(c.Body, "\n"), "\n")
		for _, bl := range bodyLines {
			bodyRow := border(indent+"│ ") + commentBodyStyle.Render(bl)
			if isHighlighted {
				bodyRow = threadHighlightStyle.Render(bodyRow)
			}
			bodyRow = m.fitRow(bodyRow)
			rows = append(rows, displayRow{text: bodyRow, diffLineIdx: -1, threadIdx: tidx})
		}
	}

	bottomBorder := border(indent + "└──")
	if isHighlighted {
		bottomBorder = threadHighlightStyle.Render(bottomBorder)
	}
	bottomBorder = m.fitRow(bottomBorder)
	rows = append(rows, displayRow{text: bottomBorder, diffLineIdx: -1, threadIdx: tidx})

	return rows
}

func (m *DiffViewModel) fitRow(s string) string {
	return lipgloss.NewStyle().MaxWidth(max(1, m.width)).Render(s)
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

// timeNow is used instead of time.Now so tests can inject a fixed time.
var timeNow = time.Now

func formatTime(isoTime string) string {
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime
	}
	d := timeNow().Sub(t)
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
