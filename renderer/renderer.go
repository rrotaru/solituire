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

	board := strings.Join([]string{
		header,
		"",
		topRow,
		"",
		tableau,
		"",
		footer,
	}, "\n")

	board = r.padBoardRight(board)

	// Overlay the drag ghost on top of the padded board so the frame-differ sees
	// a fully composed frame and clears stale ghost pixels on subsequent frames.
	if cursor.Dragging {
		ghost := r.renderGhostCard(state, cursor)
		board = applyOverlay(board, ghost, cursor.MouseY, cursor.MouseX, r.width)
	}

	return board
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

	// All top-row piles and gaps are padded to CardHeight+1 so the row height
	// never shifts when an arrow appears, and lipgloss never fills the extra
	// row with unstyled spaces.
	const topRowH = CardHeight + 1
	stock = r.padColumnHeight(stock, topRowH)
	waste = r.padColumnHeight(waste, topRowH)
	f0 = r.padColumnHeight(f0, topRowH)
	f1 = r.padColumnHeight(f1, topRowH)
	f2 = r.padColumnHeight(f2, topRowH)
	f3 = r.padColumnHeight(f3, topRowH)

	leftSection := lipgloss.JoinHorizontal(lipgloss.Top,
		stock,
		r.boardGapCol(ColGap, topRowH),
		waste,
	)

	// Position foundations at the x-offset computed from the waste visible
	// count, matching the hit-testing geometry in pileOrigins exactly.
	// computeFoundationStartX guarantees gapWidth >= 1.
	wasteVisCount := len(state.Waste.VisibleCards())
	gapWidth := computeFoundationStartX(wasteVisCount) - lipgloss.Width(leftSection)
	if gapWidth < 1 {
		gapWidth = 1
	}
	gap := r.boardGapCol(gapWidth, topRowH)
	rightSection := lipgloss.JoinHorizontal(lipgloss.Top,
		f0,
		r.boardGapCol(ColGap, topRowH),
		f1,
		r.boardGapCol(ColGap, topRowH),
		f2,
		r.boardGapCol(ColGap, topRowH),
		f3,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftSection, gap, rightSection)
}

// renderTableau renders all 7 tableau columns side by side, aligned to their tops.
func (r *Renderer) renderTableau(state *engine.GameState, cursor CursorState) string {
	cols := make([]string, 7)
	maxHeight := 0
	for i := 0; i < 7; i++ {
		cols[i] = RenderTableauPile(state.Tableau[i], i, cursor, r.theme)
		if h := strings.Count(cols[i], "\n") + 1; h > maxHeight {
			maxHeight = h
		}
	}
	// Always reserve one extra row for arrow indicators so the board height
	// is stable regardless of which column (if any) has an arrow appended.
	maxHeight++

	// Pre-pad shorter columns so JoinHorizontal doesn't add unstyled spaces
	parts := make([]string, 0, 13)
	for i, col := range cols {
		parts = append(parts, r.padColumnHeight(col, maxHeight))
		if i < 6 {
			parts = append(parts, r.boardGapCol(ColGap, maxHeight))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// boardGap returns n spaces explicitly styled with the board background color.
func (r *Renderer) boardGap(n int) string {
	return lipgloss.NewStyle().Background(r.theme.BoardBackground).Render(strings.Repeat(" ", n))
}

// boardGapCol returns a multi-line column of styled spaces (width × height).
// Use instead of boardGap when the gap is passed to JoinHorizontal alongside
// multi-line card renders, so lipgloss doesn't pad with unstyled spaces.
func (r *Renderer) boardGapCol(width, height int) string {
	line := r.boardGap(width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// padColumnHeight pads a rendered column to targetHeight with styled blank lines.
func (r *Renderer) padColumnHeight(col string, targetHeight int) string {
	lines := strings.Split(col, "\n")
	if len(lines) >= targetHeight {
		return col
	}
	w := lipgloss.Width(lines[0])
	padLine := r.boardGap(w)
	for len(lines) < targetHeight {
		lines = append(lines, padLine)
	}
	return strings.Join(lines, "\n")
}

// padBoardRight right-pads every line of board to r.width with styled spaces.
func (r *Renderer) padBoardRight(board string) string {
	bgStyle := lipgloss.NewStyle().Background(r.theme.BoardBackground)
	lines := strings.Split(board, "\n")
	for i, line := range lines {
		if w := lipgloss.Width(line); w < r.width {
			lines[i] = line + bgStyle.Render(strings.Repeat(" ", r.width-w))
		}
	}
	return strings.Join(lines, "\n")
}

// centerString centers s within width w using spaces.
func centerString(s string, w int) string {
	if len(s) >= w {
		return s
	}
	pad := (w - len(s)) / 2
	return strings.Repeat(" ", pad) + s
}
