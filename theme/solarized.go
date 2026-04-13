package theme

import "github.com/charmbracelet/lipgloss"

// SolarizedDark is the Solarized dark color scheme (Base03 background).
var SolarizedDark = Theme{
	Name: "Solarized Dark",

	CardBackground: lipgloss.Color("#fdf6e3"),
	CardForeground: lipgloss.Color("#073642"),
	CardBorder:     lipgloss.Color("#586e75"),
	CardFaceDown:   lipgloss.Color("#073642"),
	RedSuit:        lipgloss.Color("#cb4b16"),
	BlackSuit:      lipgloss.Color("#002b36"),

	HeaderBackground: lipgloss.Color("#073642"),
	HeaderForeground: lipgloss.Color("#839496"),
	FooterBackground: lipgloss.Color("#073642"),
	FooterForeground: lipgloss.Color("#839496"),
	BoardBackground:  lipgloss.Color("#002b36"),

	CursorBorder:   lipgloss.Color("#b58900"),
	SelectedBorder: lipgloss.Color("#cb4b16"),
	HintBorder:     lipgloss.Color("#859900"),

	EmptySlotBorder: lipgloss.Color("#073642"),
	EmptySlotText:   lipgloss.Color("#586e75"),
}

// SolarizedLight is the Solarized light color scheme (Base3 background).
var SolarizedLight = Theme{
	Name: "Solarized Light",

	CardBackground: lipgloss.Color("#073642"),
	CardForeground: lipgloss.Color("#eee8d5"),
	CardBorder:     lipgloss.Color("#93a1a1"),
	CardFaceDown:   lipgloss.Color("#eee8d5"),
	RedSuit:        lipgloss.Color("#cb4b16"),
	BlackSuit:      lipgloss.Color("#93a1a1"), // Base1 — contrasts against Base02 card face

	HeaderBackground: lipgloss.Color("#eee8d5"),
	HeaderForeground: lipgloss.Color("#657b83"),
	FooterBackground: lipgloss.Color("#eee8d5"),
	FooterForeground: lipgloss.Color("#657b83"),
	BoardBackground:  lipgloss.Color("#fdf6e3"),

	CursorBorder:   lipgloss.Color("#b58900"),
	SelectedBorder: lipgloss.Color("#cb4b16"),
	HintBorder:     lipgloss.Color("#859900"),

	EmptySlotBorder: lipgloss.Color("#eee8d5"),
	EmptySlotText:   lipgloss.Color("#93a1a1"),
}
