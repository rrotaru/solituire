package engine

import "errors"

// historyEntry pairs a command with the scores in effect immediately before
// and after its execution (post-clamp). Both values are needed so that both
// Undo and Redo can restore the exact score without relying on delta inversion.
type historyEntry struct {
	cmd         Command
	scoreBefore int
	scoreAfter  int
}

// History manages the undo/redo stacks for a game session.
type History struct {
	undoStack []historyEntry
	redoStack []historyEntry
}

// Push records a successfully executed command together with the score that
// was in effect immediately before (scoreBefore) and after (scoreAfter) the
// command ran. Clears the redo stack.
func (h *History) Push(cmd Command, scoreBefore, scoreAfter int) {
	h.undoStack = append(h.undoStack, historyEntry{cmd, scoreBefore, scoreAfter})
	h.redoStack = h.redoStack[:0]
}

// Undo reverses the most recent command and restores the pre-command score.
// If cmd.Undo returns an error the undo stack is left unchanged.
func (h *History) Undo(s *GameState) error {
	if len(h.undoStack) == 0 {
		return errors.New("nothing to undo")
	}
	entry := h.undoStack[len(h.undoStack)-1]
	if err := entry.cmd.Undo(s); err != nil {
		return err
	}
	s.Score = entry.scoreBefore
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	h.redoStack = append(h.redoStack, entry)
	return nil
}

// Redo re-applies the most recently undone command and restores the
// post-command score recorded at the time of the original execution.
// If cmd.Execute returns an error the redo stack is left unchanged.
func (h *History) Redo(s *GameState) error {
	if len(h.redoStack) == 0 {
		return errors.New("nothing to redo")
	}
	entry := h.redoStack[len(h.redoStack)-1]
	if err := entry.cmd.Execute(s); err != nil {
		return err
	}
	s.Score = entry.scoreAfter
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	h.undoStack = append(h.undoStack, entry)
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
