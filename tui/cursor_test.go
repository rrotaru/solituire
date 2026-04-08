package tui

import (
	"testing"

	"solituire/engine"
)

// newSeed42State builds the canonical seed-42 draw-1 game state used across cursor tests.
func newSeed42State() *engine.GameState {
	deck := engine.NewDeck()
	engine.Shuffle(deck, 42)
	state := engine.Deal(deck, 1)
	state.Seed = 42
	return state
}

// TestCursorMoveLeft_FromTableau0 verifies that pressing left from the first
// tableau column moves the cursor to the waste pile.
func TestCursorMoveLeft_FromTableau0(t *testing.T) {
	state := newSeed42State()
	c := Cursor{Pile: engine.PileTableau0, CardIndex: 0}
	c.MoveLeft(state)
	if c.Pile != engine.PileWaste {
		t.Errorf("expected PileWaste, got %v", c.Pile)
	}
}

// TestCursorMoveRight_FromTableau6 verifies that pressing right from the last
// tableau column wraps around to the stock pile.
func TestCursorMoveRight_FromTableau6(t *testing.T) {
	state := newSeed42State()
	c := Cursor{Pile: engine.PileTableau6, CardIndex: 0}
	c.MoveRight(state)
	if c.Pile != engine.PileStock {
		t.Errorf("expected PileStock (wrap), got %v", c.Pile)
	}
}

// TestCursorMoveLeft_FromStock verifies that pressing left from the stock wraps
// to the last tableau column.
func TestCursorMoveLeft_FromStock(t *testing.T) {
	state := newSeed42State()
	c := Cursor{Pile: engine.PileStock, CardIndex: 0}
	c.MoveLeft(state)
	if c.Pile != engine.PileTableau6 {
		t.Errorf("expected PileTableau6 (wrap), got %v", c.Pile)
	}
}

// TestCursorMoveRight_FromWaste verifies that pressing right from the waste pile
// moves to the first tableau column.
func TestCursorMoveRight_FromWaste(t *testing.T) {
	state := newSeed42State()
	c := Cursor{Pile: engine.PileWaste, CardIndex: 0}
	c.MoveRight(state)
	if c.Pile != engine.PileTableau0 {
		t.Errorf("expected PileTableau0, got %v", c.Pile)
	}
}

// TestCursorTabNext_Order verifies that Tab from stock visits all piles in the
// expected order: Waste → F0 → F1 → F2 → F3 → T0 → ... → T6 → (wrap to Stock).
func TestCursorTabNext_Order(t *testing.T) {
	state := newSeed42State()
	expected := tabCycleOrder // defined in cursor.go

	c := Cursor{Pile: engine.PileStock}
	for i := 1; i < len(expected); i++ {
		c.TabNext(state)
		if c.Pile != expected[i] {
			t.Errorf("Tab step %d: expected pile %v, got %v", i, expected[i], c.Pile)
		}
	}
	// One more Tab should wrap back to Stock.
	c.TabNext(state)
	if c.Pile != engine.PileStock {
		t.Errorf("Tab wrap: expected PileStock, got %v", c.Pile)
	}
}

// TestCursorTabPrev_Order verifies that Shift-Tab from stock visits piles in
// reverse order.
func TestCursorTabPrev_Order(t *testing.T) {
	state := newSeed42State()
	expected := tabCycleOrder

	c := Cursor{Pile: engine.PileStock}
	// Shift-Tab from Stock should land on T6 (last in cycle).
	c.TabPrev(state)
	if c.Pile != expected[len(expected)-1] {
		t.Errorf("Shift-Tab from Stock: expected %v, got %v", expected[len(expected)-1], c.Pile)
	}
}

// TestCursorMoveDown_Tableau verifies that pressing down in a tableau column
// with multiple face-up cards increases CardIndex.
func TestCursorMoveDown_Tableau(t *testing.T) {
	state := newSeed42State()
	// Manually make the last two cards of tableau 6 face-up so there is room
	// to move down within the face-up region.
	col := 6
	pile := state.Tableau[col]
	if len(pile.Cards) < 2 {
		t.Skip("tableau 6 needs at least 2 cards")
	}
	// Flip the second-to-last card face-up.
	pile.Cards[len(pile.Cards)-2].FaceUp = true
	fdCount := pile.FaceDownCount()
	// Start cursor at the first face-up card (not the last).
	c := Cursor{Pile: engine.PileTableau6, CardIndex: fdCount}
	c.MoveDown(state)
	if c.CardIndex != fdCount+1 {
		t.Errorf("expected CardIndex %d, got %d", fdCount+1, c.CardIndex)
	}
}

// TestCursorMoveUp_ClampsAtFaceUp verifies that pressing up in a tableau column
// cannot enter the face-down region (CardIndex must stay >= FaceDownCount).
func TestCursorMoveUp_ClampsAtFaceUp(t *testing.T) {
	state := newSeed42State()
	// Use tableau 3 (4 cards: 3 face-down + 1 face-up).
	col := 3
	pile := state.Tableau[col]
	fdCount := pile.FaceDownCount()
	if fdCount == 0 {
		t.Skip("tableau 3 has no face-down cards with this seed")
	}
	c := Cursor{Pile: engine.PileTableau3, CardIndex: fdCount}
	c.MoveUp(state) // attempt to go into face-down region
	if c.CardIndex < fdCount {
		t.Errorf("cursor entered face-down region: CardIndex %d < FaceDownCount %d", c.CardIndex, fdCount)
	}
}

// TestCursorMoveUp_NotTableau verifies that pressing up on a non-tableau pile is
// a no-op.
func TestCursorMoveUp_NotTableau(t *testing.T) {
	state := newSeed42State()
	for _, pile := range []engine.PileID{engine.PileStock, engine.PileWaste, engine.PileFoundation0} {
		c := Cursor{Pile: pile, CardIndex: 0}
		c.MoveUp(state)
		if c.CardIndex != 0 || c.Pile != pile {
			t.Errorf("MoveUp on non-tableau pile %v changed cursor state", pile)
		}
	}
}

// TestCursorMoveDown_NotTableau verifies that pressing down on a non-tableau pile
// is a no-op.
func TestCursorMoveDown_NotTableau(t *testing.T) {
	state := newSeed42State()
	for _, pile := range []engine.PileID{engine.PileStock, engine.PileWaste, engine.PileFoundation1} {
		c := Cursor{Pile: pile, CardIndex: 0}
		c.MoveDown(state)
		if c.CardIndex != 0 || c.Pile != pile {
			t.Errorf("MoveDown on non-tableau pile %v changed cursor state", pile)
		}
	}
}

// TestCursorJumpToColumn verifies that JumpToColumn positions the cursor at the
// bottom card of the target tableau column.
func TestCursorJumpToColumn(t *testing.T) {
	state := newSeed42State()
	for col := 0; col < 7; col++ {
		c := Cursor{Pile: engine.PileStock, CardIndex: 0}
		c.JumpToColumn(col, state)
		wantPile := engine.PileTableau0 + engine.PileID(col)
		if c.Pile != wantPile {
			t.Errorf("col %d: expected pile %v, got %v", col, wantPile, c.Pile)
		}
		pile := state.Tableau[col]
		wantIdx := len(pile.Cards) - 1
		if pile.IsEmpty() {
			wantIdx = 0
		}
		if c.CardIndex != wantIdx {
			t.Errorf("col %d: expected CardIndex %d, got %d", col, wantIdx, c.CardIndex)
		}
	}
}

// TestCursorMoveLeft_FromFoundation verifies that pressing left from any foundation
// lands on Waste — not a random fallback from an absent navCycleOrder entry.
func TestCursorMoveLeft_FromFoundation(t *testing.T) {
	state := newSeed42State()
	foundations := []engine.PileID{
		engine.PileFoundation0,
		engine.PileFoundation1,
		engine.PileFoundation2,
		engine.PileFoundation3,
	}
	for _, pile := range foundations {
		c := Cursor{Pile: pile, CardIndex: 0}
		c.MoveLeft(state)
		if c.Pile != engine.PileWaste {
			t.Errorf("MoveLeft from %v: expected PileWaste, got %v", pile, c.Pile)
		}
	}
}

// TestCursorMoveRight_FromFoundation verifies that pressing right from any foundation
// lands on Tableau0 — not a random fallback from an absent navCycleOrder entry.
func TestCursorMoveRight_FromFoundation(t *testing.T) {
	state := newSeed42State()
	foundations := []engine.PileID{
		engine.PileFoundation0,
		engine.PileFoundation1,
		engine.PileFoundation2,
		engine.PileFoundation3,
	}
	for _, pile := range foundations {
		c := Cursor{Pile: pile, CardIndex: 0}
		c.MoveRight(state)
		if c.Pile != engine.PileTableau0 {
			t.Errorf("MoveRight from %v: expected PileTableau0, got %v", pile, c.Pile)
		}
	}
}

// TestRendererCursor verifies that RendererCursor correctly maps all fields.
func TestRendererCursor(t *testing.T) {
	c := Cursor{
		Pile:      engine.PileTableau3,
		CardIndex: 2,
		Dragging:  true,
		ShowHint:  true,
		HintFrom:  engine.PileWaste,
		HintTo:    engine.PileFoundation1,
	}
	rc := c.RendererCursor()
	if rc.Pile != c.Pile {
		t.Errorf("Pile: got %v, want %v", rc.Pile, c.Pile)
	}
	if rc.CardIndex != c.CardIndex {
		t.Errorf("CardIndex: got %d, want %d", rc.CardIndex, c.CardIndex)
	}
	if rc.Dragging != c.Dragging {
		t.Errorf("Dragging: got %v, want %v", rc.Dragging, c.Dragging)
	}
	if rc.ShowHint != c.ShowHint {
		t.Errorf("ShowHint: got %v, want %v", rc.ShowHint, c.ShowHint)
	}
	if rc.HintFrom != c.HintFrom {
		t.Errorf("HintFrom: got %v, want %v", rc.HintFrom, c.HintFrom)
	}
	if rc.HintTo != c.HintTo {
		t.Errorf("HintTo: got %v, want %v", rc.HintTo, c.HintTo)
	}
}

// TestCursorJumpToColumn_OutOfRange verifies that out-of-range column values are
// silently ignored and leave the cursor unchanged.
func TestCursorJumpToColumn_OutOfRange(t *testing.T) {
	state := newSeed42State()
	for _, col := range []int{-1, 7, 100, -100} {
		c := Cursor{Pile: engine.PileStock, CardIndex: 0}
		c.JumpToColumn(col, state)
		if c.Pile != engine.PileStock || c.CardIndex != 0 {
			t.Errorf("JumpToColumn(%d): cursor must not change, got pile=%v cardIndex=%d",
				col, c.Pile, c.CardIndex)
		}
	}
}

// TestNaturalCardIndex verifies naturalCardIndex returns the last card index for
// non-empty tableau piles and 0 for all others.
func TestNaturalCardIndex(t *testing.T) {
	state := newSeed42State()
	// Non-empty tableau column 6 (7 cards, indices 0-6).
	idx := naturalCardIndex(engine.PileTableau6, state)
	want := len(state.Tableau[6].Cards) - 1
	if idx != want {
		t.Errorf("T6: expected %d, got %d", want, idx)
	}
	// Stock: always 0.
	if naturalCardIndex(engine.PileStock, state) != 0 {
		t.Errorf("Stock: expected 0")
	}
	// Foundation: always 0.
	if naturalCardIndex(engine.PileFoundation0, state) != 0 {
		t.Errorf("Foundation0: expected 0")
	}
}
