package engine

import "sort"

// Hint describes a suggested move with a priority for ordering.
// Higher Priority values are shown first.
type Hint struct {
	From     PileID
	CardIdx  int // index within the source pile's Cards slice
	To       PileID
	Priority int
}

// Priority levels for hint ordering (unexported; callers compare by sort order).
const (
	priorityFoundation  = 40 // move any card to a foundation pile
	priorityExposeDown  = 30 // reveal a face-down tableau card
	priorityKingToEmpty = 20 // move a King to an empty tableau column
	priorityBuildLength = 10 // extend a non-empty tableau build
	priorityStockFlip   = 5  // stock flip or recycle (last resort)
)

// FindHints returns all legal moves in state sorted by priority (highest first).
// Priority order: foundation move > expose face-down card > King to empty > build length > stock flip.
// Returns an empty slice when no moves are available.
func FindHints(state *GameState) []Hint {
	moves := ValidMoves(state)
	hints := make([]Hint, 0, len(moves))
	for _, m := range moves {
		hints = append(hints, Hint{
			From:     m.From,
			CardIdx:  cardIdxForMove(state, m),
			To:       m.To,
			Priority: assignPriority(state, m),
		})
	}
	sort.SliceStable(hints, func(i, j int) bool {
		return hints[i].Priority > hints[j].Priority
	})
	return hints
}

// assignPriority returns the priority for a move in the given state.
// Conditions are checked highest-priority first; first match wins.
func assignPriority(state *GameState, move Move) int {
	// 1. Foundation move.
	if isFoundationPile(move.To) {
		return priorityFoundation
	}

	// 2. Expose a face-down tableau card.
	if isTableauPile(move.From) {
		t := state.Tableau[tableauIndex(move.From)]
		belowIdx := len(t.Cards) - move.CardCount - 1
		if belowIdx >= 0 && !t.Cards[belowIdx].FaceUp {
			return priorityExposeDown
		}
	}

	// 3. King to empty tableau column.
	if isTableauPile(move.To) && state.Tableau[tableauIndex(move.To)].IsEmpty() {
		return priorityKingToEmpty
	}

	// 4. Regular tableau build (non-empty destination).
	if isTableauPile(move.To) {
		return priorityBuildLength
	}

	// 5. Stock flip / recycle.
	if move.From == PileStock {
		return priorityStockFlip
	}

	return 0
}

// cardIdxForMove returns the index within the source pile's Cards slice of the
// bottom card in the moved sequence (the card the TUI highlights as source).
func cardIdxForMove(state *GameState, move Move) int {
	switch {
	case move.From == PileWaste:
		return len(state.Waste.Cards) - 1
	case isTableauPile(move.From):
		t := state.Tableau[tableauIndex(move.From)]
		return len(t.Cards) - move.CardCount
	case isFoundationPile(move.From):
		f := state.Foundations[foundationIndex(move.From)]
		return len(f.Cards) - 1
	case move.From == PileStock:
		if len(state.Stock.Cards) > 0 {
			return len(state.Stock.Cards) - 1
		}
		return 0
	}
	return 0
}
