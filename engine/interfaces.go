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
}
