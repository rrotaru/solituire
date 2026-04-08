package engine

import "time"

// Game implements GameEngine. It wires together GameState, History, and StandardScorer.
// Create instances via NewGame — do not construct directly.
type Game struct {
	state   *GameState
	history History
	scorer  StandardScorer
}

// compile-time interface check
var _ GameEngine = (*Game)(nil)

// NewGame creates and returns a fully initialized Game ready to play.
func NewGame(seed int64, drawCount int) *Game {
	g := &Game{}
	g.NewGame(seed, drawCount)
	return g
}

// State returns the current game state (read-only by convention).
func (g *Game) State() *GameState { return g.state }

// Score returns the current score.
func (g *Game) Score() int { return g.state.Score }

// MoveCount returns the number of moves executed (never decremented on undo).
func (g *Game) MoveCount() int { return g.state.MoveCount }

// Seed returns the seed used to deal the current game.
func (g *Game) Seed() int64 { return g.state.Seed }

// CanUndo reports whether there is a move to undo.
func (g *Game) CanUndo() bool { return g.history.CanUndo() }

// CanRedo reports whether there is a move to redo.
func (g *Game) CanRedo() bool { return g.history.CanRedo() }

// IsWon returns true when all four foundation piles each contain 13 cards.
func (g *Game) IsWon() bool {
	for _, f := range g.state.Foundations {
		if len(f.Cards) != 13 {
			return false
		}
	}
	return true
}

// IsAutoCompletable returns true when every remaining unplayed card is face-up
// (stock is empty and no face-down cards remain in any tableau column).
func (g *Game) IsAutoCompletable() bool {
	if !g.state.Stock.IsEmpty() {
		return false
	}
	for _, t := range g.state.Tableau {
		for _, c := range t.Cards {
			if !c.FaceUp {
				return false
			}
		}
	}
	return true
}

// NewGame resets the game with a fresh deal using the given seed and draw count.
func (g *Game) NewGame(seed int64, drawCount int) {
	deck := Shuffle(NewDeck(), seed)
	g.state = Deal(deck, drawCount)
	g.state.Seed = seed
	g.history.Clear()
}

// RestartDeal re-deals using the stored seed and draw count, resetting all progress.
func (g *Game) RestartDeal() {
	g.NewGame(g.state.Seed, g.state.DrawCount)
}

// Execute runs cmd against the current state.
// On success the command (plus any auto-flip) is recorded in history and the score updated.
// On failure the state is unchanged and the error is returned.
//
// Auto-flip: if executing cmd leaves a face-down card on top of any tableau column,
// a FlipTableauCardCmd is executed immediately and the two are recorded as a CompoundCmd
// so that a single Undo reverses both atomically.
func (g *Game) Execute(cmd Command) error {
	scoreBefore := g.state.Score

	if err := cmd.Execute(g.state); err != nil {
		return err
	}

	// Auto-flip: find tableau columns that now expose a face-down top card.
	var flips []Command
	for col := 0; col < 7; col++ {
		t := g.state.Tableau[col]
		if !t.IsEmpty() && !t.Cards[len(t.Cards)-1].FaceUp {
			flip := &FlipTableauCardCmd{ColumnIdx: col}
			_ = flip.Execute(g.state) // always succeeds: top card exists and is face-down
			flips = append(flips, flip)
		}
	}

	// Build the command recorded in history.
	var recorded Command
	if len(flips) == 0 {
		recorded = cmd
	} else {
		all := make([]Command, 0, 1+len(flips))
		all = append(all, cmd)
		all = append(all, flips...)
		recorded = &CompoundCmd{Cmds: all}
	}

	delta := g.scoreForCmd(recorded)
	scoreAfter := scoreBefore + delta
	if scoreAfter < 0 {
		scoreAfter = 0
	}
	g.state.Score = scoreAfter
	g.state.MoveCount++

	g.history.Push(recorded, scoreBefore, scoreAfter)
	return nil
}

// scoreForCmd returns the total point delta for cmd by inspecting its concrete type.
func (g *Game) scoreForCmd(cmd Command) int {
	switch c := cmd.(type) {
	case *MoveCardCmd:
		return g.scorer.OnMove(Move{From: c.From, To: c.To, CardCount: c.CardCount}, g.state)
	case *MoveToFoundationCmd:
		to := PileFoundation0 + PileID(c.FoundationIdx)
		return g.scorer.OnMove(Move{From: c.From, To: to, CardCount: 1}, g.state)
	case *FlipTableauCardCmd:
		return g.scorer.OnFlipTableau()
	case *RecycleStockCmd:
		return g.scorer.OnRecycleStock()
	case *FlipStockCmd:
		return 0
	case *CompoundCmd:
		total := 0
		for _, sub := range c.Cmds {
			total += g.scoreForCmd(sub)
		}
		return total
	}
	return 0
}

// Undo reverses the most recent move, restoring the pre-move score.
// MoveCount is not decremented (standard Klondike behaviour).
func (g *Game) Undo() error { return g.history.Undo(g.state) }

// Redo re-applies the most recently undone move, restoring its recorded post-move score.
func (g *Game) Redo() error { return g.history.Redo(g.state) }

// ValidMoves returns all currently legal moves in the game state.
func (g *Game) ValidMoves() []Move { return ValidMoves(g.state) }

// IsValidMove reports whether move is legal in the current state.
func (g *Game) IsValidMove(move Move) bool { return ValidateMove(g.state, move) == nil }

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
