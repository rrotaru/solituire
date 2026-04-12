package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"solituire/config"
	"solituire/engine"
	"solituire/renderer"
	"solituire/theme"
)

// newTestApp returns a minimal AppModel for unit testing (seed 42, draw-1).
func newTestApp() AppModel {
	cfg := config.DefaultConfig()
	reg := theme.NewRegistry()
	th := reg.Get(cfg.ThemeName)
	eng := engine.NewGame(42, 1)
	rend := renderer.New(th)
	return NewAppModel(eng, rend, cfg, reg)
}

// updateApp calls Update and type-asserts the result to AppModel.
func updateApp(m AppModel, msg tea.Msg) AppModel {
	result, _ := m.Update(msg)
	return result.(AppModel)
}

// --- ChangeScreenMsg transitions (Section 7.1) ---

func TestAppModel_ChangeScreen_ToPaused(t *testing.T) {
	m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenPaused})
	if m.screen != ScreenPaused {
		t.Errorf("screen = %v, want ScreenPaused", m.screen)
	}
}

func TestAppModel_ChangeScreen_ToHelp(t *testing.T) {
	m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenHelp})
	if m.screen != ScreenHelp {
		t.Errorf("screen = %v, want ScreenHelp", m.screen)
	}
}

func TestAppModel_ChangeScreen_ToQuitConfirm(t *testing.T) {
	m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenQuitConfirm})
	if m.screen != ScreenQuitConfirm {
		t.Errorf("screen = %v, want ScreenQuitConfirm", m.screen)
	}
}

func TestAppModel_ChangeScreen_ToWin(t *testing.T) {
	m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenWin})
	if m.screen != ScreenWin {
		t.Errorf("screen = %v, want ScreenWin", m.screen)
	}
}

func TestAppModel_ChangeScreen_ToMenu(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenPlaying
	m := updateApp(app, ChangeScreenMsg{Screen: ScreenMenu})
	if m.screen != ScreenMenu {
		t.Errorf("screen = %v, want ScreenMenu", m.screen)
	}
}

func TestAppModel_ChangeScreen_ToPlaying(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenPaused
	m := updateApp(app, ChangeScreenMsg{Screen: ScreenPlaying})
	if m.screen != ScreenPlaying {
		t.Errorf("screen = %v, want ScreenPlaying", m.screen)
	}
}

// --- GameWonMsg ---

func TestAppModel_GameWonMsg_TransitionsToWin(t *testing.T) {
	m := updateApp(newTestApp(), GameWonMsg{})
	if m.screen != ScreenWin {
		t.Errorf("screen = %v, want ScreenWin after GameWonMsg", m.screen)
	}
}

// --- NewGameMsg ---

func TestAppModel_NewGameMsg_TransitionsToPlaying(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenMenu
	m := updateApp(app, NewGameMsg{Seed: 1234, DrawCount: 1})
	if m.screen != ScreenPlaying {
		t.Errorf("screen = %v, want ScreenPlaying after NewGameMsg", m.screen)
	}
}

func TestAppModel_NewGameMsg_UpdatesDrawCount(t *testing.T) {
	m := updateApp(newTestApp(), NewGameMsg{Seed: 99, DrawCount: 3})
	if m.cfg.DrawCount != 3 {
		t.Errorf("DrawCount = %d, want 3 after NewGameMsg", m.cfg.DrawCount)
	}
}

func TestAppModel_NewGameMsg_ZeroSeedDoesNotPanic(t *testing.T) {
	// Seed == 0 should trigger appSeed(); just ensure it doesn't panic.
	_ = updateApp(newTestApp(), NewGameMsg{Seed: 0, DrawCount: 1})
}

// --- RestartDealMsg ---

func TestAppModel_RestartDealMsg_TransitionsToPlaying(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenPaused
	m := updateApp(app, RestartDealMsg{})
	if m.screen != ScreenPlaying {
		t.Errorf("screen = %v, want ScreenPlaying after RestartDealMsg", m.screen)
	}
}

// --- ThemeChangedMsg / ConfigChangedMsg ---

func TestAppModel_ThemeChangedMsg_UpdatesThemeName(t *testing.T) {
	reg := theme.NewRegistry()
	dracula := reg.Get("dracula")
	m := updateApp(newTestApp(), ThemeChangedMsg{Theme: &dracula})
	if m.cfg.ThemeName != dracula.Name {
		t.Errorf("ThemeName = %q, want %q", m.cfg.ThemeName, dracula.Name)
	}
}

func TestAppModel_ThemeChangedMsg_NilThemeNoOp(t *testing.T) {
	app := newTestApp()
	original := app.cfg.ThemeName
	m := updateApp(app, ThemeChangedMsg{Theme: nil})
	if m.cfg.ThemeName != original {
		t.Errorf("ThemeName changed on nil ThemeChangedMsg: got %q", m.cfg.ThemeName)
	}
}

func TestAppModel_ConfigChangedMsg_UpdatesConfig(t *testing.T) {
	newCfg := &config.Config{DrawCount: 3, ThemeName: "nord", AutoMoveEnabled: true}
	m := updateApp(newTestApp(), ConfigChangedMsg{Config: newCfg})
	if m.cfg != newCfg {
		t.Error("cfg pointer not updated after ConfigChangedMsg")
	}
}

func TestAppModel_ConfigChangedMsg_UpdatesRendererTheme(t *testing.T) {
	// When ConfigChangedMsg carries a different ThemeName, the renderer must
	// be updated via SetTheme so that a subsequent game start renders with the
	// menu-selected theme rather than the original default theme.
	app := newTestApp() // ThemeName = "classic"
	newCfg := &config.Config{DrawCount: 1, ThemeName: "dracula", AutoMoveEnabled: false}
	app = updateApp(app, ConfigChangedMsg{Config: newCfg})

	// Trigger a new game so the board is rebuilt and renders with the new theme.
	app = updateApp(app, NewGameMsg{Seed: 42, DrawCount: 1})
	app.windowW = renderer.MinTermWidth + 10
	app.windowH = renderer.MinTermHeight + 5

	// board.View() must not panic — SetTheme wired the renderer to "dracula"
	// before NewBoardModel was called, so the board renders with the correct theme.
	got := app.board.View()
	if got == "" {
		t.Error("board.View() returned empty string after ConfigChangedMsg theme change")
	}
}

func TestAppModel_ConfigChangedMsg_NilConfigNoOp(t *testing.T) {
	app := newTestApp()
	original := app.cfg
	m := updateApp(app, ConfigChangedMsg{Config: nil})
	if m.cfg != original {
		t.Error("cfg changed on nil ConfigChangedMsg")
	}
}

// --- WindowSizeMsg + tooSmall ---

func TestAppModel_WindowSizeMsg_TooSmall(t *testing.T) {
	m := updateApp(newTestApp(), tea.WindowSizeMsg{Width: 10, Height: 10})
	if !m.tooSmall {
		t.Error("tooSmall should be true for 10×10 terminal")
	}
}

func TestAppModel_WindowSizeMsg_LargeEnough(t *testing.T) {
	m := updateApp(newTestApp(), tea.WindowSizeMsg{
		Width:  renderer.MinTermWidth + 10,
		Height: renderer.MinTermHeight + 10,
	})
	if m.tooSmall {
		t.Error("tooSmall should be false for an adequately sized terminal")
	}
}

func TestAppModel_WindowSizeMsg_ExactMinimum(t *testing.T) {
	m := updateApp(newTestApp(), tea.WindowSizeMsg{
		Width:  renderer.MinTermWidth,
		Height: renderer.MinTermHeight,
	})
	if m.tooSmall {
		t.Error("tooSmall should be false at exactly the minimum dimensions")
	}
}

func TestAppModel_WindowSizeMsg_OneLessThanMinWidth(t *testing.T) {
	m := updateApp(newTestApp(), tea.WindowSizeMsg{
		Width:  renderer.MinTermWidth - 1,
		Height: renderer.MinTermHeight,
	})
	if !m.tooSmall {
		t.Error("tooSmall should be true when width is one below minimum")
	}
}

func TestAppModel_WindowSizeMsg_OneLessThanMinHeight(t *testing.T) {
	m := updateApp(newTestApp(), tea.WindowSizeMsg{
		Width:  renderer.MinTermWidth,
		Height: renderer.MinTermHeight - 1,
	})
	if !m.tooSmall {
		t.Error("tooSmall should be true when height is one below minimum")
	}
}

// --- View returns non-empty strings for every screen ---

func TestAppModel_View_AllScreens(t *testing.T) {
	screens := []AppScreen{
		ScreenMenu, ScreenPlaying, ScreenPaused,
		ScreenHelp, ScreenQuitConfirm, ScreenWin,
	}
	for _, s := range screens {
		app := newTestApp()
		// Give the app a large terminal so tooSmall doesn't mask the per-screen view.
		app.windowW = renderer.MinTermWidth + 20
		app.windowH = renderer.MinTermHeight + 10
		app.screen = s
		v := app.View()
		if v == "" {
			t.Errorf("View() returned empty string for screen %v", s)
		}
	}
}

func TestAppModel_View_TooSmall(t *testing.T) {
	app := newTestApp()
	app.tooSmall = true
	app.windowW = 10
	app.windowH = 10
	v := app.View()
	if v == "" {
		t.Error("View() returned empty string for tooSmall terminal")
	}
}

// --- P1: non-playing screens must not drop keypresses ---

func TestAppModel_Paused_AnyKeyResumes(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenPaused
	// Simulate a key; the returned Cmd should emit ChangeScreenMsg{ScreenPlaying}.
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if cmd == nil {
		t.Fatal("ScreenPaused: keypress returned nil Cmd, expected ChangeScreenMsg")
	}
	msg := cmd()
	csm, ok := msg.(ChangeScreenMsg)
	if !ok || csm.Screen != ScreenPlaying {
		t.Errorf("ScreenPaused keypress: got %v, want ChangeScreenMsg{ScreenPlaying}", msg)
	}
}

func TestAppModel_Help_AnyKeyCloses(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenHelp
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("ScreenHelp: Esc returned nil Cmd, expected ChangeScreenMsg")
	}
	msg := cmd()
	csm, ok := msg.(ChangeScreenMsg)
	if !ok || csm.Screen != ScreenPlaying {
		t.Errorf("ScreenHelp Esc: got %v, want ChangeScreenMsg{ScreenPlaying}", msg)
	}
}

func TestAppModel_QuitConfirm_YQuits(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenQuitConfirm
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("ScreenQuitConfirm 'y': returned nil Cmd, expected tea.Quit")
	}
	// tea.Quit returns a special QuitMsg; just verify cmd is non-nil and
	// that 'n' does NOT quit.
}

func TestAppModel_QuitConfirm_NoCancelsToPlaying(t *testing.T) {
	// Open the quit dialog from ScreenPlaying via ChangeScreenMsg so prevScreen is set.
	app := newTestApp()
	app.screen = ScreenPlaying
	app = updateApp(app, ChangeScreenMsg{Screen: ScreenQuitConfirm})
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if cmd == nil {
		t.Fatal("ScreenQuitConfirm 'n': returned nil Cmd, expected ChangeScreenMsg")
	}
	msg := cmd()
	csm, ok := msg.(ChangeScreenMsg)
	if !ok || csm.Screen != ScreenPlaying {
		t.Errorf("ScreenQuitConfirm 'n' from playing: got %v, want ChangeScreenMsg{ScreenPlaying}", msg)
	}
}

func TestAppModel_QuitConfirm_CancelFromMenuReturnsToMenu(t *testing.T) {
	// Pressing q on the menu then canceling must return to ScreenMenu, not ScreenPlaying.
	app := newTestApp() // starts on ScreenMenu
	app = updateApp(app, ChangeScreenMsg{Screen: ScreenQuitConfirm})
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if cmd == nil {
		t.Fatal("cancel from menu: returned nil Cmd, expected ChangeScreenMsg")
	}
	msg := cmd()
	csm, ok := msg.(ChangeScreenMsg)
	if !ok || csm.Screen != ScreenMenu {
		t.Errorf("cancel from menu: got %v, want ChangeScreenMsg{ScreenMenu}", msg)
	}
}

func TestAppModel_Win_CtrlNStartsNewGame(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenWin
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if cmd == nil {
		t.Fatal("ScreenWin Ctrl+N: returned nil Cmd, expected NewGameMsg")
	}
	msg := cmd()
	if _, ok := msg.(NewGameMsg); !ok {
		t.Errorf("ScreenWin Ctrl+N: got %T, want NewGameMsg", msg)
	}
}

func TestAppModel_Menu_CtrlNStartsNewGame(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenMenu
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if cmd == nil {
		t.Fatal("ScreenMenu Ctrl+N: returned nil Cmd, expected NewGameMsg")
	}
	msg := cmd()
	if _, ok := msg.(NewGameMsg); !ok {
		t.Errorf("ScreenMenu Ctrl+N: got %T, want NewGameMsg", msg)
	}
}

// --- P2: board window size must be preserved after NewGame / RestartDeal ---

func TestAppModel_NewGameMsg_PreservesWindowSize(t *testing.T) {
	bigW := renderer.MinTermWidth + 40
	bigH := renderer.MinTermHeight + 20
	app := updateApp(newTestApp(), tea.WindowSizeMsg{Width: bigW, Height: bigH})
	app = updateApp(app, NewGameMsg{Seed: 7, DrawCount: 1})
	if app.board.width != bigW {
		t.Errorf("board.width after NewGameMsg = %d, want %d", app.board.width, bigW)
	}
	if app.board.height != bigH {
		t.Errorf("board.height after NewGameMsg = %d, want %d", app.board.height, bigH)
	}
}

func TestAppModel_RestartDealMsg_PreservesWindowSize(t *testing.T) {
	bigW := renderer.MinTermWidth + 40
	bigH := renderer.MinTermHeight + 20
	app := updateApp(newTestApp(), tea.WindowSizeMsg{Width: bigW, Height: bigH})
	app = updateApp(app, RestartDealMsg{})
	if app.board.width != bigW {
		t.Errorf("board.width after RestartDealMsg = %d, want %d", app.board.width, bigW)
	}
	if app.board.height != bigH {
		t.Errorf("board.height after RestartDealMsg = %d, want %d", app.board.height, bigH)
	}
}

// --- Tick chain correctness ---

// TestAppModel_TickMsg_ForwardedOnNonPlayingScreens verifies that TickMsg is
// forwarded to the board on non-playing, non-paused screens so the tick chain
// stays alive and the timer advances (help overlay, quit confirm, win, menu).
func TestAppModel_TickMsg_ForwardedOnNonPlayingScreens(t *testing.T) {
	// ScreenPaused is intentionally excluded: its tick is re-queued without
	// advancing the timer (see TestAppModel_TickMsg_PausedFreezesTimer).
	screens := []AppScreen{ScreenHelp, ScreenQuitConfirm, ScreenWin, ScreenMenu}
	for _, s := range screens {
		app := newTestApp()
		app.screen = s
		before := app.board.eng.State().ElapsedTime
		app = updateApp(app, TickMsg(time.Now()))
		after := app.board.eng.State().ElapsedTime
		if after <= before {
			t.Errorf("screen %v: TickMsg not forwarded to board (ElapsedTime unchanged)", s)
		}
	}
}

// TestAppModel_TickMsg_PausedFreezesTimer verifies that the elapsed timer does
// not advance while the game is paused.
func TestAppModel_TickMsg_PausedFreezesTimer(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenPaused
	before := app.board.eng.State().ElapsedTime
	app = updateApp(app, TickMsg(time.Now()))
	after := app.board.eng.State().ElapsedTime
	if after != before {
		t.Errorf("ScreenPaused: ElapsedTime advanced from %v to %v; timer must be frozen while paused", before, after)
	}
}

// TestAppModel_TickMsg_PausedKeepsChainAlive verifies that ScreenPaused still
// returns a non-nil Cmd so the tick chain is not permanently broken.
func TestAppModel_TickMsg_PausedKeepsChainAlive(t *testing.T) {
	app := newTestApp()
	app.screen = ScreenPaused
	_, cmd := app.Update(TickMsg(time.Now()))
	if cmd == nil {
		t.Error("ScreenPaused: TickMsg returned nil Cmd; tick chain would be broken on resume")
	}
}

// TestAppModel_NewGameMsg_NoExtraTickCmd verifies that handling NewGameMsg
// does not return a Cmd, preventing a duplicate tick chain.
func TestAppModel_NewGameMsg_NoExtraTickCmd(t *testing.T) {
	_, cmd := newTestApp().Update(NewGameMsg{Seed: 1234, DrawCount: 1})
	if cmd != nil {
		t.Error("NewGameMsg returned a non-nil Cmd; would create a duplicate tick chain")
	}
}

// TestAppModel_RestartDealMsg_NoExtraTickCmd mirrors the NewGameMsg check.
func TestAppModel_RestartDealMsg_NoExtraTickCmd(t *testing.T) {
	_, cmd := newTestApp().Update(RestartDealMsg{})
	if cmd != nil {
		t.Error("RestartDealMsg returned a non-nil Cmd; would create a duplicate tick chain")
	}
}
