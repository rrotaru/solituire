package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/golden"
	"solituire/config"
	"solituire/theme"
)

// updateMenu calls Update and type-asserts the result to MenuModel.
func updateMenu(m MenuModel, msg tea.Msg) (MenuModel, tea.Cmd) {
	result, cmd := m.Update(msg)
	return result.(MenuModel), cmd
}

// runMenuCmd executes a tea.Cmd and returns the resulting message (or nil).
func runMenuCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// newTestMenu returns a MenuModel with default config for use in tests.
// Seed is fixed at 12345 for deterministic golden output.
func newTestMenu() MenuModel {
	cfg := config.DefaultConfig()
	cfg.Seed = 12345
	return NewMenuModel(cfg, theme.NewRegistry())
}

// --- Golden render test (Pattern A) ---

func TestMenuRender(t *testing.T) {
	m := newTestMenu()
	got := m.View()
	golden.RequireEqual(t, []byte(got))
}

// --- Cursor navigation ---

func TestMenuModel_CursorDown_Advances(t *testing.T) {
	m := newTestMenu() // starts at menuItemDrawMode
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != menuItemTheme {
		t.Errorf("cursor = %d, want menuItemTheme (%d)", m.cursor, menuItemTheme)
	}
}

func TestMenuModel_CursorDown_Wraps(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemStart
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != menuItemDrawMode {
		t.Errorf("cursor should wrap to 0, got %d", m.cursor)
	}
}

func TestMenuModel_CursorUp_Retreats(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemTheme
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != menuItemDrawMode {
		t.Errorf("cursor = %d, want menuItemDrawMode (%d)", m.cursor, menuItemDrawMode)
	}
}

func TestMenuModel_CursorUp_Wraps(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemDrawMode
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != menuItemStart {
		t.Errorf("cursor should wrap to menuItemStart (%d), got %d", menuItemStart, m.cursor)
	}
}

func TestMenuModel_Tab_AdvancesCursor(t *testing.T) {
	m := newTestMenu()
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.cursor != menuItemTheme {
		t.Errorf("Tab: cursor = %d, want menuItemTheme (%d)", m.cursor, menuItemTheme)
	}
}

func TestMenuModel_ShiftTab_RetreatsWrCursor(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemTheme
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.cursor != menuItemDrawMode {
		t.Errorf("ShiftTab: cursor = %d, want menuItemDrawMode (%d)", m.cursor, menuItemDrawMode)
	}
}

func TestMenuModel_VimJ_AdvancesCursor(t *testing.T) {
	m := newTestMenu()
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != menuItemTheme {
		t.Errorf("j: cursor = %d, want %d", m.cursor, menuItemTheme)
	}
}

func TestMenuModel_VimK_RetreatsWraps(t *testing.T) {
	m := newTestMenu()
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != menuItemStart {
		t.Errorf("k from DrawMode: cursor = %d, want menuItemStart (%d)", m.cursor, menuItemStart)
	}
}

// --- Draw Mode ---

func TestMenuModel_DrawMode_RightTogglesTo3(t *testing.T) {
	m := newTestMenu() // DrawCount=1, cursor=menuItemDrawMode
	m, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.cfg.DrawCount != 3 {
		t.Errorf("DrawCount = %d, want 3", m.cfg.DrawCount)
	}
	msg := runMenuCmd(cmd)
	if _, ok := msg.(ConfigChangedMsg); !ok {
		t.Errorf("expected ConfigChangedMsg, got %T", msg)
	}
}

func TestMenuModel_DrawMode_RightWrapsBackTo1(t *testing.T) {
	m := newTestMenu()
	m.cfg.DrawCount = 3
	m, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.cfg.DrawCount != 1 {
		t.Errorf("DrawCount = %d, want 1 after wrap", m.cfg.DrawCount)
	}
	if runMenuCmd(cmd) == nil {
		t.Error("expected ConfigChangedMsg, got nil")
	}
}

func TestMenuModel_DrawMode_EnterToggles(t *testing.T) {
	m := newTestMenu()
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.cfg.DrawCount != 3 {
		t.Errorf("DrawCount = %d, want 3 after Enter", m.cfg.DrawCount)
	}
}

func TestMenuModel_DrawMode_LeftToggles(t *testing.T) {
	m := newTestMenu()
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyLeft})
	if m.cfg.DrawCount != 3 {
		t.Errorf("DrawCount = %d, want 3 after Left (only 2 options)", m.cfg.DrawCount)
	}
}

// --- Theme cycling ---

func TestMenuModel_Theme_RightCyclesToNext(t *testing.T) {
	m := newTestMenu() // ThemeName="classic"
	m.cursor = menuItemTheme
	before := m.cfg.ThemeName

	m, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})

	if m.cfg.ThemeName == before {
		t.Errorf("ThemeName unchanged after Right: %q", m.cfg.ThemeName)
	}
	if _, ok := runMenuCmd(cmd).(ConfigChangedMsg); !ok {
		t.Error("expected ConfigChangedMsg after theme change")
	}
}

func TestMenuModel_Theme_LeftCyclesToPrev(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemTheme
	// Advance once so we're not already at the first theme.
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	after := m.cfg.ThemeName

	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyLeft})
	if m.cfg.ThemeName == after {
		t.Errorf("ThemeName unchanged after Left: %q", m.cfg.ThemeName)
	}
}

func TestMenuModel_Theme_FullCycleReturnsToStart(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemTheme
	start := m.cfg.ThemeName
	names := theme.NewRegistry().List()

	for range len(names) {
		m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	}
	// DefaultConfig uses "classic" (lowercase); registry canonical name is "Classic".
	// Use case-insensitive comparison, mirroring ThemeRegistry.Next().
	if !strings.EqualFold(m.cfg.ThemeName, start) {
		t.Errorf("after full cycle ThemeName = %q, want %q (case-insensitive)", m.cfg.ThemeName, start)
	}
}

// --- Auto-Move ---

func TestMenuModel_AutoMove_RightTogglesOn(t *testing.T) {
	m := newTestMenu() // AutoMoveEnabled=false
	m.cursor = menuItemAutoMove
	m, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	if !m.cfg.AutoMoveEnabled {
		t.Error("AutoMoveEnabled should be true after Right")
	}
	if _, ok := runMenuCmd(cmd).(ConfigChangedMsg); !ok {
		t.Error("expected ConfigChangedMsg")
	}
}

func TestMenuModel_AutoMove_RightTogglesOff(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemAutoMove
	m.cfg.AutoMoveEnabled = true
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.cfg.AutoMoveEnabled {
		t.Error("AutoMoveEnabled should be false after second Right")
	}
}

func TestMenuModel_AutoMove_LeftToggles(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemAutoMove
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyLeft})
	if !m.cfg.AutoMoveEnabled {
		t.Error("AutoMoveEnabled should be true after Left")
	}
}

// --- Start New Game button ---

func TestMenuModel_Start_EmitsNewGameMsg(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemStart
	_, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyEnter})
	msg := runMenuCmd(cmd)
	if _, ok := msg.(NewGameMsg); !ok {
		t.Fatalf("expected NewGameMsg, got %T", msg)
	}
}

func TestMenuModel_Start_NewGameMsgHasCurrentDrawCount(t *testing.T) {
	m := newTestMenu()
	// Set draw count to 3 first.
	m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	m.cursor = menuItemStart
	_, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyEnter})
	ngm := runMenuCmd(cmd).(NewGameMsg)
	if ngm.DrawCount != 3 {
		t.Errorf("NewGameMsg.DrawCount = %d, want 3", ngm.DrawCount)
	}
}

func TestMenuModel_Start_NewGameMsgHasSeed(t *testing.T) {
	m := newTestMenu() // seed=12345
	m.cursor = menuItemStart
	_, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyEnter})
	ngm := runMenuCmd(cmd).(NewGameMsg)
	if ngm.Seed != 12345 {
		t.Errorf("NewGameMsg.Seed = %d, want 12345", ngm.Seed)
	}
}

func TestMenuModel_Start_SpaceAlsoActivates(t *testing.T) {
	m := newTestMenu()
	m.cursor = menuItemStart
	_, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeySpace})
	if _, ok := runMenuCmd(cmd).(NewGameMsg); !ok {
		t.Error("Space on Start should emit NewGameMsg")
	}
}

// --- CtrlN shortcut ---

func TestMenuModel_CtrlN_EmitsNewGameMsg(t *testing.T) {
	m := newTestMenu()
	_, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if _, ok := runMenuCmd(cmd).(NewGameMsg); !ok {
		t.Error("Ctrl+N should emit NewGameMsg from any cursor position")
	}
}

// --- ConfigChangedMsg carries updated config ---

func TestMenuModel_ConfigChangedMsg_HasUpdatedDrawCount(t *testing.T) {
	m := newTestMenu()
	_, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	ccm := runMenuCmd(cmd).(ConfigChangedMsg)
	if ccm.Config == nil {
		t.Fatal("ConfigChangedMsg.Config is nil")
	}
	if ccm.Config.DrawCount != 3 {
		t.Errorf("ConfigChangedMsg.Config.DrawCount = %d, want 3", ccm.Config.DrawCount)
	}
}

func TestMenuModel_ConfigChangedMsg_IsIndependentCopy(t *testing.T) {
	m := newTestMenu()
	_, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})
	ccm := runMenuCmd(cmd).(ConfigChangedMsg)
	// Mutating the returned config should not affect the menu's internal state.
	ccm.Config.DrawCount = 99
	if m.cfg.DrawCount == 99 {
		t.Error("ConfigChangedMsg.Config shares memory with MenuModel.cfg")
	}
}

// --- Non-key messages are ignored ---

func TestMenuModel_NonKeyMsg_IsNoOp(t *testing.T) {
	m := newTestMenu()
	m2, cmd := updateMenu(m, TickMsg{})
	if m2.cursor != m.cursor || m2.cfg != m.cfg {
		t.Error("non-key message should not change MenuModel state")
	}
	if cmd != nil {
		t.Error("non-key message should return nil Cmd")
	}
}

// --- App integration: initial screen is ScreenMenu ---

func TestAppModel_InitialScreen_IsMenu(t *testing.T) {
	app := newTestApp()
	if app.screen != ScreenMenu {
		t.Errorf("initial screen = %v, want ScreenMenu", app.screen)
	}
}
