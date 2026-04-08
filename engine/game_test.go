package engine

import (
	"reflect"
	"testing"
)

// newTestGame returns a Game created from a fresh deal with seed 42, draw-1.
func newTestGame() *Game {
	return NewGame(42, 1)
}

// TestGame_NewGame_InvalidDrawCount verifies that out-of-range drawCount values are
// normalized to 1, preventing FlipStockCmd from panicking on state.DrawCount.
func TestGame_NewGame_InvalidDrawCount(t *testing.T) {
	for _, bad := range []int{0, -1, 2, 4, 100} {
		g := NewGame(42, bad)
		if g.state.DrawCount != 1 {
			t.Errorf("drawCount=%d: GameState.DrawCount = %d, want 1", bad, g.state.DrawCount)
		}
		if g.state.Waste.DrawCount != 1 {
			t.Errorf("drawCount=%d: WastePile.DrawCount = %d, want 1", bad, g.state.Waste.DrawCount)
		}
		// Verify FlipStockCmd doesn't panic with the normalized state.
		if err := g.Execute(&FlipStockCmd{}); err != nil {
			t.Errorf("drawCount=%d: FlipStockCmd returned unexpected error: %v", bad, err)
		}
	}
}

// --- NewGame / initial state ---

func TestGame_NewGame_Layout(t *testing.T) {
	g := newTestGame()
	s := g.State()

	// Tableau columns 0-6 must have 1-7 cards.
	for col := 0; col < 7; col++ {
		want := col + 1
		if got := len(s.Tableau[col].Cards); got != want {
			t.Errorf("tableau[%d] has %d cards, want %d", col, got, want)
		}
		// Top card must be face-up; all others face-down.
		cards := s.Tableau[col].Cards
		for i, c := range cards {
			wantUp := i == len(cards)-1
			if c.FaceUp != wantUp {
				t.Errorf("tableau[%d][%d].FaceUp = %v, want %v", col, i, c.FaceUp, wantUp)
			}
		}
	}

	// Stock has 24 cards (52 - 1 - 2 - 3 - 4 - 5 - 6 - 7 = 24).
	if got := len(s.Stock.Cards); got != 24 {
		t.Errorf("stock has %d cards, want 24", got)
	}
	// Waste and foundations empty.
	if got := len(s.Waste.Cards); got != 0 {
		t.Errorf("waste has %d cards, want 0", got)
	}
	for i, f := range s.Foundations {
		if len(f.Cards) != 0 {
			t.Errorf("foundation[%d] has %d cards, want 0", i, len(f.Cards))
		}
	}

	// Metadata.
	if g.Score() != 0 {
		t.Errorf("initial score = %d, want 0", g.Score())
	}
	if g.MoveCount() != 0 {
		t.Errorf("initial MoveCount = %d, want 0", g.MoveCount())
	}
	if g.Seed() != 42 {
		t.Errorf("Seed = %d, want 42", g.Seed())
	}
	if g.CanUndo() {
		t.Error("CanUndo should be false on fresh game")
	}
	if g.CanRedo() {
		t.Error("CanRedo should be false on fresh game")
	}
}

// --- IsWon ---

func TestGame_IsWon_False(t *testing.T) {
	g := newTestGame()
	if g.IsWon() {
		t.Error("IsWon should be false for a freshly dealt game")
	}
}

func TestGame_IsWon_True(t *testing.T) {
	g := newTestGame()
	// Fill all four foundations with 13 cards each.
	for i := range g.state.Foundations {
		g.state.Foundations[i].Cards = make([]Card, 13)
	}
	if !g.IsWon() {
		t.Error("IsWon should be true when all foundations have 13 cards")
	}
}

// --- IsAutoCompletable ---

func TestGame_IsAutoCompletable_False_FaceDown(t *testing.T) {
	g := newTestGame()
	// Fresh game always has face-down cards in tableau.
	if g.IsAutoCompletable() {
		t.Error("IsAutoCompletable should be false when face-down tableau cards exist")
	}
}

func TestGame_IsAutoCompletable_False_StockNotEmpty(t *testing.T) {
	g := newTestGame()
	// Clear all face-down tableau cards but leave stock non-empty.
	for _, t2 := range g.state.Tableau {
		for j := range t2.Cards {
			t2.Cards[j].FaceUp = true
		}
	}
	// Stock still has 24 cards.
	if g.IsAutoCompletable() {
		t.Error("IsAutoCompletable should be false when stock is non-empty")
	}
}

func TestGame_IsAutoCompletable_True(t *testing.T) {
	g := newTestGame()
	// Flip all tableau cards and drain the stock.
	for _, t2 := range g.state.Tableau {
		for j := range t2.Cards {
			t2.Cards[j].FaceUp = true
		}
	}
	g.state.Stock.Cards = nil
	if !g.IsAutoCompletable() {
		t.Error("IsAutoCompletable should be true when stock empty and no face-down tableau cards")
	}
}

// --- Execute: basic move + score ---

func TestGame_Execute_SimpleTableauMove(t *testing.T) {
	state := buildState()
	// K♠ (black) on T0, Q♥ (red) on T1 — Q♥ can land on K♠.
	state.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
	state.Tableau[1].Cards = []Card{faceUpCard(Queen, Hearts)}
	g := &Game{state: state}

	cmd := &MoveCardCmd{From: PileTableau1, To: PileTableau0, CardCount: 1}
	if err := g.Execute(cmd); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if g.MoveCount() != 1 {
		t.Errorf("MoveCount = %d, want 1", g.MoveCount())
	}
	// Tableau→Tableau scores 0.
	if g.Score() != 0 {
		t.Errorf("Score = %d, want 0", g.Score())
	}
	// T0 should have [K♠, Q♥], T1 should be empty.
	if len(state.Tableau[0].Cards) != 2 {
		t.Errorf("T0 len = %d, want 2", len(state.Tableau[0].Cards))
	}
	if len(state.Tableau[1].Cards) != 0 {
		t.Errorf("T1 len = %d, want 0", len(state.Tableau[1].Cards))
	}
}

func TestGame_Execute_ScoreWasteToTableau(t *testing.T) {
	state := buildState()
	state.Waste.Cards = []Card{faceUpCard(Nine, Spades)}
	state.Tableau[0].Cards = []Card{faceUpCard(Ten, Hearts)}
	g := &Game{state: state}

	if err := g.Execute(&MoveCardCmd{From: PileWaste, To: PileTableau0, CardCount: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if g.Score() != 5 {
		t.Errorf("Score = %d, want 5 (Waste→Tableau)", g.Score())
	}
}

func TestGame_Execute_ScoreToFoundation(t *testing.T) {
	state := buildState()
	state.Waste.Cards = []Card{faceUpCard(Ace, Spades)}
	// F0 is Spades foundation (empty, so accepts Ace).
	g := &Game{state: state}

	if err := g.Execute(&MoveToFoundationCmd{From: PileWaste, FoundationIdx: 0}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if g.Score() != 10 {
		t.Errorf("Score = %d, want 10 (Waste→Foundation)", g.Score())
	}
}

func TestGame_Execute_ScoreTableauToFoundation(t *testing.T) {
	state := buildState()
	state.Tableau[0].Cards = []Card{faceUpCard(Ace, Hearts)}
	// F1 is Hearts foundation (empty).
	g := &Game{state: state}

	if err := g.Execute(&MoveToFoundationCmd{From: PileTableau0, FoundationIdx: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if g.Score() != 10 {
		t.Errorf("Score = %d, want 10 (Tableau→Foundation)", g.Score())
	}
}

func TestGame_Execute_ScoreFloor(t *testing.T) {
	// Foundation→Tableau costs −15. If current score is 10, result should be 0, not −5.
	state := buildState()
	// F1 (Hearts) has [A♥, 2♥]; T0 has [3♠(up)] — 2♥ can go onto 3♠.
	state.Foundations[1].Cards = []Card{faceUpCard(Ace, Hearts), faceUpCard(Two, Hearts)}
	state.Tableau[0].Cards = []Card{faceUpCard(Three, Spades)}
	state.Score = 10
	g := &Game{state: state}

	if err := g.Execute(&MoveCardCmd{From: PileFoundation1, To: PileTableau0, CardCount: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if g.Score() != 0 {
		t.Errorf("Score = %d, want 0 (floor at 0, not negative)", g.Score())
	}
}

func TestGame_Execute_Invalid(t *testing.T) {
	state := buildState()
	// Two empty tableau columns; can't move nothing.
	g := &Game{state: state}

	err := g.Execute(&MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1})
	if err == nil {
		t.Error("Execute on invalid move should return error")
	}
	if g.MoveCount() != 0 {
		t.Errorf("MoveCount = %d after failed Execute, want 0", g.MoveCount())
	}
	if g.Score() != 0 {
		t.Errorf("Score = %d after failed Execute, want 0", g.Score())
	}
}

// --- Auto-flip ---

func TestGame_Execute_AutoFlip(t *testing.T) {
	// Moving Q♥ off T0 exposes K♠ (face-down) → auto-flip fires.
	state := buildState()
	state.Tableau[0].Cards = []Card{faceDownCard(King, Spades), faceUpCard(Queen, Hearts)}
	state.Tableau[1].Cards = []Card{faceUpCard(King, Clubs)}
	g := &Game{state: state}

	if err := g.Execute(&MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// K♠ must now be face-up.
	if !state.Tableau[0].Cards[0].FaceUp {
		t.Error("auto-flip: K♠ should be face-up after Q♥ moves away")
	}
	// Score: tableau→tableau (0) + flip tableau (+5) = 5.
	if g.Score() != 5 {
		t.Errorf("Score = %d, want 5 (auto-flip +5)", g.Score())
	}
	// History recorded as compound: undo should be available.
	if !g.CanUndo() {
		t.Error("CanUndo should be true after execute with auto-flip")
	}
}

func TestGame_Execute_AutoFlip_UndoRevertsBoth(t *testing.T) {
	// Verify that undoing an auto-flip compound cmd reverses both the move and the flip.
	state := buildState()
	state.Tableau[0].Cards = []Card{faceDownCard(King, Spades), faceUpCard(Queen, Hearts)}
	state.Tableau[1].Cards = []Card{faceUpCard(King, Clubs)}
	g := &Game{state: state}

	if err := g.Execute(&MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if err := g.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}

	// Q♥ must be back on T0; K♠ must be face-down again.
	if len(state.Tableau[0].Cards) != 2 {
		t.Fatalf("T0 has %d cards after undo, want 2", len(state.Tableau[0].Cards))
	}
	if state.Tableau[0].Cards[0].FaceUp {
		t.Error("K♠ should be face-down after undo")
	}
	if state.Tableau[0].Cards[1].Rank != Queen {
		t.Error("Q♥ should be back on T0 after undo")
	}
	// Score must be restored to 0.
	if g.Score() != 0 {
		t.Errorf("Score after undo = %d, want 0", g.Score())
	}
}

// --- Undo / Redo ---

func TestGame_Undo_RevertsState(t *testing.T) {
	state := buildState()
	// K♠ (black) on T0, Q♥ (red) on T1 — Q♥ can land on K♠.
	state.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
	state.Tableau[1].Cards = []Card{faceUpCard(Queen, Hearts)}
	g := &Game{state: state}

	pre0 := append([]Card{}, state.Tableau[0].Cards...)
	pre1 := append([]Card{}, state.Tableau[1].Cards...)

	if err := g.Execute(&MoveCardCmd{From: PileTableau1, To: PileTableau0, CardCount: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if err := g.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}

	if !reflect.DeepEqual(state.Tableau[0].Cards, pre0) {
		t.Errorf("T0 after undo = %v, want %v", state.Tableau[0].Cards, pre0)
	}
	if !reflect.DeepEqual(state.Tableau[1].Cards, pre1) {
		t.Errorf("T1 after undo = %v, want %v", state.Tableau[1].Cards, pre1)
	}
	if g.Score() != 0 {
		t.Errorf("Score after undo = %d, want 0", g.Score())
	}
	if g.CanUndo() {
		t.Error("CanUndo should be false after undoing all moves")
	}
}

func TestGame_Redo(t *testing.T) {
	state := buildState()
	state.Tableau[0].Cards = []Card{faceUpCard(King, Clubs)}
	state.Tableau[1].Cards = []Card{faceUpCard(Queen, Hearts)}
	g := &Game{state: state}

	if err := g.Execute(&MoveCardCmd{From: PileTableau1, To: PileTableau0, CardCount: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	postExec := append([]Card{}, state.Tableau[0].Cards...)

	if err := g.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if err := g.Redo(); err != nil {
		t.Fatalf("Redo: %v", err)
	}

	if !reflect.DeepEqual(state.Tableau[0].Cards, postExec) {
		t.Errorf("T0 after redo = %v, want %v", state.Tableau[0].Cards, postExec)
	}
	if g.CanRedo() {
		t.Error("CanRedo should be false after redo")
	}
}

func TestGame_Undo_Empty(t *testing.T) {
	g := newTestGame()
	if err := g.Undo(); err == nil {
		t.Error("Undo on fresh game should return error")
	}
}

func TestGame_Redo_Empty(t *testing.T) {
	g := newTestGame()
	if err := g.Redo(); err == nil {
		t.Error("Redo on fresh game should return error")
	}
}

// --- RestartDeal ---

func TestGame_RestartDeal(t *testing.T) {
	g := NewGame(99, 1)

	// Snapshot initial top cards of each tableau column.
	var initialTops [7]Card
	for i, pile := range g.state.Tableau {
		initialTops[i] = pile.Cards[len(pile.Cards)-1]
	}

	// Execute a stock flip to create some history.
	if err := g.Execute(&FlipStockCmd{}); err != nil {
		t.Fatalf("Execute FlipStockCmd: %v", err)
	}

	g.RestartDeal()

	if g.Score() != 0 {
		t.Errorf("Score after RestartDeal = %d, want 0", g.Score())
	}
	if g.MoveCount() != 0 {
		t.Errorf("MoveCount after RestartDeal = %d, want 0", g.MoveCount())
	}
	if g.CanUndo() {
		t.Error("CanUndo should be false after RestartDeal")
	}
	if g.Seed() != 99 {
		t.Errorf("Seed after RestartDeal = %d, want 99", g.Seed())
	}
	// Top cards must match the original deal.
	for i, pile := range g.state.Tableau {
		got := pile.Cards[len(pile.Cards)-1]
		if got != initialTops[i] {
			t.Errorf("tableau[%d] top after restart = %v, want %v", i, got, initialTops[i])
		}
	}
}

// --- ValidMoves / IsValidMove ---

func TestGame_ValidMoves_NotEmpty(t *testing.T) {
	g := newTestGame()
	// A freshly dealt game always has at least a stock flip available.
	moves := g.ValidMoves()
	if len(moves) == 0 {
		t.Error("ValidMoves returned empty slice for fresh game")
	}
}

func TestGame_IsValidMove_True(t *testing.T) {
	state := buildState()
	state.Tableau[0].Cards = []Card{faceUpCard(King, Clubs)}
	state.Waste.Cards = []Card{faceUpCard(Queen, Hearts)}
	g := &Game{state: state}

	m := Move{From: PileWaste, To: PileTableau0, CardCount: 1}
	if !g.IsValidMove(m) {
		t.Error("IsValidMove returned false for valid Waste→Tableau move")
	}
}

func TestGame_IsValidMove_False(t *testing.T) {
	state := buildState()
	// T1 is empty; Q♥ is not a King so it can't go there.
	state.Waste.Cards = []Card{faceUpCard(Queen, Hearts)}
	g := &Game{state: state}

	m := Move{From: PileWaste, To: PileTableau1, CardCount: 1}
	if g.IsValidMove(m) {
		t.Error("IsValidMove returned true for invalid Waste→empty-tableau move (non-King)")
	}
}

// --- MoveCount does not decrement on Undo ---

func TestGame_MoveCount_NotDecrementedOnUndo(t *testing.T) {
	state := buildState()
	state.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
	state.Tableau[1].Cards = []Card{faceUpCard(Queen, Hearts)}
	g := &Game{state: state}

	if err := g.Execute(&MoveCardCmd{From: PileTableau1, To: PileTableau0, CardCount: 1}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if g.MoveCount() != 1 {
		t.Fatalf("MoveCount = %d after execute, want 1", g.MoveCount())
	}
	if err := g.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	// MoveCount stays at 1 — it is a "moves made" counter, not a "moves in play" counter.
	if g.MoveCount() != 1 {
		t.Errorf("MoveCount = %d after undo, want 1 (not decremented)", g.MoveCount())
	}
}
