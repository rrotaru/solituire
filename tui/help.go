package tui

import "github.com/charmbracelet/lipgloss"

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

// RenderKeybindHelp overlays a keybind popup centered over boardView.
func RenderKeybindHelp(boardView string, w, h int) string {
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

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, popup,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
	)
}
