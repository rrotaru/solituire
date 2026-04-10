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
	eng            engine.GameEngine
	cursor         Cursor
	rend           *renderer.Renderer
	cfg            *config.Config
	themes         *theme.ThemeRegistry
	width          int
	height         int
	autoCompleting bool // true while the auto-complete animation loop is running
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

	case AutoCompleteStepMsg:
		return m.handleAutoCompleteStep()
	}

	// Any keypress interrupts an in-progress auto-complete.
	if m.autoCompleting {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.autoCompleting = false
			return m, nil
		}
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
	// Unmapped events (mouse motion, release, scroll, unknown keys) produce
	// ActionNone. Return immediately: no state change, no automation, no extra
	// auto-complete ticks.
	if action == ActionNone {
		return m, nil
	}

	state := m.eng.State()

	// Clear any active hint on the first non-hint player action so stale
	// guidance doesn't linger across unrelated inputs.
	// ActionHint is excluded: it must see the current ShowHint value to toggle correctly.
	if action != ActionHint {
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
		return m, m.winCmd() // skip auto-move: undo must not be immediately reversed

	case ActionRedo:
		m.cursor.Dragging = false
		m.cursor.DragSource = 0
		m.cursor.DragCardCount = 0
		_ = m.eng.Redo()
		m.clampCursor()
		return m, m.winCmd() // skip auto-move: keep redo result stable

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

	// Skip auto-move while a drag is in progress: the drag source card is still
	// in the engine state, so auto-move could steal it before the user places it.
	// After placement handleSelect sets Dragging=false, so auto-move runs then.
	if !m.cursor.Dragging {
		m.applyAutoMove()
	}
	if m.eng.IsWon() {
		return m, m.winCmd()
	}
	if !m.autoCompleting && m.eng.IsAutoCompletable() {
		m.autoCompleting = true
	}
	if m.autoCompleting {
		return m, autoCompleteTickCmd()
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

// handleAutoCompleteStep executes one foundation move for the auto-complete loop,
// then schedules the next tick or ends the loop.
func (m BoardModel) handleAutoCompleteStep() (tea.Model, tea.Cmd) {
	if !m.autoCompleting {
		return m, nil
	}
	moved := m.doAutoCompleteStep()
	if !moved || m.eng.IsWon() {
		m.autoCompleting = false
		if m.eng.IsWon() {
			return m, func() tea.Msg { return GameWonMsg{} }
		}
		return m, nil
	}
	return m, autoCompleteTickCmd()
}

// doAutoCompleteStep finds the lowest-rank card that can move to a foundation and
// executes that move. Returns true if a move was made.
func (m *BoardModel) doAutoCompleteStep() bool {
	state := m.eng.State()

	var bestRank engine.Rank = engine.King + 1 // sentinel: no candidate yet
	var bestSrc engine.PileID
	bestFI := -1

	consider := func(card *engine.Card, src engine.PileID) {
		if card == nil || !card.FaceUp {
			return
		}
		for fi, f := range state.Foundations {
			if f.AcceptsCard(*card) && card.Rank < bestRank {
				bestRank = card.Rank
				bestSrc = src
				bestFI = fi
			}
		}
	}

	consider(state.Waste.TopCard(), engine.PileWaste)
	for col := 0; col < 7; col++ {
		consider(state.Tableau[col].TopCard(), engine.PileTableau0+engine.PileID(col))
	}

	if bestFI < 0 {
		return false
	}

	dest := engine.PileFoundation0 + engine.PileID(bestFI)
	cmd := m.buildMoveCmd(state, bestSrc, 1, dest)
	if cmd == nil {
		return false
	}
	_ = m.eng.Execute(cmd)
	m.clampCursor()
	return true
}

// applyAutoMove repeatedly moves safe cards to foundations while AutoMoveEnabled.
// Each call to autoMoveOneCard re-fetches state, so cascading moves are handled
// correctly (e.g., moving a 2 may make a 3 safe on the next pass).
func (m *BoardModel) applyAutoMove() {
	if !m.cfg.AutoMoveEnabled {
		return
	}
	for m.autoMoveOneCard() {
	}
}

// autoMoveOneCard checks waste and tableau tops for a single safe auto-move and
// executes the first one found. Returns true if a card was moved.
func (m *BoardModel) autoMoveOneCard() bool {
	state := m.eng.State()

	tryMove := func(card *engine.Card, src engine.PileID) bool {
		if card == nil || !card.FaceUp || !isSafeToAutoMove(*card, state) {
			return false
		}
		for fi, f := range state.Foundations {
			if f.AcceptsCard(*card) {
				dest := engine.PileFoundation0 + engine.PileID(fi)
				cmd := m.buildMoveCmd(state, src, 1, dest)
				if cmd != nil {
					_ = m.eng.Execute(cmd)
					m.clampCursor()
					return true
				}
			}
		}
		return false
	}

	if tryMove(state.Waste.TopCard(), engine.PileWaste) {
		return true
	}
	for col := 0; col < 7; col++ {
		if tryMove(state.Tableau[col].TopCard(), engine.PileTableau0+engine.PileID(col)) {
			return true
		}
	}
	return false
}

// isSafeToAutoMove reports whether card can be safely auto-moved to its foundation.
// "Safe" means BOTH opposite-colored foundation piles have rank >= card.Rank-1.
// Unstarted (nil-suit) foundations count as rank 0 and will fail the check.
// Aces and 2s are unconditionally safe.
func isSafeToAutoMove(card engine.Card, state *engine.GameState) bool {
	if card.Rank <= 2 {
		return true
	}
	oppositeColor := engine.Black
	if card.Color() == engine.Black {
		oppositeColor = engine.Red
	}
	// There are exactly 2 foundations of each color. Both must satisfy rank >= card.Rank-1.
	// Unstarted foundations (nil Suit) are skipped — they can never contribute to safeCount,
	// so requiring safeCount >= 2 correctly treats them as rank 0.
	safeCount := 0
	for _, f := range state.Foundations {
		s := f.Suit()
		if s == nil || s.Color() != oppositeColor {
			continue
		}
		top := f.TopCard()
		if top != nil && top.Rank >= card.Rank-1 {
			safeCount++
		}
	}
	return safeCount >= 2
}

// tickCmd returns a tea.Cmd that fires a TickMsg after one second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// autoCompleteTickCmd returns a Cmd that fires AutoCompleteStepMsg after 100 ms.
func autoCompleteTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return AutoCompleteStepMsg{}
	})
}
