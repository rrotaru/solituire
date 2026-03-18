package engine

import (
	"reflect"
	"testing"
)

// --- helpers ---

// stateEqual compares two GameState pile slices for equality (ignores Score/MoveCount/time).
func tabEqual(t *testing.T, label string, got, want []Card) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: got %v, want %v", label, got, want)
	}
}

// --- MoveCardCmd ---

func TestMoveCardCmd_TableauToTableau(t *testing.T) {
	state := buildState()
	state.Tableau[0].Cards = []Card{faceDownCard(King, Spades), faceUpCard(Queen, Hearts)}
	state.Tableau[1].Cards = []Card{faceUpCard(King, Clubs)}

	pre0 := append([]Card{}, state.Tableau[0].Cards...)
	pre1 := append([]Card{}, state.Tableau[1].Cards...)

	cmd := &MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	tabEqual(t, "T0 after execute", state.Tableau[0].Cards,
		[]Card{faceDownCard(King, Spades)})
	tabEqual(t, "T1 after execute", state.Tableau[1].Cards,
		[]Card{faceUpCard(King, Clubs), faceUpCard(Queen, Hearts)})

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	tabEqual(t, "T0 after undo", state.Tableau[0].Cards, pre0)
	tabEqual(t, "T1 after undo", state.Tableau[1].Cards, pre1)
}

func TestMoveCardCmd_MultiCard(t *testing.T) {
	state := buildState()
	// T0 = [A♠(down), Q♥, J♠]  T1 = [K♣]
	state.Tableau[0].Cards = []Card{
		faceDownCard(Ace, Spades),
		faceUpCard(Queen, Hearts),
		faceUpCard(Jack, Spades),
	}
	state.Tableau[1].Cards = []Card{faceUpCard(King, Clubs)}

	cmd := &MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 2}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	tabEqual(t, "T0 after execute", state.Tableau[0].Cards,
		[]Card{faceDownCard(Ace, Spades)})
	tabEqual(t, "T1 after execute", state.Tableau[1].Cards,
		[]Card{faceUpCard(King, Clubs), faceUpCard(Queen, Hearts), faceUpCard(Jack, Spades)})

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	tabEqual(t, "T0 after undo", state.Tableau[0].Cards,
		[]Card{faceDownCard(Ace, Spades), faceUpCard(Queen, Hearts), faceUpCard(Jack, Spades)})
	tabEqual(t, "T1 after undo", state.Tableau[1].Cards, []Card{faceUpCard(King, Clubs)})
}

func TestMoveCardCmd_WasteToTableau(t *testing.T) {
	state := buildState()
	state.Waste.Cards = []Card{faceUpCard(Nine, Spades)}
	state.Tableau[0].Cards = []Card{faceUpCard(Ten, Hearts)}

	cmd := &MoveCardCmd{From: PileWaste, To: PileTableau0, CardCount: 1}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(state.Waste.Cards) != 0 {
		t.Errorf("waste should be empty, got %v", state.Waste.Cards)
	}
	tabEqual(t, "T0 after execute", state.Tableau[0].Cards,
		[]Card{faceUpCard(Ten, Hearts), faceUpCard(Nine, Spades)})

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	tabEqual(t, "waste after undo", state.Waste.Cards, []Card{faceUpCard(Nine, Spades)})
	tabEqual(t, "T0 after undo", state.Tableau[0].Cards, []Card{faceUpCard(Ten, Hearts)})
}

func TestMoveCardCmd_FoundationToTableau(t *testing.T) {
	state := buildState()
	state.Foundations[0].Cards = []Card{faceUpCard(Ace, Spades), faceUpCard(Two, Spades)}
	state.Tableau[0].Cards = []Card{faceUpCard(Three, Hearts)}

	cmd := &MoveCardCmd{From: PileFoundation0, To: PileTableau0, CardCount: 1}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	tabEqual(t, "F0 after execute", state.Foundations[0].Cards,
		[]Card{faceUpCard(Ace, Spades)})
	tabEqual(t, "T0 after execute", state.Tableau[0].Cards,
		[]Card{faceUpCard(Three, Hearts), faceUpCard(Two, Spades)})

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	tabEqual(t, "F0 after undo", state.Foundations[0].Cards,
		[]Card{faceUpCard(Ace, Spades), faceUpCard(Two, Spades)})
	tabEqual(t, "T0 after undo", state.Tableau[0].Cards, []Card{faceUpCard(Three, Hearts)})
}

func TestMoveCardCmd_InvalidMove(t *testing.T) {
	state := buildState()
	// 7♥ cannot go onto 7♠ (same rank)
	state.Tableau[0].Cards = []Card{faceUpCard(Seven, Hearts)}
	state.Tableau[1].Cards = []Card{faceUpCard(Seven, Spades)}

	pre0 := append([]Card{}, state.Tableau[0].Cards...)
	pre1 := append([]Card{}, state.Tableau[1].Cards...)

	cmd := &MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1}
	if err := cmd.Execute(state); err == nil {
		t.Fatal("Execute should have returned an error")
	}
	// State must be unchanged.
	tabEqual(t, "T0 unchanged", state.Tableau[0].Cards, pre0)
	tabEqual(t, "T1 unchanged", state.Tableau[1].Cards, pre1)
}

// --- MoveToFoundationCmd ---

func TestMoveToFoundationCmd_FromWaste(t *testing.T) {
	state := buildState()
	state.Waste.Cards = []Card{faceUpCard(Ace, Hearts)}

	cmd := &MoveToFoundationCmd{From: PileWaste, FoundationIdx: 0}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(state.Waste.Cards) != 0 {
		t.Errorf("waste should be empty")
	}
	tabEqual(t, "F0 after execute", state.Foundations[0].Cards, []Card{faceUpCard(Ace, Hearts)})

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	tabEqual(t, "waste after undo", state.Waste.Cards, []Card{faceUpCard(Ace, Hearts)})
	if len(state.Foundations[0].Cards) != 0 {
		t.Errorf("F0 should be empty after undo")
	}
}

func TestMoveToFoundationCmd_FromTableau(t *testing.T) {
	state := buildState()
	state.Tableau[2].Cards = []Card{faceDownCard(King, Spades), faceUpCard(Ace, Diamonds)}

	cmd := &MoveToFoundationCmd{From: PileTableau2, FoundationIdx: 2}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	tabEqual(t, "T2 after execute", state.Tableau[2].Cards, []Card{faceDownCard(King, Spades)})
	tabEqual(t, "F2 after execute", state.Foundations[2].Cards, []Card{faceUpCard(Ace, Diamonds)})

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	tabEqual(t, "T2 after undo", state.Tableau[2].Cards,
		[]Card{faceDownCard(King, Spades), faceUpCard(Ace, Diamonds)})
	if len(state.Foundations[2].Cards) != 0 {
		t.Errorf("F2 should be empty after undo")
	}
}

// --- FlipStockCmd ---

func TestFlipStockCmd_DrawOne(t *testing.T) {
	state := buildState()
	state.DrawCount = 1
	state.Stock.Cards = []Card{
		faceDownCard(Five, Clubs),
		faceDownCard(Three, Hearts),
		faceDownCard(Ace, Spades), // top
	}

	cmd := &FlipStockCmd{}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(state.Stock.Cards) != 2 {
		t.Errorf("stock: want 2 cards, got %d", len(state.Stock.Cards))
	}
	if len(state.Waste.Cards) != 1 {
		t.Errorf("waste: want 1 card, got %d", len(state.Waste.Cards))
	}
	top := state.Waste.TopCard()
	if top == nil || top.Rank != Ace || top.Suit != Spades || !top.FaceUp {
		t.Errorf("waste top: want A♠ face-up, got %v", top)
	}

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if len(state.Stock.Cards) != 3 {
		t.Errorf("stock after undo: want 3 cards, got %d", len(state.Stock.Cards))
	}
	if len(state.Waste.Cards) != 0 {
		t.Errorf("waste after undo: want empty, got %d", len(state.Waste.Cards))
	}
	// Top of stock should be face-down A♠ again.
	if state.Stock.Cards[2].Rank != Ace || state.Stock.Cards[2].FaceUp {
		t.Errorf("stock top after undo: want A♠ face-down, got %v", state.Stock.Cards[2])
	}
}

func TestFlipStockCmd_DrawThree(t *testing.T) {
	state := buildState()
	state.DrawCount = 3
	state.Waste = &WastePile{DrawCount: 3}
	// Stock: [5♣, 3♥, A♠, K♦, Q♣]  Q♣ = top (index 4)
	state.Stock.Cards = []Card{
		faceDownCard(Five, Clubs),
		faceDownCard(Three, Hearts),
		faceDownCard(Ace, Spades),
		faceDownCard(King, Diamonds),
		faceDownCard(Queen, Clubs), // top
	}

	cmd := &FlipStockCmd{}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Stock should have 2 cards left (5♣, 3♥).
	if len(state.Stock.Cards) != 2 {
		t.Errorf("stock: want 2 cards, got %d", len(state.Stock.Cards))
	}
	// Waste should have 3 cards. Top (playable) = A♠ (deepest of drawn 3).
	if len(state.Waste.Cards) != 3 {
		t.Errorf("waste: want 3 cards, got %d", len(state.Waste.Cards))
	}
	top := state.Waste.TopCard()
	if top == nil || top.Rank != Ace || top.Suit != Spades {
		t.Errorf("waste top: want A♠, got %v", top)
	}
	if !top.FaceUp {
		t.Errorf("waste top should be face-up")
	}

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if len(state.Stock.Cards) != 5 {
		t.Errorf("stock after undo: want 5 cards, got %d", len(state.Stock.Cards))
	}
	if len(state.Waste.Cards) != 0 {
		t.Errorf("waste after undo: want empty, got %d", len(state.Waste.Cards))
	}
	// All stock cards should be face-down.
	for i, c := range state.Stock.Cards {
		if c.FaceUp {
			t.Errorf("stock[%d] should be face-down after undo", i)
		}
	}
	// Original top of stock (Q♣) should be restored.
	if state.Stock.Cards[4].Rank != Queen || state.Stock.Cards[4].Suit != Clubs {
		t.Errorf("stock top after undo: want Q♣, got %v", state.Stock.Cards[4])
	}
}

func TestFlipStockCmd_DrawThreeFewer(t *testing.T) {
	// Only 2 cards in stock with DrawCount=3 → draws 2.
	state := buildState()
	state.DrawCount = 3
	state.Waste = &WastePile{DrawCount: 3}
	state.Stock.Cards = []Card{
		faceDownCard(Five, Clubs),
		faceDownCard(Ace, Spades), // top
	}

	cmd := &FlipStockCmd{}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(state.Stock.Cards) != 0 {
		t.Errorf("stock: want empty, got %d", len(state.Stock.Cards))
	}
	if len(state.Waste.Cards) != 2 {
		t.Errorf("waste: want 2 cards, got %d", len(state.Waste.Cards))
	}
	// Top of waste = 5♣ (deepest of the 2 drawn).
	top := state.Waste.TopCard()
	if top == nil || top.Rank != Five || top.Suit != Clubs {
		t.Errorf("waste top: want 5♣, got %v", top)
	}

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if len(state.Stock.Cards) != 2 {
		t.Errorf("stock after undo: want 2 cards, got %d", len(state.Stock.Cards))
	}
	if len(state.Waste.Cards) != 0 {
		t.Errorf("waste after undo: want empty")
	}
}

func TestFlipStockCmd_EmptyStockError(t *testing.T) {
	state := buildState()
	state.Waste.Cards = []Card{faceUpCard(Five, Hearts)}
	cmd := &FlipStockCmd{}
	if err := cmd.Execute(state); err == nil {
		t.Fatal("Execute on empty stock should return error")
	}
}

// --- RecycleStockCmd ---

func TestRecycleStockCmd(t *testing.T) {
	state := buildState()
	// Waste = [A♠, 2♠, 3♠]  3♠ = top/playable
	state.Waste.Cards = []Card{
		faceUpCard(Ace, Spades),
		faceUpCard(Two, Spades),
		faceUpCard(Three, Spades), // top
	}

	cmd := &RecycleStockCmd{}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(state.Waste.Cards) != 0 {
		t.Errorf("waste should be empty after recycle")
	}
	if len(state.Stock.Cards) != 3 {
		t.Errorf("stock: want 3 cards, got %d", len(state.Stock.Cards))
	}
	// Stock top = A♠ (waste bottom → drawn first on next pass).
	if state.Stock.Cards[2].Rank != Ace || state.Stock.Cards[2].Suit != Spades {
		t.Errorf("stock top after recycle: want A♠, got %v", state.Stock.Cards[2])
	}
	// All stock cards should be face-down.
	for i, c := range state.Stock.Cards {
		if c.FaceUp {
			t.Errorf("stock[%d] should be face-down after recycle", i)
		}
	}

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if len(state.Stock.Cards) != 0 {
		t.Errorf("stock should be empty after undo")
	}
	// Waste restored: [A♠, 2♠, 3♠] with 3♠ on top, face-up.
	tabEqual(t, "waste after undo", state.Waste.Cards, []Card{
		faceUpCard(Ace, Spades),
		faceUpCard(Two, Spades),
		faceUpCard(Three, Spades),
	})
	for i, c := range state.Waste.Cards {
		if !c.FaceUp {
			t.Errorf("waste[%d] should be face-up after undo", i)
		}
	}
}

func TestRecycleStockCmd_NonEmptyStockError(t *testing.T) {
	state := buildState()
	state.Stock.Cards = []Card{faceDownCard(Ace, Spades)}
	state.Waste.Cards = []Card{faceUpCard(Two, Hearts)}
	cmd := &RecycleStockCmd{}
	if err := cmd.Execute(state); err == nil {
		t.Fatal("Execute should error when stock is not empty")
	}
}

// --- FlipTableauCardCmd ---

func TestFlipTableauCardCmd(t *testing.T) {
	state := buildState()
	state.Tableau[3].Cards = []Card{
		faceDownCard(King, Spades),
		faceDownCard(Queen, Hearts), // top, face-down
	}

	cmd := &FlipTableauCardCmd{ColumnIdx: 3}
	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	top := state.Tableau[3].TopCard()
	if !top.FaceUp {
		t.Errorf("top card should be face-up after Execute")
	}
	if top.Rank != Queen || top.Suit != Hearts {
		t.Errorf("top card: want Q♥, got %v", top)
	}

	if err := cmd.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	top = state.Tableau[3].TopCard()
	if top.FaceUp {
		t.Errorf("top card should be face-down after Undo")
	}
}

func TestFlipTableauCardCmd_AlreadyFaceUp(t *testing.T) {
	state := buildState()
	state.Tableau[3].Cards = []Card{faceUpCard(Queen, Hearts)} // already face-up
	cmd := &FlipTableauCardCmd{ColumnIdx: 3}
	if err := cmd.Execute(state); err == nil {
		t.Fatal("Execute should error when top card is already face-up")
	}
}

func TestFlipTableauCardCmd_EmptyColumn(t *testing.T) {
	state := buildState()
	cmd := &FlipTableauCardCmd{ColumnIdx: 0}
	if err := cmd.Execute(state); err == nil {
		t.Fatal("Execute should error on empty column")
	}
}

// --- CompoundCmd ---

func TestCompoundCmd_MoveAndFlip(t *testing.T) {
	state := buildState()
	// T0 = [K♠(down), 9♥(up)],  T1 = [10♣(up)]
	state.Tableau[0].Cards = []Card{
		faceDownCard(King, Spades),
		faceUpCard(Nine, Hearts),
	}
	state.Tableau[1].Cards = []Card{faceUpCard(Ten, Clubs)}

	compound := &CompoundCmd{Cmds: []Command{
		&MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1},
		&FlipTableauCardCmd{ColumnIdx: 0},
	}}

	if err := compound.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// T0 should have K♠ face-up now.
	tabEqual(t, "T0 after execute", state.Tableau[0].Cards,
		[]Card{faceUpCard(King, Spades)})
	// T1 should have [10♣, 9♥].
	tabEqual(t, "T1 after execute", state.Tableau[1].Cards,
		[]Card{faceUpCard(Ten, Clubs), faceUpCard(Nine, Hearts)})

	if err := compound.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	// Both actions reversed atomically.
	tabEqual(t, "T0 after undo", state.Tableau[0].Cards,
		[]Card{faceDownCard(King, Spades), faceUpCard(Nine, Hearts)})
	tabEqual(t, "T1 after undo", state.Tableau[1].Cards,
		[]Card{faceUpCard(Ten, Clubs)})
}

func TestCompoundCmd_RollbackOnPartialFailure(t *testing.T) {
	state := buildState()
	// T0 = [K♠(down), 9♥(up)],  T1 = [10♣(up)]
	state.Tableau[0].Cards = []Card{
		faceDownCard(King, Spades),
		faceUpCard(Nine, Hearts),
	}
	state.Tableau[1].Cards = []Card{faceUpCard(Ten, Clubs)}

	pre0 := append([]Card{}, state.Tableau[0].Cards...)
	pre1 := append([]Card{}, state.Tableau[1].Cards...)

	// Second command is invalid: T0 top will be face-down K♠ after first cmd,
	// but FlipTableauCardCmd with ColumnIdx 0 is valid. Use an impossible flip
	// instead: flip a column that's now empty (ColumnIdx 2, which is empty).
	// First cmd succeeds (moves 9♥), second cmd fails (T0 top K♠ is face-down
	// but we try to flip column 2 which is empty).
	compound := &CompoundCmd{Cmds: []Command{
		&MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1},
		&FlipTableauCardCmd{ColumnIdx: 2}, // column 2 is empty → error
	}}

	if err := compound.Execute(state); err == nil {
		t.Fatal("Execute should have returned an error")
	}
	// State must be fully rolled back.
	tabEqual(t, "T0 rolled back", state.Tableau[0].Cards, pre0)
	tabEqual(t, "T1 rolled back", state.Tableau[1].Cards, pre1)
}

func TestCompoundCmd_Description(t *testing.T) {
	cmd := &CompoundCmd{Cmds: []Command{
		&FlipTableauCardCmd{ColumnIdx: 0},
		&FlipTableauCardCmd{ColumnIdx: 1},
	}}
	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}
