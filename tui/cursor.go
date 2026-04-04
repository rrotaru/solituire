package tui

import (
	"solituire/engine"
	"solituire/renderer"
)

// navCycleOrder is the Left/Right arrow key navigation cycle.
// Foundations are omitted — they are reachable only via Tab.
var navCycleOrder = []engine.PileID{
	engine.PileStock,
	engine.PileWaste,
	engine.PileTableau0,
	engine.PileTableau1,
	engine.PileTableau2,
	engine.PileTableau3,
	engine.PileTableau4,
	engine.PileTableau5,
	engine.PileTableau6,
}

// tabCycleOrder is the Tab / Shift-Tab cycling order across all piles.
var tabCycleOrder = []engine.PileID{
	engine.PileStock,
	engine.PileWaste,
	engine.PileFoundation0,
	engine.PileFoundation1,
	engine.PileFoundation2,
	engine.PileFoundation3,
	engine.PileTableau0,
	engine.PileTableau1,
	engine.PileTableau2,
	engine.PileTableau3,
	engine.PileTableau4,
	engine.PileTableau5,
	engine.PileTableau6,
}

// Cursor holds the board cursor position plus drag and hint state.
// Call RendererCursor() to produce a renderer.CursorState for rendering.
type Cursor struct {
	Pile          engine.PileID
	CardIndex     int // 0-based within the pile's Cards slice
	Dragging      bool
	DragSource    engine.PileID
	DragCardCount int // number of cards being dragged from DragSource
	ShowHint      bool
	HintFrom      engine.PileID
	HintTo        engine.PileID
}

// RendererCursor converts the internal cursor state to renderer.CursorState
// for passing into renderer.Render.
func (c Cursor) RendererCursor() renderer.CursorState {
	return renderer.CursorState{
		Pile:      c.Pile,
		CardIndex: c.CardIndex,
		Dragging:  c.Dragging,
		HintFrom:  c.HintFrom,
		HintTo:    c.HintTo,
		ShowHint:  c.ShowHint,
	}
}

// MoveLeft cycles one step left in navCycleOrder, wrapping at the start.
// Foundations are not in navCycleOrder; when the cursor is on any foundation
// the nearest left neighbour is Waste (the last top-row pile before foundations).
func (c *Cursor) MoveLeft(state *engine.GameState) {
	if isFoundationPile(c.Pile) {
		c.Pile = engine.PileWaste
	} else {
		c.Pile = prevInCycle(navCycleOrder, c.Pile)
	}
	c.CardIndex = naturalCardIndex(c.Pile, state)
}

// MoveRight cycles one step right in navCycleOrder, wrapping at the end.
// Foundations are not in navCycleOrder; when the cursor is on any foundation
// the nearest right neighbour is Tableau0 (the first pile after foundations).
func (c *Cursor) MoveRight(state *engine.GameState) {
	if isFoundationPile(c.Pile) {
		c.Pile = engine.PileTableau0
	} else {
		c.Pile = nextInCycle(navCycleOrder, c.Pile)
	}
	c.CardIndex = naturalCardIndex(c.Pile, state)
}

// TabNext cycles one step forward in tabCycleOrder, wrapping at the end.
func (c *Cursor) TabNext(state *engine.GameState) {
	c.Pile = nextInCycle(tabCycleOrder, c.Pile)
	c.CardIndex = naturalCardIndex(c.Pile, state)
}

// TabPrev cycles one step backward in tabCycleOrder, wrapping at the start.
func (c *Cursor) TabPrev(state *engine.GameState) {
	c.Pile = prevInCycle(tabCycleOrder, c.Pile)
	c.CardIndex = naturalCardIndex(c.Pile, state)
}

// MoveUp decreases CardIndex within a tableau column, stopping at the topmost
// face-up card. No-op if the current pile is not a tableau pile.
func (c *Cursor) MoveUp(state *engine.GameState) {
	if !isTableauPile(c.Pile) {
		return
	}
	col := int(c.Pile - engine.PileTableau0)
	pile := state.Tableau[col]
	minIdx := pile.FaceDownCount()
	if c.CardIndex > minIdx {
		c.CardIndex--
	}
}

// MoveDown increases CardIndex within a tableau column, stopping at the last card.
// No-op if the current pile is not a tableau pile.
func (c *Cursor) MoveDown(state *engine.GameState) {
	if !isTableauPile(c.Pile) {
		return
	}
	col := int(c.Pile - engine.PileTableau0)
	pile := state.Tableau[col]
	if pile.IsEmpty() {
		return
	}
	max := len(pile.Cards) - 1
	if c.CardIndex < max {
		c.CardIndex++
	}
}

// JumpToColumn sets the cursor to the bottom card of tableau column col (0-based).
func (c *Cursor) JumpToColumn(col int, state *engine.GameState) {
	c.Pile = engine.PileTableau0 + engine.PileID(col)
	c.CardIndex = naturalCardIndex(c.Pile, state)
}

// naturalCardIndex returns the default CardIndex when landing on a pile:
//   - Tableau: index of the last card (the bottom / most-accessible card)
//   - All others: 0
func naturalCardIndex(pile engine.PileID, state *engine.GameState) int {
	if isTableauPile(pile) {
		col := int(pile - engine.PileTableau0)
		p := state.Tableau[col]
		if len(p.Cards) > 0 {
			return len(p.Cards) - 1
		}
	}
	return 0
}

// isTableauPile returns true if id is one of PileTableau0..PileTableau6.
func isTableauPile(id engine.PileID) bool {
	return id >= engine.PileTableau0 && id <= engine.PileTableau6
}

// isFoundationPile returns true if id is one of PileFoundation0..PileFoundation3.
func isFoundationPile(id engine.PileID) bool {
	return id >= engine.PileFoundation0 && id <= engine.PileFoundation3
}

// nextInCycle returns the PileID after current in the cycle, wrapping to the first.
// If current is not found, returns the first element.
func nextInCycle(cycle []engine.PileID, current engine.PileID) engine.PileID {
	for i, p := range cycle {
		if p == current {
			return cycle[(i+1)%len(cycle)]
		}
	}
	return cycle[0]
}

// prevInCycle returns the PileID before current in the cycle, wrapping to the last.
// If current is not found, returns the last element.
func prevInCycle(cycle []engine.PileID, current engine.PileID) engine.PileID {
	for i, p := range cycle {
		if p == current {
			return cycle[(i+len(cycle)-1)%len(cycle)]
		}
	}
	return cycle[len(cycle)-1]
}
