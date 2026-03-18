package engine

import "errors"

// History manages the undo/redo stacks for a game session.
type History struct {
	undoStack []Command
	redoStack []Command
}

// Push records a successfully executed command. Clears the redo stack.
func (h *History) Push(cmd Command) {
	h.undoStack = append(h.undoStack, cmd)
	h.redoStack = h.redoStack[:0]
}

// Undo reverses the most recent command and moves it to the redo stack.
// If cmd.Undo returns an error the undo stack is left unchanged.
func (h *History) Undo(s *GameState) error {
	if len(h.undoStack) == 0 {
		return errors.New("nothing to undo")
	}
	cmd := h.undoStack[len(h.undoStack)-1]
	if err := cmd.Undo(s); err != nil {
		return err
	}
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	h.redoStack = append(h.redoStack, cmd)
	return nil
}

// Redo re-applies the most recently undone command.
// If cmd.Execute returns an error the redo stack is left unchanged.
func (h *History) Redo(s *GameState) error {
	if len(h.redoStack) == 0 {
		return errors.New("nothing to redo")
	}
	cmd := h.redoStack[len(h.redoStack)-1]
	if err := cmd.Execute(s); err != nil {
		return err
	}
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	h.undoStack = append(h.undoStack, cmd)
	return nil
}

// CanUndo returns true when there is at least one command to undo.
func (h *History) CanUndo() bool { return len(h.undoStack) > 0 }

// CanRedo returns true when there is at least one command to redo.
func (h *History) CanRedo() bool { return len(h.redoStack) > 0 }

// Clear empties both the undo and redo stacks.
func (h *History) Clear() {
	h.undoStack = nil
	h.redoStack = nil
}
