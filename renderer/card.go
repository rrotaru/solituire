package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"solituire/engine"
	"solituire/theme"
)

// CursorState is populated by the TUI layer and consumed by the renderer.
// It describes what the cursor is pointing at and any active drag or hint.
type CursorState struct {
	Pile          engine.PileID
	CardIndex     int  // 0-based index within the pile's cards slice
	Dragging      bool // true when a card (stack) is being dragged
	DragSource    engine.PileID
	DragCardCount int // number of cards lifted from DragSource
	MouseX        int // terminal column of the mouse cursor during drag
	MouseY        int // terminal row of the mouse cursor during drag
	HintFrom      engine.PileID
	HintTo        engine.PileID
	ShowHint      bool
}

// cardVisualState enumerates the rendering mode for a single card cell.
type cardVisualState uint8

const (
	cardNormal   cardVisualState = iota
	cardFaceDown                 // face-down: hatched pattern
	cardEmpty                    // empty pile slot
	cardCursor                   // cursor hovering over card
	cardSelected                 // card is picked up / being dragged
	cardHintFrom                 // source of a hint
	cardHintTo                   // destination of a hint
)

// cardContent describes a card or slot to render.
type cardContent struct {
	card  engine.Card
	state cardVisualState
}

// renderCard renders a single card cell as a 7×5 Lipgloss string (no borders).
//
// Full card structure (CardWidth=7, CardHeight=5):
//
//	K ♠
//
//
//
//	    ♠ K
func renderCard(cc cardContent, t theme.Theme) string {
	switch cc.state {
	case cardEmpty:
		return renderEmpty(t)
	case cardFaceDown:
		return renderFaceDown(t)
	default:
		return renderFaceUp(cc, t)
	}
}

// innerWidth is used only for empty-slot dashed borders (CardWidth minus 2 border chars).
const innerWidth = CardWidth - 2

// renderEmpty renders an empty pile slot with a dashed border.
func renderEmpty(t theme.Theme) string {
	return renderEmptyWithState(cardNormal, t)
}

// renderEmptyWithState renders an empty pile slot, tinting the border to
// reflect cursor/hint interaction state (cardCursor, cardHintTo, etc.).
func renderEmptyWithState(state cardVisualState, t theme.Theme) string {
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
	bgStyle := lipgloss.NewStyle().Background(t.BoardBackground)

	top := borderStyle.Render("╭" + strings.Repeat("╌", innerWidth) + "╮")
	mid := borderStyle.Render("│") + bgStyle.Render(strings.Repeat(" ", innerWidth)) + borderStyle.Render("│")
	bot := borderStyle.Render("╰" + strings.Repeat("╌", innerWidth) + "╯")

	lines := []string{top, mid, mid, mid, bot}
	return strings.Join(lines, "\n")
}

// renderFaceDown renders a face-down card: ▇ top row, █ fill rows.
func renderFaceDown(t theme.Theme) string {
	fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown).Background(t.CardBackground)
	top := fillStyle.Render(strings.Repeat("▇", CardWidth))
	fill := fillStyle.Render(strings.Repeat("█", CardWidth))
	lines := []string{top, fill, fill, fill, fill}
	return strings.Join(lines, "\n")
}

// renderFaceUp renders a face-up card without borders.
// cursor hover: rank and suit blink.
// hint: entire card blinks.
func renderFaceUp(cc cardContent, t theme.Theme) string {
	c := cc.card
	rank := c.Rank.String()
	suit := c.Suit.Symbol()

	var suitColor lipgloss.Color
	if c.Color() == engine.Red {
		suitColor = t.RedSuit
	} else {
		suitColor = t.BlackSuit
	}

	suitStyle := lipgloss.NewStyle().Foreground(suitColor).Background(t.CardBackground)
	rankStyle := lipgloss.NewStyle().Foreground(t.CardForeground).Background(t.CardBackground)
	bgStyle := lipgloss.NewStyle().Background(t.CardBackground)

	switch cc.state {
	case cardCursor:
		rankStyle = rankStyle.Blink(true)
		suitStyle = suitStyle.Blink(true)
	case cardHintFrom, cardHintTo:
		rankStyle = rankStyle.Blink(true)
		suitStyle = suitStyle.Blink(true)
		bgStyle = bgStyle.Blink(true)
	}

	// rank strings are 1-2 chars; pad to 2 for alignment
	rankPad := fmt.Sprintf("%-2s", rank) // left-aligned (top-left)
	rankPadR := fmt.Sprintf("%2s", rank) // right-aligned (bottom-right)

	blank := bgStyle.Render(strings.Repeat(" ", CardWidth))

	// line0: rank+suit at top-left (2+1+4 = CardWidth)
	line0 := bgStyle.Render(
		rankStyle.Inline(true).Render(rankPad) +
			suitStyle.Inline(true).Render(suit) +
			bgStyle.Inline(true).Render(strings.Repeat(" ", CardWidth-3)),
	)

	// line4: suit+rank at bottom-right (4+1+2 = CardWidth)
	line4 := bgStyle.Render(
		bgStyle.Inline(true).Render(strings.Repeat(" ", CardWidth-3)) +
			suitStyle.Inline(true).Render(suit) +
			rankStyle.Inline(true).Render(rankPadR),
	)

	return strings.Join([]string{line0, blank, blank, blank, line4}, "\n")
}

// cardStubTop renders the single visible row of a face-down card stub in the tableau.
func cardStubTop(t theme.Theme) string {
	fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown).Background(t.CardBackground)
	return fillStyle.Render(strings.Repeat("▇", CardWidth))
}

// cardPeekLines renders the single peek row of a non-bottom face-up tableau card.
// cursor hover: rank and suit blink.
// hint: entire row blinks.
func cardPeekLines(c engine.Card, state cardVisualState, t theme.Theme) string {
	rank := c.Rank.String()
	suit := c.Suit.Symbol()

	var suitColor lipgloss.Color
	if c.Color() == engine.Red {
		suitColor = t.RedSuit
	} else {
		suitColor = t.BlackSuit
	}

	suitStyle := lipgloss.NewStyle().Foreground(suitColor).Background(t.CardBackground)
	rankStyle := lipgloss.NewStyle().Foreground(t.CardForeground).Background(t.CardBackground)
	bgStyle := lipgloss.NewStyle().Background(t.CardBackground)

	switch state {
	case cardCursor:
		rankStyle = rankStyle.Blink(true)
		suitStyle = suitStyle.Blink(true)
	case cardHintFrom, cardHintTo:
		rankStyle = rankStyle.Blink(true)
		suitStyle = suitStyle.Blink(true)
		bgStyle = bgStyle.Blink(true)
	}

	rankPad := fmt.Sprintf("%-2s", rank)

	return bgStyle.Render(
		rankStyle.Inline(true).Render(rankPad) +
			suitStyle.Inline(true).Render(suit) +
			bgStyle.Inline(true).Render(strings.Repeat(" ", CardWidth-3)),
	)
}
