# Agent B — T18 Win Celebration: Implementation Plan

> **Phase**: T18 (Stage 8)
> **Owner**: Agent B (Renderer Specialist)
> **Output**: `tui/celebration.go` + tests
> **Dependencies**: T12 (`engine.GameEngine.IsWon`), T13 (app shell `ScreenWin` routing)
> **Design sections**: §12.6 (Win Celebration), §7.2 (Root Model), §7.3 (Messages), §7.4 (Update Flow)

---

## 1. Context & What Already Exists

### Already implemented (do NOT recreate)

| Symbol | Location | Notes |
|--------|----------|-------|
| `AppScreen`, `ScreenWin` | `tui/messages.go:22` | Enum value already defined |
| `GameWonMsg{}` | `tui/messages.go:33` | Already defined and handled |
| `CelebrationTickMsg{}` | `tui/messages.go:44` | Already defined, not yet consumed |
| `GameWonMsg` handler | `tui/app.go:115-117` | Sets `m.screen = ScreenWin`, returns nil Cmd |
| `ScreenWin` Update stub | `tui/app.go:185-199` | Handles Ctrl+N, q, Ctrl+C directly in AppModel |
| `ScreenWin` View stub | `tui/app.go:229` | Returns plain string `"You won! …"` |
| `engine.GameEngine.IsWon()` | `engine/interfaces.go:8` | Used by BoardModel already |
| `engine.GameState.Score` | `engine/game.go:195` | `int` |
| `engine.GameState.MoveCount` | `engine/game.go:196` | `int` |
| `engine.GameState.ElapsedTime` | `engine/game.go:197` | `time.Duration` |

### What is missing

1. `tui/celebration.go` — the entire file does not exist yet.
2. `AppModel.celebration` field — not in the struct (`tui/app.go:17-29`).
3. `AppModel` does not initialize `CelebrationModel` on `GameWonMsg`.
4. `AppModel.Update()` does not delegate to `CelebrationModel` for `ScreenWin`.
5. `AppModel.View()` does not delegate to `CelebrationModel` for `ScreenWin`.
6. Golden file `tui/testdata/TestCelebrationView.golden` does not exist yet.

---

## 2. Files to Create / Modify

| Action | File |
|--------|------|
| **Create** | `tui/celebration.go` |
| **Create** | `tui/celebration_test.go` |
| **Modify** | `tui/app.go` (3 targeted edits) |
| **Auto-generated** | `tui/testdata/TestCelebrationView.golden` (via `-update`) |

---

## 3. `CelebrationModel` Design

### 3.1 Struct

```go
// tui/celebration.go
package tui

import (
    "fmt"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "solituire/theme"
)

// celebCard is one "falling" card in the cascade animation.
type celebCard struct {
    symbol string // e.g. "A♠", "K♥"
    col    int    // terminal column (0-indexed)
    row    int    // current terminal row
    speed  int    // rows advanced per tick (1 or 2)
    style  lipgloss.Style
}

// CelebrationModel is the Bubbletea sub-model for the win screen.
// It renders a congratulations message with final stats and a cascading
// card animation driven by CelebrationTickMsg ticks.
type CelebrationModel struct {
    score   int
    moves   int
    elapsed time.Duration
    th      theme.Theme

    // Animation state
    frame   int        // incremented on each CelebrationTickMsg
    cards   []celebCard // current positions of cascading cards
    windowW int
    windowH int
}
```

**Design rationale:**
- Stats are captured at construction time from `engine.GameState` so the model is self-contained and testable without a live engine.
- `theme.Theme` is passed in for Lipgloss styling, consistent with the rest of the renderer package.
- `frame int` is the primary animation driver; frame 0 is the deterministic "static" state that golden tests use.
- `cards []celebCard` tracks each falling card's column and current row so the animation can be rendered correctly at any frame.

### 3.2 Constructor

```go
// NewCelebrationModel creates a CelebrationModel with stats captured from the
// engine at the moment of win detection. The windowW/H defaults match the
// renderer minimum so it is safe to construct before the first WindowSizeMsg.
func NewCelebrationModel(score, moves int, elapsed time.Duration, th theme.Theme) CelebrationModel {
    m := CelebrationModel{
        score:   score,
        moves:   moves,
        elapsed: elapsed,
        th:      th,
        windowW: 78, // renderer.MinTermWidth — avoid import cycle by using literal
        windowH: 24,
    }
    m.cards = initCascadeCards(m.windowW, th)
    return m
}
```

**`initCascadeCards`**: Creates ~12–16 card symbols placed at evenly-spaced columns, all starting at row 0 (or slightly above). Uses a deterministic column spread derived from `windowW` so that the static (frame 0) view is reproducible for golden tests. Each card gets a speed of 1 or 2 rows/tick, cycling through a fixed pattern (not random) so tests are predictable.

Card symbols to use: `A♠ 2♥ 3♦ 4♣ 5♠ 6♥ 7♦ 8♣ 9♠ 10♥ J♦ Q♣ K♠ K♥ Q♦ J♣` — a fixed list that repeats as needed. Color alternates red/black using `th.SuitRed` and `th.SuitBlack`.

### 3.3 `Init()`

```go
func (m CelebrationModel) Init() tea.Cmd {
    return celebTickCmd()
}

// celebTickCmd fires a CelebrationTickMsg after 80ms.
func celebTickCmd() tea.Cmd {
    return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
        return CelebrationTickMsg{}
    })
}
```

`Init()` immediately schedules the first animation tick. This is returned by `AppModel` when it transitions to `ScreenWin`.

### 3.4 `Update()`

```go
func (m CelebrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case CelebrationTickMsg:
        m.frame++
        m.advanceCards()
        return m, celebTickCmd()

    case tea.WindowSizeMsg:
        m.windowW = msg.Width
        m.windowH = msg.Height
        m.cards = initCascadeCards(m.windowW, m.th)
        m.frame = 0
        return m, nil

    case tea.KeyMsg:
        switch {
        case msg.Type == tea.KeyCtrlN:
            return m, func() tea.Msg {
                return NewGameMsg{Seed: 0, DrawCount: 1}
            }
        case msg.Type == tea.KeyRunes && len(msg.Runes) > 0 &&
            (msg.Runes[0] == 'q' || msg.Runes[0] == 'Q'):
            return m, func() tea.Msg {
                return ChangeScreenMsg{Screen: ScreenQuitConfirm}
            }
        case msg.Type == tea.KeyCtrlC:
            return m, tea.Quit
        }
    }
    return m, nil
}
```

**`advanceCards()`**: Increments each `celebCard.row` by its `speed`. When a card's row exceeds `windowH`, it wraps back to `-(its height)` so the cascade loops continuously.

> **Important**: `CelebrationModel.Update()` handles key presses itself and emits `NewGameMsg` / `ChangeScreenMsg` — the same messages the existing `app.go:185-199` stub handles. After wiring, the stub in `app.go` is replaced with delegation.
>
> **DrawCount for new game**: `CelebrationModel` does not have access to `config.Config`. Pass `DrawCount: 0` in `NewGameMsg`; `AppModel.Update()` already substitutes `m.cfg.DrawCount` when it processes `NewGameMsg` — but actually looking at the code in `app.go:88-103`, `msg.DrawCount` is used directly. So we need to capture DrawCount.
>
> **Revised**: Pass `drawCount int` to `NewCelebrationModel` and store it. Emit `NewGameMsg{Seed: 0, DrawCount: m.drawCount}`.

### 3.5 `View()`

```go
func (m CelebrationModel) View() string {
    // 1. Congratulations banner
    banner := m.th.HeaderText.Render("🎉  You Win!  🎉")

    // 2. Stats block
    stats := lipgloss.JoinVertical(lipgloss.Center,
        fmt.Sprintf("Score:  %d", m.score),
        fmt.Sprintf("Moves:  %d", m.moves),
        fmt.Sprintf("Time:   %s", formatElapsed(m.elapsed)),
    )

    // 3. Key hint footer
    hint := m.th.FooterText.Render("[Ctrl+N] New Game    [Q] Quit")

    // 4. Center the text block vertically (simple padding calculation)
    center := lipgloss.NewStyle().
        Width(m.windowW).
        Align(lipgloss.Center)

    textBlock := lipgloss.JoinVertical(lipgloss.Center,
        banner, "", stats, "", hint,
    )
    centeredText := center.Render(textBlock)

    // 5. Overlay cascading cards
    if m.frame == 0 {
        // Frame 0: no cards rendered → deterministic golden output
        return centeredText
    }
    return overlayCards(centeredText, m.cards, m.windowW, m.windowH)
}
```

**Key design decision — frame 0 is card-free**: The static golden test uses frame 0 (before any `CelebrationTickMsg` is processed). This guarantees the golden file is deterministic and stable across runs. The animation only appears from frame 1 onward, where VHS testing handles validation (per spec §14.7 + todo note "Animation frames are non-deterministic and tested via VHS (T19)").

**`overlayCards`**: Builds a multi-line string by overlaying each `celebCard`'s symbol at its `(col, row)` position into the base text block. Uses a `[][]rune` grid approach: start from the rendered base text lines, then write card symbols at their row/col positions.

**`formatElapsed`**: Formats `time.Duration` as `"M:SS"` (e.g. `"3:42"`).

---

## 4. `app.go` Integration — Three Targeted Edits

### Edit 1: Add `celebration` field to `AppModel` (line ~28)

```go
// Before (tui/app.go:17-29):
type AppModel struct {
    screen     AppScreen
    prevScreen AppScreen
    engine     engine.GameEngine
    cfg        *config.Config
    themes     *theme.ThemeRegistry
    rend       *renderer.Renderer
    board      BoardModel
    menu       MenuModel
    windowW    int
    windowH    int
    tooSmall   bool
}

// After:
type AppModel struct {
    screen      AppScreen
    prevScreen  AppScreen
    engine      engine.GameEngine
    cfg         *config.Config
    themes      *theme.ThemeRegistry
    rend        *renderer.Renderer
    board       BoardModel
    menu        MenuModel
    celebration CelebrationModel          // ← add
    windowW     int
    windowH     int
    tooSmall    bool
}
```

### Edit 2: Initialize `CelebrationModel` in `GameWonMsg` handler (line ~115-117)

```go
// Before (tui/app.go:115-117):
case GameWonMsg:
    m.screen = ScreenWin
    return m, nil

// After:
case GameWonMsg:
    state := m.engine.State()
    m.celebration = NewCelebrationModel(
        m.engine.Score(),
        m.engine.MoveCount(),
        state.ElapsedTime,
        m.themes.Get(m.cfg.ThemeName),
        m.cfg.DrawCount,
    )
    m.screen = ScreenWin
    return m, m.celebration.Init()
```

**Rationale**: `Init()` starts the animation tick chain. Stats are captured at win-time so the celebration screen stays consistent even if the engine were somehow mutated later.

### Edit 3: Delegate `ScreenWin` routing in `Update()` and `View()` (lines ~185-199 and ~229)

**Update routing** — replace the existing `ScreenWin` case stub:
```go
// Before (tui/app.go:185-199):
case ScreenWin:
    if key, ok := msg.(tea.KeyMsg); ok {
        switch {
        case key.Type == tea.KeyCtrlN:
            return m, func() tea.Msg {
                return NewGameMsg{Seed: appSeed(), DrawCount: m.cfg.DrawCount}
            }
        // ... etc
        }
    }

// After:
case ScreenWin:
    updated, cmd := m.celebration.Update(msg)
    m.celebration = updated.(CelebrationModel)
    return m, cmd
```

**View routing** — replace the `ScreenWin` view stub:
```go
// Before (tui/app.go:229):
case ScreenWin:
    return "You won! Press Ctrl+N for a new game."

// After:
case ScreenWin:
    return m.celebration.View()
```

**Forward `WindowSizeMsg` to celebration** — in the existing `WindowSizeMsg` handler at `app.go:61-68`, add a celebration update alongside the board update:
```go
// In the tea.WindowSizeMsg case (after existing board update):
if m.screen == ScreenWin {
    celebUpdated, _ := m.celebration.Update(msg)
    m.celebration = celebUpdated.(CelebrationModel)
}
```

---

## 5. Tests

### 5.1 File: `tui/celebration_test.go`

**Test setup** (top of file):
```go
package tui

import (
    "testing"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/x/exp/golden"
    "github.com/muesli/termenv"
    "solituire/theme"
)

func init() {
    lipgloss.SetColorProfile(termenv.Ascii)
}

func newTestCelebration() CelebrationModel {
    reg := theme.NewRegistry()
    th := reg.Get("classic")
    return NewCelebrationModel(1250, 87, 3*time.Minute+42*time.Second, th, 1)
}
```

**Golden test for static view (frame 0)**:
```go
func TestCelebrationView(t *testing.T) {
    m := newTestCelebration()
    // frame == 0: no cascading cards, fully deterministic
    got := m.View()
    golden.RequireEqual(t, []byte(got))
}
```

**Model state tests**:

```go
// CelebrationTickMsg increments the frame counter.
func TestCelebrationModel_TickAdvancesFrame(t *testing.T) {
    m := newTestCelebration()
    if m.frame != 0 {
        t.Fatalf("initial frame = %d, want 0", m.frame)
    }
    result, cmd := m.Update(CelebrationTickMsg{})
    updated := result.(CelebrationModel)
    if updated.frame != 1 {
        t.Errorf("frame after tick = %d, want 1", updated.frame)
    }
    if cmd == nil {
        t.Error("CelebrationTickMsg: returned nil Cmd, animation tick chain broken")
    }
}

// Ctrl+N emits NewGameMsg.
func TestCelebrationModel_CtrlN_EmitsNewGameMsg(t *testing.T) {
    m := newTestCelebration()
    _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
    if cmd == nil {
        t.Fatal("Ctrl+N: returned nil Cmd, expected NewGameMsg")
    }
    msg := cmd()
    if _, ok := msg.(NewGameMsg); !ok {
        t.Errorf("Ctrl+N: got %T, want NewGameMsg", msg)
    }
}

// 'q' emits ChangeScreenMsg{ScreenQuitConfirm}.
func TestCelebrationModel_Q_EmitsQuitConfirm(t *testing.T) {
    m := newTestCelebration()
    _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    if cmd == nil {
        t.Fatal("'q': returned nil Cmd, expected ChangeScreenMsg")
    }
    msg := cmd()
    csm, ok := msg.(ChangeScreenMsg)
    if !ok || csm.Screen != ScreenQuitConfirm {
        t.Errorf("'q': got %v, want ChangeScreenMsg{ScreenQuitConfirm}", msg)
    }
}

// WindowSizeMsg resets the animation to frame 0.
func TestCelebrationModel_WindowSizeMsg_ResetsAnimation(t *testing.T) {
    m := newTestCelebration()
    // Advance a few frames first.
    for i := 0; i < 5; i++ {
        result, _ := m.Update(CelebrationTickMsg{})
        m = result.(CelebrationModel)
    }
    if m.frame == 0 {
        t.Skip("frames did not advance, skipping reset test")
    }
    result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
    updated := result.(CelebrationModel)
    if updated.frame != 0 {
        t.Errorf("frame after WindowSizeMsg = %d, want 0", updated.frame)
    }
    if updated.windowW != 100 {
        t.Errorf("windowW = %d, want 100", updated.windowW)
    }
}

// Win detection trigger: AppModel routes GameWonMsg → ScreenWin and
// the celebration model is initialized with correct stats.
func TestAppModel_GameWonMsg_InitializesCelebration(t *testing.T) {
    app := newTestApp()
    // Force engine into a "won" state isn't easy, so we directly send GameWonMsg
    // and verify the AppModel initializes the CelebrationModel properly.
    result, cmd := app.Update(GameWonMsg{})
    updated := result.(AppModel)

    if updated.screen != ScreenWin {
        t.Errorf("screen = %v, want ScreenWin", updated.screen)
    }
    if cmd == nil {
        t.Error("GameWonMsg: returned nil Cmd; celebration.Init() not called")
    }
    // The view for ScreenWin must now come from CelebrationModel, not a stub string.
    view := updated.View()
    if view == "You won! Press Ctrl+N for a new game." {
        t.Error("ScreenWin View() still returns stub string — CelebrationModel not wired")
    }
    if view == "" {
        t.Error("ScreenWin View() returned empty string after CelebrationModel init")
    }
}
```

### 5.2 Generating the golden file

Run once with the update flag to generate `tui/testdata/TestCelebrationView.golden`:

```
go test ./tui/ -run TestCelebrationView -update
```

After generation, commit the golden file. Subsequent runs assert against it without `-update`.

---

## 6. Implementation Order

Work in this sequence to keep the build compiling at each step:

1. **Create `tui/celebration.go`** — add empty `CelebrationModel` struct + stubs for `Init`, `Update`, `View`, `NewCelebrationModel`.
2. **Edit `tui/app.go` — Edit 1** — add `celebration CelebrationModel` field.  
   ✓ Build still passes (zero-value `CelebrationModel` is valid).
3. **Edit `tui/app.go` — Edit 2** — update `GameWonMsg` handler.  
   ✓ Calls `NewCelebrationModel` (now exists from step 1).
4. **Edit `tui/app.go` — Edit 3** — delegate `ScreenWin` in `Update()` and `View()`.  
   ✓ `CelebrationModel.Update()` + `View()` now called.
5. **Implement `CelebrationModel` fully** — fill in `View()`, `advanceCards()`, `initCascadeCards()`, `overlayCards()`, `formatElapsed()`, `celebTickCmd()`.
6. **Write `tui/celebration_test.go`** — add all tests from §5.1.
7. **Generate golden file** — run `go test ./tui/ -run TestCelebrationView -update`.
8. **Run full test suite** — `go test ./tui/ ./...` must pass.

---

## 7. Edge Cases & Guard Rails

| Scenario | Handling |
|----------|---------|
| Terminal resized while on win screen | `WindowSizeMsg` forwarded to `CelebrationModel.Update()`; resets card positions and frame to 0 |
| `CelebrationModel` zero value (before `GameWonMsg`) | `View()` returns empty string; `AppModel.View()` only calls it when `screen == ScreenWin` |
| Very narrow terminal (`tooSmall == true`) | `AppModel.View()` returns the "Terminal too small" error before reaching `ScreenWin` case |
| DrawCount in new game from celebration | Stored as `drawCount int` in `CelebrationModel`, passed into emitted `NewGameMsg` |
| `CelebrationModel` not yet a `tea.Model` | Ensure `Update()` returns `(tea.Model, tea.Cmd)` — the type assertion `updated.(CelebrationModel)` in app.go depends on this |

---

## 8. Contracts Produced

This phase produces no downstream contracts (T18 is a leaf node in the task graph). The outputs are consumed only by T19 (VHS tapes) and T20 (integration smoke test).

```go
// tui/celebration.go — public API
type CelebrationModel struct { ... }

func NewCelebrationModel(
    score, moves int,
    elapsed time.Duration,
    th theme.Theme,
    drawCount int,
) CelebrationModel

func (m CelebrationModel) Init() tea.Cmd
func (m CelebrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m CelebrationModel) View() string
```

---

## 9. Checklist

- [ ] `tui/celebration.go` created with `CelebrationModel` struct, constructor, and all 3 `tea.Model` methods
- [ ] `celebTickCmd()` fires `CelebrationTickMsg` at 80 ms intervals
- [ ] Frame 0 `View()` is deterministic (no card positions rendered)
- [ ] Animation cards advance each tick; wrap when they exceed `windowH`
- [ ] `AppModel.celebration` field added (`tui/app.go`)
- [ ] `GameWonMsg` handler initializes `CelebrationModel` and returns `Init()` cmd
- [ ] `ScreenWin` in `AppModel.Update()` delegates to `m.celebration.Update()`
- [ ] `ScreenWin` in `AppModel.View()` delegates to `m.celebration.View()`
- [ ] `WindowSizeMsg` forwarded to `CelebrationModel` when `screen == ScreenWin`
- [ ] `tui/celebration_test.go` created with all tests from §5.1
- [ ] Golden file `tui/testdata/TestCelebrationView.golden` generated and committed
- [ ] `go test ./tui/` passes with no failures
- [ ] `go test ./...` passes (no regressions in other packages)
