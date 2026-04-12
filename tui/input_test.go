package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTranslateInput(t *testing.T) {
	type testCase struct {
		name    string
		msg     tea.Msg
		want    GameAction
		payload interface{} // nil means "don't check payload"
	}

	tests := []testCase{
		// Cursor — arrow keys
		{"arrow left", tea.KeyMsg{Type: tea.KeyLeft}, ActionCursorLeft, nil},
		{"arrow right", tea.KeyMsg{Type: tea.KeyRight}, ActionCursorRight, nil},
		{"arrow up", tea.KeyMsg{Type: tea.KeyUp}, ActionCursorUp, nil},
		{"arrow down", tea.KeyMsg{Type: tea.KeyDown}, ActionCursorDown, nil},

		// Cursor — vim keys
		{"vim h", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}, ActionCursorLeft, nil},
		{"vim l", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}, ActionCursorRight, nil},
		{"vim k", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, ActionCursorUp, nil},
		{"vim j", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, ActionCursorDown, nil},

		// Tab cycling — uses tabCycleOrder which includes foundations
		{"tab", tea.KeyMsg{Type: tea.KeyTab}, ActionTabNext, nil},
		{"shift+tab", tea.KeyMsg{Type: tea.KeyShiftTab}, ActionTabPrev, nil},

		// Number keys 1-7 — action only (payload checked in dedicated test)
		{"key 1", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}, ActionJumpToColumn, nil},
		{"key 2", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}, ActionJumpToColumn, nil},
		{"key 3", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}, ActionJumpToColumn, nil},
		{"key 4", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}, ActionJumpToColumn, nil},
		{"key 5", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")}, ActionJumpToColumn, nil},
		{"key 6", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")}, ActionJumpToColumn, nil},
		{"key 7", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("7")}, ActionJumpToColumn, nil},

		// Selection / stock
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, ActionSelect, nil},
		{"space", tea.KeyMsg{Type: tea.KeySpace}, ActionFlipStock, nil},
		{"escape", tea.KeyMsg{Type: tea.KeyEsc}, ActionCancel, nil},

		// Shortcuts
		{"f foundation", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}, ActionMoveToFoundation, nil},

		// Undo / redo
		{"ctrl+z", tea.KeyMsg{Type: tea.KeyCtrlZ}, ActionUndo, nil},
		{"ctrl+y", tea.KeyMsg{Type: tea.KeyCtrlY}, ActionRedo, nil},

		// Hint
		{"question mark", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}, ActionHint, nil},

		// Meta
		{"F1", tea.KeyMsg{Type: tea.KeyF1}, ActionHelp, nil},
		{"H help", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")}, ActionHelp, nil},
		{"p pause", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}, ActionPause, nil},
		{"ctrl+n", tea.KeyMsg{Type: tea.KeyCtrlN}, ActionNewGame, nil},
		{"ctrl+r", tea.KeyMsg{Type: tea.KeyCtrlR}, ActionRestartDeal, nil},
		{"ctrl+a", tea.KeyMsg{Type: tea.KeyCtrlA}, ActionToggleAutoMove, nil},
		{"t theme", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}, ActionCycleTheme, nil},
		{"q quit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, ActionQuit, nil},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}, ActionQuit, nil},

		// Mouse — left click → ActionSelect (payload checked in dedicated test)
		{"mouse left click", tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonLeft,
			X:      15, Y: 10,
		}, ActionSelect, nil},

		// Unmapped keys → ActionNone
		{"unmapped rune a", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, ActionNone, nil},
		{"unmapped rune x", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}, ActionNone, nil},
		{"unmapped F2", tea.KeyMsg{Type: tea.KeyF2}, ActionNone, nil},
		{"non-key msg", tea.WindowSizeMsg{Width: 80, Height: 24}, ActionNone, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, payload := TranslateInput(tt.msg)
			if got != tt.want {
				t.Errorf("TranslateInput(%v) action = %v, want %v", tt.msg, got, tt.want)
			}
			if tt.payload != nil && payload != tt.payload {
				t.Errorf("TranslateInput(%v) payload = %v, want %v", tt.msg, payload, tt.payload)
			}
		})
	}
}

// TestTranslateInput_MousePayload verifies the mouse click payload is the original MouseMsg.
func TestTranslateInput_MousePayload(t *testing.T) {
	m := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 42, Y: 7}
	_, payload := TranslateInput(m)
	got, ok := payload.(tea.MouseMsg)
	if !ok {
		t.Fatalf("expected payload type tea.MouseMsg, got %T", payload)
	}
	if got.X != 42 || got.Y != 7 {
		t.Errorf("payload coordinates = (%d,%d), want (42,7)", got.X, got.Y)
	}
}

// TestTranslateInput_JumpColumnPayload verifies the 0-based column index payload for keys 1-7.
func TestTranslateInput_JumpColumnPayload(t *testing.T) {
	for col := 1; col <= 7; col++ {
		r := rune('0' + col)
		_, payload := TranslateInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		idx, ok := payload.(int)
		if !ok {
			t.Fatalf("key %c: expected int payload, got %T", r, payload)
		}
		if idx != col-1 {
			t.Errorf("key %c: payload = %d, want %d", r, idx, col-1)
		}
	}
}

// TestTranslateInput_PasteIgnored verifies that KeyRunes events marked as paste
// produce ActionNone, preventing bracketed paste from triggering game commands.
func TestTranslateInput_PasteIgnored(t *testing.T) {
	pasteRunes := []rune{'q', 'p', '1', 'f', 't', 'j', 'k', 'h', 'l', '?'}
	for _, r := range pasteRunes {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Paste: true}
		got, _ := TranslateInput(msg)
		if got != ActionNone {
			t.Errorf("pasted rune %q: got %v, want ActionNone", r, got)
		}
	}
}

// TestTranslateInput_AltModifiedIgnored verifies that Alt-modified keys produce
// ActionNone so terminal/tmux meta-key shortcuts are not misread as game commands.
func TestTranslateInput_AltModifiedIgnored(t *testing.T) {
	cases := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}, Alt: true},
		{Type: tea.KeyRunes, Runes: []rune{'h'}, Alt: true},
		{Type: tea.KeyRunes, Runes: []rune{'1'}, Alt: true},
		{Type: tea.KeyLeft, Alt: true},
		{Type: tea.KeyEnter, Alt: true},
	}
	for _, m := range cases {
		got, _ := TranslateInput(m)
		if got != ActionNone {
			t.Errorf("Alt+%v: got %v, want ActionNone", m, got)
		}
	}
}

// TestTranslateInput_MultiRuneIgnored verifies that IME-composed multi-rune
// KeyRunes events produce ActionNone rather than firing from the first rune.
func TestTranslateInput_MultiRuneIgnored(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q', 'u'}}
	got, _ := TranslateInput(msg)
	if got != ActionNone {
		t.Errorf("multi-rune KeyRunes: got %v, want ActionNone", got)
	}
}

// TestTranslateInput_MouseRelease verifies non-press mouse events are ignored.
func TestTranslateInput_MouseRelease(t *testing.T) {
	m := tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: 10, Y: 5}
	got, _ := TranslateInput(m)
	if got != ActionNone {
		t.Errorf("mouse release: got %v, want ActionNone", got)
	}
}
