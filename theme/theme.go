package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines all color values used by the renderer.
type Theme struct {
	Name string

	// Card colors
	CardBackground lipgloss.Color
	CardBorder     lipgloss.Color
	CardFaceDown   lipgloss.Color
	RedSuit        lipgloss.Color // Hearts, Diamonds
	BlackSuit      lipgloss.Color // Spades, Clubs

	// UI chrome
	HeaderBackground lipgloss.Color
	HeaderForeground lipgloss.Color
	FooterBackground lipgloss.Color
	FooterForeground lipgloss.Color
	BoardBackground  lipgloss.Color

	// Interactive states
	CursorBorder   lipgloss.Color
	SelectedBorder lipgloss.Color
	HintBorder     lipgloss.Color

	// Empty slot
	EmptySlotBorder lipgloss.Color
	EmptySlotText   lipgloss.Color
}
