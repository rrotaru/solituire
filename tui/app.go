package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"solituire/config"
	"solituire/engine"
	"solituire/renderer"
	"solituire/theme"
)

// AppModel is the root Bubbletea model. It owns screen state, routes messages
// to the active sub-model, and delegates rendering to the appropriate view.
// AppScreen is defined in messages.go — do not redefine it here.
type AppModel struct {
	screen   AppScreen
	engine   engine.GameEngine
	cfg      *config.Config
	themes   *theme.ThemeRegistry
	rend     *renderer.Renderer
	board    BoardModel
	windowW  int
	windowH  int
	tooSmall bool
}

// NewAppModel creates a ready-to-run AppModel starting on ScreenPlaying.
// When the Menu sub-model is added in T14, the initial screen should be
// changed to ScreenMenu.
func NewAppModel(
	eng engine.GameEngine,
	rend *renderer.Renderer,
	cfg *config.Config,
	themes *theme.ThemeRegistry,
) AppModel {
	return AppModel{
		screen:  ScreenPlaying,
		engine:  eng,
		cfg:     cfg,
		themes:  themes,
		rend:    rend,
		board:   NewBoardModel(eng, rend, cfg),
		windowW: renderer.MinTermWidth,
		windowH: renderer.MinTermHeight,
	}
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
		return m, cmd

	case ChangeScreenMsg:
		m.screen = msg.Screen
		return m, nil

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
		return m, m.board.Init()

	case RestartDealMsg:
		m.engine.RestartDeal()
		m.board = NewBoardModel(m.engine, m.rend, m.cfg)
		// Same size-restore as NewGameMsg.
		sizeUpdated, _ := m.board.Update(tea.WindowSizeMsg{Width: m.windowW, Height: m.windowH})
		m.board = sizeUpdated.(BoardModel)
		m.screen = ScreenPlaying
		return m, m.board.Init()

	case GameWonMsg:
		m.screen = ScreenWin
		return m, nil

	case ThemeChangedMsg:
		if msg.Theme != nil {
			m.cfg.ThemeName = msg.Theme.Name
		}
		return m, nil

	case ConfigChangedMsg:
		if msg.Config != nil {
			m.cfg = msg.Config
		}
		return m, nil
	}

	// Route remaining messages to the active sub-model.
	// Non-playing screens handle key input minimally here so users are never
	// trapped; these cases will be replaced by real sub-models in T14/T15/T18.
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

	case ScreenHelp:
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
			// Any other key cancels and returns to the game.
			return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenPlaying} }
		}

	case ScreenWin, ScreenMenu:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Type == tea.KeyCtrlN:
				return m, func() tea.Msg {
					return NewGameMsg{Seed: appSeed(), DrawCount: m.cfg.DrawCount}
				}
			case key.Type == tea.KeyRunes && len(key.Runes) > 0 &&
				(key.Runes[0] == 'q' || key.Runes[0] == 'Q'):
				return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenQuitConfirm} }
			case key.Type == tea.KeyCtrlC:
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// View renders the active screen. Placeholder strings for screens whose
// sub-models have not yet been implemented are replaced in later phases:
//   - ScreenMenu      → tui/menu.go (T14)
//   - ScreenPaused    → tui/pause.go (T15)
//   - ScreenHelp      → tui/help.go (T15)
//   - ScreenQuitConfirm → tui/dialog.go (T15)
//   - ScreenWin       → tui/celebration.go (T18)
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
	case ScreenHelp:
		return "Help — press Esc or F1 to close."
	case ScreenQuitConfirm:
		return "Quit? (y) Yes  (n) No"
	case ScreenWin:
		return "You won! Press Ctrl+N for a new game."
	case ScreenMenu:
		return "Klondike Solitaire\n\nPress Ctrl+N to start a new game."
	}
	return ""
}

// appSeed returns a non-deterministic seed. Isolated so tests can supply
// explicit seeds via NewGameMsg without a time dependency.
func appSeed() int64 {
	return time.Now().UnixNano()
}
