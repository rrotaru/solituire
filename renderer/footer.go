package renderer

import (
	"github.com/charmbracelet/lipgloss"
	"solituire/theme"
)

const footerText = " ←/→: move  Enter: select  Space: draw  u: undo  ?: hint  t: theme  F1: help  q: quit "

// renderFooter builds the single-line footer bar with keybinding hints.
func renderFooter(termWidth int, t theme.Theme) string {
	style := lipgloss.NewStyle().
		Background(t.FooterBackground).
		Foreground(t.FooterForeground).
		Width(termWidth)

	return style.Render(footerText)
}
