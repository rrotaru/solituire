package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"solituire/config"
	"solituire/engine"
	"solituire/renderer"
	"solituire/theme"
)

// BoardModel is the Bubbletea tea.Model for an active game board.
// It manages cursor state, delegates game commands to the engine, and
// delegates rendering to the renderer.
type BoardModel struct {
	eng      engine.GameEngine
	cursor   Cursor
	rend     *renderer.Renderer
	cfg      *config.Config
	themes   *theme.ThemeRegistry
	width    int
	height   int
}

// NewBoardModel creates a BoardModel with the cursor positioned at the bottom
// face-up card of Tableau column 0.
func NewBoardModel(eng engine.GameEngine, rend *renderer.Renderer, cfg *config.Config) BoardModel {
	m := BoardModel{
		eng:    eng,
		rend:   rend,
		cfg:    cfg,
		themes: theme.NewRegistry(),
		width:  renderer.MinTermWidth,
		height: renderer.MinTermHeight,
	}
	m.cursor.Pile = engine.PileTableau0
	m.cursor.CardIndex = naturalCardIndex(engine.PileTableau0, eng.State())
	return m
}

// Init starts the elapsed-time ticker.
func (m BoardModel) Init() tea.Cmd {
	return tickCmd()
}

// Update handles all incoming messages.
func (m BoardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rend.SetSize(msg.Width, msg.Height)
		return m, nil

	case TickMsg:
		m.eng.State().ElapsedTime += time.Second
		return m, tickCmd()
	}

	action, payload := TranslateInput(msg)
	return m.handleAction(action, payload)
}

// View delegates rendering to the renderer.
func (m BoardModel) View() string {
	return m.rend.Render(m.eng.State(), m.cursor.RendererCursor(), m.cfg)
}

// handleAction translates a GameAction into cursor movement, engine commands,
// or emitted messages.
func (m BoardModel) handleAction(action GameAction, payload interface{}) (tea.Model, tea.Cmd) {
	state := m.eng.State()

	switch action {
	case ActionCursorLeft:
		m.cursor.MoveLeft(state)

	case ActionCursorRight:
		m.cursor.MoveRight(state)

	case ActionCursorUp:
		m.cursor.MoveUp(state)

	case ActionCursorDown:
		m.cursor.MoveDown(state)

	case ActionJumpToColumn:
		if col, ok := payload.(int); ok {
			m.cursor.JumpToColumn(col, state)
		}

	case ActionSelect:
		m = m.handleSelect(state)

	case ActionCancel:
		m.cursor.Dragging = false
		m.cursor.DragSource = 0
		m.cursor.DragCardCount = 0
		m.cursor.ShowHint = false

	case ActionFlipStock:
		m.flipStock(state)

	case ActionMoveToFoundation:
		m.moveToFoundation(state)

	case ActionUndo:
		_ = m.eng.Undo()
		// Clamp cursor after undo in case piles shrank
		m.clampCursor()

	case ActionRedo:
		_ = m.eng.Redo()

	case ActionHint:
		m.toggleHint(state)

	case ActionNewGame:
		return m, func() tea.Msg {
			return NewGameMsg{Seed: time.Now().UnixNano(), DrawCount: m.cfg.DrawCount}
		}

	case ActionRestartDeal:
		return m, func() tea.Msg { return RestartDealMsg{} }

	case ActionPause:
		return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenPaused} }

	case ActionHelp:
		return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenHelp} }

	case ActionQuit:
		return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenQuitConfirm} }

	case ActionCycleTheme:
		next := m.themes.Next(m.cfg.ThemeName)
		m.cfg.ThemeName = next.Name
		m.rend.SetTheme(next)
		return m, func() tea.Msg { return ThemeChangedMsg{Theme: &next} }

	case ActionToggleAutoMove:
		m.cfg.AutoMoveEnabled = !m.cfg.AutoMoveEnabled
		cfgCopy := *m.cfg
		return m, func() tea.Msg { return ConfigChangedMsg{Config: &cfgCopy} }
	}

	return m, nil
}

// handleSelect implements the drag-style pick-up / place flow.
// First Enter picks up cards; second Enter attempts to place them.
func (m BoardModel) handleSelect(state *engine.GameState) BoardModel {
	if !m.cursor.Dragging {
		// Pressing Enter on the stock flips it instead of starting a drag.
		if m.cursor.Pile == engine.PileStock {
			m.flipStock(state)
			return m
		}
		m.cursor.DragSource = m.cursor.Pile
		m.cursor.DragCardCount = dragCount(state, m.cursor)
		if m.cursor.DragCardCount > 0 {
			m.cursor.Dragging = true
		}
	} else {
		// Attempt to place dragged cards at current cursor position.
		dest := m.cursor.Pile
		cmd := m.buildMoveCmd(state, m.cursor.DragSource, m.cursor.DragCardCount, dest)
		if cmd != nil {
			_ = m.eng.Execute(cmd) // silent rejection on error
		}
		m.cursor.Dragging = false
		m.cursor.DragSource = 0
		m.cursor.DragCardCount = 0
		m.clampCursor()
	}
	return m
}

// dragCount returns the number of cards to drag from the cursor position.
func dragCount(state *engine.GameState, c Cursor) int {
	if isTableauPile(c.Pile) {
		col := int(c.Pile - engine.PileTableau0)
		pile := state.Tableau[col]
		n := len(pile.Cards) - c.CardIndex
		if n < 1 {
			return 0
		}
		return n
	}
	// Waste and foundation: only the top card is draggable.
	return 1
}

// buildMoveCmd constructs the appropriate command for a drag-and-drop placement.
// Returns nil if the combination is unsupported (will result in a silent rejection).
func (m BoardModel) buildMoveCmd(state *engine.GameState, from engine.PileID, count int, to engine.PileID) engine.Command {
	if isFoundationPile(to) && count == 1 {
		fi := int(to - engine.PileFoundation0)
		return &engine.MoveToFoundationCmd{From: from, FoundationIdx: fi}
	}

	if isTableauPile(to) {
		moveCmd := &engine.MoveCardCmd{From: from, To: to, CardCount: count}
		if isTableauPile(from) {
			// Determine if a face-down card will be exposed after the move.
			srcCol := int(from - engine.PileTableau0)
			srcPile := state.Tableau[srcCol]
			newTopIdx := len(srcPile.Cards) - count - 1
			if newTopIdx >= 0 && !srcPile.Cards[newTopIdx].FaceUp {
				return &engine.CompoundCmd{
					Cmds: []engine.Command{
						moveCmd,
						&engine.FlipTableauCardCmd{ColumnIdx: srcCol},
					},
				}
			}
		}
		return moveCmd
	}

	return nil
}

// flipStock executes FlipStockCmd or RecycleStockCmd depending on stock state.
func (m *BoardModel) flipStock(state *engine.GameState) {
	if state.Stock.IsEmpty() {
		_ = m.eng.Execute(&engine.RecycleStockCmd{})
	} else {
		_ = m.eng.Execute(&engine.FlipStockCmd{})
	}
}

// moveToFoundation moves the top card of the cursor's pile to the matching foundation.
func (m *BoardModel) moveToFoundation(state *engine.GameState) {
	var card *engine.Card
	switch {
	case m.cursor.Pile == engine.PileWaste:
		card = state.Waste.TopCard()
	case isTableauPile(m.cursor.Pile):
		col := int(m.cursor.Pile - engine.PileTableau0)
		card = state.Tableau[col].TopCard()
	case isFoundationPile(m.cursor.Pile):
		fi := int(m.cursor.Pile - engine.PileFoundation0)
		card = state.Foundations[fi].TopCard()
	}
	if card == nil {
		return
	}
	for fi, f := range state.Foundations {
		if f.AcceptsCard(*card) {
			_ = m.eng.Execute(&engine.MoveToFoundationCmd{From: m.cursor.Pile, FoundationIdx: fi})
			m.clampCursor()
			return
		}
	}
}

// toggleHint shows the top hint from engine.FindHints, or clears it if already shown.
func (m *BoardModel) toggleHint(state *engine.GameState) {
	if m.cursor.ShowHint {
		m.cursor.ShowHint = false
		return
	}
	hints := engine.FindHints(state)
	if len(hints) > 0 {
		m.cursor.HintFrom = hints[0].From
		m.cursor.HintTo = hints[0].To
		m.cursor.ShowHint = true
	}
}

// clampCursor ensures cursor.CardIndex is within valid bounds for the current pile.
func (m *BoardModel) clampCursor() {
	state := m.eng.State()
	if isTableauPile(m.cursor.Pile) {
		col := int(m.cursor.Pile - engine.PileTableau0)
		pile := state.Tableau[col]
		if pile.IsEmpty() {
			m.cursor.CardIndex = 0
			return
		}
		max := len(pile.Cards) - 1
		if m.cursor.CardIndex > max {
			m.cursor.CardIndex = max
		}
		min := pile.FaceDownCount()
		if m.cursor.CardIndex < min && len(pile.FaceUpCards()) > 0 {
			m.cursor.CardIndex = min
		}
	}
}

// tickCmd returns a tea.Cmd that fires a TickMsg after one second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
