package engine

import "testing"

func TestSuitColors(t *testing.T) {
	tests := []struct {
		suit  Suit
		color Color
	}{
		{Spades, Black},
		{Clubs, Black},
		{Hearts, Red},
		{Diamonds, Red},
	}
	for _, tt := range tests {
		if got := tt.suit.Color(); got != tt.color {
			t.Errorf("%v.Color() = %v, want %v", tt.suit, got, tt.color)
		}
	}
}

func TestSuitSymbols(t *testing.T) {
	tests := []struct {
		suit   Suit
		symbol string
	}{
		{Spades, "♠"},
		{Hearts, "♥"},
		{Diamonds, "♦"},
		{Clubs, "♣"},
	}
	for _, tt := range tests {
		if got := tt.suit.Symbol(); got != tt.symbol {
			t.Errorf("%v.Symbol() = %q, want %q", tt.suit, got, tt.symbol)
		}
	}
}

func TestRankStrings(t *testing.T) {
	tests := []struct {
		rank Rank
		str  string
	}{
		{Ace, "A"},
		{Two, "2"},
		{Three, "3"},
		{Four, "4"},
		{Five, "5"},
		{Six, "6"},
		{Seven, "7"},
		{Eight, "8"},
		{Nine, "9"},
		{Ten, "10"},
		{Jack, "J"},
		{Queen, "Q"},
		{King, "K"},
	}
	for _, tt := range tests {
		if got := tt.rank.String(); got != tt.str {
			t.Errorf("Rank(%d).String() = %q, want %q", tt.rank, got, tt.str)
		}
	}
}

func TestAllCardsUnique(t *testing.T) {
	deck := NewDeck()
	if len(deck) != 52 {
		t.Fatalf("NewDeck() returned %d cards, want 52", len(deck))
	}
	type key struct {
		suit Suit
		rank Rank
	}
	seen := make(map[key]bool, 52)
	for _, c := range deck {
		k := key{c.Suit, c.Rank}
		if seen[k] {
			t.Errorf("duplicate card: %v", c)
		}
		seen[k] = true
	}
	if len(seen) != 52 {
		t.Errorf("only %d unique (suit,rank) pairs, want 52", len(seen))
	}
}

func TestCardColorDelegates(t *testing.T) {
	red := Card{Suit: Hearts, Rank: Ace, FaceUp: true}
	if red.Color() != Red {
		t.Errorf("Hearts card should be Red")
	}
	black := Card{Suit: Spades, Rank: King, FaceUp: true}
	if black.Color() != Black {
		t.Errorf("Spades card should be Black")
	}
}

func TestCardStringFaceDown(t *testing.T) {
	c := Card{Suit: Spades, Rank: Ace, FaceUp: false}
	if got := c.String(); got != "??" {
		t.Errorf("face-down card String() = %q, want \"??\"", got)
	}
}

func TestCardStringFaceUp(t *testing.T) {
	c := Card{Suit: Hearts, Rank: Queen, FaceUp: true}
	want := "Q♥"
	if got := c.String(); got != want {
		t.Errorf("Card.String() = %q, want %q", got, want)
	}
}
