package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"solituire/config"
	"solituire/engine"
	"solituire/theme"
)

// Renderer composes the full board view from engine state and cursor info.
// It is stateless except for the active theme and terminal dimensions.
type Renderer struct {
	theme  theme.Theme
	width  int
	height int
}

// New creates a Renderer using the given theme and a default terminal size.
func New(t theme.Theme) *Renderer {
	return &Renderer{
		theme:  t,
		width:  MinTermWidth,
		height: MinTermHeight,
	}
}

// SetTheme replaces the active theme.
func (r *Renderer) SetTheme(t theme.Theme) {
	r.theme = t
}

// SetSize updates the known terminal dimensions. Call this on tea.WindowSizeMsg.
func (r *Renderer) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// Render produces the complete board string for use as the Bubbletea View().
// If the terminal is smaller than MinTermWidth × MinTermHeight, a fallback
// message is returned instead.
func (r *Renderer) Render(state *engine.GameState, cursor CursorState, cfg *config.Config) string {
	if r.width < MinTermWidth || r.height < MinTermHeight {
		return r.renderTooSmall()
	}

	header := renderHeader(state, r.width, r.theme)
	topRow := r.renderTopRow(state, cursor)
	tableau := r.renderTableau(state, cursor)
	footer := renderFooter(r.width, r.theme)

	board := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		topRow,
		"",
		tableau,
		"",
		footer,
	)

	return lipgloss.NewStyle().
		Background(r.theme.BoardBackground).
		Width(r.width).
		Render(board)
}

// renderTooSmall returns a centered "terminal too small" message.
func (r *Renderer) renderTooSmall() string {
	msg := fmt.Sprintf("Terminal too small. Need at least %d×%d. Current: %d×%d",
		MinTermWidth, MinTermHeight, r.width, r.height)

	if r.width > 0 {
		msg = centerString(msg, r.width)
	}

	style := lipgloss.NewStyle().
		Background(r.theme.BoardBackground).
		Foreground(r.theme.HeaderForeground)

	return style.Render(msg)
}

// renderTopRow renders the stock, waste, gap, and foundations on one row.
func (r *Renderer) renderTopRow(state *engine.GameState, cursor CursorState) string {
	stock := RenderStockPile(state.Stock, cursor, r.theme)
	waste := RenderWastePile(state.Waste, cursor, r.theme)

	f0 := RenderFoundationPile(state.Foundations[0], 0, cursor, r.theme)
	f1 := RenderFoundationPile(state.Foundations[1], 1, cursor, r.theme)
	f2 := RenderFoundationPile(state.Foundations[2], 2, cursor, r.theme)
	f3 := RenderFoundationPile(state.Foundations[3], 3, cursor, r.theme)

	// Gap between stock/waste and foundations: fill remaining space
	stockWasteWidth := 2*CardWidth + ColGap
	foundationsWidth := 4*CardWidth + 3*ColGap
	gapWidth := r.width - stockWasteWidth - foundationsWidth
	if gapWidth < 1 {
		gapWidth = 1
	}
	gap := strings.Repeat(" ", gapWidth)

	leftSection := lipgloss.JoinHorizontal(lipgloss.Top,
		stock,
		strings.Repeat(" ", ColGap),
		waste,
	)
	rightSection := lipgloss.JoinHorizontal(lipgloss.Top,
		f0,
		strings.Repeat(" ", ColGap),
		f1,
		strings.Repeat(" ", ColGap),
		f2,
		strings.Repeat(" ", ColGap),
		f3,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftSection, gap, rightSection)
}

// renderTableau renders all 7 tableau columns side by side, aligned to their tops.
func (r *Renderer) renderTableau(state *engine.GameState, cursor CursorState) string {
	cols := make([]string, 7)
	for i := 0; i < 7; i++ {
		cols[i] = RenderTableauPile(state.Tableau[i], i, cursor, r.theme)
	}

	// Join with gaps
	parts := make([]string, 0, 13)
	for i, col := range cols {
		parts = append(parts, col)
		if i < 6 {
			parts = append(parts, strings.Repeat(" ", ColGap))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// centerString centers s within width w using spaces.
func centerString(s string, w int) string {
	if len(s) >= w {
		return s
	}
	pad := (w - len(s)) / 2
	return strings.Repeat(" ", pad) + s
}
