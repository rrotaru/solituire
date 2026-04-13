package theme

import "github.com/charmbracelet/lipgloss"

// Dracula is the dark purple Dracula color scheme.
var Dracula = Theme{
	Name: "Dracula",

	CardBackground: lipgloss.Color("#f8f8f2"),
	CardForeground: lipgloss.Color("#282a36"),
	CardBorder:     lipgloss.Color("#6272a4"),
	CardFaceDown:   lipgloss.Color("#44475a"),
	RedSuit:        lipgloss.Color("#ff79c6"),
	BlackSuit:      lipgloss.Color("#282a36"),

	HeaderBackground: lipgloss.Color("#191a21"),
	HeaderForeground: lipgloss.Color("#f8f8f2"),
	FooterBackground: lipgloss.Color("#191a21"),
	FooterForeground: lipgloss.Color("#f8f8f2"),
	BoardBackground:  lipgloss.Color("#282a36"),

	CursorBorder:   lipgloss.Color("#bd93f9"),
	SelectedBorder: lipgloss.Color("#ff79c6"),
	HintBorder:     lipgloss.Color("#50fa7b"),

	EmptySlotBorder: lipgloss.Color("#44475a"),
	EmptySlotText:   lipgloss.Color("#44475a"),
}
