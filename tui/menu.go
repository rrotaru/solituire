package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"solituire/config"
	"solituire/theme"
)

// menuItem identifies which row of the settings menu has keyboard focus.
type menuItem int

const (
	menuItemDrawMode menuItem = iota
	menuItemTheme
	menuItemAutoMove
	menuItemStart
	menuItemCount // sentinel — total number of navigable rows
)

var (
	menuBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			Padding(1, 3)

	menuTitleStyle = lipgloss.NewStyle().Bold(true)

	menuActiveStyle = lipgloss.NewStyle().Bold(true).Reverse(true)

	menuRowStyle = lipgloss.NewStyle()
)

// MenuModel is the Bubbletea sub-model for the settings/start screen.
// It holds a local copy of the config that is mutated as the user adjusts
// settings; a ConfigChangedMsg is emitted on every change so AppModel can
// sync its own cfg pointer.
type MenuModel struct {
	cfg    config.Config        // local working copy — mutated on toggle/cycle
	themes *theme.ThemeRegistry // for Next() cycling
	cursor menuItem             // which row has keyboard focus
}

// NewMenuModel creates a MenuModel seeded from the current app config.
func NewMenuModel(cfg *config.Config, themes *theme.ThemeRegistry) MenuModel {
	return MenuModel{
		cfg:    *cfg, // copy — MenuModel owns its own copy
		themes: themes,
		cursor: menuItemDrawMode,
	}
}

// Init implements tea.Model. The menu requires no background commands.
func (m MenuModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyUp:
		m.cursor = (m.cursor - 1 + menuItemCount) % menuItemCount
		return m, nil

	case tea.KeyDown, tea.KeyTab:
		m.cursor = (m.cursor + 1) % menuItemCount
		return m, nil

	case tea.KeyShiftTab:
		m.cursor = (m.cursor - 1 + menuItemCount) % menuItemCount
		return m, nil

	case tea.KeyLeft:
		return m.applyLeft()

	case tea.KeyRight, tea.KeyEnter, tea.KeySpace:
		return m.applyRight()

	case tea.KeyCtrlN:
		return m, func() tea.Msg {
			return NewGameMsg{Seed: m.cfg.Seed, DrawCount: m.cfg.DrawCount}
		}

	case tea.KeyRunes:
		switch string(key.Runes) {
		case "j", "J":
			m.cursor = (m.cursor + 1) % menuItemCount
		case "k", "K":
			m.cursor = (m.cursor - 1 + menuItemCount) % menuItemCount
		case "h", "H":
			return m.applyLeft()
		case "l", "L":
			return m.applyRight()
		}
	}

	return m, nil
}

// applyRight activates or increments the current menu item.
func (m MenuModel) applyRight() (MenuModel, tea.Cmd) {
	switch m.cursor {
	case menuItemDrawMode:
		if m.cfg.DrawCount == 1 {
			m.cfg.DrawCount = 3
		} else {
			m.cfg.DrawCount = 1
		}
		return m, configChangedCmd(&m.cfg)

	case menuItemTheme:
		next := m.themes.Next(m.cfg.ThemeName)
		m.cfg.ThemeName = next.Name
		return m, configChangedCmd(&m.cfg)

	case menuItemAutoMove:
		m.cfg.AutoMoveEnabled = !m.cfg.AutoMoveEnabled
		return m, configChangedCmd(&m.cfg)

	case menuItemStart:
		cfg := m.cfg
		return m, func() tea.Msg {
			return NewGameMsg{Seed: cfg.Seed, DrawCount: cfg.DrawCount}
		}
	}
	return m, nil
}

// applyLeft decrements or toggles the current menu item in reverse.
func (m MenuModel) applyLeft() (MenuModel, tea.Cmd) {
	switch m.cursor {
	case menuItemDrawMode:
		// Only two options — left and right are the same toggle.
		if m.cfg.DrawCount == 1 {
			m.cfg.DrawCount = 3
		} else {
			m.cfg.DrawCount = 1
		}
		return m, configChangedCmd(&m.cfg)

	case menuItemTheme:
		prev := m.themePrev()
		m.cfg.ThemeName = prev
		return m, configChangedCmd(&m.cfg)

	case menuItemAutoMove:
		m.cfg.AutoMoveEnabled = !m.cfg.AutoMoveEnabled
		return m, configChangedCmd(&m.cfg)

	case menuItemStart:
		// Left on the button does nothing.
	}
	return m, nil
}

// themePrev returns the theme name that precedes the current theme in the
// registry cycle.
func (m MenuModel) themePrev() string {
	names := m.themes.List()
	if len(names) == 0 {
		return m.cfg.ThemeName
	}
	for i, name := range names {
		if strings.EqualFold(name, m.cfg.ThemeName) {
			return names[(i-1+len(names))%len(names)]
		}
	}
	// Current name not found — return last (same wrap-around as Next).
	return names[len(names)-1]
}

// configChangedCmd returns a Cmd that emits a ConfigChangedMsg with a fresh
// heap copy of the config so callers receive an independent pointer.
func configChangedCmd(cfg *config.Config) tea.Cmd {
	copy := *cfg
	return func() tea.Msg {
		return ConfigChangedMsg{Config: &copy}
	}
}

// View implements tea.Model.
func (m MenuModel) View() string {
	const boxWidth = 32 // inner content width (excluding border/padding)

	center := func(s string) string {
		pad := (boxWidth - lipgloss.Width(s)) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + s
	}

	var sb strings.Builder

	// Title
	sb.WriteString(center(menuTitleStyle.Render("KLONDIKE SOLITAIRE")))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Draw Mode row
	drawRow := m.renderDrawMode()
	if m.cursor == menuItemDrawMode {
		drawRow = menuActiveStyle.Render(drawRow)
	} else {
		drawRow = menuRowStyle.Render(drawRow)
	}
	sb.WriteString(drawRow)
	sb.WriteByte('\n')

	// Theme row
	themeRow := m.renderTheme()
	if m.cursor == menuItemTheme {
		themeRow = menuActiveStyle.Render(themeRow)
	} else {
		themeRow = menuRowStyle.Render(themeRow)
	}
	sb.WriteString(themeRow)
	sb.WriteByte('\n')

	// Auto-Move row
	autoRow := m.renderAutoMove()
	if m.cursor == menuItemAutoMove {
		autoRow = menuActiveStyle.Render(autoRow)
	} else {
		autoRow = menuRowStyle.Render(autoRow)
	}
	sb.WriteString(autoRow)
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Start button
	startLabel := "[ Start New Game ]"
	startRow := center(startLabel)
	if m.cursor == menuItemStart {
		startRow = center(menuActiveStyle.Render(startLabel))
	}
	sb.WriteString(startRow)
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Seed display
	seedStr := fmt.Sprintf("Seed: %d", m.cfg.Seed)
	sb.WriteString(center(seedStr))

	return menuBoxStyle.Render(sb.String())
}

func (m MenuModel) renderDrawMode() string {
	var d1, d3 string
	if m.cfg.DrawCount == 1 {
		d1 = lipgloss.NewStyle().Bold(true).Render("[ 1 ]")
		d3 = "[ 3 ]"
	} else {
		d1 = "[ 1 ]"
		d3 = lipgloss.NewStyle().Bold(true).Render("[ 3 ]")
	}
	return fmt.Sprintf("Draw Mode:   %s %s", d1, d3)
}

func (m MenuModel) renderTheme() string {
	return fmt.Sprintf("Theme:       \u25c4 %s \u25ba", m.cfg.ThemeName)
}

func (m MenuModel) renderAutoMove() string {
	var on, off string
	if m.cfg.AutoMoveEnabled {
		on = lipgloss.NewStyle().Bold(true).Render("[ON] ")
		off = "[OFF]"
	} else {
		on = "[ON] "
		off = lipgloss.NewStyle().Bold(true).Render("[OFF]")
	}
	return fmt.Sprintf("Auto-Move:   %s %s", on, off)
}
