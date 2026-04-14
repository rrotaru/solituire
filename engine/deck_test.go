package engine

import (
	"reflect"
	"testing"
)

func TestDeterministicShuffle(t *testing.T) {
	deck1 := Shuffle(NewDeck(), 42)
	deck2 := Shuffle(NewDeck(), 42)
	if !reflect.DeepEqual(deck1, deck2) {
		t.Error("same seed produced different shuffle results")
	}
}

func TestDifferentSeedsProduceDifferentOrder(t *testing.T) {
	deck1 := Shuffle(NewDeck(), 1)
	deck2 := Shuffle(NewDeck(), 2)
	if reflect.DeepEqual(deck1, deck2) {
		t.Error("different seeds produced identical shuffle results")
	}
}

func TestShufflePreservesAllCards(t *testing.T) {
	deck := Shuffle(NewDeck(), 99)
	if len(deck) != 52 {
		t.Fatalf("shuffled deck has %d cards, want 52", len(deck))
	}
	type key struct {
		suit Suit
		rank Rank
	}
	seen := make(map[key]bool, 52)
	for _, c := range deck {
		seen[key{c.Suit, c.Rank}] = true
	}
	if len(seen) != 52 {
		t.Errorf("shuffled deck has %d unique cards, want 52", len(seen))
	}
}

func TestDealLayout(t *testing.T) {
	deck := Shuffle(NewDeck(), 42)
	state := Deal(deck, 1)

	// Tableau column sizes and face-up/down counts
	for col := 0; col < 7; col++ {
		pile := state.Tableau[col]
		wantTotal := col + 1
		if len(pile.Cards) != wantTotal {
			t.Errorf("tableau[%d] has %d cards, want %d", col, len(pile.Cards), wantTotal)
		}
		wantFaceDown := col
		if got := pile.FaceDownCount(); got != wantFaceDown {
			t.Errorf("tableau[%d] FaceDownCount = %d, want %d", col, got, wantFaceDown)
		}
		top := pile.Cards[len(pile.Cards)-1]
		if !top.FaceUp {
			t.Errorf("tableau[%d] top card is not face-up", col)
		}
	}

	// Stock has 24 cards
	if got := state.Stock.Count(); got != 24 {
		t.Errorf("stock has %d cards, want 24", got)
	}
	for _, c := range state.Stock.Cards {
		if c.FaceUp {
			t.Error("stock contains a face-up card")
		}
	}

	// Waste and foundations empty
	if !state.Waste.IsEmpty() {
		t.Error("waste pile should be empty after deal")
	}
	for i, f := range state.Foundations {
		if len(f.Cards) != 0 {
			t.Errorf("foundation[%d] should be empty after deal", i)
		}
	}
}

func TestDealDraw3PropagatesDrawCount(t *testing.T) {
	state := Deal(Shuffle(NewDeck(), 1), 3)
	if state.DrawCount != 3 {
		t.Errorf("GameState.DrawCount = %d, want 3", state.DrawCount)
	}
	if state.Waste.DrawCount != 3 {
		t.Errorf("WastePile.DrawCount = %d, want 3", state.Waste.DrawCount)
	}
}

func TestDealNoduplicateCards(t *testing.T) {
	deck := Shuffle(NewDeck(), 7)
	state := Deal(deck, 1)

	type key struct {
		suit Suit
		rank Rank
	}
	seen := make(map[key]bool, 52)
	add := func(c Card) {
		k := key{c.Suit, c.Rank}
		if seen[k] {
			panic("duplicate card in deal: " + c.String())
		}
		seen[k] = true
	}

	for _, t2 := range state.Tableau {
		for _, c := range t2.Cards {
			add(c)
		}
	}
	for _, c := range state.Stock.Cards {
		add(c)
	}
	if len(seen) != 52 {
		panic("deal total != 52")
	}
}
