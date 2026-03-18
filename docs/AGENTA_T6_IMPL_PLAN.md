# Agent A — Phase T6 Implementation Plan
## Commands and History

**Branch**: `claude/agent-a-t6-plan-MhN43`
**Dependencies**: T2 complete ✓, T3 complete ✓
**Blocks**: Agent C T11 (Board Model) — needs `Command` interface, all concrete command types, and `History`

---

## Overview

T6 replaces all `panic("not implemented")` stubs in `engine/command.go` and `engine/history.go`. No new exported types are introduced. The `Command` and `Scorer` interfaces in `engine/interfaces.go` are already correct — do **not** modify them.

Two new test files are created: `engine/command_test.go` and `engine/history_test.go`.

### Files changed

| File | Action |
|------|--------|
| `engine/command.go` | Full implementation (6 command types) |
| `engine/history.go` | Full implementation |
| `engine/command_test.go` | Create — Execute/Undo round-trips per command type |
| `engine/history_test.go` | Create — Push/Undo/Redo ordering, error cases |

`engine/interfaces.go` — **do not touch** (already correct).

---

## Data Model Decisions

### Internal state fields

Each command must store enough state on `Execute` to reverse itself on `Undo`. The table below shows what each command needs beyond its existing public fields:

| Command | New unexported fields | Reason |
|---|---|---|
| `MoveCardCmd` | none | `CardCount` + pile types are sufficient; Undo just reverses the slice ops |
| `MoveToFoundationCmd` | none | Undo reads `foundation.TopCard()` to put it back |
| `FlipStockCmd` | `flippedCount int` | Stock may have fewer cards than `DrawCount`; Undo must know exactly how many to take back from waste |
| `RecycleStockCmd` | none | Undo reverses all stock cards back to waste; count is `len(state.Stock.Cards)` at Undo time |
| `FlipTableauCardCmd` | none | Always operates on `state.Tableau[ColumnIdx].Cards[last]`; Undo just re-flips it |
| `CompoundCmd` | none | Sub-commands carry their own state |

### Stock card ordering

`StockPile.Cards[len-1]` is the **top** of the stock (drawn first). `WastePile.Cards[len-1]` is the **top** of the waste (playable).

**FlipStockCmd**: Drawing n cards from stock reverses their order onto waste — the deepest of the drawn cards (index `len-n`) becomes the new waste top. Concretely:

```
stock = [..., A, B, C]   (C = top, drawn first)
drawn from stock indices: [len-3, len-2, len-1] = [A, B, C]
appended to waste in REVERSE order: waste gains C, B, A
→ waste top = A (was deepest of drawn)
```

**Undo of FlipStockCmd**: take the last `flippedCount` cards from waste, reverse them, flip face-down, append to stock.

**RecycleStockCmd**: all waste cards go back to stock reversed and face-down:

```
waste = [W0, W1, ..., Wn]   (Wn = playable top)
stock after recycle = [Wn, Wn-1, ..., W0]   (W0 = new stock top → drawn first on next pass)
```

**Undo of RecycleStockCmd**: all stock cards reversed back to waste face-up (stock becomes empty again).

---

## Step-by-Step Implementation

### Step 1 — `MoveCardCmd`

Covers: Tableau→Tableau, Waste→Tableau, Foundation→Tableau.

**Add to `engine/command.go`** (replace existing stubs):

```go
func (c *MoveCardCmd) Execute(state *GameState) error {
    if err := ValidateMove(state, Move{From: c.From, To: c.To, CardCount: c.CardCount}); err != nil {
        return err
    }

    // Extract cards from source.
    var cards []Card
    switch {
    case c.From == PileWaste:
        w := state.Waste
        cards = []Card{w.Cards[len(w.Cards)-1]}
        state.Waste.Cards = w.Cards[:len(w.Cards)-1]
    case isFoundationPile(c.From):
        fi := foundationIndex(c.From)
        f := state.Foundations[fi]
        cards = []Card{f.Cards[len(f.Cards)-1]}
        state.Foundations[fi].Cards = f.Cards[:len(f.Cards)-1]
    case isTableauPile(c.From):
        ti := tableauIndex(c.From)
        t := state.Tableau[ti]
        // The moved sub-sequence is the last CardCount cards.
        split := len(t.Cards) - c.CardCount
        cards = make([]Card, c.CardCount)
        copy(cards, t.Cards[split:])
        state.Tableau[ti].Cards = t.Cards[:split]
    }

    // Append to destination tableau.
    ti := tableauIndex(c.To)
    state.Tableau[ti].Cards = append(state.Tableau[ti].Cards, cards...)
    return nil
}

func (c *MoveCardCmd) Undo(state *GameState) error {
    ti := tableauIndex(c.To)
    t := state.Tableau[ti]
    split := len(t.Cards) - c.CardCount
    cards := make([]Card, c.CardCount)
    copy(cards, t.Cards[split:])
    state.Tableau[ti].Cards = t.Cards[:split]

    switch {
    case c.From == PileWaste:
        state.Waste.Cards = append(state.Waste.Cards, cards...)
    case isFoundationPile(c.From):
        fi := foundationIndex(c.From)
        state.Foundations[fi].Cards = append(state.Foundations[fi].Cards, cards...)
    case isTableauPile(c.From):
        ti2 := tableauIndex(c.From)
        state.Tableau[ti2].Cards = append(state.Tableau[ti2].Cards, cards...)
    }
    return nil
}

func (c *MoveCardCmd) Description() string {
    return fmt.Sprintf("Move %d card(s) from pile %d to pile %d", c.CardCount, c.From, c.To)
}
```

---

### Step 2 — `MoveToFoundationCmd`

Covers: Waste→Foundation, Tableau→Foundation. Used by auto-move / auto-complete logic.

Note: `ValidateMove` uses `Move{From, To: PileFoundation0+PileID(FoundationIdx), CardCount: 1}`.

```go
func (c *MoveToFoundationCmd) Execute(state *GameState) error {
    to := PileFoundation0 + PileID(c.FoundationIdx)
    if err := ValidateMove(state, Move{From: c.From, To: to, CardCount: 1}); err != nil {
        return err
    }

    var card Card
    switch {
    case c.From == PileWaste:
        w := state.Waste
        card = w.Cards[len(w.Cards)-1]
        state.Waste.Cards = w.Cards[:len(w.Cards)-1]
    case isTableauPile(c.From):
        ti := tableauIndex(c.From)
        t := state.Tableau[ti]
        card = t.Cards[len(t.Cards)-1]
        state.Tableau[ti].Cards = t.Cards[:len(t.Cards)-1]
    }

    card.FaceUp = true
    state.Foundations[c.FoundationIdx].Cards = append(
        state.Foundations[c.FoundationIdx].Cards, card)
    return nil
}

func (c *MoveToFoundationCmd) Undo(state *GameState) error {
    f := state.Foundations[c.FoundationIdx]
    card := f.Cards[len(f.Cards)-1]
    state.Foundations[c.FoundationIdx].Cards = f.Cards[:len(f.Cards)-1]

    switch {
    case c.From == PileWaste:
        state.Waste.Cards = append(state.Waste.Cards, card)
    case isTableauPile(c.From):
        ti := tableauIndex(c.From)
        state.Tableau[ti].Cards = append(state.Tableau[ti].Cards, card)
    }
    return nil
}

func (c *MoveToFoundationCmd) Description() string {
    return fmt.Sprintf("Move top card from pile %d to foundation %d", c.From, c.FoundationIdx)
}
```

---

### Step 3 — `FlipStockCmd`

Add the unexported `flippedCount` field to the struct (this is an additive change; existing construction `FlipStockCmd{}` still compiles):

```go
type FlipStockCmd struct {
    flippedCount int // set during Execute; used by Undo
}
```

Implementation:

```go
func (c *FlipStockCmd) Execute(state *GameState) error {
    if err := ValidateMove(state, Move{From: PileStock, To: PileWaste}); err != nil {
        return err
    }
    if state.Stock.IsEmpty() {
        // Recycle is handled by RecycleStockCmd; FlipStockCmd is only for normal flips.
        return errors.New("use RecycleStockCmd to recycle waste to stock")
    }

    n := state.DrawCount
    if n > len(state.Stock.Cards) {
        n = len(state.Stock.Cards)
    }
    c.flippedCount = n

    // Take n cards from top of stock (last n elements).
    drawn := make([]Card, n)
    copy(drawn, state.Stock.Cards[len(state.Stock.Cards)-n:])
    state.Stock.Cards = state.Stock.Cards[:len(state.Stock.Cards)-n]

    // Reverse drawn before appending to waste: the deepest drawn card
    // (index 0 of drawn) becomes the new waste top (playable).
    for i, j := 0, len(drawn)-1; i < j; i, j = i+1, j-1 {
        drawn[i], drawn[j] = drawn[j], drawn[i]
    }
    for i := range drawn {
        drawn[i].FaceUp = true
    }
    state.Waste.Cards = append(state.Waste.Cards, drawn...)
    return nil
}

func (c *FlipStockCmd) Undo(state *GameState) error {
    if c.flippedCount == 0 {
        return errors.New("FlipStockCmd: Execute was not called")
    }
    n := c.flippedCount

    // Take n cards from top of waste.
    w := state.Waste
    taken := make([]Card, n)
    copy(taken, w.Cards[len(w.Cards)-n:])
    state.Waste.Cards = w.Cards[:len(w.Cards)-n]

    // Reverse back to restore original stock order.
    for i, j := 0, len(taken)-1; i < j; i, j = i+1, j-1 {
        taken[i], taken[j] = taken[j], taken[i]
    }
    for i := range taken {
        taken[i].FaceUp = false
    }
    state.Stock.Cards = append(state.Stock.Cards, taken...)
    c.flippedCount = 0
    return nil
}

func (c *FlipStockCmd) Description() string {
    return "Flip stock cards to waste"
}
```

**Note on FlipStockCmd vs RecycleStockCmd split**: The validator `validateStockFlip` returns nil for both "stock non-empty flip" and "stock empty + waste non-empty recycle." Commands must self-distinguish. `FlipStockCmd.Execute` returns an error if stock is empty, deferring recycle to `RecycleStockCmd`. The TUI/engine layer is responsible for constructing the correct command type based on `state.Stock.IsEmpty()`.

---

### Step 4 — `RecycleStockCmd`

```go
func (c *RecycleStockCmd) Execute(state *GameState) error {
    if !state.Stock.IsEmpty() {
        return errors.New("cannot recycle: stock is not empty")
    }
    if state.Waste.IsEmpty() {
        return errors.New("cannot recycle: waste is empty")
    }

    w := state.Waste
    // Reverse waste into stock, face-down.
    n := len(w.Cards)
    stock := make([]Card, n)
    for i, card := range w.Cards {
        card.FaceUp = false
        stock[n-1-i] = card // reverse
    }
    state.Stock.Cards = stock
    state.Waste.Cards = state.Waste.Cards[:0]
    return nil
}

func (c *RecycleStockCmd) Undo(state *GameState) error {
    // Reverse all stock cards back to waste, face-up.
    s := state.Stock
    n := len(s.Cards)
    waste := make([]Card, n)
    for i, card := range s.Cards {
        card.FaceUp = true
        waste[n-1-i] = card // reverse
    }
    state.Waste.Cards = append(state.Waste.Cards, waste...)
    state.Stock.Cards = state.Stock.Cards[:0]
    return nil
}

func (c *RecycleStockCmd) Description() string {
    return "Recycle waste to stock"
}
```

---

### Step 5 — `FlipTableauCardCmd`

```go
func (c *FlipTableauCardCmd) Execute(state *GameState) error {
    t := state.Tableau[c.ColumnIdx]
    if t.IsEmpty() {
        return errors.New("FlipTableauCardCmd: tableau column is empty")
    }
    top := &t.Cards[len(t.Cards)-1]
    if top.FaceUp {
        return errors.New("FlipTableauCardCmd: top card is already face-up")
    }
    top.FaceUp = true
    return nil
}

func (c *FlipTableauCardCmd) Undo(state *GameState) error {
    t := state.Tableau[c.ColumnIdx]
    if t.IsEmpty() {
        return errors.New("FlipTableauCardCmd Undo: tableau column is empty")
    }
    top := &t.Cards[len(t.Cards)-1]
    if !top.FaceUp {
        return errors.New("FlipTableauCardCmd Undo: top card is already face-down")
    }
    top.FaceUp = false
    return nil
}

func (c *FlipTableauCardCmd) Description() string {
    return fmt.Sprintf("Flip top card of tableau column %d face-up", c.ColumnIdx)
}
```

---

### Step 6 — `CompoundCmd`

`CompoundCmd` uses `Cmds []Command` (existing stub field name). Execute rolls back partial progress on failure. Undo runs sub-commands in reverse.

```go
func (c *CompoundCmd) Execute(state *GameState) error {
    for i, cmd := range c.Cmds {
        if err := cmd.Execute(state); err != nil {
            // Rollback previously executed sub-commands in reverse order.
            for j := i - 1; j >= 0; j-- {
                _ = c.Cmds[j].Undo(state) // best-effort rollback
            }
            return err
        }
    }
    return nil
}

func (c *CompoundCmd) Undo(state *GameState) error {
    for i := len(c.Cmds) - 1; i >= 0; i-- {
        if err := c.Cmds[i].Undo(state); err != nil {
            return err
        }
    }
    return nil
}

func (c *CompoundCmd) Description() string {
    parts := make([]string, len(c.Cmds))
    for i, cmd := range c.Cmds {
        parts[i] = cmd.Description()
    }
    return strings.Join(parts, " + ")
}
```

Add `"strings"` to the import block in `command.go`.

---

### Step 7 — `History`

```go
func (h *History) Push(cmd Command) {
    h.undoStack = append(h.undoStack, cmd)
    h.redoStack = h.redoStack[:0] // clear redo stack
}

func (h *History) Undo(s *GameState) error {
    if len(h.undoStack) == 0 {
        return errors.New("nothing to undo")
    }
    cmd := h.undoStack[len(h.undoStack)-1]
    h.undoStack = h.undoStack[:len(h.undoStack)-1]
    if err := cmd.Undo(s); err != nil {
        return err
    }
    h.redoStack = append(h.redoStack, cmd)
    return nil
}

func (h *History) Redo(s *GameState) error {
    if len(h.redoStack) == 0 {
        return errors.New("nothing to redo")
    }
    cmd := h.redoStack[len(h.redoStack)-1]
    h.redoStack = h.redoStack[:len(h.redoStack)-1]
    if err := cmd.Execute(s); err != nil {
        return err
    }
    h.undoStack = append(h.undoStack, cmd)
    return nil
}

func (h *History) CanUndo() bool { return len(h.undoStack) > 0 }
func (h *History) CanRedo() bool { return len(h.redoStack) > 0 }

func (h *History) Clear() {
    h.undoStack = nil
    h.redoStack = nil
}
```

Add `"errors"` to `history.go` imports.

---

## Imports Required

**`engine/command.go`**:
```go
import (
    "errors"
    "fmt"
    "strings"
)
```

**`engine/history.go`**:
```go
import "errors"
```

---

## Step 8 — `engine/command_test.go`

Use the same `buildState()` / `card()` / `faceDown()` helpers established in T3's `rules_test.go`. Alternatively define them in a `testhelpers_test.go` if not already shared.

### Test structure per command

Each test follows the pattern:
1. Build a known state
2. Execute the command
3. Assert post-execute state (card is where expected, pile sizes correct)
4. Undo the command
5. Assert state equals pre-execute state (compare pile slices)

### `TestMoveCardCmd_TableauToTableau`

Setup: T0 = [K♠(face-down), Q♥(face-up)], T1 = [K♣(face-up)]
Cmd: `MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1}`
Post-execute: T0 = [K♠(face-down)], T1 = [K♣, Q♥]
Post-undo: restored to original

### `TestMoveCardCmd_MultiCard`

Setup: T0 = [A♠(face-down), Q♥, J♠(face-up)], T1 = [K♣(face-up)]
Cmd: `MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 2}`
Post-execute: T0 = [A♠(face-down)], T1 = [K♣, Q♥, J♠]
Post-undo: restored

### `TestMoveCardCmd_WasteToTableau`

Setup: Waste = [9♠(face-up)], T0 = [10♥(face-up)]
Cmd: `MoveCardCmd{From: PileWaste, To: PileTableau0, CardCount: 1}`
Post-execute: Waste = [], T0 = [10♥, 9♠]
Post-undo: restored

### `TestMoveCardCmd_FoundationToTableau`

Setup: F0 = [A♠, 2♠(face-up)], T0 = [3♥(face-up)]
Cmd: `MoveCardCmd{From: PileFoundation0, To: PileTableau0, CardCount: 1}`
Post-execute: F0 = [A♠], T0 = [3♥, 2♠]
Post-undo: restored

### `TestMoveCardCmd_InvalidMove`

Setup: T0 = [7♥(face-up)], T1 = [7♠(face-up)] (same rank, wrong pairing)
Cmd: `MoveCardCmd{From: PileTableau0, To: PileTableau1, CardCount: 1}`
Execute returns non-nil error; state unchanged.

### `TestMoveToFoundationCmd_FromWaste`

Setup: Waste = [A♥(face-up)], F0 = []
Cmd: `MoveToFoundationCmd{From: PileWaste, FoundationIdx: 0}`
Post-execute: Waste = [], F0 = [A♥]
Post-undo: restored

### `TestMoveToFoundationCmd_FromTableau`

Setup: T2 = [K♠(face-down), A♦(face-up)], F2 = []
Cmd: `MoveToFoundationCmd{From: PileTableau2, FoundationIdx: 2}`
Post-execute: T2 = [K♠(face-down)], F2 = [A♦]
Post-undo: restored

### `TestFlipStockCmd_DrawOne`

Setup: Stock = [5♣, 3♥, A♠] (A♠ = top), DrawCount = 1
Execute: Stock = [5♣, 3♥], Waste = [A♠(face-up)]
Undo: restored

### `TestFlipStockCmd_DrawThree`

Setup: Stock = [5♣, 3♥, A♠, K♦, Q♣] (Q♣ = top), DrawCount = 3
Execute: Stock = [5♣, 3♥], Waste = [Q♣, K♦, A♠] (A♠ = top/playable)
Undo: restored to original 5-card stock, empty waste

### `TestFlipStockCmd_DrawThreeFewer`

Setup: Stock = [5♣, A♠] (A♠ = top), DrawCount = 3
Execute draws only 2: Stock = [], Waste = [A♠, 5♣] (5♣ = top/playable)
Undo: restored

### `TestRecycleStockCmd`

Setup: Stock = [], Waste = [A♠, 2♠, 3♠] (3♠ = top/playable)
Execute: Stock = [3♠, 2♠, A♠] (A♠ = top/drawn first), Waste = []
Undo: Stock = [], Waste = [A♠, 2♠, 3♠] (3♠ = top)

### `TestFlipTableauCardCmd`

Setup: T3 = [K♠(face-down), Q♥(face-down)]
Execute: T3 = [K♠(face-down), Q♥(face-up)]
Undo: T3 = [K♠(face-down), Q♥(face-down)]

### `TestFlipTableauCardCmd_AlreadyFaceUp`

Setup: T3 = [Q♥(face-up)]
Execute returns error; state unchanged.

### `TestCompoundCmd_MoveAndFlip`

This is the critical test: move a card off a tableau exposing a face-down card, then auto-flip.

Setup: T0 = [K♠(face-down), 9♥(face-up)], T1 = [10♣(face-up)]
Compound: [MoveCardCmd{T0→T1, 1}, FlipTableauCardCmd{ColumnIdx: 0}]
Post-execute: T0 = [K♠(face-up)], T1 = [10♣, 9♥]
Undo: T0 = [K♠(face-down), 9♥(face-up)], T1 = [10♣]  ← both actions reversed atomically

### `TestCompoundCmd_RollbackOnPartialFailure`

Compound: [ValidCmd (succeeds), InvalidCmd (fails)]
After Execute returns error, state must equal pre-execute state (ValidCmd was rolled back).

---

## Step 9 — `engine/history_test.go`

### `TestHistory_PushAndUndo`

1. Build state, execute `FlipTableauCardCmd` manually, push to history
2. `CanUndo()` == true, `CanRedo()` == false
3. Undo → state restored, `CanUndo()` == false, `CanRedo()` == true

### `TestHistory_Redo`

Continuing from above: Redo → state re-applied, `CanRedo()` == false, `CanUndo()` == true

### `TestHistory_UndoOnEmpty`

Empty `History` — `Undo()` returns non-nil error

### `TestHistory_RedoOnEmpty`

Empty `History` — `Redo()` returns non-nil error

### `TestHistory_PushClearsRedo`

1. Push cmd A, undo it (redoStack has A)
2. Push cmd B → redoStack cleared
3. `CanRedo()` == false

### `TestHistory_MultipleUndoRedoCycles`

Push 3 commands (3 `FlipTableauCardCmd` on different columns). Undo all 3 (verify states). Redo all 3 (verify states). Undo all 3 again (verify same initial state). Asserts correct LIFO ordering.

### `TestHistory_Clear`

Push 2 commands, call `Clear()`, assert `CanUndo()` == false and `CanRedo()` == false.

---

## File Checklist

| File | Action |
|------|--------|
| `engine/command.go` | Replace all 18 `panic` stubs; add `flippedCount` to `FlipStockCmd`; add imports |
| `engine/history.go` | Replace 6 `panic` stubs; add `errors` import |
| `engine/command_test.go` | Create with ≥13 test functions |
| `engine/history_test.go` | Create with ≥7 test functions |

---

## Verification Gate

```bash
go build ./...                # no new types — must compile cleanly
go test ./engine/...          # all existing tests + T6 tests must pass
go vet ./engine/...           # no vet errors
```

No other packages are modified, so no cross-package compilation issues.

---

## Handoff Contract

When T6 is complete, unblock:

**Agent C — T11 (Board Model)**: exports now fully implemented:
- `engine.Command` interface (Execute, Undo, Description)
- `engine.MoveCardCmd`, `engine.MoveToFoundationCmd`, `engine.FlipStockCmd`, `engine.RecycleStockCmd`, `engine.FlipTableauCardCmd`, `engine.CompoundCmd`
- `engine.History` (Push, Undo, Redo, CanUndo, CanRedo, Clear)

**T7 (Scoring Engine)**: can start immediately — `Scorer` interface in `interfaces.go` is already in place; `StandardScorer` in `scoring.go` does not depend on T6.

**T10 (Hint Engine)**: can start immediately — only depends on T2/T3.

**T12 (GameEngine Wiring)**: depends on T6, T7, T10.
