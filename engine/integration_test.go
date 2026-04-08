package engine

import "testing"

// moveToCmd converts a Move from ValidMoves into an executable Command.
// Stock moves are dispatched to FlipStockCmd or RecycleStockCmd based on current state.
func moveToCmd(m Move, state *GameState) Command {
	if m.From == PileStock {
		if state.Stock.IsEmpty() {
			return &RecycleStockCmd{}
		}
		return &FlipStockCmd{}
	}
	if isFoundationPile(m.To) {
		return &MoveToFoundationCmd{From: m.From, FoundationIdx: foundationIndex(m.To)}
	}
	return &MoveCardCmd{From: m.From, To: m.To, CardCount: m.CardCount}
}

// snapshotTableauTops returns the top card of each tableau column (zero Card if empty).
func snapshotTableauTops(state *GameState) [7]Card {
	var tops [7]Card
	for i, t := range state.Tableau {
		if !t.IsEmpty() {
			tops[i] = t.Cards[len(t.Cards)-1]
		}
	}
	return tops
}

// TestIntegration_Seed42_UndoAll executes up to 5 valid moves on a seed-42 game,
// then undoes all of them and asserts the state matches the original deal.
func TestIntegration_Seed42_UndoAll(t *testing.T) {
	g := NewGame(42, 1)

	initialTops := snapshotTableauTops(g.state)
	initialStockLen := len(g.state.Stock.Cards)
	initialWasteLen := len(g.state.Waste.Cards)

	const maxMoves = 5
	executed := 0
	for executed < maxMoves {
		moves := g.ValidMoves()
		if len(moves) == 0 {
			break
		}
		cmd := moveToCmd(moves[0], g.state)
		if err := g.Execute(cmd); err != nil {
			t.Fatalf("Execute move %d: %v", executed+1, err)
		}
		executed++
	}

	if executed == 0 {
		t.Fatal("no valid moves found on seed-42 game — test setup error")
	}
	if g.MoveCount() != executed {
		t.Errorf("MoveCount = %d, want %d", g.MoveCount(), executed)
	}

	// Undo all moves.
	for i := 0; i < executed; i++ {
		if err := g.Undo(); err != nil {
			t.Fatalf("Undo %d: %v", i+1, err)
		}
	}

	// State must match the original deal exactly.
	if len(g.state.Stock.Cards) != initialStockLen {
		t.Errorf("stock len after full undo = %d, want %d", len(g.state.Stock.Cards), initialStockLen)
	}
	if len(g.state.Waste.Cards) != initialWasteLen {
		t.Errorf("waste len after full undo = %d, want %d", len(g.state.Waste.Cards), initialWasteLen)
	}
	tops := snapshotTableauTops(g.state)
	for i, card := range tops {
		if card != initialTops[i] {
			t.Errorf("tableau[%d] top after full undo = %v, want %v", i, card, initialTops[i])
		}
	}
	if g.Score() != 0 {
		t.Errorf("score after full undo = %d, want 0", g.Score())
	}
	if g.CanUndo() {
		t.Error("CanUndo should be false after undoing all moves")
	}
}

// TestIntegration_ScoreAccumulates verifies that repeated moves accumulate score correctly
// and undo restores prior scores at each step.
func TestIntegration_ScoreAccumulates(t *testing.T) {
	// Construct a state where we can make two known scoring moves:
	//   1. Waste → Tableau (+5)
	//   2. Tableau → Foundation (+10) after the first move
	state := buildState()
	// T0 = [K♣(up), Q♥(up)]  — Q♥ will eventually go to foundation if suit matches
	// Actually simpler: Waste has A♥, T0 has 2♠. Move A♥ to waste then to foundation.
	// Let's do two foundation moves.
	state.Waste.Cards = []Card{faceUpCard(Ace, Spades)}
	state.Tableau[0].Cards = []Card{faceUpCard(Ace, Hearts)}
	// F0 = Spades (empty), F1 = Hearts (empty).
	g := &Game{state: state}

	// Move 1: Waste→Foundation (+10).
	if err := g.Execute(&MoveToFoundationCmd{From: PileWaste, FoundationIdx: 0}); err != nil {
		t.Fatalf("Execute move 1: %v", err)
	}
	if g.Score() != 10 {
		t.Errorf("score after move 1 = %d, want 10", g.Score())
	}

	// Move 2: Tableau→Foundation (+10).
	if err := g.Execute(&MoveToFoundationCmd{From: PileTableau0, FoundationIdx: 1}); err != nil {
		t.Fatalf("Execute move 2: %v", err)
	}
	if g.Score() != 20 {
		t.Errorf("score after move 2 = %d, want 20", g.Score())
	}

	// Undo move 2 → score restored to 10.
	if err := g.Undo(); err != nil {
		t.Fatalf("Undo move 2: %v", err)
	}
	if g.Score() != 10 {
		t.Errorf("score after undo move 2 = %d, want 10", g.Score())
	}

	// Undo move 1 → score restored to 0.
	if err := g.Undo(); err != nil {
		t.Fatalf("Undo move 1: %v", err)
	}
	if g.Score() != 0 {
		t.Errorf("score after undo move 1 = %d, want 0", g.Score())
	}
}

// TestIntegration_IsWon detects the won condition after moving the last cards to foundations.
func TestIntegration_IsWon(t *testing.T) {
	state := buildState()

	// Fill foundations[0] (Spades) with A-Q (12 cards); last card K♠ is in tableau.
	spades := make([]Card, 12)
	for r := Ace; r <= Queen; r++ {
		spades[r-1] = faceUpCard(r, Spades)
	}
	state.Foundations[0].Cards = spades

	// Fill other three foundations completely (13 cards each).
	for i := 1; i < 4; i++ {
		suit := Suit(i)
		full := make([]Card, 13)
		for r := Ace; r <= King; r++ {
			full[r-1] = faceUpCard(r, suit)
		}
		state.Foundations[i].Cards = full
	}

	// K♠ sits alone on T0.
	state.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}

	g := &Game{state: state}

	if g.IsWon() {
		t.Error("IsWon should be false before the final move")
	}

	// Move K♠ to foundation[0].
	if err := g.Execute(&MoveToFoundationCmd{From: PileTableau0, FoundationIdx: 0}); err != nil {
		t.Fatalf("Execute final move: %v", err)
	}

	if !g.IsWon() {
		t.Error("IsWon should be true after all foundations are complete")
	}
}

// TestIntegration_RecycleStock verifies recycle → flip cycle scores correctly.
func TestIntegration_RecycleStock(t *testing.T) {
	state := buildState()
	// Put one card in waste, stock empty → recycle should work.
	state.Waste.Cards = []Card{faceUpCard(Seven, Diamonds)}
	state.Stock.Cards = nil
	g := &Game{state: state}

	// Recycle: waste → stock (−100, but clamped at 0).
	if err := g.Execute(&RecycleStockCmd{}); err != nil {
		t.Fatalf("Execute RecycleStockCmd: %v", err)
	}
	if g.Score() != 0 {
		t.Errorf("score after recycle = %d, want 0 (clamped)", g.Score())
	}
	if len(g.state.Stock.Cards) != 1 {
		t.Errorf("stock len after recycle = %d, want 1", len(g.state.Stock.Cards))
	}
	if len(g.state.Waste.Cards) != 0 {
		t.Errorf("waste len after recycle = %d, want 0", len(g.state.Waste.Cards))
	}

	// Undo recycle → card back in waste.
	if err := g.Undo(); err != nil {
		t.Fatalf("Undo RecycleStockCmd: %v", err)
	}
	if len(g.state.Waste.Cards) != 1 {
		t.Errorf("waste len after undo = %d, want 1", len(g.state.Waste.Cards))
	}
	if len(g.state.Stock.Cards) != 0 {
		t.Errorf("stock len after undo = %d, want 0", len(g.state.Stock.Cards))
	}
}
