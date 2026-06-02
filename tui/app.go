package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"solituire/config"
	"solituire/engine"
	"solituire/renderer"
	"solituire/theme"
)

// AppScreen identifies which screen the application is currently showing.
type AppScreen int

const (
	ScreenMenu AppScreen = iota
	ScreenPlaying
	ScreenPaused
	ScreenKeybindHelp
	ScreenQuitConfirm
	ScreenWin
)

// AppModel is the root Bubbletea model. It owns screen state, routes messages
// to the active sub-model, and delegates rendering to the appropriate view.
type AppModel struct {
	screen      AppScreen
	prevScreen  AppScreen // screen to return to when ScreenQuitConfirm is canceled
	engine      engine.GameEngine
	cfg         *config.Config
	themes      *theme.ThemeRegistry
	rend        *renderer.Renderer
	board       BoardModel
	menu        MenuModel
	celebration CelebrationModel
	windowW     int
	windowH     int
	tooSmall    bool
}

// NewAppModel creates a ready-to-run AppModel starting on the settings menu
// (ScreenMenu). Use WithScreen to start directly on the board.
func NewAppModel(
	eng engine.GameEngine,
	rend *renderer.Renderer,
	cfg *config.Config,
	themes *theme.ThemeRegistry,
) AppModel {
	return AppModel{
		screen:  ScreenMenu,
		engine:  eng,
		cfg:     cfg,
		themes:  themes,
		rend:    rend,
		board:   NewBoardModel(eng, rend, cfg),
		menu:    NewMenuModel(cfg, themes),
		windowW: renderer.MinTermWidth,
		windowH: renderer.MinTermHeight,
	}
}

// WithScreen returns a copy of AppModel with the initial screen overridden.
// Used by main.go to bypass the menu when all config is supplied via CLI flags.
func (m AppModel) WithScreen(s AppScreen) AppModel {
	m.screen = s
	return m
}

// Init starts the elapsed-time ticker by delegating to the board sub-model.
func (m AppModel) Init() tea.Cmd {
	return m.board.Init()
}

// Update handles app-level messages first, then routes to the active sub-model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowW = msg.Width
		m.windowH = msg.Height
		m.tooSmall = msg.Width < renderer.MinTermWidth || msg.Height < renderer.MinTermHeight
		// Always propagate to board so renderer dimensions stay current.
		updated, cmd := m.board.Update(msg)
		m.board = updated.(BoardModel)
		// Propagate to celebration during win flow so card positions reflow on
		// resize. Includes ScreenQuitConfirm opened from ScreenWin: no new
		// WindowSizeMsg is emitted on screen switch, so skipping the update
		// here would leave celebration with stale dimensions when the dialog
		// is canceled and win is restored.
		if m.screen == ScreenWin ||
			(m.screen == ScreenQuitConfirm && m.prevScreen == ScreenWin) {
			celebUpdated, _ := m.celebration.Update(msg)
			m.celebration = celebUpdated.(CelebrationModel)
		}
		return m, cmd

	case ChangeScreenMsg:
		if msg.Screen == ScreenQuitConfirm {
			m.prevScreen = m.screen
		}
		m.screen = msg.Screen
		return m, nil

	case TickMsg:
		// Keep the tick chain alive on all screens. While paused, re-queue
		// the tick without forwarding to the board so the elapsed timer
		// freezes — resuming the game continues timing from where it stopped.
		if m.screen == ScreenPaused {
			return m, tickCmd()
		}
		updated, cmd := m.board.Update(msg)
		m.board = updated.(BoardModel)
		return m, cmd

	case CelebrationTickMsg:
		// Only forward during win flow: the win screen itself, or the quit-confirm
		// dialog opened from the win screen (prevScreen == ScreenWin) so that
		// canceling returns to a live animation. Any other screen (playing, menu,
		// etc.) drops the tick without requeuing so the chain terminates cleanly.
		// Without this guard every new game after a win accumulates an additional
		// concurrent 80ms tick loop.
		inWinFlow := m.screen == ScreenWin ||
			(m.screen == ScreenQuitConfirm && m.prevScreen == ScreenWin)
		if !inWinFlow {
			return m, nil
		}
		celebUpdated, cmd := m.celebration.Update(msg)
		m.celebration = celebUpdated.(CelebrationModel)
		return m, cmd

	case NewGameMsg:
		seed := msg.Seed
		if seed == 0 {
			seed = appSeed()
		}
		m.engine.NewGame(seed, msg.DrawCount)
		m.cfg.DrawCount = msg.DrawCount
		m.board = NewBoardModel(m.engine, m.rend, m.cfg)
		// Restore current terminal dimensions so mouse hit-testing stays correct
		// after the board is rebuilt (NewBoardModel defaults to MinTermWidth/Height).
		sizeUpdated, _ := m.board.Update(tea.WindowSizeMsg{Width: m.windowW, Height: m.windowH})
		m.board = sizeUpdated.(BoardModel)
		m.screen = ScreenPlaying
		// Do NOT call m.board.Init() here: the TickMsg handler above keeps the
		// tick chain alive across board rebuilds, so calling Init() would
		// schedule a second tickCmd and cause time to advance at 2× speed.
		return m, nil

	case RestartDealMsg:
		m.engine.RestartDeal()
		m.board = NewBoardModel(m.engine, m.rend, m.cfg)
		// Same size-restore as NewGameMsg.
		sizeUpdated, _ := m.board.Update(tea.WindowSizeMsg{Width: m.windowW, Height: m.windowH})
		m.board = sizeUpdated.(BoardModel)
		m.screen = ScreenPlaying
		// Same reasoning as NewGameMsg: tick chain is already in flight.
		return m, nil

	case GameWonMsg:
		state := m.engine.State()
		m.celebration = NewCelebrationModel(
			m.engine.Score(),
			m.engine.MoveCount(),
			state.ElapsedTime,
			m.themes.Get(m.cfg.ThemeName),
			m.cfg.DrawCount,
		)
		// Seed with actual terminal dimensions (NewCelebrationModel defaults to
		// 78×24). No fresh WindowSizeMsg is emitted on screen transitions, so
		// apply a synthetic resize now — same pattern as NewBoardModel handling
		// in NewGameMsg / RestartDealMsg.
		sizeUpdated, _ := m.celebration.Update(tea.WindowSizeMsg{Width: m.windowW, Height: m.windowH})
		m.celebration = sizeUpdated.(CelebrationModel)
		m.screen = ScreenWin
		return m, m.celebration.Init()

	case ThemeChangedMsg:
		if msg.Theme != nil {
			m.cfg.ThemeName = msg.Theme.Name
		}
		return m, nil

	case ConfigChangedMsg:
		if msg.Config != nil {
			if msg.Config.ThemeName != m.cfg.ThemeName {
				m.rend.SetTheme(m.themes.Get(msg.Config.ThemeName))
			}
			m.cfg = msg.Config
		}
		return m, nil
	}

	// Route remaining messages to the active sub-model. The pause and
	// quit-confirm screens are handled inline here with minimal key handling
	// (any key resumes / cancels) rather than via dedicated sub-models.
	switch m.screen {
	case ScreenPlaying:
		updated, cmd := m.board.Update(msg)
		m.board = updated.(BoardModel)
		return m, cmd

	case ScreenPaused:
		// Any keypress resumes the game.
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenPlaying} }
		}

	case ScreenKeybindHelp:
		// Any keypress closes the overlay.
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenPlaying} }
		}

	case ScreenQuitConfirm:
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.Type == tea.KeyRunes && len(key.Runes) > 0 &&
				(key.Runes[0] == 'y' || key.Runes[0] == 'Y') {
				return m, tea.Quit
			}
			// Any other key cancels — return to whichever screen opened the dialog.
			prev := m.prevScreen
			if prev == ScreenQuitConfirm {
				prev = ScreenPlaying // safeguard against self-referential prev
			}
			return m, func() tea.Msg { return ChangeScreenMsg{Screen: prev} }
		}

	case ScreenMenu:
		// Global exit keys handled before delegating to the sub-model.
		if key, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Type == tea.KeyCtrlC:
				return m, tea.Quit
			case key.Type == tea.KeyRunes && len(key.Runes) > 0 &&
				(key.Runes[0] == 'q' || key.Runes[0] == 'Q'):
				return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenQuitConfirm} }
			}
		}
		updated, cmd := m.menu.Update(msg)
		m.menu = updated.(MenuModel)
		return m, cmd

	case ScreenWin:
		celebUpdated, cmd := m.celebration.Update(msg)
		m.celebration = celebUpdated.(CelebrationModel)
		return m, cmd
	}

	return m, nil
}

// View renders the active screen:
//   - ScreenMenu        → the settings menu overlaid on a face-down board
//   - ScreenPlaying     → the board sub-model
//   - ScreenPaused      → an inline "paused" message
//   - ScreenKeybindHelp → the keybind help overlay (help.go)
//   - ScreenQuitConfirm → an inline quit-confirmation prompt
//   - ScreenWin         → the celebration sub-model
func (m AppModel) View() string {
	if m.tooSmall {
		return fmt.Sprintf(
			"Terminal too small.\nMinimum size: %d×%d  Current: %d×%d",
			renderer.MinTermWidth, renderer.MinTermHeight,
			m.windowW, m.windowH,
		)
	}

	switch m.screen {
	case ScreenPlaying:
		return m.board.View()
	case ScreenPaused:
		return "Game Paused — press any key to resume."
	case ScreenKeybindHelp:
		return RenderKeybindHelp(m.board.View(), m.windowW, m.windowH)
	case ScreenQuitConfirm:
		return "Quit? (y) Yes  (n) No"
	case ScreenWin:
		return m.celebration.View()
	case ScreenMenu:
		boardStr := m.rend.Render(newFaceDownState(m.cfg.DrawCount), renderer.CursorState{})
		menuStr := m.menu.View()
		menuLines := strings.Split(menuStr, "\n")
		menuH := len(menuLines)
		menuW := lipgloss.Width(menuLines[0])

		// Center horizontally within the board content columns (tableau spans
		// BoardWidth chars, left-aligned at x=0).
		startCol := (renderer.BoardWidth - menuW) / 2
		if startCol < 0 {
			startCol = 0
		}

		// Center vertically between the header (row 0) and footer (last row).
		boardLineCount := strings.Count(boardStr, "\n") + 1
		startRow := 1 + (boardLineCount-2-menuH)/2
		if startRow < 1 {
			startRow = 1
		}

		return renderer.Overlay(boardStr, menuStr, startRow, startCol, m.windowW)
	}
	return ""
}

// appSeed returns a non-deterministic seed. Isolated so tests can supply
// explicit seeds via NewGameMsg without a time dependency.
func appSeed() int64 {
	return time.Now().UnixNano()
}
