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
	Pile      engine.PileID
	CardIndex int  // 0-based index within the pile's cards slice
	Dragging  bool // true when a card (stack) is being dragged
	HintFrom  engine.PileID
	HintTo    engine.PileID
	ShowHint  bool
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

// renderCard renders a single card cell as a 9×7 Lipgloss string.
//
// Full card structure (CardWidth=9, CardHeight=7):
//
//	┌───────┐
//	│K      │
//	│  ♠    │
//	│       │
//	│    ♠  │
//	│      K│
//	└───────┘
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

// innerWidth is CardWidth minus the 2 border chars = 7.
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

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	top := borderStyle.Render("╭" + strings.Repeat("╌", innerWidth) + "╮")
	mid := borderStyle.Render("│") + strings.Repeat(" ", innerWidth) + borderStyle.Render("│")
	bot := borderStyle.Render("╰" + strings.Repeat("╌", innerWidth) + "╯")

	lines := []string{top, mid, mid, mid, mid, mid, bot}
	return strings.Join(lines, "\n")
}

// renderFaceDown renders a face-down card with hatched interior.
func renderFaceDown(t theme.Theme) string {
	borderStyle := lipgloss.NewStyle().Foreground(t.CardBorder)
	fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown)

	top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	fill := borderStyle.Render("│") + fillStyle.Render(strings.Repeat("░", innerWidth)) + borderStyle.Render("│")
	bot := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	lines := []string{top, fill, fill, fill, fill, fill, bot}
	return strings.Join(lines, "\n")
}

// renderFaceUp renders a face-up card with rank and suit, applying visual state borders.
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

	var borderColor lipgloss.Color
	switch cc.state {
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
	suitStyle := lipgloss.NewStyle().Foreground(suitColor).Background(t.CardBackground)
	rankStyle := lipgloss.NewStyle().Foreground(t.CardForeground).Background(t.CardBackground)
	bgStyle := lipgloss.NewStyle().Background(t.CardBackground)

	// rank strings are 1-2 chars; pad to 2 for alignment
	rankPad := fmt.Sprintf("%-2s", rank)  // left-padded rank (top-left)
	rankPadR := fmt.Sprintf("%2s", rank)  // right-padded rank (bottom-right)

	blank := strings.Repeat(" ", innerWidth)

	// line0: "K♠     " — rank+suit together at top-left (2+1+4 = 7)
	line0 := rankStyle.Inline(true).Render(rankPad) +
		suitStyle.Inline(true).Render(suit) +
		bgStyle.Inline(true).Render(strings.Repeat(" ", innerWidth-3))

	// line2: "   ♠   " — suit centered (3+1+3 = 7)
	line2 := strings.Repeat(" ", 3) +
		suitStyle.Inline(true).Render(suit) +
		bgStyle.Inline(true).Render(strings.Repeat(" ", 3))

	// line4: "     ♠K" — suit+rank together at bottom-right (4+1+2 = 7)
	line4 := strings.Repeat(" ", innerWidth-3) +
		suitStyle.Inline(true).Render(suit) +
		rankStyle.Inline(true).Render(rankPadR)

	// Each segment carries its own Background so that ANSI resets between
	// segments never expose the terminal-default background on the card face.
	top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	r0 := borderStyle.Render("│") + bgStyle.Render(line0) + borderStyle.Render("│")
	r1 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
	r2 := borderStyle.Render("│") + bgStyle.Render(line2) + borderStyle.Render("│")
	r3 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
	r4 := borderStyle.Render("│") + bgStyle.Render(line4) + borderStyle.Render("│")
	bot := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	return strings.Join([]string{top, r0, r1, r2, r3, r4, bot}, "\n")
}

// cardStubTop renders only the top border line of a face-down card (1 row).
// Used for fanned face-down cards in tableau columns.
func cardStubTop(t theme.Theme) string {
	borderStyle := lipgloss.NewStyle().Foreground(t.CardBorder)
	return borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
}

// cardPeekLines renders the top 2 lines of a face-up card (border + rank/suit line).
// Used for non-top fanned face-up cards in tableau columns.
func cardPeekLines(c engine.Card, state cardVisualState, t theme.Theme) string {
	rank := c.Rank.String()
	suit := c.Suit.Symbol()

	var suitColor lipgloss.Color
	if c.Color() == engine.Red {
		suitColor = t.RedSuit
	} else {
		suitColor = t.BlackSuit
	}

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
	suitStyle := lipgloss.NewStyle().Foreground(suitColor).Background(t.CardBackground)
	rankStyle := lipgloss.NewStyle().Foreground(t.CardForeground).Background(t.CardBackground)
	bgStyle := lipgloss.NewStyle().Background(t.CardBackground)

	rankPad := fmt.Sprintf("%-2s", rank)

	// line0: "K♠     " — rank+suit together at top-left (2+1+4 = 7)
	line0 := rankStyle.Inline(true).Render(rankPad) +
		suitStyle.Inline(true).Render(suit) +
		bgStyle.Inline(true).Render(strings.Repeat(" ", innerWidth-3))

	top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	r0 := borderStyle.Render("│") + bgStyle.Render(line0) + borderStyle.Render("│")

	return strings.Join([]string{top, r0}, "\n")
}
