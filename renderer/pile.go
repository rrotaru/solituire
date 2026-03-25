package renderer

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"solituire/engine"
	"solituire/theme"
)

// suitSymbolForIndex returns the conventional suit symbol for foundation index 0-3.
// Order: Spades, Hearts, Diamonds, Clubs.
func suitSymbolForIndex(idx int) string {
	switch idx {
	case 0:
		return "♠"
	case 1:
		return "♥"
	case 2:
		return "♦"
	case 3:
		return "♣"
	}
	return "?"
}

// cardVisualStateForCursor resolves the visual state of a card given cursor info.
func cardVisualStateForCursor(pileID engine.PileID, cardIdx int, cursor CursorState) cardVisualState {
	if cursor.ShowHint {
		if pileID == cursor.HintFrom {
			return cardHintFrom
		}
		if pileID == cursor.HintTo {
			return cardHintTo
		}
	}
	if cursor.Pile != pileID {
		return cardNormal
	}
	if cursor.CardIndex != cardIdx {
		return cardNormal
	}
	if cursor.Dragging {
		return cardSelected
	}
	return cardCursor
}

// renderStockPileFull renders the stock pile respecting cursor state on the border.
func renderStockPileFull(p *engine.StockPile, cursor CursorState, t theme.Theme) string {
	if p.IsEmpty() {
		state := cardVisualStateForCursor(engine.PileStock, 0, cursor)
		return renderEmptyWithState(state, t)
	}
	state := cardVisualStateForCursor(engine.PileStock, 0, cursor)
	return renderFaceDownWithState(state, t)
}

// renderFaceDownWithState renders a face-down card with a state-driven border color.
func renderFaceDownWithState(state cardVisualState, t theme.Theme) string {
	var borderColor lipgloss.Color
	switch state {
	case cardCursor:
		borderColor = t.CursorBorder
	case cardSelected:
		borderColor = t.SelectedBorder
	case cardHintFrom, cardHintTo:
		borderColor = t.HintBorder
	default:
		borderColor = t.CardBorder
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown)

	top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	fill := borderStyle.Render("│") + fillStyle.Render(strings.Repeat("░", innerWidth)) + borderStyle.Render("│")
	bot := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	lines := []string{top, fill, fill, fill, fill, fill, bot}
	return strings.Join(lines, "\n")
}

// RenderStockPile is the exported entry point for stock rendering.
func RenderStockPile(p *engine.StockPile, cursor CursorState, t theme.Theme) string {
	return renderStockPileFull(p, cursor, t)
}

// RenderWastePile renders the waste pile.
// Draw-1: shows only the top card.
// Draw-3: shows up to 3 cards fanned (only the top is fully playable).
func RenderWastePile(p *engine.WastePile, cursor CursorState, t theme.Theme) string {
	if p.IsEmpty() {
		state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
		return renderEmptyWithState(state, t)
	}

	visible := p.VisibleCards()
	if len(visible) == 1 {
		state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
		return renderCard(cardContent{card: visible[0], state: resolveStateForFaceUp(state)}, t)
	}

	// Draw-3: fan the cards horizontally with overlap.
	// Render each card and join them with overlap using lipgloss.JoinHorizontal.
	// For simplicity, render each as a full card side by side (Agent C can refine).
	parts := make([]string, len(visible))
	for i, c := range visible {
		var state cardVisualState
		if i == len(visible)-1 {
			state = cardVisualStateForCursor(engine.PileWaste, 0, cursor)
			state = resolveStateForFaceUp(state)
		}
		parts[i] = renderCard(cardContent{card: c, state: state}, t)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// RenderFoundationPile renders a foundation pile.
func RenderFoundationPile(p *engine.FoundationPile, idx int, cursor CursorState, t theme.Theme) string {
	pid := engine.PileID(engine.PileFoundation0 + engine.PileID(idx))
	if p.TopCard() == nil {
		state := cardVisualStateForCursor(pid, 0, cursor)
		sym := suitSymbolForIndex(idx)
		return renderEmptyWithSuit(sym, state, t)
	}
	top := p.TopCard()
	state := cardVisualStateForCursor(pid, 0, cursor)
	return renderCard(cardContent{card: *top, state: resolveStateForFaceUp(state)}, t)
}

// renderEmptyWithSuit renders an empty foundation slot with a suit symbol hint,
// tinting the border according to the active cursor/hint state.
func renderEmptyWithSuit(suit string, state cardVisualState, t theme.Theme) string {
	var borderColor lipgloss.Color
	switch state {
	case cardCursor:
		borderColor = t.CursorBorder
	case cardHintFrom, cardHintTo:
		borderColor = t.HintBorder
	default:
		borderColor = t.EmptySlotBorder
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	textStyle := lipgloss.NewStyle().Foreground(t.EmptySlotText)

	top := borderStyle.Render("╭" + strings.Repeat("╌", innerWidth) + "╮")
	blank := borderStyle.Render("│") + strings.Repeat(" ", innerWidth) + borderStyle.Render("│")
	// center the suit symbol on row 3 (middle of 5 inner rows)
	midContent := centerInWidth(textStyle.Render(suit), innerWidth)
	mid := borderStyle.Render("│") + midContent + borderStyle.Render("│")
	bot := borderStyle.Render("╰" + strings.Repeat("╌", innerWidth) + "╯")

	lines := []string{top, blank, blank, mid, blank, blank, bot}
	return strings.Join(lines, "\n")
}

// centerInWidth centers s within width w, padding with spaces.
// s is assumed to be a single rune visually (suit symbol).
func centerInWidth(s string, w int) string {
	// suit symbol is 1 visual column wide; we have w columns total
	left := (w - 1) / 2
	right := w - 1 - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// RenderTableauPile renders a tableau column as a vertical stack.
// Face-down cards show as single-line stubs; face-up cards fan with 2-line peeks
// except the bottom card which is fully rendered.
func RenderTableauPile(p *engine.TableauPile, colIdx int, cursor CursorState, t theme.Theme) string {
	pid := engine.PileID(engine.PileTableau0 + engine.PileID(colIdx))

	if p.IsEmpty() {
		state := cardVisualStateForCursor(pid, 0, cursor)
		return renderEmptyWithState(state, t)
	}

	var rows []string

	fdCount := p.FaceDownCount()
	fuCards := p.FaceUpCards()

	// Face-down stubs: each is 1 row (top border line only)
	for i := 0; i < fdCount; i++ {
		rows = append(rows, cardStubTop(t))
	}

	// Face-up cards: all but last get 2-line peek; last gets full render
	for fi, c := range fuCards {
		cardIdx := fdCount + fi
		state := cardVisualStateForCursor(pid, cardIdx, cursor)
		state = resolveStateForFaceUp(state)

		if fi < len(fuCards)-1 {
			rows = append(rows, cardPeekLines(c, state, t))
		} else {
			rows = append(rows, renderCard(cardContent{card: c, state: state}, t))
		}
	}

	return strings.Join(rows, "\n")
}

// resolveStateForFaceUp maps a raw cardVisualState to one valid for a face-up card.
// Face-up cards never render as cardFaceDown or cardEmpty.
func resolveStateForFaceUp(s cardVisualState) cardVisualState {
	switch s {
	case cardFaceDown, cardEmpty:
		return cardNormal
	}
	return s
}
