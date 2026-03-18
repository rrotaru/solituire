package engine

// History manages the undo/redo stacks for a game session.
type History struct {
	undoStack []Command
	redoStack []Command
}

// Push records a successfully executed command. Clears the redo stack.
func (h *History) Push(cmd Command) { panic("not implemented") }

// Undo reverses the most recent command and moves it to the redo stack.
func (h *History) Undo(s *GameState) error { panic("not implemented") }

// Redo re-applies the most recently undone command.
func (h *History) Redo(s *GameState) error { panic("not implemented") }

// CanUndo returns true when there is at least one command to undo.
func (h *History) CanUndo() bool { panic("not implemented") }

// CanRedo returns true when there is at least one command to redo.
func (h *History) CanRedo() bool { panic("not implemented") }

// Clear empties both the undo and redo stacks.
func (h *History) Clear() { panic("not implemented") }
