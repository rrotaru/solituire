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
	return renderStockFaceDown(t)
}

// renderStockFaceDown renders the stock pile face-down card as all ▇ rows,
// visually distinguishing it from tableau face-down cards (▇ top + █ fill).
// The stock card body has no cursor/selection variant — the hover arrow is
// drawn separately by RenderStockPile — so it takes no cursor state.
func renderStockFaceDown(t theme.Theme) string {
	fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown).Background(t.BoardBackground)
	row := fillStyle.Render(strings.Repeat("█", CardWidth))
	lines := []string{row, row, row, row, row}
	return strings.Join(lines, "\n")
}

// pileShowsArrow reports whether an arrow indicator is drawn for pid. It mirrors
// pileArrowColor's boolean decision exactly but needs no theme, so hit-testing
// and rendering can share it and never disagree on the layout.
func pileShowsArrow(pid engine.PileID, cursor CursorState) bool {
	if cursor.ShowHint && (pid == cursor.HintFrom || pid == cursor.HintTo) {
		return true
	}
	if cursor.Selecting {
		// During a keyboard pick-up the source stays marked and the cursor's
		// current pile shows where the cards would be placed.
		return pid == cursor.DragSource || pid == cursor.Pile
	}
	return cursor.Pile == pid && !cursor.Dragging
}

// pileArrowColor returns (color, true) when an arrow indicator should be rendered for pid.
func pileArrowColor(pid engine.PileID, cursor CursorState, t theme.Theme) (lipgloss.Color, bool) {
	if !pileShowsArrow(pid, cursor) {
		return "", false
	}
	switch {
	case cursor.ShowHint && (pid == cursor.HintFrom || pid == cursor.HintTo):
		return t.HintBorder, true
	case cursor.Selecting && pid == cursor.DragSource:
		return t.SelectedBorder, true
	default:
		return t.CursorBorder, true
	}
}

// tableauFocalFi returns the index *within the face-up slice* of the card that
// should be rendered in full with the arrow directly beneath it. fuLen is the
// number of visible face-up cards (after any drag truncation). It returns -1
// when the column has no face-up cards.
//
// By default the focal card is the bottom card, so the arrow sits at the bottom
// of the column (the historical behavior). When the keyboard cursor hovers the
// column, or a keyboard pick-up is active on it, the focal card is "lifted" to
// the selected card / top of the picked-up run instead.
func tableauFocalFi(pid engine.PileID, fdCount, fuLen int, cursor CursorState) int {
	if fuLen <= 0 {
		return -1
	}
	focal := fuLen - 1
	switch {
	case cursor.Selecting && cursor.DragSource == pid && cursor.DragCardCount > 0:
		if f := fuLen - cursor.DragCardCount; f >= 0 && f < fuLen {
			focal = f // top of the picked-up run
		}
	case !cursor.Dragging && !cursor.Selecting && cursor.Pile == pid:
		if f := cursor.CardIndex - fdCount; f >= 0 && f < fuLen {
			focal = f // card under the keyboard cursor
		}
	}
	return focal
}

// arrowRow builds a centered bold ↑ row of the given width.
func arrowRow(width int, color lipgloss.Color, bg lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	pad := (width - 1) / 2
	blankStyle := lipgloss.NewStyle().Background(bg)
	arrowStyle := lipgloss.NewStyle().Foreground(color).Background(bg).Bold(true)
	return blankStyle.Render(strings.Repeat(" ", pad)) +
		arrowStyle.Render("↑") +
		blankStyle.Render(strings.Repeat(" ", width-pad-1))
}

// appendArrow appends a centered bold ↑ row directly below s (1 row below).
// The width is derived from the first rendered line of s.
func appendArrow(s string, color lipgloss.Color, bg lipgloss.Color) string {
	firstLine := strings.Split(s, "\n")[0]
	return s + "\n" + arrowRow(lipgloss.Width(firstLine), color, bg)
}

// RenderStockPile is the exported entry point for stock rendering.
func RenderStockPile(p *engine.StockPile, cursor CursorState, t theme.Theme) string {
	s := renderStockPileFull(p, cursor, t)
	if color, ok := pileArrowColor(engine.PileStock, cursor, t); ok {
		s = appendArrow(s, color, t.BoardBackground)
	}
	return s
}

// RenderWastePile renders the waste pile.
// Draw-1: shows only the top card.
// Draw-3: shows up to 3 cards fanned (only the top is fully playable).
// During a drag from waste, the top card is omitted so the pile looks depleted.
func RenderWastePile(p *engine.WastePile, cursor CursorState, t theme.Theme) string {
	draggingFromHere := cursor.Dragging && cursor.DragSource == engine.PileWaste

	var s string
	if p.IsEmpty() || (draggingFromHere && len(p.Cards) == 1) {
		state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
		s = renderEmptyWithState(state, t)
	} else {
		visible := p.VisibleCards()
		if draggingFromHere {
			visible = visible[:len(visible)-1]
		}

		if len(visible) == 0 {
			state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
			s = renderEmptyWithState(state, t)
		} else if len(visible) == 1 {
			state := cardVisualStateForCursor(engine.PileWaste, 0, cursor)
			s = renderCard(cardContent{card: visible[0], state: resolveStateForFaceUp(state)}, t)
		} else {
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
			s = lipgloss.JoinHorizontal(lipgloss.Top, parts...)
		}
	}

	if color, ok := pileArrowColor(engine.PileWaste, cursor, t); ok {
		s = appendArrow(s, color, t.BoardBackground)
	}
	return s
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
		s := renderEmptyWithSuit(sym, state, t)
		if color, ok := pileArrowColor(pid, cursor, t); ok {
			s = appendArrow(s, color, t.BoardBackground)
		}
		return s
	}

	// Show the card at cardCount-1 (i.e. the new top after the drag lift).
	top := p.Cards[cardCount-1]
	state := cardVisualStateForCursor(pid, 0, cursor)
	s := renderCard(cardContent{card: top, state: resolveStateForFaceUp(state)}, t)
	if color, ok := pileArrowColor(pid, cursor, t); ok {
		s = appendArrow(s, color, t.BoardBackground)
	}
	return s
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
// Face-down cards show as single-line stubs; face-up cards fan with 1-line peeks
// except the focal card, which is fully rendered with the cursor arrow directly
// beneath it. The focal card defaults to the bottom card but follows the
// keyboard cursor (or the top of a keyboard pick-up), so the cards below it form
// a small stack under the arrow showing what would be picked up.
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
		s := renderEmptyWithState(state, t)
		if color, ok := pileArrowColor(pid, cursor, t); ok {
			s = appendArrow(s, color, t.BoardBackground)
		}
		return s
	}

	// When a drag has removed all face-up cards, the top face-down card is
	// newly exposed — render it as a full card so the player can see it clearly.
	topFDFull := len(fuCards) == 0 && fdCount > 0

	// The focal card renders in full with the arrow beneath it; every other
	// face-up card is a 1-line peek. Cards below the focal card therefore form a
	// small stack under the arrow, showing what would be picked up with it.
	arrowColor, showArrow := pileArrowColor(pid, cursor, t)
	focalFi := tableauFocalFi(pid, fdCount, len(fuCards), cursor)

	var rows []string

	// Face-down stubs: each is 1 row except the topmost when promoted to full.
	for i := 0; i < fdCount; i++ {
		if topFDFull && i == fdCount-1 {
			rows = append(rows, renderFaceDown(t))
		} else {
			rows = append(rows, cardStubTop(t))
		}
	}

	// Face-up cards: the focal card renders full (with the arrow directly below
	// it); the rest render as 1-line peeks.
	for fi, c := range fuCards {
		if fi == focalFi {
			state := resolveStateForFaceUp(cardVisualStateForCursor(pid, fdCount+fi, cursor))
			rows = append(rows, renderCard(cardContent{card: c, state: state}, t))
			if showArrow {
				rows = append(rows, arrowRow(CardWidth, arrowColor, t.BoardBackground))
			}
		} else {
			// Peek rows are always rendered in the card's natural colors; the
			// focal card and the arrow convey cursor/selection state.
			rows = append(rows, cardTopLine(c, t))
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
