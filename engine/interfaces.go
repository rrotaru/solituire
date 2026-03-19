package engine

// GameEngine is the primary interface through which the TUI shell interacts
// with game logic. The engine package implements this via the Game struct.
type GameEngine interface {
	// State queries
	State() *GameState
	IsWon() bool
	IsAutoCompletable() bool
	Score() int
	MoveCount() int
	Seed() int64

	// Command execution
	Execute(cmd Command) error
	Undo() error
	Redo() error
	CanUndo() bool
	CanRedo() bool

	// Query helpers
	ValidMoves() []Move
	IsValidMove(move Move) bool

	// Game lifecycle
	NewGame(seed int64, drawCount int)
	RestartDeal()
}

// Command represents a reversible game action.
// Every mutation to GameState must go through a Command so Undo is always possible.
type Command interface {
	Execute(state *GameState) error
	Undo(state *GameState) error
	Description() string // human-readable, e.g. "Move K♠ from tableau[3] to tableau[0]"
}

// Scorer computes point deltas for scoring events.
// The interface exists so alternative scoring systems (e.g. Vegas) can be added
// without changing existing code.
type Scorer interface {
	OnMove(move Move, state *GameState) int // returns point delta
	OnUndo(move Move, state *GameState) int // returns point delta (negative of original)
	OnFlipTableau() int                     // returns +5 for standard scoring
	OnRecycleStock() int                    // returns -100 for standard scoring
}
