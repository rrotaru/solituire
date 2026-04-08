# Agent C — Phase T13 Implementation Plan
## App Shell + Screen Routing

**Branch**: `claude/agent-c-t13-plan-53zkc`
**Stage**: 7
**Estimate**: Medium
**Dependencies**: T4 ✓ (`config/config.go`), T11 ✓ (`tui/board.go`, `tui/cursor.go`), T12 ✓ (`engine/game.go` implements `GameEngine`)
**Blocks**: T14 (Menu), T15 (Help/Pause/Dialog), T16 (Mouse), T17 (Auto-Complete)

---

## Overview

T13 wires together all prior work into a running program. It adds `tui/app.go` (the root Bubbletea model) and fills in `main.go` (currently a no-op stub). No engine, renderer, config, or theme code is modified. The only new TUI surface added is the `AppModel` routing shell; the sub-models for Menu, Help, Pause, Dialog, and Win are stubs or placeholders that later phases will replace.

### Files created / modified

| File | Status | Contents |
|------|--------|----------|
| `tui/app.go` | **New** | `AppModel` struct, `NewAppModel`, `Init`, `Update`, `View` |
| `main.go` | **Modified** | Full program entry point |
| `tui/app_test.go` | **New** | Pattern C (model state) screen-transition tests |

### What T13 does NOT touch

- `tui/messages.go` — `AppScreen` and all `Msg` types already defined in T9. **Do not redefine `AppScreen` in `app.go`.**
- `tui/input.go`, `tui/cursor.go`, `tui/board.go` — already complete.
- `engine/`, `renderer/`, `config/`, `theme/` — zero changes.

---

## Codebase State at T13 Start

The following contracts are available and must be used as-is:

```go
// engine/interfaces.go — GameEngine
engine.NewGame(seed int64, drawCount int) *engine.Game
eng.NewGame(seed int64, drawCount int)   // on existing instance
eng.RestartDeal()
eng.State() *engine.GameState
eng.IsWon() bool

// renderer/renderer.go
renderer.New(t theme.Theme) *renderer.Renderer
renderer.MinTermWidth  = 78
renderer.MinTermHeight = 24
rend.SetSize(w, h int)

// theme/registry.go
theme.NewRegistry() *theme.ThemeRegistry
reg.Get(name string) theme.Theme

// config/config.go
config.DefaultConfig() *config.Config

// tui/board.go
tui.NewBoardModel(eng engine.GameEngine, rend *renderer.Renderer, cfg *config.Config) tui.BoardModel

// tui/messages.go — already defined, do not redefine
tui.AppScreen  (ScreenMenu, ScreenPlaying, ScreenPaused, ScreenHelp, ScreenQuitConfirm, ScreenWin)
tui.ChangeScreenMsg{Screen AppScreen}
tui.NewGameMsg{Seed int64, DrawCount int}
tui.RestartDealMsg{}
tui.GameWonMsg{}
tui.ThemeChangedMsg{Theme *theme.Theme}
tui.ConfigChangedMsg{Config *config.Config}
```

---

## Step 1 — `tui/app.go`

### 1a — `AppModel` struct

```go
package tui

import (
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "solituire/config"
    "solituire/engine"
    "solituire/renderer"
    "solituire/theme"
)

// AppModel is the root Bubbletea model. It owns screen state, routes messages
// to the active sub-model, and delegates rendering.
type AppModel struct {
    screen   AppScreen
    engine   engine.GameEngine
    cfg      *config.Config
    themes   *theme.ThemeRegistry
    rend     *renderer.Renderer
    board    BoardModel
    windowW  int
    windowH  int
    tooSmall bool
}
```

**Key field notes:**
- `screen` — uses the `AppScreen` iota already defined in `messages.go`
- `board` — the only real sub-model in T13 scope; later phases add menu, help, pause, dialog, celebration
- `tooSmall` — derived from `windowW`/`windowH` vs `renderer.MinTermWidth`/`MinTermHeight`; recomputed on every `tea.WindowSizeMsg`
- No sub-model fields for Menu/Help/Pause/Dialog/Win — these are added as stubs when those phases land

### 1b — `NewAppModel` constructor

```go
// NewAppModel creates a ready-to-run AppModel. The initial screen is ScreenPlaying.
// When the Menu sub-model is added in T14, this should be changed to ScreenMenu.
func NewAppModel(
    eng engine.GameEngine,
    rend *renderer.Renderer,
    cfg *config.Config,
    themes *theme.ThemeRegistry,
) AppModel {
    return AppModel{
        screen:  ScreenPlaying,
        engine:  eng,
        cfg:     cfg,
        themes:  themes,
        rend:    rend,
        board:   NewBoardModel(eng, rend, cfg),
        windowW: renderer.MinTermWidth,
        windowH: renderer.MinTermHeight,
    }
}
```

> **T14 handoff note**: change the initial `screen` to `ScreenMenu` once `tui/menu.go` is implemented.

### 1c — `Init()`

Delegate to the board's `Init` (which starts the elapsed-time ticker):

```go
func (m AppModel) Init() tea.Cmd {
    return m.board.Init()
}
```

### 1d — `Update()`

App-level messages are handled first (before routing to sub-models). Any message not matched at the app level is forwarded to the active sub-model.

```go
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        m.windowW = msg.Width
        m.windowH = msg.Height
        m.tooSmall = msg.Width < renderer.MinTermWidth || msg.Height < renderer.MinTermHeight
        // Always propagate to board so renderer dimensions stay current.
        updated, cmd := m.board.Update(msg)
        m.board = updated.(BoardModel)
        return m, cmd

    case ChangeScreenMsg:
        m.screen = msg.Screen
        return m, nil

    case NewGameMsg:
        seed := msg.Seed
        if seed == 0 {
            seed = newSeed()
        }
        m.engine.NewGame(seed, msg.DrawCount)
        m.cfg.DrawCount = msg.DrawCount
        m.board = NewBoardModel(m.engine, m.rend, m.cfg)
        m.screen = ScreenPlaying
        return m, m.board.Init()

    case RestartDealMsg:
        m.engine.RestartDeal()
        m.board = NewBoardModel(m.engine, m.rend, m.cfg)
        m.screen = ScreenPlaying
        return m, m.board.Init()

    case GameWonMsg:
        m.screen = ScreenWin
        return m, nil

    case ThemeChangedMsg:
        if msg.Theme != nil {
            m.cfg.ThemeName = msg.Theme.Name
        }
        return m, nil

    case ConfigChangedMsg:
        if msg.Config != nil {
            m.cfg = msg.Config
        }
        return m, nil
    }

    // Route to active sub-model.
    switch m.screen {
    case ScreenPlaying:
        updated, cmd := m.board.Update(msg)
        m.board = updated.(BoardModel)
        return m, cmd
    }

    // All other screens have no sub-model in T13; messages are silently dropped.
    return m, nil
}
```

Private helper (place at bottom of file):

```go
// newSeed returns a non-deterministic seed for new games when cfg.Seed == 0.
// Isolated here so tests can verify the NewGameMsg branch without time dependency.
func newSeed() int64 {
    return time.Now().UnixNano()
}
```

Add `"time"` to imports.

### 1e — `View()`

```go
func (m AppModel) View() string {
    if m.tooSmall {
        return fmt.Sprintf(
            "Terminal too small.\nMinimum size: %d×%d\nCurrent: %d×%d",
            renderer.MinTermWidth, renderer.MinTermHeight,
            m.windowW, m.windowH,
        )
    }

    switch m.screen {
    case ScreenPlaying:
        return m.board.View()
    case ScreenPaused:
        return "Game Paused — press any key to resume." // replaced by tui/pause.go in T15
    case ScreenHelp:
        return "Help — press Esc or F1 to close."       // replaced by tui/help.go in T15
    case ScreenQuitConfirm:
        return "Quit? (y) Yes  (n) No"                  // replaced by tui/dialog.go in T15
    case ScreenWin:
        return "You won! Press Ctrl+N for a new game."  // replaced by tui/celebration.go in T18
    case ScreenMenu:
        return "Klondike Solitaire\n\nPress Ctrl+N to start a new game." // replaced by tui/menu.go in T14
    }
    return ""
}
```

**Full import block for `tui/app.go`:**

```go
import (
    "fmt"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "solituire/config"
    "solituire/engine"
    "solituire/renderer"
    "solituire/theme"
)
```

---

## Step 2 — `main.go`

Replace the current empty `main()`:

```go
package main

import (
    "fmt"
    "math/rand"
    "os"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "solituire/config"
    "solituire/engine"
    "solituire/renderer"
    "solituire/theme"
    "solituire/tui"
)

func main() {
    cfg := config.DefaultConfig()

    reg := theme.NewRegistry()
    th := reg.Get(cfg.ThemeName)

    seed := cfg.Seed
    if seed == 0 {
        // Use a seeded source for reproducibility in tests; rand.Int63() is fine
        // for production because we just need any non-zero value here.
        seed = time.Now().UnixNano()
        _ = rand.New(rand.NewSource(seed)) // satisfies staticcheck unused-import check
        seed = time.Now().UnixNano()
    }

    eng := engine.NewGame(seed, cfg.DrawCount)
    rend := renderer.New(th)
    app := tui.NewAppModel(eng, rend, cfg, reg)

    p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "solituire: %v\n", err)
        os.Exit(1)
    }
}
```

> **Simpler seed logic**: since `rand` is not actually needed in main (the seed is just `time.Now().UnixNano()`), drop the `math/rand` import and just use `time.Now().UnixNano()` directly.

**Revised, cleaner `main.go`:**

```go
package main

import (
    "fmt"
    "os"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "solituire/config"
    "solituire/engine"
    "solituire/renderer"
    "solituire/theme"
    "solituire/tui"
)

func main() {
    cfg := config.DefaultConfig()

    reg := theme.NewRegistry()
    th := reg.Get(cfg.ThemeName)

    seed := cfg.Seed
    if seed == 0 {
        seed = time.Now().UnixNano()
    }

    eng := engine.NewGame(seed, cfg.DrawCount)
    rend := renderer.New(th)
    app := tui.NewAppModel(eng, rend, cfg, reg)

    p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "solituire: %v\n", err)
        os.Exit(1)
    }
}
```

---

## Step 3 — `tui/app_test.go`

Pattern C: call `model.Update(msg)`, type-assert the result back to `AppModel`, inspect exported state.

```go
package tui

import (
    "testing"

    tea "github.com/charmbracelet/bubbletea"
    "solituire/config"
    "solituire/engine"
    "solituire/renderer"
    "solituire/theme"
)

// newTestApp returns a minimal AppModel suitable for unit testing.
func newTestApp() AppModel {
    cfg := config.DefaultConfig()
    reg := theme.NewRegistry()
    th := reg.Get(cfg.ThemeName)
    eng := engine.NewGame(42, 1)
    rend := renderer.New(th)
    return NewAppModel(eng, rend, cfg, reg)
}

// updateApp is a helper that calls Update and type-asserts the result.
func updateApp(m AppModel, msg tea.Msg) AppModel {
    result, _ := m.Update(msg)
    return result.(AppModel)
}

// --- ChangeScreenMsg transitions ---

func TestAppModel_ChangeScreen_ToPaused(t *testing.T) {
    m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenPaused})
    if m.screen != ScreenPaused {
        t.Errorf("screen = %v, want ScreenPaused", m.screen)
    }
}

func TestAppModel_ChangeScreen_ToHelp(t *testing.T) {
    m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenHelp})
    if m.screen != ScreenHelp {
        t.Errorf("screen = %v, want ScreenHelp", m.screen)
    }
}

func TestAppModel_ChangeScreen_ToQuitConfirm(t *testing.T) {
    m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenQuitConfirm})
    if m.screen != ScreenQuitConfirm {
        t.Errorf("screen = %v, want ScreenQuitConfirm", m.screen)
    }
}

func TestAppModel_ChangeScreen_ToWin(t *testing.T) {
    m := updateApp(newTestApp(), ChangeScreenMsg{Screen: ScreenWin})
    if m.screen != ScreenWin {
        t.Errorf("screen = %v, want ScreenWin", m.screen)
    }
}

func TestAppModel_ChangeScreen_ToPlaying(t *testing.T) {
    // Start on a non-Playing screen, then transition back.
    app := newTestApp()
    app.screen = ScreenPaused
    m := updateApp(app, ChangeScreenMsg{Screen: ScreenPlaying})
    if m.screen != ScreenPlaying {
        t.Errorf("screen = %v, want ScreenPlaying", m.screen)
    }
}

// --- GameWonMsg ---

func TestAppModel_GameWonMsg_TransitionsToWin(t *testing.T) {
    m := updateApp(newTestApp(), GameWonMsg{})
    if m.screen != ScreenWin {
        t.Errorf("screen = %v, want ScreenWin after GameWonMsg", m.screen)
    }
}

// --- NewGameMsg ---

func TestAppModel_NewGameMsg_TransitionsToPlaying(t *testing.T) {
    app := newTestApp()
    app.screen = ScreenMenu
    m := updateApp(app, NewGameMsg{Seed: 1234, DrawCount: 1})
    if m.screen != ScreenPlaying {
        t.Errorf("screen = %v, want ScreenPlaying after NewGameMsg", m.screen)
    }
}

func TestAppModel_NewGameMsg_UpdatesDrawCount(t *testing.T) {
    app := newTestApp()
    m := updateApp(app, NewGameMsg{Seed: 99, DrawCount: 3})
    if m.cfg.DrawCount != 3 {
        t.Errorf("DrawCount = %d, want 3 after NewGameMsg", m.cfg.DrawCount)
    }
}

// --- RestartDealMsg ---

func TestAppModel_RestartDealMsg_TransitionsToPlaying(t *testing.T) {
    app := newTestApp()
    app.screen = ScreenPaused
    m := updateApp(app, RestartDealMsg{})
    if m.screen != ScreenPlaying {
        t.Errorf("screen = %v, want ScreenPlaying after RestartDealMsg", m.screen)
    }
}

// --- WindowSizeMsg + tooSmall ---

func TestAppModel_WindowSizeMsg_TooSmall(t *testing.T) {
    m := updateApp(newTestApp(), tea.WindowSizeMsg{Width: 10, Height: 10})
    if !m.tooSmall {
        t.Error("tooSmall should be true for 10×10 terminal")
    }
}

func TestAppModel_WindowSizeMsg_LargeEnough(t *testing.T) {
    m := updateApp(newTestApp(), tea.WindowSizeMsg{
        Width:  renderer.MinTermWidth + 10,
        Height: renderer.MinTermHeight + 10,
    })
    if m.tooSmall {
        t.Error("tooSmall should be false for an adequately sized terminal")
    }
}

func TestAppModel_WindowSizeMsg_ExactMinimum(t *testing.T) {
    m := updateApp(newTestApp(), tea.WindowSizeMsg{
        Width:  renderer.MinTermWidth,
        Height: renderer.MinTermHeight,
    })
    if m.tooSmall {
        t.Error("tooSmall should be false at exactly minimum dimensions")
    }
}

// --- View returns non-empty strings for every screen ---

func TestAppModel_View_AllScreens(t *testing.T) {
    screens := []AppScreen{
        ScreenMenu, ScreenPlaying, ScreenPaused,
        ScreenHelp, ScreenQuitConfirm, ScreenWin,
    }
    for _, s := range screens {
        app := newTestApp()
        app.screen = s
        // Ensure View doesn't panic and returns a non-empty string.
        v := app.View()
        if v == "" {
            t.Errorf("View() returned empty string for screen %v", s)
        }
    }
}
```

---

## Pitfalls and Constraints

| Concern | Resolution |
|---------|------------|
| `AppScreen` already defined in `messages.go` | **Do not redefine it** in `app.go`; just use it directly |
| `BoardModel` is in the same `tui` package | No import needed; reference as `BoardModel` |
| `board.Update()` returns `(tea.Model, tea.Cmd)` | Type-assert: `updated.(BoardModel)` — safe because `BoardModel` always returns itself |
| `NewGameMsg.Seed == 0` | Generate a live seed (`time.Now().UnixNano()`) in `Update`; the `newSeed()` helper isolates this |
| `tooSmall` — boundary condition | `<` not `<=`: exactly `MinTermWidth × MinTermHeight` is acceptable |
| `tea.WithMouseCellMotion()` in `main.go` | Required for T16 mouse support; harmless if T16 hasn't landed yet |
| Sub-models for Menu/Help/Pause/Dialog/Win are missing | Use placeholder `View()` strings; each will be replaced when those phases land |
| `time` import in `tui/app.go` | Needed for `newSeed()` |

---

## Verification Gate

```bash
go build ./...          # must compile cleanly (main.go + all packages)
go test ./tui/...       # all existing + new T13 tests must pass
go vet ./...            # zero warnings
```

The binary produced by `go build` should launch a playable game in the terminal.

---

## Handoff Contract

When T13 is complete, the following are available:

- `tui.AppModel` — root model, exported, usable from `main.go`
- `tui.NewAppModel(eng, rend, cfg, themes) AppModel` — canonical constructor
- Screen routing via `ChangeScreenMsg` — all six `AppScreen` values handled
- `main.go` — binary launches with alt-screen + mouse support
- Placeholder `View()` strings for Menu, Paused, Help, QuitConfirm, Win screens

**T14 (Menu)**: add `menu MenuModel` field to `AppModel`, route `ScreenMenu` in `Update`/`View`, change `NewAppModel` initial screen to `ScreenMenu`.

**T15 (Help/Pause/Dialog)**: add corresponding sub-model fields to `AppModel`, replace placeholder strings in `View()`.

**T18 (Celebration)**: replace `ScreenWin` placeholder in `View()` with `celebration.View()`.
