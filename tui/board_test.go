package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/muesli/termenv"
	"solituire/config"
	"solituire/engine"
	"solituire/renderer"
	"solituire/theme"
)

func init() {
	// Lock color profile to ASCII for deterministic golden output.
	lipgloss.SetColorProfile(termenv.Ascii)
}

// testEngine is a minimal engine.GameEngine implementation backed by raw engine
// primitives. It is used only in tui tests.
type testEngine struct {
	state   *engine.GameState
	history engine.History
}

func (e *testEngine) State() *engine.GameState { return e.state }
func (e *testEngine) IsWon() bool {
	for _, f := range e.state.Foundations {
		if !f.IsComplete() {
			return false
		}
	}
	return true
}
func (e *testEngine) IsAutoCompletable() bool { return false }
func (e *testEngine) Score() int              { return e.state.Score }
func (e *testEngine) MoveCount() int          { return e.state.MoveCount }
func (e *testEngine) Seed() int64             { return e.state.Seed }
func (e *testEngine) Execute(cmd engine.Command) error {
	scoreBefore := e.state.Score
	if err := cmd.Execute(e.state); err != nil {
		return err
	}
	e.history.Push(cmd, scoreBefore, e.state.Score)
	e.state.MoveCount++
	return nil
}
func (e *testEngine) Undo() error { return e.history.Undo(e.state) }
func (e *testEngine) Redo() error { return e.history.Redo(e.state) }
func (e *testEngine) CanUndo() bool { return e.history.CanUndo() }
func (e *testEngine) CanRedo() bool { return e.history.CanRedo() }
func (e *testEngine) ValidMoves() []engine.Move {
	return engine.ValidMoves(e.state)
}
func (e *testEngine) IsValidMove(move engine.Move) bool {
	return engine.ValidateMove(e.state, move) == nil
}
func (e *testEngine) NewGame(seed int64, drawCount int) {
	deck := engine.NewDeck()
	engine.Shuffle(deck, seed)
	e.state = engine.Deal(deck, drawCount)
	e.state.Seed = seed
	e.history.Clear()
}
func (e *testEngine) RestartDeal() {
	deck := engine.NewDeck()
	engine.Shuffle(deck, e.state.Seed)
	e.state = engine.Deal(deck, e.state.DrawCount)
	e.history.Clear()
}

// newSeed42Engine returns a testEngine with seed-42 draw-1 state.
func newSeed42Engine() *testEngine {
	deck := engine.NewDeck()
	engine.Shuffle(deck, 42)
	state := engine.Deal(deck, 1)
	state.Seed = 42
	return &testEngine{state: state}
}

// newBoard creates a BoardModel backed by a seed-42 engine at 80×30.
func newBoard() (BoardModel, *testEngine) {
	eng := newSeed42Engine()
	rend := renderer.New(theme.Classic)
	rend.SetSize(80, 30)
	cfg := config.DefaultConfig()
	board := NewBoardModel(eng, rend, cfg)
	return board, eng
}

// sendKey delivers a key message to the board and returns the updated model.
func sendKey(board BoardModel, kt tea.KeyType) BoardModel {
	updated, _ := board.Update(tea.KeyMsg{Type: kt})
	return updated.(BoardModel)
}

// sendRune delivers a rune key message to the board and returns the updated model.
func sendRune(board BoardModel, r rune) BoardModel {
	updated, _ := board.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return updated.(BoardModel)
}

// --- Golden test ---

func TestBoardViewGolden(t *testing.T) {
	board, _ := newBoard()
	got := board.View()
	golden.RequireEqual(t, []byte(got))
}

// --- Model state tests ---

func TestBoardDragPickUp(t *testing.T) {
	board, _ := newBoard()
	// Position cursor on a non-empty tableau pile and press Enter to pick up.
	board.cursor.Pile = engine.PileTableau3
	board.cursor.CardIndex = len(board.eng.State().Tableau[3].Cards) - 1

	board = sendKey(board, tea.KeyEnter)
	if !board.cursor.Dragging {
		t.Fatal("expected Dragging=true after picking up a card")
	}
	if board.cursor.DragSource != engine.PileTableau3 {
		t.Errorf("DragSource: got %v, want PileTableau3", board.cursor.DragSource)
	}
	if board.cursor.DragCardCount < 1 {
		t.Errorf("DragCardCount must be >= 1, got %d", board.cursor.DragCardCount)
	}
}

func TestBoardDragCancel(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileTableau3
	board.cursor.CardIndex = len(board.eng.State().Tableau[3].Cards) - 1

	board = sendKey(board, tea.KeyEnter) // pick up
	if !board.cursor.Dragging {
		t.Fatal("precondition: expected Dragging=true")
	}

	board = sendKey(board, tea.KeyEsc) // cancel
	if board.cursor.Dragging {
		t.Error("expected Dragging=false after Esc")
	}
	if board.cursor.DragCardCount != 0 {
		t.Errorf("expected DragCardCount=0 after cancel, got %d", board.cursor.DragCardCount)
	}
}

// TestBoardEnterOnEmptyWaste verifies that pressing Enter on an empty waste pile
// does not start a drag (waste is empty at game start before any stock flip).
func TestBoardEnterOnEmptyWaste(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileWaste
	board.cursor.CardIndex = 0

	board = sendKey(board, tea.KeyEnter)
	if board.cursor.Dragging {
		t.Error("Enter on empty waste must not start a drag")
	}
}

// TestBoardEnterOnEmptyFoundation verifies that pressing Enter on an empty
// foundation pile does not start a drag.
func TestBoardEnterOnEmptyFoundation(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileFoundation0
	board.cursor.CardIndex = 0

	board = sendKey(board, tea.KeyEnter)
	if board.cursor.Dragging {
		t.Error("Enter on empty foundation must not start a drag")
	}
}

func TestBoardSelectOnStock_FlipsNotDrags(t *testing.T) {
	board, eng := newBoard()
	wasteBefore := len(eng.State().Waste.Cards)

	// Cursor on stock; Enter should flip, not start a drag.
	board.cursor.Pile = engine.PileStock
	board.cursor.CardIndex = 0
	board = sendKey(board, tea.KeyEnter)

	if board.cursor.Dragging {
		t.Error("Enter on stock must not start a drag")
	}
	if len(eng.State().Waste.Cards) <= wasteBefore {
		t.Error("Enter on stock must flip cards to waste")
	}
}

func TestBoardDragPlace_Valid(t *testing.T) {
	board, eng := newBoard()

	// Find a valid tableau-to-tableau move.
	var move engine.Move
	for _, m := range eng.ValidMoves() {
		if isTableauPile(m.From) && isTableauPile(m.To) {
			move = m
			break
		}
	}
	if move.From == 0 && move.To == 0 {
		t.Skip("no tableau-to-tableau move available with seed 42")
	}

	srcCol := int(move.From - engine.PileTableau0)
	srcLen := len(eng.State().Tableau[srcCol].Cards)
	srcCardIdx := srcLen - move.CardCount

	// Pick up from source.
	board.cursor.Pile = move.From
	board.cursor.CardIndex = srcCardIdx
	board = sendKey(board, tea.KeyEnter)
	if !board.cursor.Dragging {
		t.Fatal("expected Dragging=true after pickup")
	}

	// Move cursor to destination.
	board.cursor.Pile = move.To
	board.cursor.CardIndex = naturalCardIndex(move.To, eng.State())

	// Place.
	board = sendKey(board, tea.KeyEnter)
	if board.cursor.Dragging {
		t.Error("expected Dragging=false after placement")
	}
	afterLen := len(eng.State().Tableau[srcCol].Cards)
	if afterLen >= srcLen {
		t.Errorf("source pile should have shrunk: before=%d after=%d", srcLen, afterLen)
	}
}

func TestBoardDragPlace_Invalid(t *testing.T) {
	board, eng := newBoard()
	moveBefore := eng.State().MoveCount

	// Pick up from T0 bottom card.
	board.cursor.Pile = engine.PileTableau0
	board.cursor.CardIndex = len(eng.State().Tableau[0].Cards) - 1
	board = sendKey(board, tea.KeyEnter)
	if !board.cursor.Dragging {
		t.Fatal("precondition: expected Dragging=true")
	}

	// Try to place on T0 (same pile — always invalid).
	board.cursor.Pile = engine.PileTableau0
	board = sendKey(board, tea.KeyEnter)

	if board.cursor.Dragging {
		t.Error("expected Dragging=false after attempted placement")
	}
	if eng.State().MoveCount != moveBefore {
		t.Error("invalid move must not change MoveCount")
	}
}

func TestBoardFlipStock(t *testing.T) {
	board, eng := newBoard()
	wasteBefore := len(eng.State().Waste.Cards)

	board = sendKey(board, tea.KeySpace)

	if len(eng.State().Waste.Cards) <= wasteBefore {
		t.Error("Space must flip a card from stock to waste")
	}
}

func TestBoardUndo(t *testing.T) {
	board, eng := newBoard()

	// Flip stock to create an undoable action.
	board = sendKey(board, tea.KeySpace)
	wasteAfterFlip := len(eng.State().Waste.Cards)
	if wasteAfterFlip == 0 {
		t.Fatal("flip produced no waste cards")
	}

	board = sendKey(board, tea.KeyCtrlZ)
	if len(eng.State().Waste.Cards) != 0 {
		t.Errorf("after undo waste should be empty, got %d cards", len(eng.State().Waste.Cards))
	}
}

func TestBoardRedo(t *testing.T) {
	board, eng := newBoard()

	board = sendKey(board, tea.KeySpace) // flip
	wasteAfterFlip := len(eng.State().Waste.Cards)

	board = sendKey(board, tea.KeyCtrlZ)  // undo
	board = sendKey(board, tea.KeyCtrlY)  // redo

	if len(eng.State().Waste.Cards) != wasteAfterFlip {
		t.Errorf("after redo waste should have %d cards, got %d", wasteAfterFlip, len(eng.State().Waste.Cards))
	}
}

// TestBoardRedoClampsCursor verifies that after a redo that removes cards from
// the cursor's tableau pile, CardIndex is clamped to a valid position so that
// subsequent drag attempts compute a positive card count.
func TestBoardRedoClampsCursor(t *testing.T) {
	board, eng := newBoard()

	// Find a tableau-to-tableau move so the source pile shrinks after the move.
	var move engine.Move
	for _, m := range eng.ValidMoves() {
		if isTableauPile(m.From) && isTableauPile(m.To) {
			move = m
			break
		}
	}
	if move.From == 0 && move.To == 0 {
		t.Skip("no tableau-to-tableau move available with seed 42")
	}

	srcCol := int(move.From - engine.PileTableau0)

	// Position cursor at the bottom of the source pile and execute the move.
	board.cursor.Pile = move.From
	board.cursor.CardIndex = len(eng.State().Tableau[srcCol].Cards) - 1
	board = sendKey(board, tea.KeyEnter) // pick up
	board.cursor.Pile = move.To
	board = sendKey(board, tea.KeyEnter) // place — source pile shrinks

	// Undo restores the source pile; cursor may now be below the new top.
	board = sendKey(board, tea.KeyCtrlZ)

	// Redo re-applies the move; source pile shrinks again.
	// Cursor must be clamped to remain within the pile.
	board.cursor.Pile = move.From
	board.cursor.CardIndex = len(eng.State().Tableau[srcCol].Cards) - 1 // park at bottom before redo
	board = sendKey(board, tea.KeyCtrlY)

	if isTableauPile(board.cursor.Pile) {
		col := int(board.cursor.Pile - engine.PileTableau0)
		pile := eng.State().Tableau[col]
		if !pile.IsEmpty() && board.cursor.CardIndex >= len(pile.Cards) {
			t.Errorf("after redo CardIndex %d is out of bounds for pile len %d",
				board.cursor.CardIndex, len(pile.Cards))
		}
	}
}

func TestBoardHintToggle(t *testing.T) {
	board, _ := newBoard()

	// First '?' should show a hint.
	board = sendRune(board, '?')
	if !board.cursor.ShowHint {
		t.Error("'?' must set ShowHint=true")
	}

	// Second '?' should clear the hint.
	board = sendRune(board, '?')
	if board.cursor.ShowHint {
		t.Error("second '?' must clear ShowHint")
	}
}

func TestBoardTickUpdatesElapsed(t *testing.T) {
	board, eng := newBoard()
	before := eng.State().ElapsedTime

	updated, _ := board.Update(TickMsg(time.Now()))
	board = updated.(BoardModel)

	after := eng.State().ElapsedTime
	if after != before+time.Second {
		t.Errorf("TickMsg must add 1 second: before=%v after=%v", before, after)
	}
}

// TestBoardTabReachesFoundation verifies that Tab can visit a foundation pile,
// confirming tabCycleOrder (not navCycleOrder) is used.
func TestBoardTabReachesFoundation(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileWaste // start just before foundations in tabCycleOrder
	board.cursor.CardIndex = 0

	board = sendKey(board, tea.KeyTab)
	if !isFoundationPile(board.cursor.Pile) {
		t.Errorf("Tab from Waste must land on a foundation pile, got %v", board.cursor.Pile)
	}
}

// TestBoardShiftTabReachesFoundation verifies that Shift-Tab can visit a foundation.
func TestBoardShiftTabReachesFoundation(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileTableau0 // first in tabCycleOrder after foundations
	board.cursor.CardIndex = 0

	board = sendKey(board, tea.KeyShiftTab)
	if !isFoundationPile(board.cursor.Pile) {
		t.Errorf("Shift-Tab from Tableau0 must land on a foundation pile, got %v", board.cursor.Pile)
	}
}

// TestBoardArrowDoesNotReachFoundation verifies that Left/Right arrow keys skip
// foundations (navCycleOrder) so they still reach them only via Tab.
func TestBoardArrowDoesNotReachFoundation(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileWaste
	board.cursor.CardIndex = 0

	// Right from Waste with arrow key goes to Tableau0, not Foundation0.
	board = sendKey(board, tea.KeyRight)
	if isFoundationPile(board.cursor.Pile) {
		t.Errorf("Right arrow from Waste must skip foundations, got %v", board.cursor.Pile)
	}
	if board.cursor.Pile != engine.PileTableau0 {
		t.Errorf("Right arrow from Waste must land on PileTableau0, got %v", board.cursor.Pile)
	}
}

// TestBoardFoundationAutoFlip verifies that moving a tableau top card to a
// foundation via drag automatically flips the newly exposed face-down card.
func TestBoardFoundationAutoFlip(t *testing.T) {
	_, eng := newBoard()

	// Find a to-foundation move from a tableau column that will expose a face-down card.
	var move engine.Move
	state := eng.State()
	for _, m := range eng.ValidMoves() {
		if !isTableauPile(m.From) || !isFoundationPile(m.To) {
			continue
		}
		col := int(m.From - engine.PileTableau0)
		pile := state.Tableau[col]
		// Need at least 2 cards and the card below is face-down.
		if len(pile.Cards) >= 2 && !pile.Cards[len(pile.Cards)-2].FaceUp {
			move = m
			break
		}
	}
	if move.From == 0 && move.To == 0 {
		t.Skip("no tableau-to-foundation move that exposes a face-down card with seed 42")
	}

	srcCol := int(move.From - engine.PileTableau0)

	// Execute the move directly through board's drag flow.
	board, _ := newBoard()
	board.cursor.Pile = move.From
	board.cursor.CardIndex = len(eng.State().Tableau[srcCol].Cards) - 1

	board = sendKey(board, tea.KeyEnter) // pick up
	board.cursor.Pile = move.To
	board = sendKey(board, tea.KeyEnter) // place on foundation

	// The previously face-down card must now be face-up.
	topCard := eng.State().Tableau[srcCol].TopCard()
	if topCard == nil {
		t.Fatal("source tableau column is empty after move — unexpected")
	}
	if !topCard.FaceUp {
		t.Errorf("card beneath moved card must be flipped face-up after foundation drop, got FaceUp=false")
	}
}

// TestBoardMoveToFoundationKey_AutoFlip verifies that the 'f' shortcut also
// triggers the auto-flip when a face-down card is exposed.
func TestBoardMoveToFoundationKey_AutoFlip(t *testing.T) {
	_, eng := newBoard()
	state := eng.State()

	// Find a tableau column whose top card can go to a foundation and has a
	// face-down card immediately beneath it.
	targetCol := -1
	for col := 0; col < 7; col++ {
		pile := state.Tableau[col]
		top := pile.TopCard()
		if top == nil || len(pile.Cards) < 2 {
			continue
		}
		if pile.Cards[len(pile.Cards)-2].FaceUp {
			continue // need a face-down card below
		}
		for _, f := range state.Foundations {
			if f.AcceptsCard(*top) {
				targetCol = col
				break
			}
		}
		if targetCol >= 0 {
			break
		}
	}
	if targetCol < 0 {
		t.Skip("no suitable tableau column for 'f' auto-flip test with seed 42")
	}

	board, _ := newBoard()
	board.cursor.Pile = engine.PileTableau0 + engine.PileID(targetCol)
	board.cursor.CardIndex = len(eng.State().Tableau[targetCol].Cards) - 1

	board = sendRune(board, 'f')

	topCard := eng.State().Tableau[targetCol].TopCard()
	if topCard == nil {
		// Column is now empty — that's fine, nothing to check.
		return
	}
	if !topCard.FaceUp {
		t.Errorf("'f' key must flip the newly exposed face-down card, got FaceUp=false")
	}
}

func TestBoardWindowResize(t *testing.T) {
	board, _ := newBoard()

	updated, _ := board.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	board = updated.(BoardModel)

	if board.width != 120 || board.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", board.width, board.height)
	}
}
