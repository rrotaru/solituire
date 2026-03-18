package engine

// TableauPile is one of the seven columns in Klondike solitaire.
// Face-down cards sit at the bottom; face-up cards sit on top.
type TableauPile struct {
	Cards []Card
}

// FaceDownCount returns the number of face-down cards at the bottom of the column.
func (t *TableauPile) FaceDownCount() int {
	count := 0
	for _, c := range t.Cards {
		if !c.FaceUp {
			count++
		}
	}
	return count
}

// FaceUpCards returns the slice of face-up cards at the top of the column.
// Returns a sub-slice of the underlying array — callers must not mutate.
func (t *TableauPile) FaceUpCards() []Card {
	for i, c := range t.Cards {
		if c.FaceUp {
			return t.Cards[i:]
		}
	}
	return nil
}

// TopCard returns a pointer to the top (last) card, or nil if the pile is empty.
func (t *TableauPile) TopCard() *Card {
	if len(t.Cards) == 0 {
		return nil
	}
	return &t.Cards[len(t.Cards)-1]
}

// IsEmpty returns true when the column has no cards.
func (t *TableauPile) IsEmpty() bool {
	return len(t.Cards) == 0
}
