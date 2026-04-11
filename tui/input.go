package tui

import tea "github.com/charmbracelet/bubbletea"

// GameAction represents a game-level action translated from a raw input event.
type GameAction int

const (
	ActionNone GameAction = iota

	// Cursor movement — arrow keys and vim keys
	ActionCursorUp
	ActionCursorDown
	ActionCursorLeft
	ActionCursorRight

	// Tab cycling — visits all 13 piles including foundations
	ActionTabNext // Tab
	ActionTabPrev // Shift-Tab

	// Selection
	ActionSelect // Enter or click — pick up or place card(s)
	ActionCancel // Esc — deselect current selection

	// Shortcuts
	ActionFlipStock        // Spacebar
	ActionJumpToColumn     // 1-7 number keys; payload = int column index (0-based)
	ActionMoveToFoundation // 'f' — auto-move selected to foundation

	// Meta
	ActionUndo           // Ctrl+Z
	ActionRedo           // Ctrl+Y
	ActionHint           // '?'
	ActionNewGame        // Ctrl+N
	ActionRestartDeal    // Ctrl+R
	ActionPause          // 'p'
	ActionHelp           // F1 or 'H'
	ActionQuit           // 'q' or Ctrl+C
	ActionToggleAutoMove // Ctrl+A
	ActionCycleTheme     // 't'
)

// TranslateInput maps a raw Bubbletea message to a GameAction and an optional
// payload. The payload is non-nil only for:
//   - ActionJumpToColumn: int column index (0-based, 0–6)
//   - ActionSelect from mouse: the original tea.MouseMsg (for hit-testing)
func TranslateInput(msg tea.Msg) (GameAction, interface{}) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		return translateKey(m)
	case tea.MouseMsg:
		return translateMouse(m)
	}
	return ActionNone, nil
}

func translateKey(m tea.KeyMsg) (GameAction, interface{}) {
	if m.Paste || m.Alt {
		return ActionNone, nil
	}
	switch m.Type {
	case tea.KeyLeft:
		return ActionCursorLeft, nil
	case tea.KeyRight:
		return ActionCursorRight, nil
	case tea.KeyUp:
		return ActionCursorUp, nil
	case tea.KeyDown:
		return ActionCursorDown, nil
	case tea.KeyTab:
		return ActionTabNext, nil
	case tea.KeyShiftTab:
		return ActionTabPrev, nil
	case tea.KeyEnter:
		return ActionSelect, nil
	case tea.KeySpace:
		return ActionFlipStock, nil
	case tea.KeyEsc:
		return ActionCancel, nil
	case tea.KeyCtrlZ:
		return ActionUndo, nil
	case tea.KeyCtrlY:
		return ActionRedo, nil
	case tea.KeyF1:
		return ActionHelp, nil
	case tea.KeyCtrlN:
		return ActionNewGame, nil
	case tea.KeyCtrlR:
		return ActionRestartDeal, nil
	case tea.KeyCtrlA:
		return ActionToggleAutoMove, nil
	case tea.KeyCtrlC:
		return ActionQuit, nil
	case tea.KeyRunes:
		if len(m.Runes) != 1 {
			// Multi-rune events come from IME composition; ignore rather than
			// firing a command from the first rune of an unrelated sequence.
			return ActionNone, nil
		}
		return translateRune(m.Runes[0])
	}
	return ActionNone, nil
}

func translateRune(r rune) (GameAction, interface{}) {
	switch r {
	case 'h':
		return ActionCursorLeft, nil
	case 'l':
		return ActionCursorRight, nil
	case 'k':
		return ActionCursorUp, nil
	case 'j':
		return ActionCursorDown, nil
	case 'f':
		return ActionMoveToFoundation, nil
	case '?':
		return ActionHint, nil
	case 'H':
		return ActionHelp, nil
	case 'p':
		return ActionPause, nil
	case 't':
		return ActionCycleTheme, nil
	case 'q':
		return ActionQuit, nil
	case '1', '2', '3', '4', '5', '6', '7':
		return ActionJumpToColumn, int(r - '1')
	}
	return ActionNone, nil
}

func translateMouse(m tea.MouseMsg) (GameAction, interface{}) {
	if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
		return ActionSelect, m
	}
	return ActionNone, nil
}
