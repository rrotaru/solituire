package engine

// StockPile is the face-down draw pile. The top of the draw pile is the last element.
type StockPile struct {
	Cards []Card
}

// IsEmpty returns true when the stock has no cards.
func (s *StockPile) IsEmpty() bool {
	return len(s.Cards) == 0
}

// Count returns the number of cards remaining in the stock.
func (s *StockPile) Count() int {
	return len(s.Cards)
}

// WastePile holds cards that have been drawn from the stock.
// The top (last) card is the only playable card; DrawCount controls how many are rendered.
type WastePile struct {
	Cards     []Card
	DrawCount int // 1 or 3
}

// TopCard returns a pointer to the top playable card (last element), or nil if empty.
func (w *WastePile) TopCard() *Card {
	if len(w.Cards) == 0 {
		return nil
	}
	return &w.Cards[len(w.Cards)-1]
}

// VisibleCards returns up to DrawCount cards from the top of the waste for rendering.
// Only the last card in the returned slice is playable.
func (w *WastePile) VisibleCards() []Card {
	if len(w.Cards) == 0 {
		return nil
	}
	n := w.DrawCount
	if n <= 0 {
		n = 1
	}
	start := len(w.Cards) - n
	if start < 0 {
		start = 0
	}
	return w.Cards[start:]
}

// IsEmpty returns true when the waste has no cards.
func (w *WastePile) IsEmpty() bool {
	return len(w.Cards) == 0
}
