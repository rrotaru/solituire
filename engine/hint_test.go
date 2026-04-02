package engine

import "testing"

// TestFindHints_FoundationHighestPriority verifies foundation moves rank above all others.
func TestFindHints_FoundationHighestPriority(t *testing.T) {
	s := buildState()
	// Waste has A♠ which can go to F0 (Spades, empty).
	s.Waste.Cards = []Card{faceUpCard(Ace, Spades)}
	// T0 has Q♠ that could go to T1 (build-length), giving a competing non-foundation hint.
	s.Tableau[0].Cards = []Card{faceUpCard(Queen, Spades)}
	s.Tableau[1].Cards = []Card{faceUpCard(King, Hearts)}

	hints := FindHints(s)
	if len(hints) == 0 {
		t.Fatal("expected at least one hint, got none")
	}
	if hints[0].Priority != priorityFoundation {
		t.Errorf("top hint priority = %d, want %d (foundation)", hints[0].Priority, priorityFoundation)
	}
	if hints[0].From != PileWaste {
		t.Errorf("top hint From = %v, want PileWaste", hints[0].From)
	}
	if !isFoundationPile(hints[0].To) {
		t.Errorf("top hint To = %v, want a foundation pile", hints[0].To)
	}
	// CardIdx: only card in waste is at index 0.
	if hints[0].CardIdx != 0 {
		t.Errorf("top hint CardIdx = %d, want 0", hints[0].CardIdx)
	}
}

// TestFindHints_ExposeFaceDownSecondPriority verifies expose-face-down outranks build-length.
func TestFindHints_ExposeFaceDownSecondPriority(t *testing.T) {
	s := buildState()
	// T0: face-down K♠ under face-up Q♥ — moving Q♥ exposes the face-down card.
	s.Tableau[0].Cards = []Card{faceDownCard(King, Spades), faceUpCard(Queen, Hearts)}
	// T1: K♣ accepts Q♥ (valid build target).
	s.Tableau[1].Cards = []Card{faceUpCard(King, Clubs)}
	// T2: empty — Q♥ is NOT a King so it can't go here; no King-to-empty competition here.

	hints := FindHints(s)
	if len(hints) == 0 {
		t.Fatal("expected at least one hint, got none")
	}

	// The T0→T1 move exposes a face-down card; it must be the top hint.
	top := hints[0]
	if top.Priority != priorityExposeDown {
		t.Errorf("top hint priority = %d, want %d (exposeDown)", top.Priority, priorityExposeDown)
	}
	if top.From != PileTableau0 {
		t.Errorf("top hint From = %v, want PileTableau0", top.From)
	}
	if top.To != PileTableau1 {
		t.Errorf("top hint To = %v, want PileTableau1", top.To)
	}
	// Q♥ is at index 1 (bottom of 1-card sequence: len(T0.Cards)-1 = 2-1 = 1).
	if top.CardIdx != 1 {
		t.Errorf("top hint CardIdx = %d, want 1", top.CardIdx)
	}
}

// TestFindHints_KingToEmptyColumn verifies King-to-empty ranks above build-length.
func TestFindHints_KingToEmptyColumn(t *testing.T) {
	s := buildState()
	// T0: K♥ — can move to empty T1 (King-to-empty).
	s.Tableau[0].Cards = []Card{faceUpCard(King, Hearts)}
	// T1: empty.
	// T2: Q♠ that K♥ can't accept (no competing build here since K♥ is the only face-up card).

	hints := FindHints(s)
	if len(hints) == 0 {
		t.Fatal("expected at least one hint, got none")
	}
	top := hints[0]
	if top.Priority != priorityKingToEmpty {
		t.Errorf("top hint priority = %d, want %d (kingToEmpty)", top.Priority, priorityKingToEmpty)
	}
	if top.From != PileTableau0 {
		t.Errorf("top hint From = %v, want PileTableau0", top.From)
	}
	if !isTableauPile(top.To) || !s.Tableau[tableauIndex(top.To)].IsEmpty() {
		t.Errorf("top hint To = %v, want an empty tableau pile", top.To)
	}
}

// TestFindHints_KingToEmptyBeforeBuildLength verifies King-to-empty beats a plain build-length hint.
func TestFindHints_KingToEmptyBeforeBuildLength(t *testing.T) {
	s := buildState()
	// T0: Q♥ — can go onto K♠ in T1 (build-length).
	s.Tableau[0].Cards = []Card{faceUpCard(Queen, Hearts)}
	s.Tableau[1].Cards = []Card{faceUpCard(King, Spades)}
	// T2: K♦ — can go to empty T3 (King-to-empty).
	s.Tableau[2].Cards = []Card{faceUpCard(King, Diamonds)}
	// T3: empty.

	hints := FindHints(s)
	if len(hints) == 0 {
		t.Fatal("expected hints, got none")
	}
	if hints[0].Priority != priorityKingToEmpty {
		t.Errorf("top hint priority = %d, want %d (kingToEmpty)", hints[0].Priority, priorityKingToEmpty)
	}
}

// TestFindHints_BuildLength verifies ordinary tableau builds get priorityBuildLength.
func TestFindHints_BuildLength(t *testing.T) {
	s := buildState()
	// T0: 6♥ can go onto T1: 7♠ — valid build-length, no face-down exposure.
	// T2–T6 hold A♠ face-down so they are non-empty (no King-to-empty destinations)
	// and the face-down Ace tops accept no card (rank 0 doesn't exist).
	s.Tableau[0].Cards = []Card{faceUpCard(Six, Hearts)}
	s.Tableau[1].Cards = []Card{faceUpCard(Seven, Spades)}
	for i := 2; i <= 6; i++ {
		s.Tableau[i].Cards = []Card{faceDownCard(Ace, Spades)}
	}

	hints := FindHints(s)
	if len(hints) == 0 {
		t.Fatal("expected at least one hint, got none")
	}
	top := hints[0]
	if top.Priority != priorityBuildLength {
		t.Errorf("top hint priority = %d, want %d (buildLength)", top.Priority, priorityBuildLength)
	}
	if top.From != PileTableau0 || top.To != PileTableau1 {
		t.Errorf("top hint = {%v→%v}, want PileTableau0→PileTableau1", top.From, top.To)
	}
}

// TestFindHints_StockFlipLowestPriority verifies stock flip is last resort.
func TestFindHints_StockFlipLowestPriority(t *testing.T) {
	s := buildState()
	s.Stock.Cards = []Card{faceDownCard(Seven, Clubs)}
	// No tableau or waste moves possible.

	hints := FindHints(s)
	if len(hints) != 1 {
		t.Fatalf("expected exactly 1 hint, got %d: %+v", len(hints), hints)
	}
	if hints[0].From != PileStock {
		t.Errorf("hint From = %v, want PileStock", hints[0].From)
	}
	if hints[0].Priority != priorityStockFlip {
		t.Errorf("hint priority = %d, want %d (stockFlip)", hints[0].Priority, priorityStockFlip)
	}
	// CardIdx points to the top stock card (last element, index 0 here).
	if hints[0].CardIdx != 0 {
		t.Errorf("hint CardIdx = %d, want 0", hints[0].CardIdx)
	}
}

// TestFindHints_NoMoves verifies an empty (non-nil) slice is returned when no moves exist.
func TestFindHints_NoMoves(t *testing.T) {
	s := buildState()
	// Stranded 2♠ in T0, no foundation base, no tableau target.
	s.Tableau[0].Cards = []Card{faceUpCard(Two, Spades)}

	hints := FindHints(s)
	if hints == nil {
		t.Error("FindHints returned nil, want empty non-nil slice")
	}
	if len(hints) != 0 {
		t.Errorf("expected 0 hints, got %d: %+v", len(hints), hints)
	}
}

// TestFindHints_CardIdx_MultiCardSequence verifies CardIdx points to the bottom of a multi-card sequence.
func TestFindHints_CardIdx_MultiCardSequence(t *testing.T) {
	s := buildState()
	// T0: [A♦(face-down), Q♥(face-up), J♠(face-up)] — moving Q♥+J♠ exposes A♦.
	s.Tableau[0].Cards = []Card{
		faceDownCard(Ace, Diamonds),
		faceUpCard(Queen, Hearts),
		faceUpCard(Jack, Spades),
	}
	// T1: K♣ accepts Q♥ (bottom of 2-card sequence).
	s.Tableau[1].Cards = []Card{faceUpCard(King, Clubs)}

	hints := FindHints(s)
	if len(hints) == 0 {
		t.Fatal("expected hints, got none")
	}

	// Find the T0→T1 hint.
	var found *Hint
	for i := range hints {
		if hints[i].From == PileTableau0 && hints[i].To == PileTableau1 {
			found = &hints[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected hint T0→T1, not found")
	}
	// Q♥ is at index 1: len(T0.Cards)=3, CardCount=2, so index = 3-2 = 1.
	if found.CardIdx != 1 {
		t.Errorf("CardIdx = %d, want 1 (Q♥, bottom of 2-card sequence)", found.CardIdx)
	}
	if found.Priority != priorityExposeDown {
		t.Errorf("priority = %d, want %d (exposeDown)", found.Priority, priorityExposeDown)
	}
}

// TestFindHints_OrderingStability verifies equal-priority hints preserve ValidMoves order.
func TestFindHints_OrderingStability(t *testing.T) {
	s := buildState()
	// Two foundation moves: A♥ from waste and A♦ from T3.
	s.Waste.Cards = []Card{faceUpCard(Ace, Hearts)}
	s.Tableau[3].Cards = []Card{faceUpCard(Ace, Diamonds)}
	// Both F1 and F2 are empty and will accept these Aces.

	hints := FindHints(s)
	// Must have at least two foundation hints.
	foundationCount := 0
	for _, h := range hints {
		if h.Priority == priorityFoundation {
			foundationCount++
		}
	}
	if foundationCount < 2 {
		t.Errorf("expected ≥2 foundation hints, got %d", foundationCount)
	}
	// All hints must be sorted highest-first.
	for i := 1; i < len(hints); i++ {
		if hints[i].Priority > hints[i-1].Priority {
			t.Errorf("hints not sorted: hints[%d].Priority=%d > hints[%d].Priority=%d",
				i, hints[i].Priority, i-1, hints[i-1].Priority)
		}
	}
}

// TestFindHints_AllHintsAreLegal verifies every returned hint corresponds to a valid move.
func TestFindHints_AllHintsAreLegal(t *testing.T) {
	deck := Shuffle(NewDeck(), 42)
	state := Deal(deck, 1)

	hints := FindHints(state)
	for _, h := range hints {
		var cardCount int
		switch {
		case h.From == PileStock:
			cardCount = 0 // stock flip/recycle uses CardCount 0
		case isTableauPile(h.From):
			ti := tableauIndex(h.From)
			cardCount = len(state.Tableau[ti].Cards) - h.CardIdx
		default:
			cardCount = 1
		}
		m := Move{From: h.From, To: h.To, CardCount: cardCount}
		if err := ValidateMove(state, m); err != nil {
			t.Errorf("hint {From:%v CardIdx:%d To:%v Pri:%d} → move %+v is invalid: %v",
				h.From, h.CardIdx, h.To, h.Priority, m, err)
		}
	}
}
