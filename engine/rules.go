package engine

// Move describes a single game action: moving CardCount cards from pile From to pile To.
// For stock flip/recycle, CardCount is 0 and To/From may equal PileStock or PileWaste.
type Move struct {
	From      PileID
	To        PileID
	CardCount int
}

// ValidateMove returns nil if move is legal in state, or a descriptive error otherwise.
func ValidateMove(state *GameState, move Move) error { panic("not implemented") }

// ValidMoves returns all currently legal moves in state.
func ValidMoves(state *GameState) []Move { panic("not implemented") }
