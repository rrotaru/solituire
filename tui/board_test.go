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
func (e *testEngine) IsAutoCompletable() bool {
	if !e.state.Stock.IsEmpty() {
		return false
	}
	for _, t := range e.state.Tableau {
		for _, c := range t.Cards {
			if !c.FaceUp {
				return false
			}
		}
	}
	return true
}
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

// TestBoardEnterOnFaceDownTableauCard verifies that pressing Enter (or a mouse
// click resolved to a face-down stub) on a face-down tableau card does not
// start a drag, since face-down cards are not legal drag sources.
func TestBoardEnterOnFaceDownTableauCard(t *testing.T) {
	board, eng := newBoard()

	// Find a tableau column with at least one face-down card.
	targetCol := -1
	for col := 0; col < 7; col++ {
		if eng.State().Tableau[col].FaceDownCount() > 0 {
			targetCol = col
			break
		}
	}
	if targetCol < 0 {
		t.Skip("no tableau column with a face-down card at deal time")
	}

	board.cursor.Pile = engine.PileTableau0 + engine.PileID(targetCol)
	board.cursor.CardIndex = 0 // always face-down in a freshly dealt game

	board = sendKey(board, tea.KeyEnter)
	if board.cursor.Dragging {
		t.Error("Enter on a face-down card must not start a drag")
	}
	if board.cursor.DragCardCount != 0 {
		t.Errorf("DragCardCount must be 0 after no-op, got %d", board.cursor.DragCardCount)
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

// TestBoardFlipStockCancelsDrag verifies that pressing Space while a drag is
// active cancels the drag before flipping the stock, preventing stale drag
// state from referencing a card that is no longer the waste top card.
func TestBoardFlipStockCancelsDrag(t *testing.T) {
	board, eng := newBoard()

	// Flip a card to waste so we have something to drag from there.
	board = sendKey(board, tea.KeySpace)
	if len(eng.State().Waste.Cards) == 0 {
		t.Fatal("precondition: expected a card on waste after stock flip")
	}

	// Start a drag from waste.
	board.cursor.Pile = engine.PileWaste
	board.cursor.CardIndex = 0
	board = sendKey(board, tea.KeyEnter)
	if !board.cursor.Dragging {
		t.Fatal("precondition: expected Dragging=true after picking up from waste")
	}

	// Flip stock again while drag is active.
	board = sendKey(board, tea.KeySpace)

	if board.cursor.Dragging {
		t.Error("Space while dragging must cancel the drag")
	}
	if board.cursor.DragSource != 0 || board.cursor.DragCardCount != 0 {
		t.Error("Space while dragging must clear DragSource and DragCardCount")
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

// TestBoardUndoClearsDrag verifies that pressing Ctrl+Z while a drag is active
// cancels the drag before reverting the last move.
func TestBoardUndoClearsDrag(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileTableau3
	board.cursor.CardIndex = len(board.eng.State().Tableau[3].Cards) - 1
	board = sendKey(board, tea.KeyEnter) // pick up
	if !board.cursor.Dragging {
		t.Fatal("precondition: expected Dragging=true")
	}

	board = sendKey(board, tea.KeyCtrlZ)
	if board.cursor.Dragging {
		t.Error("Ctrl+Z must clear Dragging")
	}
	if board.cursor.DragSource != 0 || board.cursor.DragCardCount != 0 {
		t.Error("Ctrl+Z must clear DragSource and DragCardCount")
	}
}

// TestBoardRedoClearsDrag verifies that pressing Ctrl+Y while a drag is active
// cancels the drag before re-applying the last undone move.
func TestBoardRedoClearsDrag(t *testing.T) {
	board, eng := newBoard()

	// Create an undone action to redo.
	board = sendKey(board, tea.KeySpace)   // flip stock
	board = sendKey(board, tea.KeyCtrlZ)   // undo

	// Start a drag.
	board.cursor.Pile = engine.PileTableau3
	board.cursor.CardIndex = len(eng.State().Tableau[3].Cards) - 1
	board = sendKey(board, tea.KeyEnter)
	if !board.cursor.Dragging {
		t.Fatal("precondition: expected Dragging=true")
	}

	board = sendKey(board, tea.KeyCtrlY)
	if board.cursor.Dragging {
		t.Error("Ctrl+Y must clear Dragging")
	}
	if board.cursor.DragSource != 0 || board.cursor.DragCardCount != 0 {
		t.Error("Ctrl+Y must clear DragSource and DragCardCount")
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

// TestBoardFKeyWhileDragging_SingleCard verifies that pressing 'f' during a
// single-card drag completes the move to the foundation from DragSource (not
// the hovered pile) and clears the drag state.
func TestBoardFKeyWhileDragging_SingleCard(t *testing.T) {
	board, eng := newBoard()

	// Find a tableau top card that can go to a foundation.
	var srcPile engine.PileID
	for col := 0; col < 7; col++ {
		pile := eng.State().Tableau[col]
		top := pile.TopCard()
		if top == nil {
			continue
		}
		for _, f := range eng.State().Foundations {
			if f.AcceptsCard(*top) {
				srcPile = engine.PileTableau0 + engine.PileID(col)
				break
			}
		}
		if srcPile != 0 {
			break
		}
	}
	if srcPile == 0 {
		t.Skip("no tableau card can go to a foundation with seed 42 at deal time")
	}

	srcCol := int(srcPile - engine.PileTableau0)
	board.cursor.Pile = srcPile
	board.cursor.CardIndex = len(eng.State().Tableau[srcCol].Cards) - 1

	board = sendKey(board, tea.KeyEnter) // pick up one card
	if !board.cursor.Dragging {
		t.Fatal("precondition: expected Dragging=true")
	}

	// Move cursor to an unrelated pile to prove 'f' uses DragSource, not cursor.
	board.cursor.Pile = engine.PileStock

	board = sendRune(board, 'f') // should complete the drag to foundation

	if board.cursor.Dragging {
		t.Error("'f' while dragging must clear Dragging")
	}
	if board.cursor.DragSource != 0 {
		t.Error("'f' while dragging must clear DragSource")
	}
}

// TestBoardFKeyWhileDragging_MultiCard verifies that pressing 'f' during a
// multi-card drag (count > 1, which cannot go to a foundation) still clears
// the drag without moving any card.
func TestBoardFKeyWhileDragging_MultiCard(t *testing.T) {
	board, eng := newBoard()

	// Find a tableau pile with at least 2 face-up cards to drag a stack.
	var srcPile engine.PileID
	for col := 0; col < 7; col++ {
		pile := eng.State().Tableau[col]
		if len(pile.FaceUpCards()) >= 2 {
			srcPile = engine.PileTableau0 + engine.PileID(col)
			break
		}
	}
	if srcPile == 0 {
		t.Skip("no tableau column with 2+ face-up cards at deal time with seed 42")
	}

	srcCol := int(srcPile - engine.PileTableau0)
	fdCount := eng.State().Tableau[srcCol].FaceDownCount()
	// Pick up from the first face-up card so DragCardCount >= 2.
	board.cursor.Pile = srcPile
	board.cursor.CardIndex = fdCount
	board = sendKey(board, tea.KeyEnter)
	if !board.cursor.Dragging || board.cursor.DragCardCount < 2 {
		t.Skip("could not start a multi-card drag")
	}

	movesBefore := eng.State().MoveCount
	board = sendRune(board, 'f')

	if board.cursor.Dragging {
		t.Error("'f' while dragging must clear Dragging even for multi-card drag")
	}
	if eng.State().MoveCount != movesBefore {
		t.Error("multi-card drag + 'f' must not move any card")
	}
}

// TestBoardHintClearedByAction verifies that any non-hint player action
// clears an active hint so stale guidance does not persist on screen.
func TestBoardHintClearedByAction(t *testing.T) {
	board, _ := newBoard()

	board = sendRune(board, '?') // show hint
	if !board.cursor.ShowHint {
		t.Fatal("precondition: ShowHint must be true after '?'")
	}

	// A cursor movement is a non-hint action and must clear the hint.
	board = sendKey(board, tea.KeyRight)
	if board.cursor.ShowHint {
		t.Error("cursor movement must clear ShowHint")
	}

	// '?' after the hint was cleared should show a fresh hint (not toggle off).
	board = sendRune(board, '?')
	if !board.cursor.ShowHint {
		t.Error("'?' after hint was cleared must re-show a hint")
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

// TestBoardJumpToColumn_OutOfRangePayload verifies that an out-of-range column
// payload is silently ignored and leaves the cursor unchanged.
func TestBoardJumpToColumn_OutOfRangePayload(t *testing.T) {
	board, _ := newBoard()
	board.cursor.Pile = engine.PileStock
	board.cursor.CardIndex = 0

	for _, col := range []int{-1, 7, 100} {
		updated, _ := board.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{rune('1' + col)}, // won't map to ActionJumpToColumn, so inject directly
		})
		// Inject the action directly to bypass TranslateInput's own guard.
		model, _ := board.handleAction(ActionJumpToColumn, col)
		b := model.(BoardModel)
		if b.cursor.Pile != engine.PileStock {
			t.Errorf("out-of-range col %d: cursor pile changed to %v", col, b.cursor.Pile)
		}
		_ = updated
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
	board, eng := newBoard()

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

	// Execute the move through the same board/engine pair used for discovery.
	board.cursor.Pile = move.From
	board.cursor.CardIndex = len(state.Tableau[srcCol].Cards) - 1

	board = sendKey(board, tea.KeyEnter) // pick up
	board.cursor.Pile = move.To
	board = sendKey(board, tea.KeyEnter) // place on foundation

	// Assert through the same engine that board.Update mutated.
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
	board, eng := newBoard()
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

	// Execute through the same board/engine pair used for discovery.
	board.cursor.Pile = engine.PileTableau0 + engine.PileID(targetCol)
	board.cursor.CardIndex = len(state.Tableau[targetCol].Cards) - 1

	board = sendRune(board, 'f')

	// Assert through the same engine that board.Update mutated.
	topCard := eng.State().Tableau[targetCol].TopCard()
	if topCard == nil {
		// Column is now empty — that's fine, nothing to check.
		return
	}
	if !topCard.FaceUp {
		t.Errorf("'f' key must flip the newly exposed face-down card, got FaceUp=false")
	}
}

// TestBoardFKeyOnNonTopCard verifies that pressing 'f' when the cursor is on a
// non-top face-up card is a no-op — the action target must match the highlighted card.
func TestBoardFKeyOnNonTopCard(t *testing.T) {
	board, eng := newBoard()

	// Manually flip the second-to-last card of T6 face-up so there are two
	// accessible face-up cards in the column.
	col := 6
	pile := eng.State().Tableau[col]
	if len(pile.Cards) < 2 {
		t.Skip("T6 needs at least 2 cards")
	}
	pile.Cards[len(pile.Cards)-2].FaceUp = true

	moveBefore := eng.State().MoveCount
	board.cursor.Pile = engine.PileTableau6
	board.cursor.CardIndex = len(pile.Cards) - 2 // second-to-last — not the top

	board = sendRune(board, 'f')

	if eng.State().MoveCount != moveBefore {
		t.Error("'f' on a non-top card must be a no-op; engine state must not change")
	}
}

// TestBoardWinEmitsGameWonMsg verifies that a move which completes all four
// foundations causes Update to return a Cmd that emits GameWonMsg.
func TestBoardWinEmitsGameWonMsg(t *testing.T) {
	board, eng := newBoard()
	state := eng.State()

	// Clear all piles so only the cards we place exist.
	for i := range state.Tableau {
		state.Tableau[i].Cards = nil
	}
	state.Stock.Cards = nil
	state.Waste.Cards = nil

	// Fill foundations 1-3 completely (Hearts, Diamonds, Clubs).
	for fi, suit := range []engine.Suit{engine.Hearts, engine.Diamonds, engine.Clubs} {
		state.Foundations[fi+1].Cards = nil
		for rank := engine.Ace; rank <= engine.King; rank++ {
			state.Foundations[fi+1].Cards = append(state.Foundations[fi+1].Cards,
				engine.Card{Suit: suit, Rank: rank, FaceUp: true})
		}
	}

	// Fill foundation 0 with Ace-Queen of Spades (12 cards — one short).
	state.Foundations[0].Cards = nil
	for rank := engine.Ace; rank <= engine.Queen; rank++ {
		state.Foundations[0].Cards = append(state.Foundations[0].Cards,
			engine.Card{Suit: engine.Spades, Rank: rank, FaceUp: true})
	}

	// Place the King of Spades face-up on waste — the final winning card.
	state.Waste.Cards = []engine.Card{{Suit: engine.Spades, Rank: engine.King, FaceUp: true}}

	board.cursor.Pile = engine.PileWaste
	board.cursor.CardIndex = 0

	_, cmd := board.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd == nil {
		t.Fatal("winning move must return a non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(GameWonMsg); !ok {
		t.Errorf("winning move cmd must emit GameWonMsg, got %T", msg)
	}
}

// TestBoardMouseClickMovesAndSelects verifies that a left-click mouse event
// translates coordinates via PileHitTestWithWidth and runs select on the
// clicked pile, not the current keyboard cursor position.
func TestBoardMouseClickMovesAndSelects(t *testing.T) {
	board, eng := newBoard()
	wasteBefore := len(eng.State().Waste.Cards)

	// Cursor starts somewhere other than stock.
	board.cursor.Pile = engine.PileTableau3
	board.cursor.CardIndex = 0

	// Click at the stock pile coordinates (X=1, Y=2 is inside the stock region).
	// Stock is at origin (0, 2) with CardWidth=9, CardHeight=7.
	// Clicking stock while drag is false runs flipStock (not drag pick-up).
	updated, _ := board.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      1,
		Y:      2,
	})
	board = updated.(BoardModel)

	if board.cursor.Pile != engine.PileStock {
		t.Errorf("mouse click on stock must move cursor to PileStock, got %v", board.cursor.Pile)
	}
	if len(eng.State().Waste.Cards) <= wasteBefore {
		t.Error("mouse click on stock must flip a card to waste")
	}
}

// TestBoardMouseClickOutsidePile verifies that a left-click outside any pile
// region is silently ignored and leaves cursor and game state unchanged.
func TestBoardMouseClickOutsidePile(t *testing.T) {
	board, eng := newBoard()
	board.cursor.Pile = engine.PileStock
	moveBefore := eng.State().MoveCount

	// Click far outside any pile region.
	updated, _ := board.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      79,
		Y:      29,
	})
	board = updated.(BoardModel)

	if board.cursor.Pile != engine.PileStock {
		t.Errorf("miss-click must not change cursor pile, got %v", board.cursor.Pile)
	}
	if eng.State().MoveCount != moveBefore {
		t.Error("miss-click must not change engine state")
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

// tableauCardY returns the terminal Y coordinate for cardIdx within tableau
// column col. tabRow is the Y origin of the tableau row.
func tableauCardY(state *engine.GameState, col, cardIdx, tabRow int) int {
	pile := state.Tableau[col]
	fdCount := pile.FaceDownCount()
	row := tabRow
	// Face-down stubs occupy exactly 1 row each.
	if cardIdx < fdCount {
		return row + cardIdx
	}
	row += fdCount
	// Face-up cards: all but last occupy 2 rows; last occupies CardHeight rows.
	fuCards := pile.FaceUpCards()
	for fi := range fuCards {
		height := 2
		if fi == len(fuCards)-1 {
			height = renderer.CardHeight
		}
		ci := fdCount + fi
		if ci == cardIdx {
			return row + height/2
		}
		row += height
	}
	return tabRow // fallback: top of column
}

// clickPile delivers a left-press mouse click aimed at a specific card within
// pileID and returns the updated BoardModel. Coordinates are derived from the
// renderer layout constants so the hit-test inside BoardModel.Update resolves
// to the intended pile and card index.
//
// Layout reference (mirrors renderer.pileOrigins):
//
//	topRow = 2
//	tabRow = topRow + renderer.CardHeight + 1 = 10
func clickPile(board BoardModel, pileID engine.PileID, cardIdx int) BoardModel {
	const (
		topRow = 2
		tabRow = topRow + renderer.CardHeight + 1 // 10
	)

	state := board.eng.State()
	var x, y int

	switch {
	case pileID == engine.PileStock:
		x = renderer.CardWidth / 2
		y = topRow + renderer.CardHeight/2

	case pileID == engine.PileWaste:
		x = renderer.CardWidth + renderer.ColGap + renderer.CardWidth/2
		y = topRow + renderer.CardHeight/2

	case isFoundationPile(pileID):
		fi := int(pileID - engine.PileFoundation0)
		fStartX := board.width - (4*renderer.CardWidth + 3*renderer.ColGap)
		x = fStartX + fi*(renderer.CardWidth+renderer.ColGap) + renderer.CardWidth/2
		y = topRow + renderer.CardHeight/2

	case isTableauPile(pileID):
		col := int(pileID - engine.PileTableau0)
		x = col*(renderer.CardWidth+renderer.ColGap) + renderer.CardWidth/2
		y = tableauCardY(state, col, cardIdx, tabRow)
	}

	updated, _ := board.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      x,
		Y:      y,
	})
	return updated.(BoardModel)
}

// TestBoardMousePickUp verifies that a left-click on a face-up tableau card
// starts a drag (Dragging becomes true) with the correct DragSource.
func TestBoardMousePickUp(t *testing.T) {
	board, eng := newBoard()

	// T0 has exactly 1 card (0 face-down, 1 face-up) with seed 42.
	state := eng.State()
	if state.Tableau[0].FaceDownCount() != 0 || len(state.Tableau[0].Cards) == 0 {
		t.Skip("T0 layout does not match seed-42 expectation")
	}

	board = clickPile(board, engine.PileTableau0, 0)

	if !board.cursor.Dragging {
		t.Fatal("mouse click on face-up card must set Dragging=true")
	}
	if board.cursor.DragSource != engine.PileTableau0 {
		t.Errorf("DragSource: got %v, want PileTableau0", board.cursor.DragSource)
	}
	if board.cursor.DragCardCount < 1 {
		t.Errorf("DragCardCount must be >= 1, got %d", board.cursor.DragCardCount)
	}
}

// TestBoardMouseDragPlace_Valid verifies that two sequential mouse clicks
// (pick up then place) execute a valid tableau-to-tableau move, mirroring
// the keyboard-driven TestBoardDragPlace_Valid but via mouse events.
func TestBoardMouseDragPlace_Valid(t *testing.T) {
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

	// Click 1: pick up the source card(s).
	board = clickPile(board, move.From, srcCardIdx)
	if !board.cursor.Dragging {
		t.Fatal("first mouse click must pick up card (Dragging=true)")
	}

	// Click 2: place on destination.
	destCardIdx := naturalCardIndex(move.To, eng.State())
	board = clickPile(board, move.To, destCardIdx)

	if board.cursor.Dragging {
		t.Error("second mouse click must clear Dragging")
	}
	afterLen := len(eng.State().Tableau[srcCol].Cards)
	if afterLen >= srcLen {
		t.Errorf("source pile must shrink after valid move: before=%d after=%d", srcLen, afterLen)
	}
}

// TestBoardMouseDragPlace_FaceDownCard verifies that clicking a face-down
// tableau card does not start a drag (face-down cards are not legal drag sources).
func TestBoardMouseDragPlace_FaceDownCard(t *testing.T) {
	board, eng := newBoard()

	// Find any column with a face-down card.
	targetCol := -1
	for col := 0; col < 7; col++ {
		if eng.State().Tableau[col].FaceDownCount() > 0 {
			targetCol = col
			break
		}
	}
	if targetCol < 0 {
		t.Skip("no face-down cards at deal time")
	}

	pileID := engine.PileTableau0 + engine.PileID(targetCol)
	board = clickPile(board, pileID, 0) // cardIdx=0 is always face-down in a fresh deal

	if board.cursor.Dragging {
		t.Error("mouse click on face-down card must not start a drag")
	}
	if board.cursor.DragCardCount != 0 {
		t.Errorf("DragCardCount must be 0 after no-op, got %d", board.cursor.DragCardCount)
	}
}

// ── T17: Auto-Complete + Auto-Move ───────────────────────────────────────────

// newNearWonBoard creates a BoardModel where only 4 Kings remain (one per suit,
// face-up in tableau[0..3]). All other 48 cards are on their foundations.
// IsAutoCompletable() == true, IsWon() == false.
func newNearWonBoard() (BoardModel, *testEngine) {
	state := &engine.GameState{
		Stock:     &engine.StockPile{},
		Waste:     &engine.WastePile{DrawCount: 1},
		DrawCount: 1,
	}
	suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
	for i := range state.Foundations {
		state.Foundations[i] = &engine.FoundationPile{}
	}
	for i := range state.Tableau {
		state.Tableau[i] = &engine.TableauPile{}
	}
	// Fill foundations Ace–Queen for each suit.
	for fi, suit := range suits {
		for r := engine.Ace; r <= engine.Queen; r++ {
			state.Foundations[fi].Cards = append(state.Foundations[fi].Cards,
				engine.Card{Suit: suit, Rank: r, FaceUp: true})
		}
	}
	// Place each King face-up in its own tableau column.
	for col, suit := range suits {
		state.Tableau[col].Cards = []engine.Card{
			{Suit: suit, Rank: engine.King, FaceUp: true},
		}
	}

	eng := &testEngine{state: state}
	rend := renderer.New(theme.Classic)
	rend.SetSize(80, 30)
	cfg := config.DefaultConfig()
	board := NewBoardModel(eng, rend, cfg)
	return board, eng
}

// TestBoardAutoCompleteInterruptByKeypress verifies that any keypress while
// autoCompleting is true clears the flag and returns a nil cmd.
func TestBoardAutoCompleteInterruptByKeypress(t *testing.T) {
	board, _ := newNearWonBoard()
	board.autoCompleting = true

	updated, cmd := board.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	board = updated.(BoardModel)

	if board.autoCompleting {
		t.Error("keypress must clear autoCompleting")
	}
	if cmd != nil {
		t.Error("interrupt must return nil cmd (no further ticks)")
	}
}

// TestBoardAutoCompleteStep_MovesToFoundation verifies that one AutoCompleteStepMsg
// moves exactly one card from tableau to a foundation.
func TestBoardAutoCompleteStep_MovesToFoundation(t *testing.T) {
	board, eng := newNearWonBoard()
	board.autoCompleting = true

	before := 0
	for _, f := range eng.State().Foundations {
		before += len(f.Cards)
	}

	updated, _ := board.Update(AutoCompleteStepMsg{})
	_ = updated

	after := 0
	for _, f := range eng.State().Foundations {
		after += len(f.Cards)
	}
	if after != before+1 {
		t.Errorf("one AutoCompleteStepMsg must move exactly 1 card to foundation: before=%d after=%d", before, after)
	}
}

// TestBoardAutoCompleteStep_EmitsGameWonMsg verifies that the final auto-complete
// step emits GameWonMsg and clears autoCompleting.
func TestBoardAutoCompleteStep_EmitsGameWonMsg(t *testing.T) {
	// Build a state with only the King of Spades remaining.
	state := &engine.GameState{
		Stock:     &engine.StockPile{},
		Waste:     &engine.WastePile{DrawCount: 1},
		DrawCount: 1,
	}
	suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
	for i := range state.Foundations {
		state.Foundations[i] = &engine.FoundationPile{}
	}
	for i := range state.Tableau {
		state.Tableau[i] = &engine.TableauPile{}
	}
	// All suits complete (Ace–King), except Spades stops at Queen.
	for fi, suit := range suits {
		limit := engine.King
		if suit == engine.Spades {
			limit = engine.Queen
		}
		for r := engine.Ace; r <= limit; r++ {
			state.Foundations[fi].Cards = append(state.Foundations[fi].Cards,
				engine.Card{Suit: suit, Rank: r, FaceUp: true})
		}
	}
	// The lone King of Spades sits face-up in tableau[0].
	state.Tableau[0].Cards = []engine.Card{
		{Suit: engine.Spades, Rank: engine.King, FaceUp: true},
	}

	eng := &testEngine{state: state}
	rend := renderer.New(theme.Classic)
	rend.SetSize(80, 30)
	board := NewBoardModel(eng, rend, config.DefaultConfig())
	board.autoCompleting = true

	updated, cmd := board.Update(AutoCompleteStepMsg{})
	board = updated.(BoardModel)

	if cmd == nil {
		t.Fatal("final auto-complete step must return a non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(GameWonMsg); !ok {
		t.Errorf("final step must emit GameWonMsg, got %T", msg)
	}
	if board.autoCompleting {
		t.Error("autoCompleting must be false after game is won")
	}
}

// TestBoardAutoMove_MovesCardAfterAction verifies that with AutoMoveEnabled = true
// a safe tableau card is automatically moved to its foundation after a player action.
func TestBoardAutoMove_MovesCardAfterAction(t *testing.T) {
	// State: four Aces on foundations; 2♠ (safe) and 3♥ (not yet safe) in tableau.
	state := &engine.GameState{
		Stock:     &engine.StockPile{},
		Waste:     &engine.WastePile{DrawCount: 1},
		DrawCount: 1,
	}
	for i := range state.Foundations {
		state.Foundations[i] = &engine.FoundationPile{}
	}
	for i := range state.Tableau {
		state.Tableau[i] = &engine.TableauPile{}
	}
	suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
	for fi, suit := range suits {
		state.Foundations[fi].Cards = []engine.Card{
			{Suit: suit, Rank: engine.Ace, FaceUp: true},
		}
	}
	// 2♠ is safe: both Red Aces (Ace >= 2-1=1) are on foundations.
	state.Tableau[0].Cards = []engine.Card{
		{Suit: engine.Spades, Rank: engine.Two, FaceUp: true},
	}
	// 3♥ is NOT yet safe: Clubs foundation only has Ace (rank 1 < 3-1=2).
	state.Tableau[1].Cards = []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Three, FaceUp: true},
	}

	eng := &testEngine{state: state}
	rend := renderer.New(theme.Classic)
	rend.SetSize(80, 30)
	cfg := config.DefaultConfig()
	cfg.AutoMoveEnabled = true
	board := NewBoardModel(eng, rend, cfg)

	before := 0
	for _, f := range eng.State().Foundations {
		before += len(f.Cards)
	}

	// Any player action triggers auto-move at end of handleAction.
	board = sendKey(board, tea.KeyLeft)

	after := 0
	for _, f := range eng.State().Foundations {
		after += len(f.Cards)
	}
	if after != before+1 {
		t.Errorf("2♠ must be auto-moved after action: before=%d after=%d", before, after)
	}
	// 3♥ must not have been moved: Clubs only has Ace (rank 1 < 2).
	if eng.State().Tableau[1].IsEmpty() {
		t.Error("3♥ must not be auto-moved yet (opposite-color min rank insufficient)")
	}
}

// TestBoardAutoMove_DisabledDoesNotMove verifies that with AutoMoveEnabled = false
// (the default) safe cards remain in place after a player action.
func TestBoardAutoMove_DisabledDoesNotMove(t *testing.T) {
	state := &engine.GameState{
		Stock:     &engine.StockPile{},
		Waste:     &engine.WastePile{DrawCount: 1},
		DrawCount: 1,
	}
	for i := range state.Foundations {
		state.Foundations[i] = &engine.FoundationPile{}
	}
	for i := range state.Tableau {
		state.Tableau[i] = &engine.TableauPile{}
	}
	suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
	for fi, suit := range suits {
		state.Foundations[fi].Cards = []engine.Card{
			{Suit: suit, Rank: engine.Ace, FaceUp: true},
		}
	}
	state.Tableau[0].Cards = []engine.Card{
		{Suit: engine.Spades, Rank: engine.Two, FaceUp: true},
	}

	eng := &testEngine{state: state}
	rend := renderer.New(theme.Classic)
	rend.SetSize(80, 30)
	cfg := config.DefaultConfig() // AutoMoveEnabled = false
	board := NewBoardModel(eng, rend, cfg)

	before := len(eng.State().Foundations[0].Cards) // Spades foundation: 1 card

	board = sendKey(board, tea.KeyLeft)

	after := len(eng.State().Foundations[0].Cards)
	if after != before {
		t.Errorf("auto-move disabled: Spades foundation must not grow: before=%d after=%d", before, after)
	}
}
