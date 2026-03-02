package diff

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
	LineHunkHeader
)

type DiffLine struct {
	Content    string
	Type       LineType
	OldLineNum int // 0 if not applicable (e.g., added line)
	NewLineNum int // 0 if not applicable (e.g., removed line)
}

var (
	addedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	removedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	hunkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	lineNumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func Parse(patch string) []DiffLine {
	if patch == "" {
		return nil
	}

	var result []DiffLine
	lines := strings.Split(patch, "\n")
	var oldLine, newLine int

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "@@"):
			oldStart, newStart := parseHunkHeader(line)
			oldLine = oldStart
			newLine = newStart
			result = append(result, DiffLine{
				Content: line,
				Type:    LineHunkHeader,
			})

		case strings.HasPrefix(line, "+"):
			result = append(result, DiffLine{
				Content:    line,
				Type:       LineAdded,
				NewLineNum: newLine,
			})
			newLine++

		case strings.HasPrefix(line, "-"):
			result = append(result, DiffLine{
				Content:    line,
				Type:       LineRemoved,
				OldLineNum: oldLine,
			})
			oldLine++

		default:
			if line == `\ No newline at end of file` {
				result = append(result, DiffLine{
					Content: line,
					Type:    LineContext,
				})
				continue
			}
			result = append(result, DiffLine{
				Content:    line,
				Type:       LineContext,
				OldLineNum: oldLine,
				NewLineNum: newLine,
			})
			oldLine++
			newLine++
		}
	}

	return result
}

func parseHunkHeader(line string) (oldStart, newStart int) {
	// Parse @@ -10,6 +10,8 @@
	parts := strings.SplitN(line, "@@", 3)
	if len(parts) < 2 {
		return 1, 1
	}
	ranges := strings.TrimSpace(parts[1])
	rangeParts := strings.Fields(ranges)

	for _, r := range rangeParts {
		if strings.HasPrefix(r, "-") {
			nums := strings.SplitN(r[1:], ",", 2)
			oldStart, _ = strconv.Atoi(nums[0]) // falls back to 0 on parse error
		} else if strings.HasPrefix(r, "+") {
			nums := strings.SplitN(r[1:], ",", 2)
			newStart, _ = strconv.Atoi(nums[0]) // falls back to 0 on parse error
		}
	}

	// Default to 1 for new/deleted files where the hunk header shows 0
	if oldStart == 0 {
		oldStart = 1
	}
	if newStart == 0 {
		newStart = 1
	}
	return
}

func RenderLine(dl DiffLine, width int, highlighted bool) string {
	if width < 1 {
		width = 1
	}

	// Line numbers
	var oldNum, newNum string
	if dl.OldLineNum > 0 {
		oldNum = fmt.Sprintf("%4d", dl.OldLineNum)
	} else {
		oldNum = "    "
	}
	if dl.NewLineNum > 0 {
		newNum = fmt.Sprintf("%4d", dl.NewLineNum)
	} else {
		newNum = "    "
	}

	var prefix string
	if dl.Type == LineHunkHeader {
		prefix = "        "
	} else {
		prefix = lineNumStyle.Render(oldNum) + " " + lineNumStyle.Render(newNum) + " "
	}

	content := dl.Content
	// Truncate if too wide
	maxContent := width - 12
	if maxContent < 1 {
		maxContent = 1
	}
	content = truncateDisplay(content, maxContent)

	var line string
	switch dl.Type {
	case LineAdded:
		line = prefix + addedStyle.Render(content)
	case LineRemoved:
		line = prefix + removedStyle.Render(content)
	case LineHunkHeader:
		line = hunkStyle.Render(content)
	default:
		line = prefix + content
	}

	if highlighted {
		line = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Render(line)
	}

	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

func truncateDisplay(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth == 1 {
		return "…"
	}

	var b strings.Builder
	currentWidth := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			break
		}
		rw := lipgloss.Width(string(r))
		if currentWidth+rw+1 > maxWidth {
			break
		}
		b.WriteRune(r)
		currentWidth += rw
		s = s[size:]
	}
	return b.String() + "…"
}
