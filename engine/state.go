package engine

import "time"

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
	DrawCount   int // 1 or 3
	Seed        int64
}
