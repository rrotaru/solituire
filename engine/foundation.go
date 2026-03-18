package engine

// FoundationPile holds cards built up from Ace to King in a single suit.
// All cards in the pile are face-up.
type FoundationPile struct {
	Cards []Card
}

// TopCard returns a pointer to the top card, or nil if the foundation is empty.
func (f *FoundationPile) TopCard() *Card {
	if len(f.Cards) == 0 {
		return nil
	}
	return &f.Cards[len(f.Cards)-1]
}

// AcceptsCard returns true if card can be legally placed on this foundation.
// An empty foundation accepts only an Ace. A non-empty foundation accepts
// the next rank of the same suit.
func (f *FoundationPile) AcceptsCard(card Card) bool {
	if len(f.Cards) == 0 {
		return card.Rank == Ace
	}
	top := f.Cards[len(f.Cards)-1]
	return card.Suit == top.Suit && card.Rank == top.Rank+1
}

// IsComplete returns true when the foundation holds all 13 cards (Ace through King).
func (f *FoundationPile) IsComplete() bool {
	return len(f.Cards) == 13
}

// Suit returns a pointer to the suit of the first card, or nil if empty.
func (f *FoundationPile) Suit() *Suit {
	if len(f.Cards) == 0 {
		return nil
	}
	s := f.Cards[0].Suit
	return &s
}
