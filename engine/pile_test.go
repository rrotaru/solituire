package engine

import "testing"

// --- TableauPile tests ---

func TestTableauIsEmpty(t *testing.T) {
	tp := &TableauPile{}
	if !tp.IsEmpty() {
		t.Error("new TableauPile should be empty")
	}
	tp.Cards = append(tp.Cards, Card{Suit: Spades, Rank: Ace, FaceUp: true})
	if tp.IsEmpty() {
		t.Error("TableauPile with a card should not be empty")
	}
}

func TestTableauTopCard(t *testing.T) {
	tp := &TableauPile{}
	if tp.TopCard() != nil {
		t.Error("empty TableauPile TopCard should be nil")
	}
	c := Card{Suit: Hearts, Rank: King, FaceUp: true}
	tp.Cards = append(tp.Cards, c)
	if got := tp.TopCard(); got == nil || *got != c {
		t.Errorf("TopCard() = %v, want %v", got, c)
	}
}

func TestTableauFaceUpCards(t *testing.T) {
	tp := &TableauPile{Cards: []Card{
		{Suit: Spades, Rank: Five, FaceUp: false},
		{Suit: Spades, Rank: Six, FaceUp: false},
		{Suit: Hearts, Rank: Seven, FaceUp: true},
		{Suit: Clubs, Rank: Six, FaceUp: true},
	}}
	fu := tp.FaceUpCards()
	if len(fu) != 2 {
		t.Fatalf("FaceUpCards() returned %d cards, want 2", len(fu))
	}
	if fu[0].Rank != Seven || fu[1].Rank != Six {
		t.Errorf("unexpected FaceUpCards: %v", fu)
	}
}

func TestTableauFaceUpCardsAllDown(t *testing.T) {
	tp := &TableauPile{Cards: []Card{
		{Suit: Spades, Rank: Two, FaceUp: false},
	}}
	if fu := tp.FaceUpCards(); fu != nil {
		t.Errorf("all face-down pile FaceUpCards() = %v, want nil", fu)
	}
}

func TestTableauFaceDownCount(t *testing.T) {
	tp := &TableauPile{Cards: []Card{
		{FaceUp: false},
		{FaceUp: false},
		{FaceUp: true},
	}}
	if got := tp.FaceDownCount(); got != 2 {
		t.Errorf("FaceDownCount() = %d, want 2", got)
	}
}

// --- FoundationPile tests ---

func TestFoundationAcceptsAceOnEmpty(t *testing.T) {
	fp := &FoundationPile{}
	ace := Card{Suit: Spades, Rank: Ace, FaceUp: true}
	if !fp.AcceptsCard(ace) {
		t.Error("empty foundation should accept Ace")
	}
	two := Card{Suit: Spades, Rank: Two, FaceUp: true}
	if fp.AcceptsCard(two) {
		t.Error("empty foundation should not accept Two")
	}
}

func TestFoundationAcceptsNextRankSameSuit(t *testing.T) {
	fp := &FoundationPile{Cards: []Card{
		{Suit: Hearts, Rank: Ace, FaceUp: true},
	}}
	two := Card{Suit: Hearts, Rank: Two, FaceUp: true}
	if !fp.AcceptsCard(two) {
		t.Error("should accept Two of Hearts on Ace of Hearts")
	}
	wrongSuit := Card{Suit: Spades, Rank: Two, FaceUp: true}
	if fp.AcceptsCard(wrongSuit) {
		t.Error("should not accept Two of Spades on Hearts foundation")
	}
	wrongRank := Card{Suit: Hearts, Rank: Three, FaceUp: true}
	if fp.AcceptsCard(wrongRank) {
		t.Error("should not accept Three when Two is needed")
	}
}

func TestFoundationIsComplete(t *testing.T) {
	fp := &FoundationPile{}
	if fp.IsComplete() {
		t.Error("empty foundation should not be complete")
	}
	for r := Ace; r <= King; r++ {
		fp.Cards = append(fp.Cards, Card{Suit: Spades, Rank: r, FaceUp: true})
	}
	if !fp.IsComplete() {
		t.Error("13-card foundation should be complete")
	}
}

func TestFoundationTopCard(t *testing.T) {
	fp := &FoundationPile{}
	if fp.TopCard() != nil {
		t.Error("empty foundation TopCard should be nil")
	}
	c := Card{Suit: Diamonds, Rank: Five, FaceUp: true}
	fp.Cards = append(fp.Cards, c)
	if got := fp.TopCard(); got == nil || *got != c {
		t.Errorf("TopCard() = %v, want %v", got, c)
	}
}

func TestFoundationSuit(t *testing.T) {
	fp := &FoundationPile{}
	if fp.Suit() != nil {
		t.Error("empty foundation Suit should be nil")
	}
	fp.Cards = append(fp.Cards, Card{Suit: Clubs, Rank: Ace, FaceUp: true})
	if s := fp.Suit(); s == nil || *s != Clubs {
		t.Errorf("Suit() = %v, want Clubs", s)
	}
}

// --- StockPile tests ---

func TestStockIsEmpty(t *testing.T) {
	sp := &StockPile{}
	if !sp.IsEmpty() {
		t.Error("new StockPile should be empty")
	}
}

func TestStockCount(t *testing.T) {
	sp := &StockPile{Cards: []Card{{}, {}, {}}}
	if got := sp.Count(); got != 3 {
		t.Errorf("Count() = %d, want 3", got)
	}
}

// --- WastePile tests ---

func TestWasteTopCardEmpty(t *testing.T) {
	wp := &WastePile{DrawCount: 1}
	if wp.TopCard() != nil {
		t.Error("empty WastePile TopCard should be nil")
	}
}

func TestWasteTopCard(t *testing.T) {
	c := Card{Suit: Diamonds, Rank: Jack, FaceUp: true}
	wp := &WastePile{Cards: []Card{{Suit: Clubs, Rank: Two, FaceUp: true}, c}, DrawCount: 1}
	if got := wp.TopCard(); got == nil || *got != c {
		t.Errorf("TopCard() = %v, want %v", got, c)
	}
}

func TestWasteVisibleCardsDraw1(t *testing.T) {
	wp := &WastePile{
		Cards:     []Card{{Suit: Clubs, Rank: Two, FaceUp: true}, {Suit: Hearts, Rank: Three, FaceUp: true}},
		DrawCount: 1,
	}
	vis := wp.VisibleCards()
	if len(vis) != 1 {
		t.Fatalf("draw-1 VisibleCards() returned %d cards, want 1", len(vis))
	}
	if vis[0].Rank != Three {
		t.Errorf("VisibleCards()[0] = %v, want Three", vis[0])
	}
}

func TestWasteVisibleCardsDraw3Full(t *testing.T) {
	wp := &WastePile{
		Cards: []Card{
			{Suit: Spades, Rank: Ace, FaceUp: true},
			{Suit: Hearts, Rank: Two, FaceUp: true},
			{Suit: Clubs, Rank: Three, FaceUp: true},
			{Suit: Diamonds, Rank: Four, FaceUp: true},
		},
		DrawCount: 3,
	}
	vis := wp.VisibleCards()
	if len(vis) != 3 {
		t.Fatalf("draw-3 VisibleCards() returned %d cards, want 3", len(vis))
	}
	// Top (last) should be Four
	if vis[len(vis)-1].Rank != Four {
		t.Errorf("top visible card = %v, want Four", vis[len(vis)-1])
	}
}

func TestWasteVisibleCardsDraw3Fewer(t *testing.T) {
	wp := &WastePile{
		Cards: []Card{
			{Suit: Spades, Rank: Ace, FaceUp: true},
			{Suit: Hearts, Rank: Two, FaceUp: true},
		},
		DrawCount: 3,
	}
	vis := wp.VisibleCards()
	if len(vis) != 2 {
		t.Fatalf("draw-3 with 2 cards VisibleCards() returned %d, want 2", len(vis))
	}
}

func TestWasteIsEmpty(t *testing.T) {
	wp := &WastePile{DrawCount: 1}
	if !wp.IsEmpty() {
		t.Error("new WastePile should be empty")
	}
}
