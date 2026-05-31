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

// renderStockPileFull renders the stock pile respecting cursor state.
func renderStockPileFull(p *engine.StockPile, cursor CursorState, t theme.Theme) string {
	state := cardVisualStateForCursor(engine.PileStock, 0, cursor)
	if p.IsEmpty() {
		return renderEmptyWithState(state, t)
	}
	return renderStockFaceDown(state, t)
}

// renderStockFaceDown renders the stock pile face-down card as all ▇ rows,
// visually distinguishing it from tableau face-down cards (▇ top + █ fill).
func renderStockFaceDown(state cardVisualState, t theme.Theme) string {
	fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown).Background(t.BoardBackground)
	switch state {
	case cardCursor, cardHintFrom, cardHintTo:
		fillStyle = fillStyle.Blink(true)
	}
	row := fillStyle.Render(strings.Repeat("▇", CardWidth))
	lines := []string{row, row, row, row, row}
	return strings.Join(lines, "\n")
}

// RenderStockPile is the exported entry point for stock rendering.
func RenderStockPile(p *engine.StockPile, cursor CursorState, t theme.Theme) string {
	return renderStockPileFull(p, cursor, t)
}

// RenderWastePile renders the waste pile.
// Draw-1: shows only the top card.
// Draw-3: shows up to 3 cards fanned (only the top is fully playable).
// During a drag from waste, the top card is omitted so the pile looks depleted.
func RenderWastePile(p *engine.WastePile, cursor CursorState, t theme.Theme) string {
	draggingFromHere := cursor.Dragging && cursor.DragSource == engine.PileWaste

	if p.IsEmpty() || (draggingFromHere && len(p.Cards) == 1) {
		state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
		return renderEmptyWithState(state, t)
	}

	visible := p.VisibleCards()
	if draggingFromHere {
		// Drop the top visible card — it is shown as the ghost.
		visible = visible[:len(visible)-1]
	}

	if len(visible) == 0 {
		state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
		return renderEmptyWithState(state, t)
	}
	if len(visible) == 1 {
		state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
		return renderCard(cardContent{card: visible[0], state: resolveStateForFaceUp(state)}, t)
	}

	// Draw-3: fan the cards horizontally with overlap.
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
// During a drag from this foundation the top card is omitted so the pile looks depleted.
func RenderFoundationPile(p *engine.FoundationPile, idx int, cursor CursorState, t theme.Theme) string {
	pid := engine.PileID(engine.PileFoundation0 + engine.PileID(idx))
	draggingFromHere := cursor.Dragging && cursor.DragSource == pid

	sym := suitSymbolForIndex(idx)

	// Determine how many cards we logically have (minus any being dragged).
	cardCount := len(p.Cards)
	if draggingFromHere {
		cardCount--
	}

	if cardCount <= 0 {
		state := cardVisualStateForCursor(pid, 0, cursor)
		return renderEmptyWithSuit(sym, state, t)
	}

	// Show the card at cardCount-1 (i.e. the new top after the drag lift).
	top := p.Cards[cardCount-1]
	state := cardVisualStateForCursor(pid, 0, cursor)
	return renderCard(cardContent{card: top, state: resolveStateForFaceUp(state)}, t)
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
	borderStyle := lipgloss.NewStyle().Foreground(borderColor).Background(t.BoardBackground)
	textStyle := lipgloss.NewStyle().Foreground(t.EmptySlotText).Background(t.BoardBackground)
	bgStyle := lipgloss.NewStyle().Background(t.BoardBackground)

	top := borderStyle.Render("╭" + strings.Repeat("╌", innerWidth) + "╮")
	blank := borderStyle.Render("│") + bgStyle.Render(strings.Repeat(" ", innerWidth)) + borderStyle.Render("│")
	// center the suit symbol on the middle row
	left := (innerWidth - 1) / 2
	right := innerWidth - 1 - left
	midContent := bgStyle.Render(strings.Repeat(" ", left)) + textStyle.Render(suit) + bgStyle.Render(strings.Repeat(" ", right))
	mid := borderStyle.Render("│") + midContent + borderStyle.Render("│")
	bot := borderStyle.Render("╰" + strings.Repeat("╌", innerWidth) + "╯")

	lines := []string{top, blank, mid, blank, bot}
	return strings.Join(lines, "\n")
}

// RenderTableauPile renders a tableau column as a vertical stack.
// Face-down cards show as single-line stubs; face-up cards fan with 2-line peeks
// except the bottom card which is fully rendered.
// During a drag from this column the lifted cards are omitted so the pile looks depleted.
func RenderTableauPile(p *engine.TableauPile, colIdx int, cursor CursorState, t theme.Theme) string {
	pid := engine.PileID(engine.PileTableau0 + engine.PileID(colIdx))

	// When dragging from this column, hide the lifted cards.
	draggingFromHere := cursor.Dragging && cursor.DragSource == pid

	fdCount := p.FaceDownCount()
	fuCards := p.FaceUpCards()
	if draggingFromHere && cursor.DragCardCount > 0 && cursor.DragCardCount <= len(fuCards) {
		fuCards = fuCards[:len(fuCards)-cursor.DragCardCount]
	}

	if fdCount == 0 && len(fuCards) == 0 {
		state := cardVisualStateForCursor(pid, 0, cursor)
		return renderEmptyWithState(state, t)
	}

	// When a drag has removed all face-up cards, the top face-down card is
	// newly exposed — render it as a full card so the player can see it clearly.
	topFDFull := draggingFromHere && len(fuCards) == 0 && fdCount > 0

	var rows []string

	// Face-down stubs: each is 1 row except the topmost when promoted to full.
	for i := 0; i < fdCount; i++ {
		if topFDFull && i == fdCount-1 {
			rows = append(rows, renderFaceDown(t))
		} else {
			rows = append(rows, cardStubTop(t))
		}
	}

	// Face-up cards: all but last get 1-line peek; last gets full render
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
