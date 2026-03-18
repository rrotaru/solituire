package engine

import (
	"errors"
	"fmt"
	"strings"
)

// MoveCardCmd moves CardCount cards from pile From to pile To.
// Covers tableau↔tableau, waste→tableau, and foundation→tableau moves.
type MoveCardCmd struct {
	From      PileID
	To        PileID
	CardCount int
}

func (c *MoveCardCmd) Execute(state *GameState) error {
	if !isTableauPile(c.To) {
		return errors.New("MoveCardCmd: destination must be a tableau pile; use MoveToFoundationCmd for foundation moves")
	}
	if err := ValidateMove(state, Move{From: c.From, To: c.To, CardCount: c.CardCount}); err != nil {
		return err
	}

	// Extract cards from source.
	var cards []Card
	switch {
	case c.From == PileWaste:
		w := state.Waste
		cards = []Card{w.Cards[len(w.Cards)-1]}
		state.Waste.Cards = w.Cards[:len(w.Cards)-1]
	case isFoundationPile(c.From):
		fi := foundationIndex(c.From)
		f := state.Foundations[fi]
		cards = []Card{f.Cards[len(f.Cards)-1]}
		state.Foundations[fi].Cards = f.Cards[:len(f.Cards)-1]
	case isTableauPile(c.From):
		ti := tableauIndex(c.From)
		t := state.Tableau[ti]
		split := len(t.Cards) - c.CardCount
		cards = make([]Card, c.CardCount)
		copy(cards, t.Cards[split:])
		state.Tableau[ti].Cards = t.Cards[:split]
	}

	// Append to destination tableau.
	ti := tableauIndex(c.To)
	state.Tableau[ti].Cards = append(state.Tableau[ti].Cards, cards...)
	return nil
}

func (c *MoveCardCmd) Undo(state *GameState) error {
	ti := tableauIndex(c.To)
	t := state.Tableau[ti]
	split := len(t.Cards) - c.CardCount
	cards := make([]Card, c.CardCount)
	copy(cards, t.Cards[split:])
	state.Tableau[ti].Cards = t.Cards[:split]

	switch {
	case c.From == PileWaste:
		state.Waste.Cards = append(state.Waste.Cards, cards...)
	case isFoundationPile(c.From):
		fi := foundationIndex(c.From)
		state.Foundations[fi].Cards = append(state.Foundations[fi].Cards, cards...)
	case isTableauPile(c.From):
		ti2 := tableauIndex(c.From)
		state.Tableau[ti2].Cards = append(state.Tableau[ti2].Cards, cards...)
	}
	return nil
}

func (c *MoveCardCmd) Description() string {
	return fmt.Sprintf("Move %d card(s) from pile %d to pile %d", c.CardCount, c.From, c.To)
}

// MoveToFoundationCmd moves the top card of pile From to the foundation at FoundationIdx.
// Specialized command used by auto-move logic.
type MoveToFoundationCmd struct {
	From          PileID
	FoundationIdx int
}

func (c *MoveToFoundationCmd) Execute(state *GameState) error {
	if c.FoundationIdx < 0 || c.FoundationIdx >= 4 {
		return errors.New("MoveToFoundationCmd: FoundationIdx out of range [0, 3]")
	}
	to := PileFoundation0 + PileID(c.FoundationIdx)
	if err := ValidateMove(state, Move{From: c.From, To: to, CardCount: 1}); err != nil {
		return err
	}

	var card Card
	switch {
	case c.From == PileWaste:
		w := state.Waste
		card = w.Cards[len(w.Cards)-1]
		state.Waste.Cards = w.Cards[:len(w.Cards)-1]
	case isTableauPile(c.From):
		ti := tableauIndex(c.From)
		t := state.Tableau[ti]
		card = t.Cards[len(t.Cards)-1]
		state.Tableau[ti].Cards = t.Cards[:len(t.Cards)-1]
	}

	card.FaceUp = true
	state.Foundations[c.FoundationIdx].Cards = append(state.Foundations[c.FoundationIdx].Cards, card)
	return nil
}

func (c *MoveToFoundationCmd) Undo(state *GameState) error {
	if c.FoundationIdx < 0 || c.FoundationIdx >= 4 {
		return errors.New("MoveToFoundationCmd: FoundationIdx out of range [0, 3]")
	}
	f := state.Foundations[c.FoundationIdx]
	card := f.Cards[len(f.Cards)-1]
	state.Foundations[c.FoundationIdx].Cards = f.Cards[:len(f.Cards)-1]

	switch {
	case c.From == PileWaste:
		state.Waste.Cards = append(state.Waste.Cards, card)
	case isTableauPile(c.From):
		ti := tableauIndex(c.From)
		state.Tableau[ti].Cards = append(state.Tableau[ti].Cards, card)
	}
	return nil
}

func (c *MoveToFoundationCmd) Description() string {
	return fmt.Sprintf("Move top card from pile %d to foundation %d", c.From, c.FoundationIdx)
}

// FlipStockCmd draws DrawCount cards from the stock to the waste pile.
type FlipStockCmd struct {
	flippedCount int // set during Execute; used by Undo
}

func (c *FlipStockCmd) Execute(state *GameState) error {
	if state.Stock.IsEmpty() {
		return errors.New("use RecycleStockCmd to recycle waste to stock")
	}

	n := state.DrawCount
	if n > len(state.Stock.Cards) {
		n = len(state.Stock.Cards)
	}
	c.flippedCount = n

	// Take n cards from top of stock (last n elements).
	drawn := make([]Card, n)
	copy(drawn, state.Stock.Cards[len(state.Stock.Cards)-n:])
	state.Stock.Cards = state.Stock.Cards[:len(state.Stock.Cards)-n]

	// Reverse drawn before appending to waste: the deepest drawn card
	// (index 0 of drawn) becomes the new waste top (playable).
	for i, j := 0, len(drawn)-1; i < j; i, j = i+1, j-1 {
		drawn[i], drawn[j] = drawn[j], drawn[i]
	}
	for i := range drawn {
		drawn[i].FaceUp = true
	}
	state.Waste.Cards = append(state.Waste.Cards, drawn...)
	return nil
}

func (c *FlipStockCmd) Undo(state *GameState) error {
	if c.flippedCount == 0 {
		return errors.New("FlipStockCmd: Execute was not called")
	}
	n := c.flippedCount

	// Take n cards from top of waste.
	w := state.Waste
	taken := make([]Card, n)
	copy(taken, w.Cards[len(w.Cards)-n:])
	state.Waste.Cards = w.Cards[:len(w.Cards)-n]

	// Reverse back to restore original stock order.
	for i, j := 0, len(taken)-1; i < j; i, j = i+1, j-1 {
		taken[i], taken[j] = taken[j], taken[i]
	}
	for i := range taken {
		taken[i].FaceUp = false
	}
	state.Stock.Cards = append(state.Stock.Cards, taken...)
	c.flippedCount = 0
	return nil
}

func (c *FlipStockCmd) Description() string {
	return "Flip stock cards to waste"
}

// RecycleStockCmd moves all waste cards back to the stock (face-down, reversed).
type RecycleStockCmd struct{}

func (c *RecycleStockCmd) Execute(state *GameState) error {
	if !state.Stock.IsEmpty() {
		return errors.New("cannot recycle: stock is not empty")
	}
	if state.Waste.IsEmpty() {
		return errors.New("cannot recycle: waste is empty")
	}

	w := state.Waste
	n := len(w.Cards)
	// Reverse waste into stock, face-down: waste top (last) becomes stock bottom (first).
	stock := make([]Card, n)
	for i, card := range w.Cards {
		card.FaceUp = false
		stock[n-1-i] = card
	}
	state.Stock.Cards = stock
	state.Waste.Cards = state.Waste.Cards[:0]
	return nil
}

func (c *RecycleStockCmd) Undo(state *GameState) error {
	// Reverse all stock cards back to waste, face-up.
	s := state.Stock
	n := len(s.Cards)
	waste := make([]Card, n)
	for i, card := range s.Cards {
		card.FaceUp = true
		waste[n-1-i] = card
	}
	state.Waste.Cards = append(state.Waste.Cards, waste...)
	state.Stock.Cards = state.Stock.Cards[:0]
	return nil
}

func (c *RecycleStockCmd) Description() string {
	return "Recycle waste to stock"
}

// FlipTableauCardCmd flips the top face-down card of tableau column ColumnIdx face-up.
// Auto-triggered after a move exposes a face-down card.
type FlipTableauCardCmd struct {
	ColumnIdx int
}

func (c *FlipTableauCardCmd) Execute(state *GameState) error {
	if c.ColumnIdx < 0 || c.ColumnIdx >= len(state.Tableau) {
		return fmt.Errorf("FlipTableauCardCmd: ColumnIdx %d out of range [0, %d)", c.ColumnIdx, len(state.Tableau))
	}
	t := state.Tableau[c.ColumnIdx]
	if t.IsEmpty() {
		return errors.New("FlipTableauCardCmd: tableau column is empty")
	}
	top := &t.Cards[len(t.Cards)-1]
	if top.FaceUp {
		return errors.New("FlipTableauCardCmd: top card is already face-up")
	}
	top.FaceUp = true
	return nil
}

func (c *FlipTableauCardCmd) Undo(state *GameState) error {
	if c.ColumnIdx < 0 || c.ColumnIdx >= len(state.Tableau) {
		return fmt.Errorf("FlipTableauCardCmd: ColumnIdx %d out of range [0, %d)", c.ColumnIdx, len(state.Tableau))
	}
	t := state.Tableau[c.ColumnIdx]
	if t.IsEmpty() {
		return errors.New("FlipTableauCardCmd Undo: tableau column is empty")
	}
	top := &t.Cards[len(t.Cards)-1]
	if !top.FaceUp {
		return errors.New("FlipTableauCardCmd Undo: top card is already face-down")
	}
	top.FaceUp = false
	return nil
}

func (c *FlipTableauCardCmd) Description() string {
	return fmt.Sprintf("Flip top card of tableau column %d face-up", c.ColumnIdx)
}

// CompoundCmd groups multiple atomic commands so they undo as a single logical action.
// Example: move + auto-flip must undo together with one Ctrl+Z.
type CompoundCmd struct {
	Cmds []Command
}

func (c *CompoundCmd) Execute(state *GameState) error {
	for i, cmd := range c.Cmds {
		if err := cmd.Execute(state); err != nil {
			// Rollback previously executed sub-commands in reverse order.
			for j := i - 1; j >= 0; j-- {
				_ = c.Cmds[j].Undo(state) // best-effort rollback
			}
			return err
		}
	}
	return nil
}

func (c *CompoundCmd) Undo(state *GameState) error {
	for i := len(c.Cmds) - 1; i >= 0; i-- {
		if err := c.Cmds[i].Undo(state); err != nil {
			return err
		}
	}
	return nil
}

func (c *CompoundCmd) Description() string {
	parts := make([]string, len(c.Cmds))
	for i, cmd := range c.Cmds {
		parts[i] = cmd.Description()
	}
	return strings.Join(parts, " + ")
}
