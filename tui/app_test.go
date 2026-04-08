package tui

import (
	"testing"

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
