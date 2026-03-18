package engine

import (
	"testing"
)

// buildFlipCmd returns a FlipTableauCardCmd for column i and a state where it can execute.
// The column will have a single face-down card.
func buildFlippableState(colIdx int) (*GameState, *FlipTableauCardCmd) {
	state := buildState()
	state.Tableau[colIdx].Cards = []Card{faceDownCard(King, Spades)}
	return state, &FlipTableauCardCmd{ColumnIdx: colIdx}
}

func TestHistory_PushAndUndo(t *testing.T) {
	state, cmd := buildFlippableState(0)
	h := &History{}

	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	h.Push(cmd)

	if !h.CanUndo() {
		t.Error("CanUndo should be true after push")
	}
	if h.CanRedo() {
		t.Error("CanRedo should be false before any undo")
	}

	if err := h.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if !state.Tableau[0].Cards[0].FaceUp == false {
		// card should be face-down again
	}
	if state.Tableau[0].Cards[0].FaceUp {
		t.Error("card should be face-down after undo")
	}
	if h.CanUndo() {
		t.Error("CanUndo should be false after undoing all")
	}
	if !h.CanRedo() {
		t.Error("CanRedo should be true after undo")
	}
}

func TestHistory_Redo(t *testing.T) {
	state, cmd := buildFlippableState(0)
	h := &History{}

	if err := cmd.Execute(state); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	h.Push(cmd)
	if err := h.Undo(state); err != nil {
		t.Fatalf("Undo: %v", err)
	}

	// Card is face-down; redo should flip it face-up again.
	if err := h.Redo(state); err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if !state.Tableau[0].Cards[0].FaceUp {
		t.Error("card should be face-up after redo")
	}
	if !h.CanUndo() {
		t.Error("CanUndo should be true after redo")
	}
	if h.CanRedo() {
		t.Error("CanRedo should be false after redo")
	}
}

func TestHistory_UndoOnEmpty(t *testing.T) {
	state := buildState()
	h := &History{}
	if err := h.Undo(state); err == nil {
		t.Error("Undo on empty history should return error")
	}
}

func TestHistory_RedoOnEmpty(t *testing.T) {
	state := buildState()
	h := &History{}
	if err := h.Redo(state); err == nil {
		t.Error("Redo on empty history should return error")
	}
}

func TestHistory_PushClearsRedo(t *testing.T) {
	state, cmdA := buildFlippableState(0)
	h := &History{}

	// Execute and push cmdA.
	if err := cmdA.Execute(state); err != nil {
		t.Fatalf("Execute A: %v", err)
	}
	h.Push(cmdA)

	// Undo it → redo stack now has cmdA.
	if err := h.Undo(state); err != nil {
		t.Fatalf("Undo A: %v", err)
	}
	if !h.CanRedo() {
		t.Fatal("CanRedo should be true after undo")
	}

	// Push a new command → redo stack should be cleared.
	state.Tableau[1].Cards = []Card{faceDownCard(Queen, Hearts)}
	cmdB := &FlipTableauCardCmd{ColumnIdx: 1}
	if err := cmdB.Execute(state); err != nil {
		t.Fatalf("Execute B: %v", err)
	}
	h.Push(cmdB)

	if h.CanRedo() {
		t.Error("CanRedo should be false after new push")
	}
}

func TestHistory_MultipleUndoRedoCycles(t *testing.T) {
	state := buildState()
	// Set up 3 columns each with a face-down card.
	for i := 0; i < 3; i++ {
		state.Tableau[i].Cards = []Card{faceDownCard(King, Spades)}
	}
	h := &History{}

	// Execute and push 3 flip commands.
	cmds := []*FlipTableauCardCmd{
		{ColumnIdx: 0},
		{ColumnIdx: 1},
		{ColumnIdx: 2},
	}
	for i, cmd := range cmds {
		if err := cmd.Execute(state); err != nil {
			t.Fatalf("Execute %d: %v", i, err)
		}
		h.Push(cmd)
	}

	// All 3 columns face-up.
	for i := 0; i < 3; i++ {
		if !state.Tableau[i].Cards[0].FaceUp {
			t.Errorf("col %d should be face-up before undos", i)
		}
	}

	// Undo all 3 in reverse order (col2, col1, col0).
	for i := 2; i >= 0; i-- {
		if err := h.Undo(state); err != nil {
			t.Fatalf("Undo %d: %v", i, err)
		}
		if state.Tableau[i].Cards[0].FaceUp {
			t.Errorf("col %d should be face-down after undo", i)
		}
	}
	if h.CanUndo() {
		t.Error("CanUndo should be false after undoing all")
	}

	// Redo all 3 (col0, col1, col2).
	for i := 0; i < 3; i++ {
		if err := h.Redo(state); err != nil {
			t.Fatalf("Redo %d: %v", i, err)
		}
		if !state.Tableau[i].Cards[0].FaceUp {
			t.Errorf("col %d should be face-up after redo", i)
		}
	}
	if h.CanRedo() {
		t.Error("CanRedo should be false after redoing all")
	}

	// Undo again to verify we can cycle.
	for i := 2; i >= 0; i-- {
		if err := h.Undo(state); err != nil {
			t.Fatalf("Second undo %d: %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		if state.Tableau[i].Cards[0].FaceUp {
			t.Errorf("col %d should be face-down after second undo cycle", i)
		}
	}
}

func TestHistory_Clear(t *testing.T) {
	state, cmdA := buildFlippableState(0)
	h := &History{}

	if err := cmdA.Execute(state); err != nil {
		t.Fatalf("Execute A: %v", err)
	}
	h.Push(cmdA)

	state.Tableau[1].Cards = []Card{faceDownCard(Queen, Hearts)}
	cmdB := &FlipTableauCardCmd{ColumnIdx: 1}
	if err := cmdB.Execute(state); err != nil {
		t.Fatalf("Execute B: %v", err)
	}
	h.Push(cmdB)

	h.Clear()

	if h.CanUndo() {
		t.Error("CanUndo should be false after Clear")
	}
	if h.CanRedo() {
		t.Error("CanRedo should be false after Clear")
	}
}
