package engine

import (
	"errors"
	"fmt"
)

// Move describes a single game action: moving CardCount cards from pile From to pile To.
// For stock flip/recycle, CardCount is 0 and To/From may equal PileStock or PileWaste.
type Move struct {
	From      PileID
	To        PileID
	CardCount int
}

// ValidateMove returns nil if move is legal in state, or a descriptive error otherwise.
func ValidateMove(state *GameState, move Move) error {
	switch {
	case move.From == PileStock:
		return validateStockFlip(state, move)
	case isTableauPile(move.From) && isTableauPile(move.To):
		return validateTableauToTableau(state, move)
	case move.From == PileWaste && isTableauPile(move.To):
		return validateWasteToTableau(state, move)
	case (move.From == PileWaste || isTableauPile(move.From)) && isFoundationPile(move.To):
		return validateToFoundation(state, move)
	case isFoundationPile(move.From) && isTableauPile(move.To):
		return validateFoundationToTableau(state, move)
	default:
		return fmt.Errorf("unsupported move: from %d to %d", move.From, move.To)
	}
}

// ValidMoves returns all currently legal moves in state.
func ValidMoves(state *GameState) []Move {
	var moves []Move

	// 1. Stock flip or recycle.
	stockMove := Move{From: PileStock, To: PileWaste}
	if ValidateMove(state, stockMove) == nil {
		moves = append(moves, stockMove)
	}

	// 2. Waste → Tableau (single card).
	for col := 0; col < 7; col++ {
		m := Move{From: PileWaste, To: PileTableau0 + PileID(col), CardCount: 1}
		if ValidateMove(state, m) == nil {
			moves = append(moves, m)
		}
	}

	// 3. Waste → Foundation.
	for f := 0; f < 4; f++ {
		m := Move{From: PileWaste, To: PileFoundation0 + PileID(f), CardCount: 1}
		if ValidateMove(state, m) == nil {
			moves = append(moves, m)
		}
	}

	// 4. Tableau → Foundation (top card only).
	for col := 0; col < 7; col++ {
		for f := 0; f < 4; f++ {
			m := Move{
				From:      PileTableau0 + PileID(col),
				To:        PileFoundation0 + PileID(f),
				CardCount: 1,
			}
			if ValidateMove(state, m) == nil {
				moves = append(moves, m)
			}
		}
	}

	// 5. Foundation → Tableau (top card only).
	for f := 0; f < 4; f++ {
		for col := 0; col < 7; col++ {
			m := Move{
				From:      PileFoundation0 + PileID(f),
				To:        PileTableau0 + PileID(col),
				CardCount: 1,
			}
			if ValidateMove(state, m) == nil {
				moves = append(moves, m)
			}
		}
	}

	// 6. Tableau → Tableau (all valid sequence lengths).
	for srcCol := 0; srcCol < 7; srcCol++ {
		faceUp := state.Tableau[srcCol].FaceUpCards()
		for count := 1; count <= len(faceUp); count++ {
			for dstCol := 0; dstCol < 7; dstCol++ {
				if srcCol == dstCol {
					continue
				}
				m := Move{
					From:      PileTableau0 + PileID(srcCol),
					To:        PileTableau0 + PileID(dstCol),
					CardCount: count,
				}
				if ValidateMove(state, m) == nil {
					moves = append(moves, m)
				}
			}
		}
	}

	return moves
}

// isTableauPile returns true if id is one of PileTableau0..PileTableau6.
func isTableauPile(id PileID) bool {
	return id >= PileTableau0 && id <= PileTableau6
}

// isFoundationPile returns true if id is one of PileFoundation0..PileFoundation3.
func isFoundationPile(id PileID) bool {
	return id >= PileFoundation0 && id <= PileFoundation3
}

// tableauIndex returns the 0-based column index for a tableau PileID.
func tableauIndex(id PileID) int {
	return int(id - PileTableau0)
}

// foundationIndex returns the 0-based index for a foundation PileID.
func foundationIndex(id PileID) int {
	return int(id - PileFoundation0)
}

// isValidTableauPlacement reports whether card can be placed onto dest.
// Empty dest accepts only Kings. Non-empty dest requires opposite color
// and exactly one rank lower than the destination top card.
func isValidTableauPlacement(card Card, dest *TableauPile) bool {
	if dest.IsEmpty() {
		return card.Rank == King
	}
	top := dest.TopCard()
	return card.Color() != top.Color() && card.Rank == top.Rank-1
}

// isValidFaceUpSequence returns true if cards form a valid built sequence:
// each successive card is one rank lower and the opposite color from the previous.
// A single card or empty slice is always valid.
func isValidFaceUpSequence(cards []Card) bool {
	for i := 1; i < len(cards); i++ {
		prev, curr := cards[i-1], cards[i]
		if curr.Color() == prev.Color() || curr.Rank != prev.Rank-1 {
			return false
		}
	}
	return true
}

// validateTableauToTableau validates a Tableau→Tableau move (Section 13.1).
func validateTableauToTableau(state *GameState, move Move) error {
	src := state.Tableau[tableauIndex(move.From)]
	dst := state.Tableau[tableauIndex(move.To)]

	faceUp := src.FaceUpCards()
	if len(faceUp) == 0 {
		return errors.New("source tableau has no face-up cards")
	}
	if move.CardCount < 1 || move.CardCount > len(faceUp) {
		return errors.New("card count out of range for source face-up cards")
	}

	// seq is the sub-sequence being moved; seq[0] is its bottom (lowest rank) card.
	seq := faceUp[len(faceUp)-move.CardCount:]

	if !isValidFaceUpSequence(seq) {
		return errors.New("moved cards do not form a valid sequence")
	}
	if !isValidTableauPlacement(seq[0], dst) {
		return errors.New("destination tableau does not accept this card")
	}
	return nil
}

// validateWasteToTableau validates a Waste→Tableau move (Section 13.2).
func validateWasteToTableau(state *GameState, move Move) error {
	top := state.Waste.TopCard()
	if top == nil {
		return errors.New("waste pile is empty")
	}
	if move.CardCount != 1 {
		return errors.New("can only move one card from waste")
	}
	dst := state.Tableau[tableauIndex(move.To)]
	if !isValidTableauPlacement(*top, dst) {
		return errors.New("destination tableau does not accept this card")
	}
	return nil
}

// validateToFoundation validates a Waste→Foundation or Tableau→Foundation move (Section 13.3).
func validateToFoundation(state *GameState, move Move) error {
	var card Card
	switch {
	case move.From == PileWaste:
		top := state.Waste.TopCard()
		if top == nil {
			return errors.New("waste pile is empty")
		}
		card = *top
	case isTableauPile(move.From):
		src := state.Tableau[tableauIndex(move.From)]
		top := src.TopCard()
		if top == nil {
			return errors.New("source tableau is empty")
		}
		if !top.FaceUp {
			return errors.New("top card of source tableau is face-down")
		}
		if move.CardCount != 1 {
			return errors.New("can only move one card to foundation")
		}
		card = *top
	default:
		return errors.New("invalid source for foundation move")
	}

	foundation := state.Foundations[foundationIndex(move.To)]
	if !foundation.AcceptsCard(card) {
		return errors.New("foundation does not accept this card")
	}
	return nil
}

// validateFoundationToTableau validates a Foundation→Tableau move (Section 13.4).
func validateFoundationToTableau(state *GameState, move Move) error {
	foundation := state.Foundations[foundationIndex(move.From)]
	top := foundation.TopCard()
	if top == nil {
		return errors.New("source foundation is empty")
	}
	dst := state.Tableau[tableauIndex(move.To)]
	if !isValidTableauPlacement(*top, dst) {
		return errors.New("destination tableau does not accept this card")
	}
	return nil
}

// validateStockFlip validates a stock flip or recycle move (Section 13.5).
func validateStockFlip(state *GameState, move Move) error {
	if move.To != PileWaste {
		return errors.New("stock flip destination must be waste")
	}
	if !state.Stock.IsEmpty() {
		return nil // Normal flip: draw DrawCount cards.
	}
	if state.Waste.IsEmpty() {
		return errors.New("stock and waste are both empty")
	}
	return nil // Recycle: all waste cards go back to stock reversed.
}
