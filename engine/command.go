package engine

// MoveCardCmd moves CardCount cards from pile From to pile To.
// Covers tableau↔tableau, waste→tableau, and foundation→tableau moves.
type MoveCardCmd struct {
	From      PileID
	To        PileID
	CardCount int
}

func (c *MoveCardCmd) Execute(state *GameState) error { panic("not implemented") }
func (c *MoveCardCmd) Undo(state *GameState) error    { panic("not implemented") }
func (c *MoveCardCmd) Description() string            { panic("not implemented") }

// MoveToFoundationCmd moves the top card of pile From to the foundation at FoundationIdx.
// Specialized command used by auto-move logic.
type MoveToFoundationCmd struct {
	From         PileID
	FoundationIdx int
}

func (c *MoveToFoundationCmd) Execute(state *GameState) error { panic("not implemented") }
func (c *MoveToFoundationCmd) Undo(state *GameState) error    { panic("not implemented") }
func (c *MoveToFoundationCmd) Description() string            { panic("not implemented") }

// FlipStockCmd draws DrawCount cards from the stock to the waste pile.
type FlipStockCmd struct{}

func (c *FlipStockCmd) Execute(state *GameState) error { panic("not implemented") }
func (c *FlipStockCmd) Undo(state *GameState) error    { panic("not implemented") }
func (c *FlipStockCmd) Description() string            { panic("not implemented") }

// RecycleStockCmd moves all waste cards back to the stock (face-down, reversed).
type RecycleStockCmd struct{}

func (c *RecycleStockCmd) Execute(state *GameState) error { panic("not implemented") }
func (c *RecycleStockCmd) Undo(state *GameState) error    { panic("not implemented") }
func (c *RecycleStockCmd) Description() string            { panic("not implemented") }

// FlipTableauCardCmd flips the top face-down card of tableau column ColumnIdx face-up.
// Auto-triggered after a move exposes a face-down card.
type FlipTableauCardCmd struct {
	ColumnIdx int
}

func (c *FlipTableauCardCmd) Execute(state *GameState) error { panic("not implemented") }
func (c *FlipTableauCardCmd) Undo(state *GameState) error    { panic("not implemented") }
func (c *FlipTableauCardCmd) Description() string            { panic("not implemented") }

// CompoundCmd groups multiple atomic commands so they undo as a single logical action.
// Example: move + auto-flip must undo together with one Ctrl+Z.
type CompoundCmd struct {
	Cmds []Command
}

func (c *CompoundCmd) Execute(state *GameState) error { panic("not implemented") }
func (c *CompoundCmd) Undo(state *GameState) error    { panic("not implemented") }
func (c *CompoundCmd) Description() string            { panic("not implemented") }
