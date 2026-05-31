package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"solituire/renderer"
)

var keybinds = [][2]string{
	{"← →", "Move between piles"},
	{"↑ ↓", "Move within column"},
	{"1–7", "Jump to column"},
	{"Tab", "Cycle pile"},
	{"Enter", "Pick up / place"},
	{"Space", "Draw from stock"},
	{"f", "Move to foundation"},
	{"h", "Show hint"},
	{"u", "Undo"},
	{"r", "Redo"},
	{"t", "Cycle theme"},
	{"p", "Pause / resume"},
	{"?", "Toggle this help"},
	{"Esc", "Cancel / dismiss"},
	{"q", "Quit"},
}

// RenderKeybindHelp overlays a keybind popup centered over the board content
// area (within BoardWidth columns, between header and footer rows).
func RenderKeybindHelp(boardView string, termW, termH int) string {
	keyStyle := lipgloss.NewStyle().Width(7).Bold(true)
	descStyle := lipgloss.NewStyle().Width(22)

	rows := make([]string, len(keybinds))
	for i, kb := range keybinds {
		rows[i] = keyStyle.Render(kb[0]) + descStyle.Render(kb[1])
	}
	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	popup := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(0, 1).
		Render(content)

	popupW := lipgloss.Width(strings.SplitN(popup, "\n", 2)[0])
	popupH := strings.Count(popup, "\n") + 1

	// Center horizontally within the board content width (BoardWidth = 55).
	startCol := (renderer.BoardWidth - popupW) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Center vertically within the visible terminal height, between header
	// (rows 0-1) and footer (last 2 rows). Using termH rather than the board's
	// rendered line count prevents the popup from extending past the screen when
	// the tableau is taller than the terminal.
	const topPad = 2  // header line + blank spacer
	const botPad = 2  // blank spacer + footer line
	playH := termH - topPad - botPad
	startRow := topPad + (playH-popupH)/2
	if startRow < topPad {
		startRow = topPad
	}

	return renderer.Overlay(boardView, popup, startRow, startCol, termW)
}
