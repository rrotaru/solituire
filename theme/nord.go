package theme

import "github.com/charmbracelet/lipgloss"

// Nord is the arctic blue Nord color palette.
var Nord = Theme{
	Name: "Nord",

	CardBackground: lipgloss.Color("#eceff4"),
	CardBorder:     lipgloss.Color("#d8dee9"),
	CardFaceDown:   lipgloss.Color("#3b4252"),
	RedSuit:        lipgloss.Color("#bf616a"),
	BlackSuit:      lipgloss.Color("#2e3440"),

	HeaderBackground: lipgloss.Color("#2e3440"),
	HeaderForeground: lipgloss.Color("#d8dee9"),
	FooterBackground: lipgloss.Color("#2e3440"),
	FooterForeground: lipgloss.Color("#d8dee9"),
	BoardBackground:  lipgloss.Color("#2e3440"),

	CursorBorder:   lipgloss.Color("#88c0d0"),
	SelectedBorder: lipgloss.Color("#81a1c1"),
	HintBorder:     lipgloss.Color("#a3be8c"),

	EmptySlotBorder: lipgloss.Color("#3b4252"),
	EmptySlotText:   lipgloss.Color("#4c566a"),
}
