# Agent C — T15: Help Overlay, Pause Screen, Quit Dialog Implementation Plan

**Stage 8 | Phase 6 | Estimate: Small | Dependencies: T13 ✓**

---

## 1. Overview

T15 implements three UI overlays that sit on top of the playing board:

- **Help overlay** (`tui/help.go`) — full keybinding reference, dismiss on Esc/F1
- **Pause screen** (`tui/pause.go`) — hides cards, freezes the timer, any key resumes
- **Quit dialog** (`tui/dialog.go`) — centered Yes/No confirmation before exiting

**Goal**: Replace the three placeholder strings currently returned by `AppModel.View()` with proper sub-model views. Fix the timer so it does not advance while the game is paused.

**Scope**:
- **New**: `tui/help.go` — `HelpModel` struct and View
- **New**: `tui/pause.go` — `PauseModel` struct and View
- **New**: `tui/dialog.go` — `DialogModel` struct, Update, and View
- **New**: `tui/help_test.go`, `tui/pause_test.go`, `tui/dialog_test.go`
- **Modify**: `tui/app.go` — add sub-model fields, fix timer freeze, update View, extend `prevScreen` tracking to ScreenHelp, delegate QuitConfirm Update to DialogModel
- **Modify**: `tui/app_test.go` — remove ScreenPaused from forward-tick test; add frozen-timer test

No changes to `engine/`, `renderer/`, `config/`, or `theme/`.

---

## 2. Target Visual Layouts

### 2.1 Help Overlay

Full-width box showing every binding from §8.2:

```
╔══════════════════════════════════════════════╗
║                  KEY BINDINGS                ║
║                                              ║
║  ← / → / h / l     Move cursor left/right   ║
║  ↑ / ↓ / k / j     Move cursor up/down      ║
║  Tab / Shift+Tab    Cycle pile               ║
║  1-7                Jump to tableau column   ║
║  Enter              Select / Place card      ║
║  Space              Flip stock               ║
║  Esc                Cancel selection         ║
║  f                  Move to foundation       ║
║  Ctrl+Z             Undo                     ║
║  Ctrl+Y             Redo                     ║
║  ?                  Hint                     ║
║  F1                 Help (this screen)       ║
║  p                  Pause                    ║
║  Ctrl+N             New game                 ║
║  Ctrl+R             Restart deal             ║
║  Ctrl+A             Toggle auto-foundation   ║
║  t                  Cycle theme              ║
║  q                  Quit                     ║
║  Mouse click        Select / Place           ║
║                                              ║
║             Esc or F1 to close              ║
╚══════════════════════════════════════════════╝
```

### 2.2 Pause Screen

Centered message; no game state visible:

```
╔══════════════════════════════════╗
║                                  ║
║          GAME  PAUSED            ║
║                                  ║
║    Press any key to resume.      ║
║                                  ║
╚══════════════════════════════════╝
```

### 2.3 Quit Confirmation Dialog

Centered dialog overlaid on the board (§12.7):

```
╔═════════════════════════╗
║   Quit current game?    ║
║                         ║
║    [Yes]      [No]      ║
╚═════════════════════════╝
```

The active choice is highlighted (bold + reverse video). Arrow keys and `y`/`n` change selection; Enter or `y`/`n` confirms.

---

## 3. Data Models

### 3.1 HelpModel

No mutable state — purely a view factory.

```go
// tui/help.go
package tui

import tea "github.com/charmbracelet/bubbletea"

// HelpModel renders the keybinding reference overlay.
// It has no internal state; all routing is handled by AppModel.
type HelpModel struct{}

func NewHelpModel() HelpModel { return HelpModel{} }

func (m HelpModel) Init() tea.Cmd                        { return nil }
func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m HelpModel) View() string                         { /* see §5.1 */ }
```

AppModel owns the dismiss logic (Esc/F1 → `ChangeScreenMsg{prevScreen}`) so `HelpModel.Update` is a no-op. This mirrors how `PauseModel` works.

### 3.2 PauseModel

Also stateless — the board is simply not rendered.

```go
// tui/pause.go
package tui

import tea "github.com/charmbracelet/bubbletea"

// PauseModel renders the "game paused" overlay.
// AppModel handles keypress routing (any key → ChangeScreenMsg{ScreenPlaying}).
type PauseModel struct{}

func NewPauseModel() PauseModel { return PauseModel{} }

func (m PauseModel) Init() tea.Cmd                        { return nil }
func (m PauseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m PauseModel) View() string                         { /* see §5.2 */ }
```

### 3.3 DialogModel

Tracks the cursor (Yes vs No) and handles key input.

```go
// tui/dialog.go
package tui

import tea "github.com/charmbracelet/bubbletea"

type dialogChoice int

const (
    dialogYes dialogChoice = iota
    dialogNo
)

// DialogModel is the Bubbletea sub-model for the quit-confirmation dialog.
// It tracks which button is highlighted and emits tea.Quit or a
// ChangeScreenMsg when the player confirms or cancels.
type DialogModel struct {
    choice   dialogChoice
    prevScreen AppScreen // screen to return to on cancel
}

func NewDialogModel(prev AppScreen) DialogModel {
    return DialogModel{choice: dialogNo, prevScreen: prev}
}

func (m DialogModel) Init() tea.Cmd { return nil }
func (m DialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m DialogModel) View() string
```

**Why default to No?** Prevents accidental quit on Enter-mash after opening the dialog.

---

## 4. Bubbletea Interface Details

### 4.1 HelpModel — no Update logic

AppModel handles `ScreenHelp` key events directly:

```
Esc  → ChangeScreenMsg{prevScreen}
F1   → ChangeScreenMsg{prevScreen}
(all other keys absorbed — no action)
```

This is a deliberate departure from "any key closes" so the user can read the overlay without accidentally dismissing it.

### 4.2 PauseModel — no Update logic

AppModel handles `ScreenPaused` key events directly:

```
any key → ChangeScreenMsg{ScreenPlaying}
```

No change from current stub — but the View improves from a raw string to `PauseModel.View()`.

### 4.3 DialogModel.Update — key routing

| Key                   | Effect                                           |
|-----------------------|--------------------------------------------------|
| `←` / `h`            | Move cursor to Yes                               |
| `→` / `l`            | Move cursor to No                                |
| `y` / `Y`            | Select Yes → emit `tea.Quit`                     |
| `n` / `N`            | Select No → emit `ChangeScreenMsg{prevScreen}`   |
| Enter                 | Confirm current selection (Yes → quit, No → cancel) |
| Esc                   | Cancel → emit `ChangeScreenMsg{prevScreen}`      |

```go
func (m DialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    key, ok := msg.(tea.KeyMsg)
    if !ok {
        return m, nil
    }
    switch key.Type {
    case tea.KeyLeft:
        m.choice = dialogYes
    case tea.KeyRight:
        m.choice = dialogNo
    case tea.KeyEnter:
        return m.confirm()
    case tea.KeyEsc:
        return m, m.cancelCmd()
    case tea.KeyRunes:
        switch string(key.Runes) {
        case "y", "Y":
            m.choice = dialogYes
            return m.confirm()
        case "n", "N":
            m.choice = dialogNo
            return m, m.cancelCmd()
        case "h", "H":
            m.choice = dialogYes
        case "l", "L":
            m.choice = dialogNo
        }
    }
    return m, nil
}

func (m DialogModel) confirm() (DialogModel, tea.Cmd) {
    if m.choice == dialogYes {
        return m, tea.Quit
    }
    return m, m.cancelCmd()
}

func (m DialogModel) cancelCmd() tea.Cmd {
    prev := m.prevScreen
    return func() tea.Msg { return ChangeScreenMsg{Screen: prev} }
}
```

---

## 5. View Implementations

### 5.1 HelpModel.View

Build a two-column table of bindings using `lipgloss`. Each row is `"  key    description"`. Wrap in a double-border box. The dismiss footer ("Esc or F1 to close") is separated by a blank line at the bottom.

```go
var (
    helpBoxStyle = lipgloss.NewStyle().
        Border(lipgloss.DoubleBorder()).
        Padding(1, 2)
    helpTitleStyle = lipgloss.NewStyle().Bold(true)
    helpKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)

// binding is a key→description pair rendered as one row.
type binding struct{ key, desc string }

var helpBindings = []binding{
    {"← / → / h / l", "Move cursor left/right"},
    {"↑ / ↓ / k / j", "Move cursor up/down"},
    {"Tab / Shift+Tab", "Cycle pile"},
    {"1-7", "Jump to tableau column"},
    {"Enter", "Select / Place card"},
    {"Space", "Flip stock"},
    {"Esc", "Cancel selection"},
    {"f", "Move to foundation"},
    {"Ctrl+Z", "Undo"},
    {"Ctrl+Y", "Redo"},
    {"?", "Hint"},
    {"F1", "Help (this screen)"},
    {"p", "Pause"},
    {"Ctrl+N", "New game"},
    {"Ctrl+R", "Restart deal"},
    {"Ctrl+A", "Toggle auto-foundation"},
    {"t", "Cycle theme"},
    {"q", "Quit"},
    {"Mouse click", "Select / Place"},
}
```

Use a fixed key-column width (e.g. 18 chars, left-aligned) so descriptions line up cleanly.

### 5.2 PauseModel.View

```go
var pauseBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.DoubleBorder()).
    Padding(1, 4)

func (m PauseModel) View() string {
    content := lipgloss.JoinVertical(lipgloss.Center,
        lipgloss.NewStyle().Bold(true).Render("GAME  PAUSED"),
        "",
        "Press any key to resume.",
    )
    return pauseBoxStyle.Render(content)
}
```

### 5.3 DialogModel.View

```go
var (
    dialogBoxStyle = lipgloss.NewStyle().
        Border(lipgloss.DoubleBorder()).
        Padding(0, 2)
    dialogActiveStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
)

func (m DialogModel) View() string {
    yes := "[ Yes ]"
    no  := "[ No  ]"
    if m.choice == dialogYes {
        yes = dialogActiveStyle.Render(yes)
    } else {
        no = dialogActiveStyle.Render(no)
    }
    buttons := yes + "    " + no

    content := lipgloss.JoinVertical(lipgloss.Center,
        "Quit current game?",
        "",
        buttons,
    )
    return dialogBoxStyle.Render(content)
}
```

---

## 6. AppModel Modifications

Four targeted changes to `tui/app.go`.

### 6a — Add sub-model fields to AppModel struct

```go
type AppModel struct {
    screen     AppScreen
    prevScreen AppScreen
    engine     engine.GameEngine
    cfg        *config.Config
    themes     *theme.ThemeRegistry
    rend       *renderer.Renderer
    board      BoardModel
    menu       MenuModel
    help       HelpModel   // ← add
    pause      PauseModel  // ← add
    dialog     DialogModel // ← add
    windowW    int
    windowH    int
    tooSmall   bool
}
```

### 6b — Initialize sub-models in NewAppModel

```go
return AppModel{
    // ... existing fields ...
    help:   NewHelpModel(),
    pause:  NewPauseModel(),
    dialog: NewDialogModel(ScreenMenu), // prevScreen set properly on transition
    // ...
}
```

### 6c — Fix TickMsg handler to freeze timer when paused

Current code always forwards `TickMsg` to the board. Replace with:

```go
case TickMsg:
    if m.screen == ScreenPaused {
        // Keep the tick chain alive but do NOT advance ElapsedTime.
        return m, tickCmd()
    }
    updated, cmd := m.board.Update(msg)
    m.board = updated.(BoardModel)
    return m, cmd
```

`tickCmd()` is package-private in `board.go` but accessible here since both files are in `package tui`.

### 6d — Extend prevScreen tracking to ScreenHelp

In the `ChangeScreenMsg` handler:

```go
case ChangeScreenMsg:
    if msg.Screen == ScreenQuitConfirm || msg.Screen == ScreenHelp {
        m.prevScreen = m.screen
    }
    m.screen = msg.Screen
    return m, nil
```

### 6e — Update ScreenHelp routing in Update

Replace the stub:

```go
case ScreenHelp:
    if key, ok := msg.(tea.KeyMsg); ok {
        switch key.Type {
        case tea.KeyEsc, tea.KeyF1:
            prev := m.prevScreen
            if prev == ScreenHelp {
                prev = ScreenPlaying
            }
            return m, func() tea.Msg { return ChangeScreenMsg{Screen: prev} }
        }
        // All other keys are absorbed — user is reading the overlay.
    }
```

### 6f — Delegate ScreenQuitConfirm Update to DialogModel

Replace the stub:

```go
case ScreenQuitConfirm:
    // Sync prevScreen into dialog each time (prevScreen may have changed
    // since the dialog was last opened).
    m.dialog.prevScreen = m.prevScreen
    updated, cmd := m.dialog.Update(msg)
    m.dialog = updated.(DialogModel)
    return m, cmd
```

This removes the inline y/n handling from app.go — `DialogModel.Update` owns it now.

### 6g — Replace stub View cases

```go
case ScreenPaused:
    return m.pause.View()
case ScreenHelp:
    return m.help.View()
case ScreenQuitConfirm:
    return m.dialog.View()
```

---

## 7. Test Plan

### 7.1 `tui/help_test.go`

#### Pattern A — Golden render

```go
func init() {
    lipgloss.SetColorProfile(termenv.Ascii)
}

func TestHelpRender(t *testing.T) {
    m := NewHelpModel()
    golden.RequireEqual(t, []byte(m.View()))
}
```

#### Pattern C — Dismiss behavior via AppModel

```go
// Esc closes the overlay and returns to prevScreen.
func TestAppModel_Help_EscReturnsToPrevScreen(t *testing.T) {
    app := newTestApp()
    app.screen = ScreenPlaying
    app = updateApp(app, ChangeScreenMsg{Screen: ScreenHelp})
    // prevScreen should now be ScreenPlaying
    _, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
    msg := cmd()
    csm, ok := msg.(ChangeScreenMsg)
    if !ok || csm.Screen != ScreenPlaying {
        t.Errorf("Esc: got %v, want ChangeScreenMsg{ScreenPlaying}", msg)
    }
}

// F1 closes the overlay too.
func TestAppModel_Help_F1Closes(t *testing.T) { /* same pattern, key.Type = tea.KeyF1 */ }

// Non-Esc/F1 key is absorbed (nil cmd).
func TestAppModel_Help_OtherKeyAbsorbed(t *testing.T) {
    app := newTestApp()
    app.screen = ScreenHelp
    _, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
    if cmd != nil {
        t.Error("non-dismiss key should be absorbed (nil cmd) in ScreenHelp")
    }
}
```

### 7.2 `tui/pause_test.go`

#### Pattern A — Golden render

```go
func TestPauseRender(t *testing.T) {
    m := NewPauseModel()
    golden.RequireEqual(t, []byte(m.View()))
}
```

#### Pattern C — Timer behavior via AppModel

```go
// Timer must NOT advance when paused.
func TestAppModel_TickMsg_FrozenWhenPaused(t *testing.T) {
    app := newTestApp()
    app.screen = ScreenPaused
    before := app.board.eng.State().ElapsedTime
    app = updateApp(app, TickMsg(time.Now()))
    after := app.board.eng.State().ElapsedTime
    if after != before {
        t.Errorf("ElapsedTime advanced during pause: %v → %v", before, after)
    }
}

// Tick chain must stay alive during pause (cmd returned is non-nil).
func TestAppModel_TickMsg_ChainKeptAliveWhenPaused(t *testing.T) {
    app := newTestApp()
    app.screen = ScreenPaused
    _, cmd := app.Update(TickMsg(time.Now()))
    if cmd == nil {
        t.Error("TickMsg on ScreenPaused returned nil Cmd — tick chain would die")
    }
}

// Any key while paused resumes (returns ChangeScreenMsg{ScreenPlaying}).
func TestAppModel_Paused_AnyKeyResumes(t *testing.T) { /* already in app_test.go — keep as-is */ }
```

#### App_test.go modification

Remove `ScreenPaused` from `TestAppModel_TickMsg_ForwardedOnNonPlayingScreens`:

```go
// ScreenPaused intentionally excluded — timer is frozen during pause (see TestAppModel_TickMsg_FrozenWhenPaused).
screens := []AppScreen{ScreenHelp, ScreenQuitConfirm, ScreenWin, ScreenMenu}
```

### 7.3 `tui/dialog_test.go`

#### Pattern A — Golden render (default cursor on No)

```go
func TestDialogRender_DefaultNo(t *testing.T) {
    m := NewDialogModel(ScreenPlaying)
    golden.RequireEqual(t, []byte(m.View()))
}

func TestDialogRender_CursorOnYes(t *testing.T) {
    m := NewDialogModel(ScreenPlaying)
    m.choice = dialogYes
    golden.RequireEqual(t, []byte(m.View()))
}
```

#### Pattern C — Model state tests

```go
// Helper
func updateDialog(m DialogModel, msg tea.Msg) (DialogModel, tea.Cmd) {
    result, cmd := m.Update(msg)
    return result.(DialogModel), cmd
}

func runCmd(cmd tea.Cmd) tea.Msg {
    if cmd == nil { return nil }
    return cmd()
}

// Arrow left selects Yes.
func TestDialogModel_LeftSelectsYes(t *testing.T) {
    m, _ := updateDialog(NewDialogModel(ScreenPlaying), tea.KeyMsg{Type: tea.KeyLeft})
    if m.choice != dialogYes {
        t.Errorf("choice = %v, want dialogYes", m.choice)
    }
}

// Arrow right selects No.
func TestDialogModel_RightSelectsNo(t *testing.T) {
    m := NewDialogModel(ScreenPlaying)
    m.choice = dialogYes
    m, _ = updateDialog(m, tea.KeyMsg{Type: tea.KeyRight})
    if m.choice != dialogNo {
        t.Errorf("choice = %v, want dialogNo", m.choice)
    }
}

// 'y' immediately confirms quit.
func TestDialogModel_YQuits(t *testing.T) {
    _, cmd := updateDialog(NewDialogModel(ScreenPlaying),
        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
    // tea.Quit returns a special msg; just verify cmd is non-nil
    if cmd == nil {
        t.Error("'y' returned nil Cmd, expected tea.Quit")
    }
}

// 'n' emits ChangeScreenMsg to prevScreen.
func TestDialogModel_NCancels(t *testing.T) {
    _, cmd := updateDialog(NewDialogModel(ScreenPlaying),
        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
    msg := runCmd(cmd)
    csm, ok := msg.(ChangeScreenMsg)
    if !ok || csm.Screen != ScreenPlaying {
        t.Errorf("'n': got %v, want ChangeScreenMsg{ScreenPlaying}", msg)
    }
}

// Enter on Yes quits.
func TestDialogModel_EnterOnYesQuits(t *testing.T) {
    m := NewDialogModel(ScreenPlaying)
    m.choice = dialogYes
    _, cmd := updateDialog(m, tea.KeyMsg{Type: tea.KeyEnter})
    if cmd == nil {
        t.Error("Enter on Yes returned nil Cmd")
    }
}

// Enter on No cancels.
func TestDialogModel_EnterOnNoCancels(t *testing.T) {
    m := NewDialogModel(ScreenPlaying)
    m.choice = dialogNo
    _, cmd := updateDialog(m, tea.KeyMsg{Type: tea.KeyEnter})
    msg := runCmd(cmd)
    csm, ok := msg.(ChangeScreenMsg)
    if !ok || csm.Screen != ScreenPlaying {
        t.Errorf("Enter on No: got %v, want ChangeScreenMsg{ScreenPlaying}", msg)
    }
}

// Esc always cancels regardless of cursor.
func TestDialogModel_EscCancels(t *testing.T) {
    m := NewDialogModel(ScreenMenu)
    m.choice = dialogYes // cursor is on Yes, but Esc should still cancel
    _, cmd := updateDialog(m, tea.KeyMsg{Type: tea.KeyEsc})
    msg := runCmd(cmd)
    csm, ok := msg.(ChangeScreenMsg)
    if !ok || csm.Screen != ScreenMenu {
        t.Errorf("Esc: got %v, want ChangeScreenMsg{ScreenMenu}", msg)
    }
}

// prevScreen is passed through on cancel.
func TestDialogModel_CancelReturnsToCorrectPrevScreen(t *testing.T) {
    _, cmd := updateDialog(NewDialogModel(ScreenPaused),
        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
    msg := runCmd(cmd)
    csm, ok := msg.(ChangeScreenMsg)
    if !ok || csm.Screen != ScreenPaused {
        t.Errorf("cancel from paused: got %v, want ChangeScreenMsg{ScreenPaused}", msg)
    }
}
```

---

## 8. Edge Cases and Constraints

| Situation | Handling |
|-----------|----------|
| Help opened from non-Playing screen | `prevScreen` is set correctly by ChangeScreenMsg handler (§6d); Esc returns there |
| `prevScreen == ScreenHelp` (self-referential) | Guard in ScreenHelp Update: fall back to `ScreenPlaying` |
| `prevScreen == ScreenQuitConfirm` | Guard in cancelCmd: `DialogModel.prevScreen` is synced from `app.prevScreen` each dispatch (§6f) |
| Non-key messages received by help/pause/dialog | All return `(m, nil)` — models ignore non-key input |
| `DialogModel.prevScreen` zero-value | Default is `ScreenMenu` (iota 0) — harmless; `NewDialogModel` always receives a real prev |
| Dialog opened from ScreenMenu | `prevScreen = ScreenMenu`; cancel returns there (existing test `TestAppModel_QuitConfirm_CancelFromMenuReturnsToMenu` must still pass) |
| `tickCmd()` accessibility | It's package-private in `board.go` but `app.go` is in the same `package tui` — no export needed |

---

## 9. File Checklist

```
tui/
├── help.go           ← NEW  (HelpModel + View with binding table)
├── help_test.go      ← NEW  (golden render + 3 dismiss behavior tests)
├── pause.go          ← NEW  (PauseModel + View)
├── pause_test.go     ← NEW  (golden render + timer freeze + chain alive tests)
├── dialog.go         ← NEW  (DialogModel + Update + View)
├── dialog_test.go    ← NEW  (2 golden renders + 8 state tests)
├── app.go            ← MODIFY (6 targeted edits: struct, init, TickMsg, ChangeScreenMsg, Update routing, View)
├── app_test.go       ← MODIFY (remove ScreenPaused from forward-tick test; add frozen-timer tests)
└── testdata/
    ├── TestHelpRender.golden          ← GENERATED (first -update run)
    ├── TestPauseRender.golden         ← GENERATED
    ├── TestDialogRender_DefaultNo.golden  ← GENERATED
    └── TestDialogRender_CursorOnYes.golden ← GENERATED
```

---

## 10. Execution Order

1. **Create `tui/pause.go`** — `PauseModel` struct, `Init`, `Update`, `View`.
2. **Create `tui/help.go`** — `HelpModel` struct, `helpBindings` slice, `Init`, `Update`, `View`.
3. **Create `tui/dialog.go`** — `DialogModel` struct, `Init`, `Update` (confirm/cancel helpers), `View`.
4. **Modify `tui/app.go`** — apply edits 6a–6g in order:
   - Add fields to `AppModel` struct.
   - Initialize in `NewAppModel`.
   - Fix `TickMsg` case.
   - Extend `ChangeScreenMsg` to record `prevScreen` for `ScreenHelp`.
   - Replace `ScreenHelp` Update stub with Esc/F1-only dismiss.
   - Replace `ScreenQuitConfirm` Update stub with delegation to `DialogModel`.
   - Replace three stub View cases with sub-model views.
5. **Build** — `go build ./...` — fix any compile errors before writing tests.
6. **Modify `tui/app_test.go`** — remove `ScreenPaused` from `TestAppModel_TickMsg_ForwardedOnNonPlayingScreens`; verify `TestAppModel_QuitConfirm_*` tests still pass.
7. **Create `tui/pause_test.go`** — timer freeze + chain tests.
8. **Create `tui/help_test.go`** — dismiss tests.
9. **Create `tui/dialog_test.go`** — state tests.
10. **Run `go test ./tui/ -run "TestHelpRender|TestPauseRender|TestDialogRender" -update`** — generate four golden files.
11. **Run `go test ./tui/`** — all tests must pass, including pre-existing `TestAppModel_*` tests.
12. Commit and push on `claude/agent-c-t15-plan-9meJt`.

---

## 11. Pre-existing Test Compatibility

The following existing `app_test.go` tests are affected by T15 changes and must remain green:

| Test | Change needed |
|------|--------------|
| `TestAppModel_TickMsg_ForwardedOnNonPlayingScreens` | Remove `ScreenPaused` from `screens` slice |
| `TestAppModel_Paused_AnyKeyResumes` | No change — AppModel still handles any-key routing |
| `TestAppModel_Help_AnyKeyCloses` | **Rename + narrow**: only Esc/F1 now dismiss; rename to `TestAppModel_Help_EscCloses` and move to `help_test.go`; delete from `app_test.go` |
| `TestAppModel_QuitConfirm_YQuits` | No change — DialogModel emits `tea.Quit` the same as the stub |
| `TestAppModel_QuitConfirm_NoCancelsToPlaying` | No change — DialogModel emits `ChangeScreenMsg{prevScreen}` |
| `TestAppModel_QuitConfirm_CancelFromMenuReturnsToMenu` | No change — `prevScreen` syncing in §6f preserves this behavior |
| `TestAppModel_View_AllScreens` | No change — all screens now return real views, not empty strings |

---

## 12. T16 Handoff Note

After T15 merges:
- `ScreenHelp`, `ScreenPaused`, `ScreenQuitConfirm` are fully implemented.
- T16 (Mouse Input) touches `tui/input.go` and `tui/cursor.go` only — no conflict.
- T17 (Auto-Complete) touches `tui/board.go` only — no conflict.
- The only remaining placeholder is `ScreenWin` → `tui/celebration.go` (T18).
