package renderer

import (
	"github.com/charmbracelet/lipgloss"
	"solituire/theme"
)

const footerText = " ←/→: move  Enter: select  Space: draw  u: undo  ?: hint  t: theme  F1: help  q: quit "

// renderFooter builds the single-line footer bar with keybinding hints.
// The hint text is rune-truncated to termWidth before styling so lipgloss
// never has a reason to wrap it into a second line.
func renderFooter(termWidth int, t theme.Theme) string {
	text := footerText
	if runes := []rune(text); len(runes) > termWidth {
		text = string(runes[:termWidth])
	}

	style := lipgloss.NewStyle().
		Background(t.FooterBackground).
		Foreground(t.FooterForeground).
		Width(termWidth)

	return style.Render(text)
}
