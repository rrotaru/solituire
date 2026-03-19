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
	borderStyle := lipgloss.NewStyle().Foreground(t.EmptySlotBorder)
	textStyle := lipgloss.NewStyle().Foreground(t.EmptySlotText)
	_ = textStyle

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
	suitStyle := lipgloss.NewStyle().Foreground(suitColor)
	bgStyle := lipgloss.NewStyle().Background(t.CardBackground)

	// rank strings are 1-2 chars; pad to 2 for alignment
	rankPad := fmt.Sprintf("%-2s", rank)  // left-padded rank (top-left)
	rankPadR := fmt.Sprintf("%2s", rank)  // right-padded rank (bottom-right)

	// Inner content lines (innerWidth = 7 chars each):
	// line0: "K  ♠   " — rank left, suit right
	// line1-3: blank
	// line4: "   ♠  K" — suit left, rank right
	line0 := rankPad + strings.Repeat(" ", innerWidth-2-1) + suitStyle.Render(suit) // 2 + spaces + 1 suit = 7 - but suit may be multi-byte
	line4 := suitStyle.Render(suit) + strings.Repeat(" ", innerWidth-1-2) + rankPadR
	blank := strings.Repeat(" ", innerWidth)

	top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	r0 := borderStyle.Render("│") + bgStyle.Render(line0) + borderStyle.Render("│")
	r1 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
	r2 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
	r3 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
	r4 := borderStyle.Render("│") + bgStyle.Render(line4) + borderStyle.Render("│")
	bot := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	return strings.Join([]string{top, r0, r1, r2, r3, r4, bot}, "\n")
}

// cardStubTop renders only the top border line of a face-down card (1 row).
// Used for fanned face-down cards in tableau columns.
func cardStubTop(t theme.Theme) string {
	borderStyle := lipgloss.NewStyle().Foreground(t.CardBorder)
	fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown)
	return borderStyle.Render("┌") + fillStyle.Render(strings.Repeat("░", innerWidth)) + borderStyle.Render("┐")
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
	suitStyle := lipgloss.NewStyle().Foreground(suitColor)
	bgStyle := lipgloss.NewStyle().Background(t.CardBackground)

	rankPad := fmt.Sprintf("%-2s", rank)
	line0 := rankPad + strings.Repeat(" ", innerWidth-2-1) + suitStyle.Render(suit)

	top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	r0 := borderStyle.Render("│") + bgStyle.Render(line0) + borderStyle.Render("│")

	return strings.Join([]string{top, r0}, "\n")
}
