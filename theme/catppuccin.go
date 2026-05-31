package theme

import "github.com/charmbracelet/lipgloss"

// Catppuccin is the Catppuccin Mocha (dark) flavor.
// Palette: https://github.com/catppuccin/catppuccin
var Catppuccin = Theme{
	Name: "Catppuccin",

	CardBackground: lipgloss.Color("#cdd6f4"), // Text
	CardForeground: lipgloss.Color("#11111b"), // Crust
	CardBorder:     lipgloss.Color("#585b70"), // Surface2
	CardFaceDown:   lipgloss.Color("#89b4fa"), // Blue — 7.83:1 vs Base board
	RedSuit:        lipgloss.Color("#f38ba8"), // Red
	BlackSuit:      lipgloss.Color("#11111b"), // Crust

	HeaderBackground: lipgloss.Color("#181825"), // Mantle
	HeaderForeground: lipgloss.Color("#cdd6f4"), // Text
	FooterBackground: lipgloss.Color("#181825"), // Mantle
	FooterForeground: lipgloss.Color("#cdd6f4"), // Text
	BoardBackground:  lipgloss.Color("#1e1e2e"), // Base

	CursorBorder:   lipgloss.Color("#f9e2af"), // Yellow
	SelectedBorder: lipgloss.Color("#fab387"), // Peach
	HintBorder:     lipgloss.Color("#a6e3a1"), // Green

	EmptySlotBorder: lipgloss.Color("#45475a"), // Surface1
	EmptySlotText:   lipgloss.Color("#6c7086"), // Overlay0
}
