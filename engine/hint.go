package engine

// Hint describes a suggested move with a priority for ordering.
// Higher Priority values are shown first.
type Hint struct {
	From     PileID
	CardIdx  int // index within the source pile's Cards slice
	To       PileID
	Priority int
}

// FindHints returns all legal moves in state sorted by priority (highest first).
// Priority order: foundation move > expose face-down card > King to empty > build length > stock flip.
// Returns an empty slice when no moves are available.
func FindHints(state *GameState) []Hint { panic("not implemented") }
