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

func (s StandardScorer) OnMove(move Move, state *GameState) int { panic("not implemented") }
func (s StandardScorer) OnUndo(move Move, state *GameState) int { panic("not implemented") }
func (s StandardScorer) OnRecycleStock() int                    { panic("not implemented") }
