package theme

import "github.com/charmbracelet/lipgloss"

// Classic is the green felt theme — the traditional solitaire look.
var Classic = Theme{
	Name: "Classic",

	CardBackground: lipgloss.Color("#ffffff"),
	CardForeground: lipgloss.Color("#1a1a1a"),
	CardBorder:     lipgloss.Color("#cccccc"),
	CardFaceDown:   lipgloss.Color("#1a5276"),
	RedSuit:        lipgloss.Color("#c0392b"),
	BlackSuit:      lipgloss.Color("#1a1a1a"),

	HeaderBackground: lipgloss.Color("#1e3a2a"),
	HeaderForeground: lipgloss.Color("#f0e6c8"),
	FooterBackground: lipgloss.Color("#1e3a2a"),
	FooterForeground: lipgloss.Color("#f0e6c8"),
	BoardBackground:  lipgloss.Color("#35654d"),

	CursorBorder:   lipgloss.Color("#f1c40f"),
	SelectedBorder: lipgloss.Color("#e67e22"),
	HintBorder:     lipgloss.Color("#2ecc71"),

	EmptySlotBorder: lipgloss.Color("#5d8a6a"),
	EmptySlotText:   lipgloss.Color("#5d8a6a"),
}
