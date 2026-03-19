package engine

// StandardScorer implements Scorer with standard Klondike point values:
//
//	Waste → Tableau:         +5
//	Waste → Foundation:      +10
//	Tableau → Foundation:    +10
//	Foundation → Tableau:    −15
//	Flip tableau card:       +5  (via FlipTableauCardCmd)
//	Recycle stock:           −100
//
// Score is floored at 0.
type StandardScorer struct{}

func (s StandardScorer) OnMove(move Move, _ *GameState) int {
	switch {
	case move.From == PileWaste && isTableauPile(move.To):
		return 5
	case move.From == PileWaste && isFoundationPile(move.To):
		return 10
	case isTableauPile(move.From) && isFoundationPile(move.To):
		return 10
	case isFoundationPile(move.From) && isTableauPile(move.To):
		return -15
	default:
		return 0 // tableau→tableau, stock flip: no points
	}
}

func (s StandardScorer) OnFlipTableau() int {
	return 5
}

func (s StandardScorer) OnRecycleStock() int {
	return -100
}
