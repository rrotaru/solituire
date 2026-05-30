package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"solituire/engine"
)

// ghostCardHeight is the number of lines in the compact ghost card render.
const ghostCardHeight = 3

// renderGhostCard produces a compact 3-line card representation for the drag ghost.
//
// Single-card drag:
//
//	┌───────┐
//	│K♠     │
//	└───────┘
//
// Multi-card drag (N cards):
//
//	┌───────┐
//	│K♠  (3)│
//	└───────┘
func (r *Renderer) renderGhostCard(state *engine.GameState, cursor CursorState) string {
	card, count := ghostCardInfo(state, cursor)
	if card == nil {
		return ""
	}

	borderStyle := lipgloss.NewStyle().Foreground(r.theme.SelectedBorder).Background(r.theme.BoardBackground)
	bgStyle := lipgloss.NewStyle().Background(r.theme.CardBackground)

	var suitColor lipgloss.Color
	if card.Color() == engine.Red {
		suitColor = r.theme.RedSuit
	} else {
		suitColor = r.theme.BlackSuit
	}
	suitStyle := lipgloss.NewStyle().Foreground(suitColor).Background(r.theme.CardBackground)
	rankStyle := lipgloss.NewStyle().Foreground(r.theme.CardForeground).Background(r.theme.CardBackground)

	rank := card.Rank.String()
	suit := card.Suit.Symbol()

	var inner string
	if count > 1 {
		// e.g. "K♠  (3)" — rank+suit left, count right, padded to innerWidth
		countStr := fmt.Sprintf("(%d)", count)
		mid := strings.Repeat(" ", innerWidth-len(rank)-1-len(countStr))
		inner = rankStyle.Inline(true).Render(rank) +
			suitStyle.Inline(true).Render(suit) +
			bgStyle.Inline(true).Render(mid) +
			bgStyle.Inline(true).Render(countStr)
	} else {
		// e.g. "K♠     " — rank+suit at left, spaces to fill
		pad := strings.Repeat(" ", innerWidth-len(rank)-1)
		inner = rankStyle.Inline(true).Render(rank) +
			suitStyle.Inline(true).Render(suit) +
			bgStyle.Inline(true).Render(pad)
	}

	top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	// Use inner directly (already composed of inline-styled pieces) rather than
	// wrapping in a second bgStyle.Render, which could inject block formatting.
	mid := borderStyle.Render("│") + inner + borderStyle.Render("│")
	bot := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	return strings.Join([]string{top, mid, bot}, "\n")
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

	case src >= engine.PileFoundation0 && src <= engine.PileFoundation3:
		fi := int(src - engine.PileFoundation0)
		return state.Foundations[fi].TopCard(), 1

	case src >= engine.PileTableau0 && src <= engine.PileTableau6:
		col := int(src - engine.PileTableau0)
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

// applyOverlay overlays the multi-line overlay string at (startRow, startCol)
// within base, using ANSI-aware string manipulation so existing terminal styling
// in base is preserved outside the overlay region.
func applyOverlay(base, overlay string, startRow, startCol, termWidth int) string {
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
	// Clamp against the actual board line count so the ghost never tries to
	// write past the end of the rendered board string. termHeight can exceed
	// len(baseLines) when the board content is shorter than the terminal.
	maxRow := len(baseLines) - ghostCardHeight
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
