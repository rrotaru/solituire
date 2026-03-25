package renderer

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"solituire/engine"
	"solituire/theme"
)

// renderHeader builds the single-line header bar showing game stats.
func renderHeader(state *engine.GameState, termWidth int, t theme.Theme) string {
	elapsed := formatDuration(state.ElapsedTime)
	content := fmt.Sprintf(
		" Score: %-6d  Moves: %-5d  Time: %s  Seed: %-10d  Draw: %d ",
		state.Score,
		state.MoveCount,
		elapsed,
		state.Seed,
		state.DrawCount,
	)

	style := lipgloss.NewStyle().
		Background(t.HeaderBackground).
		Foreground(t.HeaderForeground).
		Width(termWidth)

	return style.Render(content)
}

// formatDuration formats a duration as mm:ss.
func formatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
