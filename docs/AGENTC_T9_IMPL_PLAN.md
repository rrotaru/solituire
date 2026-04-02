# Agent C — Phase T9 Implementation Plan
## Input Translator

**Branch**: `claude/agent-c-phase-t9-plan-Ura2B`
**Dependencies**: T1 complete ✓ (Bubbletea is in `go.mod`)
**Blocks**: Agent C T11 (Board Model) — needs `GameAction`, `TranslateInput`, and all custom `Msg` types

---

## Overview

T9 delivers the pure input-translation layer and the full set of TUI message types. No engine or renderer code is touched. The only package imported outside the standard library is `github.com/charmbracelet/bubbletea`.

### Files created

| File | Contents |
|------|----------|
| `tui/input.go` | `GameAction` enum, `TranslateInput` function |
| `tui/messages.go` | `AppScreen` enum, all custom `Msg` types |
| `tui/input_test.go` | Table-driven tests for every binding in Section 8.2 |

`tui/tui.go` already exists as a `package tui` stub — do **not** modify it.

---

## Design Decisions

### `TranslateInput` signature

The design spec notation `(GameAction, ...payload)` is not valid Go. The second return is `interface{}` carrying an optional payload:

```go
func TranslateInput(msg tea.Msg) (GameAction, interface{})
```

The payload is only meaningful for `ActionJumpToColumn` (carries `int` column index 0–6) and `ActionSelect` from a mouse click (carries `tea.MouseMsg` for downstream hit-testing). All other actions return `nil` as the second value.

### `AppScreen` placement

`ChangeScreenMsg` (defined in `messages.go`) references `AppScreen`. Since `tui/app.go` does not exist until T13, `AppScreen` must be defined somewhere that compiles in T9. Define it **in `messages.go`** rather than creating a stub `app.go`. When T13 is implemented, it uses `AppScreen` from `messages.go` — do **not** redefine it in `app.go`.

### Import path for `config` and `theme`

`ConfigChangedMsg` and `ThemeChangedMsg` carry pointers to types from sibling packages:

```go
import (
    "solituire/config"
    "solituire/theme"
)
```

Both packages exist and compile; this is safe.

---

## Step 1 — `tui/messages.go`

Define `AppScreen` first (needed by `ChangeScreenMsg`), then all message types from Section 7.3.

```go
package tui

import (
    "time"

    "solituire/config"
    "solituire/theme"
)

// AppScreen identifies which screen the application is currently showing.
// Defined here (not app.go) so that ChangeScreenMsg compiles in T9.
// T13 must NOT redefine this type.
type AppScreen int

const (
    ScreenMenu AppScreen = iota
    ScreenPlaying
    ScreenPaused
    ScreenHelp
    ScreenQuitConfirm
    ScreenWin
)

// Game lifecycle
type NewGameMsg struct {
    Seed      int64
    DrawCount int
}
type RestartDealMsg struct{}
type GameWonMsg struct{}

// Navigation
type ChangeScreenMsg struct{ Screen AppScreen }

// Ticks
type TickMsg time.Time          // elapsed timer updates
type CelebrationTickMsg struct{} // win animation frames

// Config
type ConfigChangedMsg struct{ Config *config.Config }
type ThemeChangedMsg struct{ Theme *theme.Theme }

// Auto-complete
type AutoCompleteStepMsg struct{} // triggers one foundation move per tick
```

---

## Step 2 — `tui/input.go`

### 2a — `GameAction` enum

All 19 values from Section 8.1 in the exact order listed:

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

type GameAction int

const (
    ActionNone GameAction = iota

    // Cursor movement
    ActionCursorUp
    ActionCursorDown
    ActionCursorLeft
    ActionCursorRight

    // Selection
    ActionSelect    // Enter or click — pick up or place card(s)
    ActionCancel    // Esc — deselect current selection

    // Shortcuts
    ActionFlipStock        // Spacebar
    ActionJumpToColumn     // 1-7 number keys; payload = int column index (0-based)
    ActionMoveToFoundation // 'f' — auto-move selected to foundation

    // Meta
    ActionUndo           // Ctrl+Z
    ActionRedo           // Ctrl+Y or Ctrl+Shift+Z
    ActionHint           // 'h' or '?'
    ActionNewGame        // Ctrl+N
    ActionRestartDeal    // Ctrl+R
    ActionPause          // 'p'
    ActionHelp           // F1
    ActionQuit           // 'q' or Ctrl+C
    ActionToggleAutoMove // Ctrl+A
    ActionCycleTheme     // 't'
)
```

### 2b — `TranslateInput`

Pure function; no side effects; no global state. Handle `tea.KeyMsg` then `tea.MouseMsg`; anything else returns `(ActionNone, nil)`.

**Key-to-action mapping** (derive directly from Section 8.2):

```go
func TranslateInput(msg tea.Msg) (GameAction, interface{}) {
    switch m := msg.(type) {
    case tea.KeyMsg:
        return translateKey(m)
    case tea.MouseMsg:
        return translateMouse(m)
    }
    return ActionNone, nil
}
```

#### `translateKey` mapping table

| Key expression | Action | Payload |
|---|---|---|
| `tea.KeyLeft`, rune `'h'` | `ActionCursorLeft` | nil |
| `tea.KeyRight`, rune `'l'` | `ActionCursorRight` | nil |
| `tea.KeyUp`, rune `'k'` | `ActionCursorUp` | nil |
| `tea.KeyDown`, rune `'j'` | `ActionCursorDown` | nil |
| `tea.KeyTab` | `ActionCursorRight` (next pile) | nil |
| `tea.KeyShiftTab` | `ActionCursorLeft` (prev pile) | nil |
| runes `'1'`–`'7'` | `ActionJumpToColumn` | `int(r - '1')` (0-based) |
| `tea.KeyEnter` | `ActionSelect` | nil |
| `tea.KeySpace` | `ActionFlipStock` | nil |
| `tea.KeyEsc` | `ActionCancel` | nil |
| rune `'f'` | `ActionMoveToFoundation` | nil |
| `tea.KeyCtrlZ` | `ActionUndo` | nil |
| `tea.KeyCtrlY` | `ActionRedo` | nil |
| rune `'h'`, rune `'?'` | `ActionHint` | nil |
| `tea.KeyF1` | `ActionHelp` | nil |
| rune `'p'` | `ActionPause` | nil |
| `tea.KeyCtrlN` | `ActionNewGame` | nil |
| `tea.KeyCtrlR` | `ActionRestartDeal` | nil |
| `tea.KeyCtrlA` | `ActionToggleAutoMove` | nil |
| rune `'t'` | `ActionCycleTheme` | nil |
| rune `'q'`, `tea.KeyCtrlC` | `ActionQuit` | nil |
| anything else | `ActionNone` | nil |

> **Conflict note**: `'h'` maps to both `ActionCursorLeft` and `ActionHint` in the spec table (Section 8.2 lists `h/l` for left/right and `?` for hint). Resolve by mapping `'h'` → `ActionCursorLeft` only (vim navigation takes priority); hint uses `'?'` exclusively. This matches the hint-dedicated `?` binding and keeps vim navigation unambiguous.

#### `translateMouse` implementation

For T9, mouse support is basic (extended hit-testing in T16). Map `tea.MouseLeft` button-down events to `ActionSelect` and pass the raw `tea.MouseMsg` as the payload for downstream coordinate resolution:

```go
func translateMouse(m tea.MouseMsg) (GameAction, interface{}) {
    if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
        return ActionSelect, m
    }
    return ActionNone, nil
}
```

Full pile hit-testing is added in T16 via `renderer.PileHitTest`; the T9 contract is just that a left click produces `ActionSelect` with coordinates preserved.

---

## Step 3 — `tui/input_test.go`

Table-driven test covering every row of Section 8.2 plus unmapped-key guard.

### Test structure

```go
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
        {"arrow left",  tea.KeyMsg{Type: tea.KeyLeft},  ActionCursorLeft,  nil},
        {"arrow right", tea.KeyMsg{Type: tea.KeyRight}, ActionCursorRight, nil},
        {"arrow up",    tea.KeyMsg{Type: tea.KeyUp},    ActionCursorUp,    nil},
        {"arrow down",  tea.KeyMsg{Type: tea.KeyDown},  ActionCursorDown,  nil},

        // Cursor — vim keys
        {"vim h", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}, ActionCursorLeft,  nil},
        {"vim l", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}, ActionCursorRight, nil},
        {"vim k", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, ActionCursorUp,    nil},
        {"vim j", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}, ActionCursorDown,  nil},

        // Tab cycling
        {"tab",       tea.KeyMsg{Type: tea.KeyTab},      ActionCursorRight, nil},
        {"shift+tab", tea.KeyMsg{Type: tea.KeyShiftTab}, ActionCursorLeft,  nil},

        // Number keys 1-7 → ActionJumpToColumn with 0-based index
        {"key 1", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}, ActionJumpToColumn, 0},
        {"key 2", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}, ActionJumpToColumn, 1},
        {"key 3", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}, ActionJumpToColumn, 2},
        {"key 4", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}, ActionJumpToColumn, 3},
        {"key 5", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")}, ActionJumpToColumn, 4},
        {"key 6", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")}, ActionJumpToColumn, 5},
        {"key 7", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("7")}, ActionJumpToColumn, 6},

        // Selection / stock
        {"enter",  tea.KeyMsg{Type: tea.KeyEnter}, ActionSelect,    nil},
        {"space",  tea.KeyMsg{Type: tea.KeySpace}, ActionFlipStock, nil},
        {"escape", tea.KeyMsg{Type: tea.KeyEsc},   ActionCancel,    nil},

        // Shortcuts
        {"f foundation", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}, ActionMoveToFoundation, nil},

        // Undo / redo
        {"ctrl+z", tea.KeyMsg{Type: tea.KeyCtrlZ}, ActionUndo, nil},
        {"ctrl+y", tea.KeyMsg{Type: tea.KeyCtrlY}, ActionRedo, nil},

        // Hint
        {"question mark", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}, ActionHint, nil},

        // Meta
        {"F1",     tea.KeyMsg{Type: tea.KeyF1},    ActionHelp,  nil},
        {"p pause",tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}, ActionPause, nil},
        {"ctrl+n", tea.KeyMsg{Type: tea.KeyCtrlN}, ActionNewGame,     nil},
        {"ctrl+r", tea.KeyMsg{Type: tea.KeyCtrlR}, ActionRestartDeal, nil},
        {"ctrl+a", tea.KeyMsg{Type: tea.KeyCtrlA}, ActionToggleAutoMove, nil},
        {"t theme",tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}, ActionCycleTheme, nil},
        {"q quit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, ActionQuit, nil},
        {"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}, ActionQuit, nil},

        // Mouse — left click → ActionSelect with MouseMsg payload
        {"mouse left click", tea.MouseMsg{
            Action: tea.MouseActionPress,
            Button: tea.MouseButtonLeft,
            X: 15, Y: 10,
        }, ActionSelect, nil}, // payload is tea.MouseMsg; checked separately below

        // Unmapped keys → ActionNone
        {"unmapped rune a",  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, ActionNone, nil},
        {"unmapped rune x",  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}, ActionNone, nil},
        {"unmapped F2",      tea.KeyMsg{Type: tea.KeyF2},                        ActionNone, nil},
        {"non-key msg",      tea.WindowSizeMsg{Width: 80, Height: 24},           ActionNone, nil},
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

// TestTranslateInput_JumpColumnPayload verifies payload is the correct 0-based column index.
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
```

---

## Imports Required

**`tui/input.go`**:
```go
import tea "github.com/charmbracelet/bubbletea"
```

**`tui/messages.go`**:
```go
import (
    "time"

    "solituire/config"
    "solituire/theme"
)
```

**`tui/input_test.go`**:
```go
import (
    "testing"

    tea "github.com/charmbracelet/bubbletea"
)
```

---

## Verification Gate

```bash
go build ./tui/...                # must compile cleanly
go test ./tui/...                 # all T9 tests must pass
go vet ./tui/...                  # no vet errors
```

No engine or renderer code is touched, so no cross-package regressions.

---

## Handoff Contract

When T9 is complete, T11 (Board Model) can proceed. T11 needs:

- `GameAction` enum (all 19 values) from `tui/input.go`
- `TranslateInput(tea.Msg) (GameAction, interface{})` from `tui/input.go`
- `AppScreen` enum and all `Msg` types from `tui/messages.go`

**T13 (App Shell) authors**: `AppScreen` is defined in `tui/messages.go`. Do **not** redefine it in `tui/app.go` — just use it directly.
