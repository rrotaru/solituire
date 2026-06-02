package renderer

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"solituire/engine"
)

// ghostCardHeight is the number of lines in the ghost card render.
const ghostCardHeight = CardHeight

// renderGhostCard produces a full-height card for the drag ghost.
func (r *Renderer) renderGhostCard(state *engine.GameState, cursor CursorState) string {
	card, _ := ghostCardInfo(state, cursor)
	if card == nil {
		return ""
	}
	return renderCard(cardContent{card: *card, state: cardSelected}, r.theme)
}

// ghostCardInfo returns the card to display in the ghost and the total drag count.
// The displayed card is the top card of the drag (i.e. lowest index in the pile for
// tableau, or the only card for waste/foundation).
func ghostCardInfo(state *engine.GameState, cursor CursorState) (*engine.Card, int) {
	src := cursor.DragSource
	count := cursor.DragCardCount

	switch {
	case src == engine.PileWaste:
		return state.Waste.TopCard(), 1

	case src.IsFoundation():
		fi := src.FoundationIndex()
		return state.Foundations[fi].TopCard(), 1

	case src.IsTableau():
		col := src.TableauIndex()
		pile := state.Tableau[col]
		fuCards := pile.FaceUpCards()
		if count > len(fuCards) || count == 0 {
			return nil, 0
		}
		// Top of the drag is the card at index (len(fuCards) - count)
		top := fuCards[len(fuCards)-count]
		return &top, count
	}

	return nil, 0
}

// Overlay composites the multi-line overlay string at (startRow, startCol)
// within base, preserving existing terminal styling in base outside the overlay
// region. Exported for use by TUI layers that need to place panels over the board.
func Overlay(base, overlay string, startRow, startCol, termWidth int) string {
	return applyOverlayN(base, overlay, startRow, startCol, termWidth, 0)
}

// applyOverlay overlays the multi-line overlay string at (startRow, startCol)
// within base, using ANSI-aware string manipulation so existing terminal styling
// in base is preserved outside the overlay region.
func applyOverlay(base, overlay string, startRow, startCol, termWidth int) string {
	return applyOverlayN(base, overlay, startRow, startCol, termWidth, ghostCardHeight)
}

func applyOverlayN(base, overlay string, startRow, startCol, termWidth, clampHeight int) string {
	if overlay == "" {
		return base
	}

	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	overlayWidth := lipgloss.Width(overlayLines[0])

	// Clamp ghost so it does not escape the visible area.
	if startCol < 0 {
		startCol = 0
	}
	if startCol+overlayWidth > termWidth {
		startCol = termWidth - overlayWidth
		if startCol < 0 {
			startCol = 0
		}
	}
	if startRow < 0 {
		startRow = 0
	}
	// Clamp against the actual board line count so the overlay never tries to
	// write past the end of the rendered board string.
	clamp := clampHeight
	if clamp == 0 {
		clamp = len(overlayLines)
	}
	maxRow := len(baseLines) - clamp
	if maxRow < 0 {
		maxRow = 0
	}
	if startRow > maxRow {
		startRow = maxRow
	}

	for i, ovLine := range overlayLines {
		row := startRow + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		baseLines[row] = injectAt(baseLines[row], ovLine, startCol)
	}
	return strings.Join(baseLines, "\n")
}

// injectAt replaces the columns [col, col+width(inject)) of base with inject,
// using ANSI-aware truncation so escape sequences in base are handled correctly.
func injectAt(base, inject string, col int) string {
	injectW := lipgloss.Width(inject)
	prefix := ansi.Truncate(base, col, "")
	suffix := ansi.TruncateLeft(base, col+injectW, "")
	return prefix + inject + suffix
}
