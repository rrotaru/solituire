# Agent C — T17: Auto-Complete + Auto-Move Implementation Plan

**Stage 8 | Phase 8 | Estimate: Medium | Dependencies: T12 ✓, T13 ✓**

---

## 1. Overview

T17 adds two related automation features to `tui/board.go`:

1. **Auto-Move** (§12.2): After every player action, if `Config.AutoMoveEnabled`, any top
   card on the waste or a tableau column that is "safe" to move to a foundation is moved
   automatically. "Safe" means both suits of the opposite color already have at least
   `card.Rank − 1` on their foundation (the standard Klondike safety rule). Cards are moved
   via the existing `buildMoveCmd` + `eng.Execute` path so scoring and undo both work.

2. **Auto-Complete** (§12.3): When `eng.IsAutoCompletable()` returns true (empty stock,
   no face-down tableau cards), the board enters an animation loop. A `tea.Tick` fires
   `AutoCompleteStepMsg` every 100 ms; each tick moves the lowest-rank card that can go
   to a foundation. The loop stops when `IsWon()` is true (emits `GameWonMsg`) or when
   the player presses any key (interrupts cleanly).

**Scope — only `tui/board.go` is modified in production code:**

| File | Change |
|---|---|
| `tui/board.go` | Add `autoCompleting` field; new methods + helpers; modify `Update` and `handleAction` |
| `tui/board_test.go` | Fix `testEngine.IsAutoCompletable`; add 5 new tests + `newNearWonBoard` helper |

No other files require changes. `AutoCompleteStepMsg` is already declared in
`tui/messages.go` (T9).

---

## 2. Audit: What Is Already Implemented

### 2.1 `tui/messages.go` — `AutoCompleteStepMsg` ✅

```go
type AutoCompleteStepMsg struct{} // triggers one foundation move per tick
```

Already declared. No changes needed.

### 2.2 `tui/board.go` — `winCmd()` ✅

```go
func (m BoardModel) winCmd() tea.Cmd {
    if m.eng.IsWon() {
        return func() tea.Msg { return GameWonMsg{} }
    }
    return nil
}
```

`handleAction` currently ends with `return m, m.winCmd()`. T17 inserts auto-move and
auto-complete checks immediately before this line.

### 2.3 `tui/board.go` — `buildMoveCmd` and `moveToFoundation` ✅

Both exist and correctly handle `FlipTableauCardCmd` wrapping for auto-flip. During
auto-complete all cards are face-up, so `buildMoveCmd` will always return a bare
`MoveToFoundationCmd` — the flip branch simply never triggers. Auto-move uses the same
path.

### 2.4 `engine/interfaces.go` — `IsAutoCompletable()` ✅

```go
IsAutoCompletable() bool
```

Already part of `GameEngine`. The `Game.IsAutoCompletable` implementation in
`engine/game.go:53` returns true when stock is empty and every tableau card is face-up.

### 2.5 `tui/board_test.go` — `testEngine.IsAutoCompletable` ⚠️ Stub

```go
func (e *testEngine) IsAutoCompletable() bool { return false }
```

This hard-coded stub prevents any auto-complete test from detecting the completable
state. It must be replaced with a real implementation (§6.1 below) before T17 tests
can pass.

---

## 3. Implementation: `BoardModel` State Extension

### 3.1 Add `autoCompleting bool` field

```go
// tui/board.go — BoardModel struct

type BoardModel struct {
    eng           engine.GameEngine
    cursor        Cursor
    rend          *renderer.Renderer
    cfg           *config.Config
    themes        *theme.ThemeRegistry
    width         int
    height        int
    autoCompleting bool // true while the auto-complete animation loop is running
}
```

`autoCompleting` starts as `false`. It is set to `true` when `IsAutoCompletable()`
becomes true after a player action, and cleared to `false` on a keypress or when
`IsWon()` is detected.

---

## 4. Implementation: Auto-Complete Tick Loop

### 4.1 `autoCompleteTickCmd` — 100 ms tick

Add at the bottom of `board.go` alongside `tickCmd`:

```go
// autoCompleteTickCmd returns a Cmd that fires AutoCompleteStepMsg after 100 ms.
func autoCompleteTickCmd() tea.Cmd {
    return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
        return AutoCompleteStepMsg{}
    })
}
```

### 4.2 `handleAutoCompleteStep` — one step of the animation loop

```go
// handleAutoCompleteStep executes one foundation move for the auto-complete loop,
// then schedules the next tick or ends the loop.
func (m BoardModel) handleAutoCompleteStep() (tea.Model, tea.Cmd) {
    if !m.autoCompleting {
        return m, nil
    }
    moved := m.doAutoCompleteStep()
    if !moved || m.eng.IsWon() {
        m.autoCompleting = false
        if m.eng.IsWon() {
            return m, func() tea.Msg { return GameWonMsg{} }
        }
        return m, nil
    }
    return m, autoCompleteTickCmd()
}
```

### 4.3 `doAutoCompleteStep` — lowest-rank foundation move

Design §12.3 specifies "the lowest-rank eligible card". We scan waste and all seven
tableau tops, find the minimum-rank card that any foundation accepts, then execute it.

```go
// doAutoCompleteStep finds the lowest-rank card that can move to a foundation and
// executes that move. Returns true if a move was made.
func (m *BoardModel) doAutoCompleteStep() bool {
    state := m.eng.State()

    var bestRank engine.Rank = engine.King + 1 // sentinel: no candidate yet
    var bestSrc engine.PileID
    bestFI := -1

    consider := func(card *engine.Card, src engine.PileID) {
        if card == nil || !card.FaceUp {
            return
        }
        for fi, f := range state.Foundations {
            if f.AcceptsCard(*card) && card.Rank < bestRank {
                bestRank = card.Rank
                bestSrc = src
                bestFI = fi
            }
        }
    }

    consider(state.Waste.TopCard(), engine.PileWaste)
    for col := 0; col < 7; col++ {
        consider(state.Tableau[col].TopCard(), engine.PileTableau0+engine.PileID(col))
    }

    if bestFI < 0 {
        return false
    }

    dest := engine.PileFoundation0 + engine.PileID(bestFI)
    cmd := m.buildMoveCmd(state, bestSrc, 1, dest)
    if cmd == nil {
        return false
    }
    _ = m.eng.Execute(cmd)
    m.clampCursor()
    return true
}
```

### 4.4 Wire `AutoCompleteStepMsg` and keypress interrupt into `Update`

```go
func (m BoardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.rend.SetSize(msg.Width, msg.Height)
        return m, nil

    case TickMsg:
        m.eng.State().ElapsedTime += time.Second
        return m, tickCmd()

    case AutoCompleteStepMsg:                 // ← NEW
        return m.handleAutoCompleteStep()     // ← NEW
    }

    // Any keypress interrupts an in-progress auto-complete.   // ← NEW
    if m.autoCompleting {                                       // ← NEW
        if _, ok := msg.(tea.KeyMsg); ok {                      // ← NEW
            m.autoCompleting = false                            // ← NEW
            return m, nil                                       // ← NEW
        }                                                       // ← NEW
    }                                                           // ← NEW

    action, payload := TranslateInput(msg)
    return m.handleAction(action, payload)
}
```

---

## 5. Implementation: Auto-Move on Player Action

### 5.1 `isSafeToAutoMove` — standard Klondike safety check

A card of rank R is safe to auto-move when every foundation pile of the opposite color
already has rank ≥ R−1. Aces and 2s are unconditionally safe (nothing can need them
on the tableau). Add as a package-level helper in `board.go`:

```go
// isSafeToAutoMove reports whether card can be safely auto-moved to its foundation.
// "Safe" means all opposite-colored foundations have rank >= card.Rank-1, ensuring
// nothing on the tableau can still need those cards as alternating-color targets.
func isSafeToAutoMove(card engine.Card, state *engine.GameState) bool {
    if card.Rank <= 2 {
        return true
    }
    oppositeColor := engine.Black
    if card.Color() == engine.Black {
        oppositeColor = engine.Red
    }
    found := false
    for _, f := range state.Foundations {
        s := f.Suit()
        if s == nil || s.Color() != oppositeColor {
            continue
        }
        found = true
        top := f.TopCard()
        oppRank := engine.Rank(0) // 0 = empty foundation (below Ace)
        if top != nil {
            oppRank = top.Rank
        }
        if oppRank < card.Rank-1 {
            return false
        }
    }
    return found // false if no opposite-color foundation has been started yet
}
```

**Safety rule walkthrough:**

| Card | Opposite color | Required min opp rank | Condition |
|---|---|---|---|
| A (rank 1) | — | always safe | unconditional |
| 2 (rank 2) | — | always safe | unconditional |
| 3♠ (Black, rank 3) | Red | ≥ 2 | both ♥ and ♦ foundations have ≥ 2 |
| 5♥ (Red, rank 5) | Black | ≥ 4 | both ♠ and ♣ foundations have ≥ 4 |

### 5.2 `autoMoveOneCard` — move one safe card, return whether anything moved

```go
// autoMoveOneCard checks waste and tableau tops for a single safe auto-move and
// executes the first one found. Returns true if a card was moved.
func (m *BoardModel) autoMoveOneCard() bool {
    state := m.eng.State()

    tryMove := func(card *engine.Card, src engine.PileID) bool {
        if card == nil || !card.FaceUp || !isSafeToAutoMove(*card, state) {
            return false
        }
        for fi, f := range state.Foundations {
            if f.AcceptsCard(*card) {
                dest := engine.PileFoundation0 + engine.PileID(fi)
                cmd := m.buildMoveCmd(state, src, 1, dest)
                if cmd != nil {
                    _ = m.eng.Execute(cmd)
                    m.clampCursor()
                    return true
                }
            }
        }
        return false
    }

    if tryMove(state.Waste.TopCard(), engine.PileWaste) {
        return true
    }
    for col := 0; col < 7; col++ {
        if tryMove(state.Tableau[col].TopCard(), engine.PileTableau0+engine.PileID(col)) {
            return true
        }
    }
    return false
}
```

### 5.3 `applyAutoMove` — loop until no more safe cards

```go
// applyAutoMove repeatedly moves safe cards to foundations while AutoMoveEnabled.
// Each call to autoMoveOneCard re-fetches state, so cascading moves are handled
// correctly (e.g., moving a 2 may make a 3 safe on the next pass).
func (m *BoardModel) applyAutoMove() {
    if !m.cfg.AutoMoveEnabled {
        return
    }
    for m.autoMoveOneCard() {
    }
}
```

---

## 6. Implementation: `handleAction` Integration

Replace the final line of `handleAction`:

```go
// BEFORE:
return m, m.winCmd()

// AFTER:
m.applyAutoMove()
if !m.autoCompleting && m.eng.IsAutoCompletable() {
    m.autoCompleting = true
}
if m.autoCompleting {
    return m, autoCompleteTickCmd()
}
return m, m.winCmd()
```

**Why this ordering is correct:**

1. `applyAutoMove()` runs first — it may move cards that bring the game closer to
   completable or won.
2. `IsAutoCompletable()` is checked after auto-move. If auto-move itself finishes the
   game, `IsWon()` will be true, `IsAutoCompletable()` will be false (no cards left to
   loop over), and `winCmd()` emits `GameWonMsg`.
3. Screen-changing actions (`ActionPause`, `ActionHelp`, `ActionQuit`, `ActionNewGame`,
   `ActionRestartDeal`, `ActionCycleTheme`, `ActionToggleAutoMove`) all return early
   inside the switch — they never reach this block. Auto-move does not run after them.

---

## 7. `tui/board_test.go` Changes

### 7.1 Fix `testEngine.IsAutoCompletable`

Replace the hard-coded stub with a real implementation matching `engine.Game`:

```go
// BEFORE:
func (e *testEngine) IsAutoCompletable() bool { return false }

// AFTER:
func (e *testEngine) IsAutoCompletable() bool {
    if !e.state.Stock.IsEmpty() {
        return false
    }
    for _, t := range e.state.Tableau {
        for _, c := range t.Cards {
            if !c.FaceUp {
                return false
            }
        }
    }
    return true
}
```

### 7.2 Helper: `newNearWonBoard`

Provides a board in a near-won state: all four foundations hold Ace through Queen, the
remaining four Kings sit face-up in tableau columns 0–3, stock and waste are empty.
`IsAutoCompletable()` is true; `IsWon()` is false.

```go
// newNearWonBoard creates a BoardModel where only 4 Kings remain (face-up in
// tableau[0..3]). All other 48 cards are on their respective foundations.
// IsAutoCompletable() == true, IsWon() == false.
func newNearWonBoard() (BoardModel, *testEngine) {
    state := &engine.GameState{
        Stock:     &engine.StockPile{},
        Waste:     &engine.WastePile{DrawCount: 1},
        DrawCount: 1,
    }
    suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
    for i := range state.Foundations {
        state.Foundations[i] = &engine.FoundationPile{}
    }
    for i := range state.Tableau {
        state.Tableau[i] = &engine.TableauPile{}
    }
    // Fill foundations Ace–Queen for each suit.
    for fi, suit := range suits {
        for r := engine.Ace; r <= engine.Queen; r++ {
            state.Foundations[fi].Cards = append(state.Foundations[fi].Cards,
                engine.Card{Suit: suit, Rank: r, FaceUp: true})
        }
    }
    // Place each King face-up in its own tableau column.
    for col, suit := range suits {
        state.Tableau[col].Cards = []engine.Card{
            {Suit: suit, Rank: engine.King, FaceUp: true},
        }
    }

    eng := &testEngine{state: state}
    rend := renderer.New(theme.Classic)
    rend.SetSize(80, 30)
    cfg := config.DefaultConfig()
    board := NewBoardModel(eng, rend, cfg)
    return board, eng
}
```

### 7.3 Test: `TestBoardAutoCompleteInterruptByKeypress`

Verifies that any keypress clears `autoCompleting` and returns a nil cmd.

```go
func TestBoardAutoCompleteInterruptByKeypress(t *testing.T) {
    board, _ := newNearWonBoard()
    board.autoCompleting = true

    updated, cmd := board.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    board = updated.(BoardModel)

    if board.autoCompleting {
        t.Error("keypress must clear autoCompleting")
    }
    if cmd != nil {
        t.Error("interrupt must return nil cmd (no further ticks)")
    }
}
```

### 7.4 Test: `TestBoardAutoCompleteStep_MovesToFoundation`

Verifies that one `AutoCompleteStepMsg` moves a card from tableau to foundation.

```go
func TestBoardAutoCompleteStep_MovesToFoundation(t *testing.T) {
    board, eng := newNearWonBoard()
    board.autoCompleting = true

    // Count foundation cards before.
    before := 0
    for _, f := range eng.State().Foundations {
        before += len(f.Cards)
    }

    updated, _ := board.Update(AutoCompleteStepMsg{})
    board = updated.(BoardModel)

    after := 0
    for _, f := range eng.State().Foundations {
        after += len(f.Cards)
    }

    if after != before+1 {
        t.Errorf("one AutoCompleteStepMsg must move exactly 1 card to foundation: before=%d after=%d", before, after)
    }
}
```

### 7.5 Test: `TestBoardAutoCompleteStep_EmitsGameWonMsg`

Verifies that the final step emits `GameWonMsg` and clears `autoCompleting`. Uses a
board where only one card remains.

```go
func TestBoardAutoCompleteStep_EmitsGameWonMsg(t *testing.T) {
    // Build a state with only the King of Spades remaining.
    state := &engine.GameState{
        Stock:     &engine.StockPile{},
        Waste:     &engine.WastePile{DrawCount: 1},
        DrawCount: 1,
    }
    suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
    for i := range state.Foundations {
        state.Foundations[i] = &engine.FoundationPile{}
    }
    for i := range state.Tableau {
        state.Tableau[i] = &engine.TableauPile{}
    }
    // All suits complete (Ace–King), except Spades stops at Queen.
    for fi, suit := range suits {
        limit := engine.King
        if suit == engine.Spades {
            limit = engine.Queen
        }
        for r := engine.Ace; r <= limit; r++ {
            state.Foundations[fi].Cards = append(state.Foundations[fi].Cards,
                engine.Card{Suit: suit, Rank: r, FaceUp: true})
        }
    }
    // The lone King of Spades sits face-up in tableau[0].
    state.Tableau[0].Cards = []engine.Card{
        {Suit: engine.Spades, Rank: engine.King, FaceUp: true},
    }

    eng := &testEngine{state: state}
    rend := renderer.New(theme.Classic)
    rend.SetSize(80, 30)
    board := NewBoardModel(eng, rend, config.DefaultConfig())
    board.autoCompleting = true

    _, cmd := board.Update(AutoCompleteStepMsg{})

    if cmd == nil {
        t.Fatal("final auto-complete step must return a non-nil cmd")
    }
    msg := cmd()
    if _, ok := msg.(GameWonMsg); !ok {
        t.Errorf("final step must emit GameWonMsg, got %T", msg)
    }
    if board.autoCompleting {
        t.Error("autoCompleting must be false after game is won")
    }
}
```

### 7.6 Test: `TestBoardAutoMove_MovesCardAfterAction`

Verifies that with `AutoMoveEnabled = true`, a safe tableau card is automatically moved
to its foundation after a player action.

```go
func TestBoardAutoMove_MovesCardAfterAction(t *testing.T) {
    // Build state: four Aces on foundations, 2 of Spades face-up in tableau[0].
    state := &engine.GameState{
        Stock:     &engine.StockPile{},
        Waste:     &engine.WastePile{DrawCount: 1},
        DrawCount: 1,
    }
    for i := range state.Foundations {
        state.Foundations[i] = &engine.FoundationPile{}
    }
    for i := range state.Tableau {
        state.Tableau[i] = &engine.TableauPile{}
    }
    suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
    // Place one Ace on each foundation.
    for fi, suit := range suits {
        state.Foundations[fi].Cards = []engine.Card{
            {Suit: suit, Rank: engine.Ace, FaceUp: true},
        }
    }
    // 2 of Spades face-up in tableau[0].
    state.Tableau[0].Cards = []engine.Card{
        {Suit: engine.Spades, Rank: engine.Two, FaceUp: true},
    }
    // 3 of Hearts face-up in tableau[1] (not yet safe: needs 2♠ on foundation first)
    state.Tableau[1].Cards = []engine.Card{
        {Suit: engine.Hearts, Rank: engine.Three, FaceUp: true},
    }

    eng := &testEngine{state: state}
    rend := renderer.New(theme.Classic)
    rend.SetSize(80, 30)
    cfg := config.DefaultConfig()
    cfg.AutoMoveEnabled = true
    board := NewBoardModel(eng, rend, cfg)

    // Record foundation card count before.
    before := 0
    for _, f := range eng.State().Foundations {
        before += len(f.Cards)
    }

    // Any player action triggers auto-move; use a cursor key (no-op for game state).
    board = sendKey(board, tea.KeyLeft)

    after := 0
    for _, f := range eng.State().Foundations {
        after += len(f.Cards)
    }
    if after != before+1 {
        t.Errorf("2♠ must be auto-moved after action: before=%d after=%d", before, after)
    }
    // 3♥ must NOT have been moved (Black foundations only have rank 2 now for Spades,
    // Clubs still has rank 1 — min Black rank is 1, need >= 2 for a rank-3 Red card).
    if eng.State().Tableau[1].IsEmpty() {
        t.Error("3♥ must not be auto-moved yet (opposite-color min rank insufficient)")
    }
}
```

### 7.7 Test: `TestBoardAutoMove_DisabledDoesNotMove`

Verifies that with `AutoMoveEnabled = false` (the default), safe cards stay put.

```go
func TestBoardAutoMove_DisabledDoesNotMove(t *testing.T) {
    // Same state as §7.6 but AutoMoveEnabled = false.
    state := &engine.GameState{
        Stock:     &engine.StockPile{},
        Waste:     &engine.WastePile{DrawCount: 1},
        DrawCount: 1,
    }
    for i := range state.Foundations {
        state.Foundations[i] = &engine.FoundationPile{}
    }
    for i := range state.Tableau {
        state.Tableau[i] = &engine.TableauPile{}
    }
    suits := []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs}
    for fi, suit := range suits {
        state.Foundations[fi].Cards = []engine.Card{
            {Suit: suit, Rank: engine.Ace, FaceUp: true},
        }
    }
    state.Tableau[0].Cards = []engine.Card{
        {Suit: engine.Spades, Rank: engine.Two, FaceUp: true},
    }

    eng := &testEngine{state: state}
    rend := renderer.New(theme.Classic)
    rend.SetSize(80, 30)
    cfg := config.DefaultConfig() // AutoMoveEnabled = false by default
    board := NewBoardModel(eng, rend, cfg)

    before := len(eng.State().Foundations[0].Cards) // Spades foundation has 1 card

    board = sendKey(board, tea.KeyLeft)

    after := len(eng.State().Foundations[0].Cards)
    if after != before {
        t.Errorf("auto-move disabled: Spades foundation must not grow: before=%d after=%d", before, after)
    }
}
```

---

## 8. Execution Order

1. **Fix `testEngine.IsAutoCompletable`** in `tui/board_test.go` (§7.1).
   Run `go test ./tui/ -run TestBoard` to confirm no regressions before writing new code.

2. **Add `autoCompleting` field** to `BoardModel` struct in `tui/board.go` (§3.1).

3. **Add `autoCompleteTickCmd`** to `tui/board.go` (§4.1).

4. **Add `doAutoCompleteStep`** to `tui/board.go` (§4.3).

5. **Add `handleAutoCompleteStep`** to `tui/board.go` (§4.2).

6. **Modify `Update`** in `tui/board.go`: add `AutoCompleteStepMsg` case and keypress
   interrupt block (§4.4).

7. **Add `isSafeToAutoMove`** to `tui/board.go` (§5.1).

8. **Add `autoMoveOneCard`** to `tui/board.go` (§5.2).

9. **Add `applyAutoMove`** to `tui/board.go` (§5.3).

10. **Modify `handleAction`** in `tui/board.go`: replace final `return m, m.winCmd()`
    with auto-move + auto-complete integration block (§6).

11. **Run `go build ./...`** — must compile cleanly.

12. **Add `newNearWonBoard` helper** to `tui/board_test.go` (§7.2).

13. **Add five new test functions** to `tui/board_test.go` (§7.3–7.7).

14. **Run `go test ./tui/ -run TestBoardAutoComplete`** — new auto-complete tests pass.

15. **Run `go test ./tui/ -run TestBoardAutoMove`** — new auto-move tests pass.

16. **Run `go test ./...`** — full suite green, no regressions.

17. Commit and push on `claude/agent-c-t17-plan-EjXoz`.

---

## 9. Edge Cases and Constraints

| Situation | Handling |
|---|---|
| Auto-move wins the game | `applyAutoMove` moves last card; `IsAutoCompletable()` returns false; `winCmd()` emits `GameWonMsg` |
| Auto-complete tick arrives after interrupt | `handleAutoCompleteStep` checks `autoCompleting`; if false returns `(m, nil)` — stale ticks are ignored |
| Player presses key during auto-complete | `Update` intercepts `tea.KeyMsg` before `TranslateInput`; clears `autoCompleting`, returns `nil` cmd |
| `isSafeToAutoMove` when opposite-color foundations are empty | `found` remains false → returns false (no Ace → unsafe) |
| `isSafeToAutoMove` for Aces and 2s | Returns true unconditionally (§5.1) |
| Auto-move cascade (moving 2♠ makes 3♥ safe) | `applyAutoMove` loops via `autoMoveOneCard` until no move is made in a full pass |
| `autoCompleting` set while board is mid-drag | Drag state and cursor are not modified; auto-complete runs after drag commits (or is cancelled by the next keypress) |
| `handleAction` early returns (screen changes, new game, etc.) | These return before the auto-move + auto-complete block — auto-move never fires on navigation actions |
| `doAutoCompleteStep` with no valid foundation move | Returns false; `handleAutoCompleteStep` clears `autoCompleting`, returns `nil` cmd — defensive exit |
| Multiple Rank ties in `doAutoCompleteStep` | `consider` selects the first (waste before tableau, tableau[0] before tableau[6]) — deterministic |

---

## 10. File Checklist

```
tui/
├── board.go       ← MODIFY  (autoCompleting field; 6 new functions/methods; Update + handleAction edits)
└── board_test.go  ← MODIFY  (fix IsAutoCompletable stub; newNearWonBoard helper; 5 new test functions)
```

No other files are affected. All production changes are confined to `tui/board.go`.

---

## 11. Pre-existing Test Compatibility

Fixing `testEngine.IsAutoCompletable` from a hard-coded `false` to a real
implementation is the only change to existing test infrastructure. Because no existing
test sets up a state where stock is empty and all tableau cards are face-up (the seed-42
deal always has face-down cards at the start), every existing test's `IsAutoCompletable`
call will still return `false`. No existing test behaviour changes.

The new `handleAction` tail block runs after every action that falls through the switch.
For all existing tests the engine state after each action is not auto-completable and no
safe auto-move cards exist (deck is not in a near-won position), so `applyAutoMove` is a
no-op and `autoCompleting` stays false. `winCmd()` is reached as before.

---

## 12. T18 Handoff Note

After T17 merges:

- `GameWonMsg` is now emitted both by the existing `winCmd()` path (manual last move)
  and by `handleAutoCompleteStep` (animated last step).
- T18 (Win Celebration) consumes `GameWonMsg` in `AppModel.Update` to transition to
  `ScreenWin`. No changes to `board.go` are needed for T18.
- `tui/celebration.go` uses `CelebrationTickMsg` (already declared in `messages.go`),
  which is a different tick from `AutoCompleteStepMsg` — no conflict.
