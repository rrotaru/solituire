package renderer

import (
	"github.com/charmbracelet/lipgloss"
	"solituire/theme"
)

const footerText = " ←/→ Enter Space (u)ndo (h)int (t)heme (q)uit "

// renderFooter builds the single-line footer bar with keybinding hints.
// The hint text is rune-truncated to boardWidth before styling so lipgloss
// never has a reason to wrap it into a second line.
func renderFooter(boardWidth int, t theme.Theme) string {
	text := footerText
	if runes := []rune(text); len(runes) > boardWidth {
		text = string(runes[:boardWidth])
	}

	style := lipgloss.NewStyle().
		Background(t.FooterBackground).
		Foreground(t.FooterForeground).
		Width(boardWidth)

	return style.Render(text)
}
