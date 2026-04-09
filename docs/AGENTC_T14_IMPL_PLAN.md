# Agent C — T14: Settings Menu Implementation Plan

**Stage 8 | Phase 5 | Estimate: Small | Dependencies: T13 ✓**

---

## 1. Overview

T14 implements the `MenuModel` Bubbletea sub-model that renders the settings/start screen shown at application launch. The design spec is in DESIGN.md §11.1.

**Goal**: When the app starts the user sees a menu, can configure draw mode, theme, and auto-move, and can launch a new game — all before the board is shown.

**Scope**:
- **New**: `tui/menu.go` — `MenuModel` struct and Bubbletea implementation
- **New**: `tui/menu_test.go` — golden render test + model-state tests
- **Modify**: `tui/app.go` — wire `MenuModel` into `AppModel`; change initial screen to `ScreenMenu`

No changes to `engine/`, `renderer/`, `config/`, or `theme/`.

---

## 2. Target Visual Layout

From DESIGN.md §11.1:

```
╔══════════════════════════════╗
║     KLONDIKE SOLITAIRE       ║
║                              ║
║  Draw Mode:   [ 1 ] [ 3 ]   ║
║  Theme:       ◀ Classic ▶    ║
║  Auto-Move:   [ON] [OFF]    ║
║                              ║
║      [ Start New Game ]      ║
║                              ║
║         Seed: 12345          ║
╚══════════════════════════════╝
```

Selected/active option is visually highlighted (bold or inverted). The cursor position determines which row is active.

---

## 3. Data Model

### 3.1 MenuModel Struct

```go
// tui/menu.go

package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "solituire/config"
    "solituire/theme"
)

// menuItem identifies which row of the menu has keyboard focus.
type menuItem int

const (
    menuItemDrawMode menuItem = iota
    menuItemTheme
    menuItemAutoMove
    menuItemStart
    menuItemCount // sentinel — total number of navigable rows
)

// MenuModel is the Bubbletea sub-model for the settings/start screen.
// It holds a local copy of the config that is mutated as the user adjusts
// settings; a ConfigChangedMsg is emitted on every change so AppModel can
// sync its own cfg pointer.
type MenuModel struct {
    cfg    config.Config        // local working copy — mutated on toggle/cycle
    themes *theme.ThemeRegistry // for Next() cycling
    cursor menuItem             // which row has keyboard focus
}

// NewMenuModel creates a MenuModel seeded from the current app config.
func NewMenuModel(cfg *config.Config, themes *theme.ThemeRegistry) MenuModel {
    return MenuModel{
        cfg:    *cfg,   // copy — MenuModel owns its own copy
        themes: themes,
        cursor: menuItemDrawMode,
    }
}
```

**Why copy config?** The menu mutates settings interactively; using a local copy lets the caller (AppModel) decide when to apply the change via `ConfigChangedMsg` without worrying about aliased mutation.

### 3.2 Settings Fields

| Field           | Config field        | Options                    | Default   |
|-----------------|---------------------|----------------------------|-----------|
| Draw Mode       | `DrawCount int`     | 1 or 3 (toggle)            | 1         |
| Theme           | `ThemeName string`  | all registered (cycle)     | "classic" |
| Auto-Move       | `AutoMoveEnabled`   | true / false (toggle)      | false     |
| Seed (display)  | `Seed int64`        | read-only; 0 shown as 0    | 0         |

---

## 4. Bubbletea Interface

### 4.1 Init

```go
func (m MenuModel) Init() tea.Cmd { return nil }
```

The menu requires no background commands on startup.

### 4.2 Update — Keyboard Routing

| Key                   | Effect                                                          |
|-----------------------|-----------------------------------------------------------------|
| `↑` / `k` / `Shift+Tab` | Move cursor up (wraps)                                       |
| `↓` / `j` / `Tab`    | Move cursor down (wraps)                                        |
| `←` / `h`            | Decrement / toggle current item (same as Enter, reverse direction for Draw Mode; wraps theme backward) |
| `→` / `l` / `Enter` / `Space` | Increment / toggle / activate current item             |
| Enter on `menuItemStart` | Emit `NewGameMsg`                                          |
| Right/Enter on `menuItemDrawMode` | Toggle DrawCount 1↔3                              |
| Right/Enter on `menuItemTheme` | Cycle to next theme (`themes.Next`)                 |
| Left on `menuItemTheme` | Cycle theme backward (find previous in registry list)        |
| Right/Enter on `menuItemAutoMove` | Toggle AutoMoveEnabled                           |

> **Note on Left on Theme**: `ThemeRegistry` only exposes `Next()`; implement a `prev()` helper inside `menu.go` that walks `themes.List()` to find the predecessor.

### 4.3 Emitted Messages

| Trigger                    | Message                         |
|----------------------------|---------------------------------|
| Any setting changes        | `ConfigChangedMsg{Config: &cfg}` (pointer to heap copy) |
| "Start New Game" activated | `NewGameMsg{Seed: cfg.Seed, DrawCount: cfg.DrawCount}` |

Both may be returned as `tea.Cmd` from `Update()`.

When a setting changes **and** "Start New Game" is activated in one keypress — that can't happen because Start is its own row. But if it did, `NewGameMsg` takes priority.

### 4.4 View

Render using `lipgloss` borders and styles. The active row is highlighted (bold or reverse video on the selected option within that row). Use a fixed-width box centred with the rest of the terminal via `lipgloss.Place` or left-padded string construction. Keep it simple: a vertical stack of lines inside a border.

The active `menuItem` determines which bracket/arrow pair is highlighted:
- Draw Mode: highlight `[ 1 ]` if DrawCount==1, else `[ 3 ]`; bold the active bracket
- Theme: surround name with `◀ name ▶`; bold when cursor is on this row
- Auto-Move: highlight `[ON]` or `[OFF]` depending on value; bold when active
- Start button: bold `[ Start New Game ]` when cursor is on this row

---

## 5. Detailed Implementation Steps

### Step 1 — Create `tui/menu.go`

1. Define `menuItem` type and constants.
2. Define `MenuModel` struct with fields: `cfg config.Config`, `themes *theme.ThemeRegistry`, `cursor menuItem`.
3. Implement `NewMenuModel(cfg *config.Config, themes *theme.ThemeRegistry) MenuModel`.
4. Implement `Init() tea.Cmd` → `nil`.
5. Implement `Update(msg tea.Msg) (tea.Model, tea.Cmd)`:
   - Type-switch on `tea.KeyMsg` only; return `(m, nil)` for all other message types.
   - Navigation keys move `m.cursor` (modulo `menuItemCount`).
   - Action keys call internal helpers `m.applyLeft()` / `m.applyRight()`.
   - `applyRight()`: DrawMode toggle (`1→3→1`), Theme `themes.Next()`, AutoMove negate, Start emit `NewGameMsg`.
   - `applyLeft()`: DrawMode toggle (same as right since there are only 2 options), Theme cycle backward, AutoMove negate.
   - After any setting mutation emit `ConfigChangedMsg` by returning the cmd from a helper.
6. Implement `View() string`:
   - Build each row string with active-state highlighting.
   - Wrap in a lipgloss border box.
   - Return the full string.

### Step 2 — Modify `tui/app.go`

Four targeted changes (all minimal — don't restructure the file):

**2a. Add `menu MenuModel` field to `AppModel` struct** (after `board BoardModel`, line ~24):
```go
type AppModel struct {
    screen   AppScreen
    engine   engine.GameEngine
    cfg      *config.Config
    themes   *theme.ThemeRegistry
    rend     *renderer.Renderer
    board    BoardModel
    menu     MenuModel   // ← add
    windowW  int
    windowH  int
    tooSmall bool
}
```

**2b. Initialize `menu` in `NewAppModel` and change initial screen** (lines ~38-47):
```go
return AppModel{
    screen:  ScreenMenu,   // ← was ScreenPlaying
    engine:  eng,
    cfg:     cfg,
    themes:  themes,
    rend:    rend,
    board:   NewBoardModel(eng, rend, cfg),
    menu:    NewMenuModel(cfg, themes),  // ← add
    windowW: renderer.MinTermWidth,
    windowH: renderer.MinTermHeight,
}
```

**2c. Route `ScreenMenu` to `MenuModel` in `Update()`** (replace the `case ScreenWin, ScreenMenu:` block, lines ~157-171):

The current combined `ScreenWin, ScreenMenu` fallback handles Ctrl+N and q/Ctrl+C. After T14, `ScreenMenu` gets its own branch that delegates to `MenuModel.Update()` (which will emit `NewGameMsg` when the user activates Start). The `ScreenWin` case stays as-is:

```go
case ScreenMenu:
    updated, cmd := m.menu.Update(msg)
    m.menu = updated.(MenuModel)
    // Sync cfg pointer when menu emits ConfigChangedMsg.
    // ConfigChangedMsg is also handled at the top of Update() globally,
    // so no extra handling needed here.
    return m, cmd

case ScreenWin:
    if key, ok := msg.(tea.KeyMsg); ok {
        switch {
        case key.Type == tea.KeyCtrlN:
            return m, func() tea.Msg {
                return NewGameMsg{Seed: appSeed(), DrawCount: m.cfg.DrawCount}
            }
        case key.Type == tea.KeyRunes && len(key.Runes) > 0 &&
            (key.Runes[0] == 'q' || key.Runes[0] == 'Q'):
            return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenQuitConfirm} }
        case key.Type == tea.KeyCtrlC:
            return m, tea.Quit
        }
    }
```

**2d. Replace placeholder in `View()` for `ScreenMenu`** (line ~204):
```go
case ScreenMenu:
    return m.menu.View()
```

### Step 3 — Create `tui/menu_test.go`

#### Pattern A — Golden render test

```go
func TestMenuRender(t *testing.T) {
    cfg := config.DefaultConfig()
    cfg.Seed = 12345   // deterministic seed display
    reg := theme.NewRegistry()
    m := NewMenuModel(cfg, reg)
    got := m.View()
    golden.RequireEqual(t, []byte(got))
}
```

Run once with `-update` flag to generate `testdata/TestMenuRender.golden`. The `init()` in `board_test.go` already sets `lipgloss.SetColorProfile(termenv.Ascii)` — because all `_test.go` files in `package tui` share the same `init()`, this applies to the menu golden test too.

#### Pattern C — Model state tests

```go
// --- Cursor navigation ---
func TestMenuModel_CursorDown_Wraps(t *testing.T)
func TestMenuModel_CursorUp_Wraps(t *testing.T)
func TestMenuModel_Tab_AdvancesCursor(t *testing.T)

// --- Draw mode ---
func TestMenuModel_DrawMode_RightTogglesTo3(t *testing.T)
func TestMenuModel_DrawMode_RightWrapsBackTo1(t *testing.T)
func TestMenuModel_DrawMode_EnterToggles(t *testing.T)

// --- Theme cycling ---
func TestMenuModel_Theme_RightCyclesToNextTheme(t *testing.T)
func TestMenuModel_Theme_LeftCyclesToPrevTheme(t *testing.T)

// --- Auto-move ---
func TestMenuModel_AutoMove_RightTogglesOn(t *testing.T)
func TestMenuModel_AutoMove_RightTogglesOffAgain(t *testing.T)

// --- Start New Game ---
func TestMenuModel_Start_EmitsNewGameMsg(t *testing.T)
func TestMenuModel_Start_NewGameMsgContainsCurrentConfig(t *testing.T)

// --- Config changed ---
func TestMenuModel_SettingChange_EmitsConfigChangedMsg(t *testing.T)
func TestMenuModel_ConfigChangedMsg_HasUpdatedConfig(t *testing.T)
```

**Test helper pattern**:

```go
func updateMenu(m MenuModel, msg tea.Msg) (MenuModel, tea.Cmd) {
    result, cmd := m.Update(msg)
    return result.(MenuModel), cmd
}

func runCmd(cmd tea.Cmd) tea.Msg {
    if cmd == nil {
        return nil
    }
    return cmd()
}
```

**Example state test**:

```go
func TestMenuModel_DrawMode_RightTogglesTo3(t *testing.T) {
    cfg := config.DefaultConfig() // DrawCount = 1
    m := NewMenuModel(cfg, theme.NewRegistry())
    // cursor starts on menuItemDrawMode

    m, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyRight})

    if m.cfg.DrawCount != 3 {
        t.Errorf("DrawCount = %d, want 3", m.cfg.DrawCount)
    }
    msg := runCmd(cmd)
    if _, ok := msg.(ConfigChangedMsg); !ok {
        t.Errorf("expected ConfigChangedMsg, got %T", msg)
    }
}

func TestMenuModel_Start_EmitsNewGameMsg(t *testing.T) {
    cfg := config.DefaultConfig()
    cfg.Seed = 99
    m := NewMenuModel(cfg, theme.NewRegistry())
    // Navigate to Start row
    for range 3 { // 3 downs: DrawMode→Theme→AutoMove→Start
        m, _ = updateMenu(m, tea.KeyMsg{Type: tea.KeyDown})
    }
    _, cmd := updateMenu(m, tea.KeyMsg{Type: tea.KeyEnter})
    msg := runCmd(cmd)
    ngm, ok := msg.(NewGameMsg)
    if !ok {
        t.Fatalf("expected NewGameMsg, got %T", msg)
    }
    if ngm.DrawCount != cfg.DrawCount {
        t.Errorf("DrawCount = %d, want %d", ngm.DrawCount, cfg.DrawCount)
    }
}
```

### Step 4 — Integration: verify AppModel routes correctly

The existing `TestAppModel_Menu_CtrlNStartsNewGame` in `app_test.go` exercises the old ScreenMenu fallback path. After T14:
- `ScreenMenu` routes to `MenuModel.Update()`.
- The menu's Update emits `NewGameMsg` when Enter is pressed on the Start row.
- Ctrl+N no longer bypasses the menu; it is absorbed by the menu as a key event.
  - **Decision**: Keep Ctrl+N working from ScreenMenu as a convenience. The simplest way is to handle `tea.KeyCtrlN` in `MenuModel.Update()` and emit `NewGameMsg` directly — so the existing `app_test.go` test passes without modification.

Add one new integration test:

```go
func TestAppModel_InitialScreen_IsMenu(t *testing.T) {
    app := newTestApp()
    if app.screen != ScreenMenu {
        t.Errorf("initial screen = %v, want ScreenMenu", app.screen)
    }
}
```

---

## 6. Lipgloss Style Guide for the Menu

Keep it minimal — lipgloss is already imported in `app.go`.

```go
var (
    menuBoxStyle = lipgloss.NewStyle().
        Border(lipgloss.DoubleBorder()).
        BorderForeground(lipgloss.Color("6")).  // cyan
        Padding(1, 2)

    menuTitleStyle = lipgloss.NewStyle().Bold(true).Align(lipgloss.Center)

    menuActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")) // green

    menuDimStyle = lipgloss.NewStyle().Faint(true)
)
```

These are unexported package-level vars, defined at the top of `menu.go`. No global state — lipgloss styles are immutable value types.

---

## 7. Edge Cases and Constraints

| Situation                          | Handling                                                         |
|------------------------------------|------------------------------------------------------------------|
| Seed == 0 in display               | Show `0` (random) — no special handling needed                  |
| Non-key messages (TickMsg, etc.)   | Return `(m, nil)` — menu ignores non-key messages               |
| ThemeRegistry empty                | Can't happen — `NewRegistry()` always loads 5 built-in themes   |
| `applyLeft` on Theme at index 0    | Wrap to last theme in `themes.List()`                            |
| Unknown key                        | No-op, return `(m, nil)`                                         |
| Ctrl+N shortcut from ScreenMenu    | Handled in `MenuModel.Update()` — emit `NewGameMsg` directly     |
| q / Ctrl+C from ScreenMenu         | Still handled in `AppModel.Update()` via the `ScreenMenu` case (after delegating to menu and checking if cmd is nil, or handled before delegation) |

> **Note on q/Ctrl+C**: The cleanest approach is to keep global quit-key handling in `AppModel.Update()` _before_ delegating to the menu sub-model. In the `ScreenMenu` case, check for q and Ctrl+C first, then fall through to `m.menu.Update(msg)`. This mirrors how the board handles global keys.

---

## 8. AppModel.Update — Revised ScreenMenu Case (full)

```go
case ScreenMenu:
    // Global exit keys handled before delegating to sub-model.
    if key, ok := msg.(tea.KeyMsg); ok {
        switch {
        case key.Type == tea.KeyCtrlC:
            return m, tea.Quit
        case key.Type == tea.KeyRunes && len(key.Runes) > 0 &&
            (key.Runes[0] == 'q' || key.Runes[0] == 'Q'):
            return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenQuitConfirm} }
        }
    }
    updated, cmd := m.menu.Update(msg)
    m.menu = updated.(MenuModel)
    return m, cmd
```

---

## 9. File Checklist

```
tui/
├── menu.go          ← NEW  (MenuModel + View)
├── menu_test.go     ← NEW  (golden + state tests)
├── app.go           ← MODIFY (4 targeted edits: struct field, NewAppModel, Update routing, View)
└── testdata/
    └── TestMenuRender.golden  ← GENERATED (first run with -update flag)
```

---

## 10. Execution Order

1. Write `tui/menu.go` (MenuModel struct → Init → Update → View).
2. Modify `tui/app.go` (4 edits in steps 2a–2d).
3. Build and `go vet ./...` — fix any compilation errors.
4. Write `tui/menu_test.go` (helpers → state tests → golden test).
5. Run `go test ./tui/ -run TestMenuRender -update` to generate the golden file.
6. Run `go test ./tui/` — all tests must pass.
7. Commit and push on `claude/agent-c-t14-plan-wThLF`.

---

## 11. T15 Handoff Note

After T14 merges:
- `ScreenMenu` routes fully to `MenuModel`; the old ScreenMenu/ScreenWin combined fallback is gone.
- T15 (Help/Pause/Dialog) can start immediately — it only touches `ScreenPaused`, `ScreenHelp`, `ScreenQuitConfirm`, which still use placeholder stubs.
- T16 (Mouse) touches `tui/input.go` and `tui/cursor.go` — no conflict with menu.
- T17 (Auto-Complete) touches `tui/board.go` — no conflict.
