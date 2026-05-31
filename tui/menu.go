package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"solituire/config"
	"solituire/engine"
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

const (
	// menuInnerWidth is the text-content width of the menu box (excludes padding
	// and border). Wide enough for the longest theme name "Solarized Light" with
	// comfortable margins; intentionally a round number so the box never resizes.
	menuInnerWidth = 40
	menuPadH       = 3 // horizontal padding each side inside the border
)

var (
	// Fixed outer width = inner + 2×padding. Border adds 2 more on top of that.
	menuBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			Padding(1, menuPadH).
			Width(menuInnerWidth + 2*menuPadH)

	menuTitleStyle = lipgloss.NewStyle().Bold(true)
	menuBoldStyle  = lipgloss.NewStyle().Bold(true)
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
	// center returns s centered within menuInnerWidth using space padding.
	center := func(s string) string {
		pad := (menuInnerWidth - lipgloss.Width(s)) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + s
	}

	// sel returns the cursor triangle prefix for a given menu item.
	// Active row gets "▶ ", all others get two spaces so text stays aligned.
	sel := func(item menuItem) string {
		if m.cursor == item {
			return "▶ "
		}
		return "  "
	}

	// centerAfterCursor centers s in the space to the right of the cursor prefix.
	const cursorWidth = 2
	centerAfterCursor := func(s string) string {
		available := menuInnerWidth - cursorWidth
		pad := (available - lipgloss.Width(s)) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + s
	}

	var sb strings.Builder

	// Title — centered across the full inner width, bold.
	sb.WriteString(center(menuTitleStyle.Render("KLONDIKE SOLITAIRE")))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Option rows: cursor prefix + content.
	sb.WriteString(sel(menuItemDrawMode) + m.renderDrawMode())
	sb.WriteByte('\n')
	sb.WriteString(sel(menuItemTheme) + m.renderTheme())
	sb.WriteByte('\n')
	sb.WriteString(sel(menuItemAutoMove) + m.renderAutoMove())
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Start button — centered to the right of the cursor column; bold when active.
	startLabel := "[ Start New Game ]"
	if m.cursor == menuItemStart {
		startLabel = menuBoldStyle.Render(startLabel)
	}
	sb.WriteString(sel(menuItemStart) + centerAfterCursor(startLabel))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Seed display — centered, no cursor.
	sb.WriteString(center(fmt.Sprintf("Seed: %d", m.cfg.Seed)))

	return menuBoxStyle.Render(sb.String())
}

func (m MenuModel) renderDrawMode() string {
	var d1, d3 string
	if m.cfg.DrawCount == 1 {
		d1 = menuBoldStyle.Render("[ 1 ]")
		d3 = "[ 3 ]"
	} else {
		d1 = "[ 1 ]"
		d3 = menuBoldStyle.Render("[ 3 ]")
	}
	return fmt.Sprintf("Draw Mode:   %s %s", d1, d3)
}

func (m MenuModel) renderTheme() string {
	return fmt.Sprintf("Theme:       \u25c4 %s \u25ba", m.cfg.ThemeName)
}

// newFaceDownState returns a standard Klondike layout where every card is
// face-down, used to render the decorative board background on the menu screen.
func newFaceDownState(drawCount int) *engine.GameState {
	state := engine.Deal(engine.NewDeck(), drawCount)
	for _, pile := range state.Tableau {
		if len(pile.Cards) > 0 {
			pile.Cards[len(pile.Cards)-1].FaceUp = false
		}
	}
	return state
}

func (m MenuModel) renderAutoMove() string {
	var on, off string
	if m.cfg.AutoMoveEnabled {
		on = menuBoldStyle.Render("[ON] ")
		off = "[OFF]"
	} else {
		on = "[ON] "
		off = menuBoldStyle.Render("[OFF]")
	}
	return fmt.Sprintf("Auto-Move:   %s %s", on, off)
}
