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

	// Clear any active hint on the first non-hint player action so stale
	// guidance doesn't linger across unrelated inputs.
	// ActionNone (no recognised key) and ActionHint itself are excluded:
	// ActionHint must see the current ShowHint value to toggle correctly.
	if action != ActionNone && action != ActionHint {
		m.cursor.ShowHint = false
	}

	switch action {
	case ActionCursorLeft:
		m.cursor.MoveLeft(state)

	case ActionCursorRight:
		m.cursor.MoveRight(state)

	case ActionTabNext:
		m.cursor.TabNext(state)

	case ActionTabPrev:
		m.cursor.TabPrev(state)

	case ActionCursorUp:
		m.cursor.MoveUp(state)

	case ActionCursorDown:
		m.cursor.MoveDown(state)

	case ActionJumpToColumn:
		if col, ok := payload.(int); ok && col >= 0 && col <= 6 {
			m.cursor.JumpToColumn(col, state)
		}

	case ActionSelect:
		// For mouse clicks, hit-test the click coordinates to move the cursor to
		// the target pile/card before running the select logic. Clicks outside any
		// pile are ignored.
		if mouse, ok := payload.(tea.MouseMsg); ok {
			pile, cardIdx, hit := renderer.PileHitTestWithWidth(mouse.X, mouse.Y, state, m.width)
			if !hit {
				break
			}
			m.cursor.Pile = pile
			m.cursor.CardIndex = cardIdx
		}
		m = m.handleSelect(state)

	case ActionCancel:
		m.cursor.Dragging = false
		m.cursor.DragSource = 0
		m.cursor.DragCardCount = 0
		m.cursor.ShowHint = false

	case ActionFlipStock:
		// Cancel any active drag before flipping so the drag source (which may be
		// PileWaste) is not left pointing at a card that is no longer the top card.
		m.cursor.Dragging = false
		m.cursor.DragSource = 0
		m.cursor.DragCardCount = 0
		m.flipStock(state)

	case ActionMoveToFoundation:
		if m.cursor.Dragging {
			// Only a single-card drag can go to a foundation.
			if m.cursor.DragCardCount == 1 {
				m.moveToFoundation(state, m.cursor.DragSource)
			}
			// Clear drag regardless — 'f' always ends the drag gesture.
			m.cursor.Dragging = false
			m.cursor.DragSource = 0
			m.cursor.DragCardCount = 0
		} else {
			// For tableau piles the shortcut only applies when the cursor sits on
			// the top card; pressing 'f' on a non-top face-up card is a no-op so
			// that the action target always matches the highlighted card.
			if isTableauPile(m.cursor.Pile) {
				col := int(m.cursor.Pile - engine.PileTableau0)
				pile := state.Tableau[col]
				if pile.IsEmpty() || m.cursor.CardIndex != len(pile.Cards)-1 {
					break
				}
			}
			m.moveToFoundation(state, m.cursor.Pile)
		}

	case ActionUndo:
		m.cursor.Dragging = false
		m.cursor.DragSource = 0
		m.cursor.DragCardCount = 0
		_ = m.eng.Undo()
		m.clampCursor()

	case ActionRedo:
		m.cursor.Dragging = false
		m.cursor.DragSource = 0
		m.cursor.DragCardCount = 0
		_ = m.eng.Redo()
		m.clampCursor()

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

	return m, m.winCmd()
}

// winCmd returns a Cmd that emits GameWonMsg if the engine reports a win.
func (m BoardModel) winCmd() tea.Cmd {
	if m.eng.IsWon() {
		return func() tea.Msg { return GameWonMsg{} }
	}
	return nil
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
// Returns 0 when the pile has no card to pick up.
func dragCount(state *engine.GameState, c Cursor) int {
	if isTableauPile(c.Pile) {
		col := int(c.Pile - engine.PileTableau0)
		pile := state.Tableau[col]
		if c.CardIndex < pile.FaceDownCount() {
			return 0 // face-down cards cannot be dragged
		}
		n := len(pile.Cards) - c.CardIndex
		if n < 1 {
			return 0
		}
		return n
	}
	if c.Pile == engine.PileWaste {
		if state.Waste.TopCard() == nil {
			return 0
		}
		return 1
	}
	if isFoundationPile(c.Pile) {
		fi := int(c.Pile - engine.PileFoundation0)
		if state.Foundations[fi].TopCard() == nil {
			return 0
		}
		return 1
	}
	return 0
}

// buildMoveCmd constructs the appropriate command for a drag-and-drop placement.
// Returns nil if the combination is unsupported (will result in a silent rejection).
func (m BoardModel) buildMoveCmd(state *engine.GameState, from engine.PileID, count int, to engine.PileID) engine.Command {
	if isFoundationPile(to) && count == 1 {
		fi := int(to - engine.PileFoundation0)
		moveCmd := &engine.MoveToFoundationCmd{From: from, FoundationIdx: fi}
		if isTableauPile(from) {
			srcCol := int(from - engine.PileTableau0)
			srcPile := state.Tableau[srcCol]
			newTopIdx := len(srcPile.Cards) - 2 // after removing the one moved card
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

// moveToFoundation moves the top card of src to the matching foundation.
func (m *BoardModel) moveToFoundation(state *engine.GameState, src engine.PileID) {
	var card *engine.Card
	switch {
	case src == engine.PileWaste:
		card = state.Waste.TopCard()
	case isTableauPile(src):
		col := int(src - engine.PileTableau0)
		card = state.Tableau[col].TopCard()
	case isFoundationPile(src):
		fi := int(src - engine.PileFoundation0)
		card = state.Foundations[fi].TopCard()
	}
	if card == nil {
		return
	}
	for fi, f := range state.Foundations {
		if f.AcceptsCard(*card) {
			to := engine.PileFoundation0 + engine.PileID(fi)
			cmd := m.buildMoveCmd(state, src, 1, to)
			if cmd != nil {
				_ = m.eng.Execute(cmd)
			}
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
