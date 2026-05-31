package theme

import "github.com/charmbracelet/lipgloss"

// TokyoNight is the Tokyo Night (Night variant) color scheme.
// Palette: https://github.com/enkia/tokyo-night-vscode-theme
var TokyoNight = Theme{
	Name: "Tokyo Night",

	CardBackground: lipgloss.Color("#c0caf5"), // fg
	CardForeground: lipgloss.Color("#1a1b26"), // bg
	CardBorder:     lipgloss.Color("#565f89"), // comment
	CardFaceDown:   lipgloss.Color("#7aa2f7"), // blue — 6.79:1 vs bg board
	RedSuit:        lipgloss.Color("#f7768e"), // red
	BlackSuit:      lipgloss.Color("#1a1b26"), // bg

	HeaderBackground: lipgloss.Color("#16161e"), // bg_dark
	HeaderForeground: lipgloss.Color("#c0caf5"), // fg
	FooterBackground: lipgloss.Color("#16161e"), // bg_dark
	FooterForeground: lipgloss.Color("#c0caf5"), // fg
	BoardBackground:  lipgloss.Color("#1a1b26"), // bg

	CursorBorder:   lipgloss.Color("#e0af68"), // yellow
	SelectedBorder: lipgloss.Color("#ff9e64"), // orange
	HintBorder:     lipgloss.Color("#9ece6a"), // green

	EmptySlotBorder: lipgloss.Color("#3b4261"), // fg_gutter
	EmptySlotText:   lipgloss.Color("#545c7e"), // dark3
}
