package engine

import "time"

// PileID identifies a pile in GameState without pointer equality.
type PileID uint8

const (
	PileStock       PileID = iota
	PileWaste
	PileFoundation0 // foundations are PileFoundation0 + index (0-3)
	PileFoundation1
	PileFoundation2
	PileFoundation3
	PileTableau0 // tableau columns are PileTableau0 + index (0-6)
	PileTableau1
	PileTableau2
	PileTableau3
	PileTableau4
	PileTableau5
	PileTableau6
)

// GameState holds the complete, serialisable state of a Klondike game.
// All mutable operations go through Command.Execute so that Undo is always possible.
type GameState struct {
	Tableau     [7]*TableauPile
	Foundations [4]*FoundationPile
	Stock       *StockPile
	Waste       *WastePile
	Score       int
	MoveCount   int
	ElapsedTime time.Duration
	DrawCount   int   // 1 or 3
	Seed        int64
}

// deepCopyState returns a fully independent copy of s.
// Every pile is a new allocation with its own backing slice so mutations
// to the copy do not affect the original, and vice-versa.
func deepCopyState(s *GameState) *GameState {
	cp := &GameState{
		Score:       s.Score,
		MoveCount:   s.MoveCount,
		ElapsedTime: s.ElapsedTime,
		DrawCount:   s.DrawCount,
		Seed:        s.Seed,
	}

	// Tableau
	for i, t := range s.Tableau {
		cards := make([]Card, len(t.Cards))
		copy(cards, t.Cards)
		cp.Tableau[i] = &TableauPile{Cards: cards}
	}

	// Foundations
	for i, f := range s.Foundations {
		cards := make([]Card, len(f.Cards))
		copy(cards, f.Cards)
		cp.Foundations[i] = &FoundationPile{Cards: cards}
	}

	// Stock
	stockCards := make([]Card, len(s.Stock.Cards))
	copy(stockCards, s.Stock.Cards)
	cp.Stock = &StockPile{Cards: stockCards}

	// Waste
	wasteCards := make([]Card, len(s.Waste.Cards))
	copy(wasteCards, s.Waste.Cards)
	cp.Waste = &WastePile{Cards: wasteCards, DrawCount: s.Waste.DrawCount}

	return cp
}
