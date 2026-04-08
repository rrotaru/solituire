# Agent A — Phase T12 Implementation Plan
## GameEngine Wiring

**Branch**: `claude/agent-a-phase-t12-plan-vSg2R`
**Dependencies**: T6 ✓, T7 ✓, T10 ✓ (all complete — `command.go`, `history.go`, `scoring.go`, `hint.go` are fully implemented)
**Blocks**: Agent C T13 (App Shell) — needs `engine.GameEngine` interface + `engine.NewGame` constructor

---

## Overview

T12 introduces the `Game` struct in `engine/game.go`, which implements the `GameEngine` interface
defined in `engine/interfaces.go`. `Game` wires together `GameState`, `History`, and
`StandardScorer` into the single entry point the TUI uses for all game operations.

Two new test files are created: `engine/game_test.go` and `engine/integration_test.go`.

### Files changed

| File | Action |
|------|--------|
| `engine/game.go` | Append `Game` struct + all `GameEngine` method implementations |
| `engine/game_test.go` | Create — unit tests for every `Game` method |
| `engine/integration_test.go` | Create — scripted playthrough with seed 42 |

No other files are modified.

---

## Codebase State Entering T12

| File | Status |
|------|--------|
| `engine/interfaces.go` | Complete — `GameEngine`, `Command`, `Scorer` interfaces all correct |
| `engine/game.go` | Partial — has `GameState`, `PileID` constants, `deepCopyState`; **no `Game` struct** |
| `engine/engine.go` | Empty (`package engine` only); not used |
| `engine/command.go` | Complete — all six command types |
| `engine/history.go` | Complete — score-snapshot undo/redo (`Push(cmd, scoreBefore, scoreAfter)`) |
| `engine/scoring.go` | Complete — `StandardScorer` |
| `engine/hint.go` | Complete — `FindHints`, priority constants |
| `engine/rules.go` | Complete — `ValidateMove`, `ValidMoves` |
| `engine/deck.go` | Complete — `NewDeck`, `Shuffle`, `Deal` |

**Critical note**: `History.Push` takes `scoreBefore, scoreAfter int` in addition to the command.
The `Game.Execute` method must pass both values so undo/redo snapshot-restores score correctly
(prevents the "undo inflates score" bug documented in `history_test.go`).

---

## `Game` Struct

```go
// Game implements GameEngine. It wires together GameState, History, and StandardScorer.
// Create instances via NewGame — do not construct directly.
type Game struct {
    state   *GameState
    history History
    scorer  StandardScorer
}

// NewGame creates and returns a fully initialized Game.
// It exists as a package-level constructor so the TUI can call engine.NewGame(seed, drawCount).
func NewGame(seed int64, drawCount int) *Game {
    g := &Game{}
    g.NewGame(seed, drawCount)
    return g
}
```

`engine.go` remains empty — the stub constructor above lives in `game.go` alongside the struct.

---

## Method Implementations

### State-query methods (trivial)

```go
func (g *Game) State() *GameState        { return g.state }
func (g *Game) Score() int               { return g.state.Score }
func (g *Game) MoveCount() int           { return g.state.MoveCount }
func (g *Game) Seed() int64              { return g.state.Seed }
func (g *Game) CanUndo() bool            { return g.history.CanUndo() }
func (g *Game) CanRedo() bool            { return g.history.CanRedo() }
```

---

### `IsWon() bool`

All four foundation piles each contain exactly 13 cards.

```go
func (g *Game) IsWon() bool {
    for _, f := range g.state.Foundations {
        if len(f.Cards) != 13 {
            return false
        }
    }
    return true
}
```

---

### `IsAutoCompletable() bool`

Every remaining unplayed card is face-up (i.e., no hidden cards remain). This requires:
- Stock is empty (stock cards are always face-down).
- No face-down cards in any tableau column.

Waste and Foundation cards are always face-up by invariant, so they need no check.

```go
func (g *Game) IsAutoCompletable() bool {
    if !g.state.Stock.IsEmpty() {
        return false
    }
    for _, t := range g.state.Tableau {
        for _, c := range t.Cards {
            if !c.FaceUp {
                return false
            }
        }
    }
    return true
}
```

---

### `ValidMoves() []Move`

Calls the package-level `ValidMoves` function from `rules.go` directly (not `FindHints`).
This avoids the `Hint` → `Move` conversion loss of `CardCount`, which `FindHints` does not
preserve in the `Hint` struct.

```go
func (g *Game) ValidMoves() []Move {
    return ValidMoves(g.state)
}
```

---

### `IsValidMove(move Move) bool`

```go
func (g *Game) IsValidMove(move Move) bool {
    return ValidateMove(g.state, move) == nil
}
```

---

### `NewGame(seed int64, drawCount int)`

```go
func (g *Game) NewGame(seed int64, drawCount int) {
    deck := Shuffle(NewDeck(), seed)
    g.state = Deal(deck, drawCount)
    g.state.Seed = seed
    g.history.Clear()
}
```

`Deal` does not set `Seed` (it has no knowledge of it), so the caller sets it afterward.

---

### `RestartDeal()`

Re-deals using the stored seed and draw count, then clears history.

```go
func (g *Game) RestartDeal() {
    g.NewGame(g.state.Seed, g.state.DrawCount)
}
```

---

### `Execute(cmd Command) error`  ← most complex method

The execute flow (from Design §6.2):

1. Save `scoreBefore`.
2. Call `cmd.Execute(g.state)`. On error: return error immediately (no state mutation recorded).
3. Detect tableau columns that now have a face-down top card (need auto-flip).
4. Execute a `FlipTableauCardCmd` for each such column.
5. If any flips occurred, wrap original cmd + flip cmds into `CompoundCmd` as the recorded command.
6. Compute the total score delta for the recorded command.
7. Clamp new score: `max(0, scoreBefore + delta)`.
8. Update `state.Score`, increment `state.MoveCount`.
9. Push recorded command to history with `scoreBefore` / `scoreAfter`.

```go
func (g *Game) Execute(cmd Command) error {
    scoreBefore := g.state.Score

    if err := cmd.Execute(g.state); err != nil {
        return err
    }

    // Auto-flip: find tableau columns that now expose a face-down top card.
    var flips []Command
    for col := 0; col < 7; col++ {
        t := g.state.Tableau[col]
        if !t.IsEmpty() && !t.Cards[len(t.Cards)-1].FaceUp {
            flip := &FlipTableauCardCmd{ColumnIdx: col}
            _ = flip.Execute(g.state) // always succeeds: top card exists and is face-down
            flips = append(flips, flip)
        }
    }

    // Build the command to record in history.
    var recorded Command
    if len(flips) == 0 {
        recorded = cmd
    } else {
        all := make([]Command, 0, 1+len(flips))
        all = append(all, cmd)
        all = append(all, flips...)
        recorded = &CompoundCmd{Cmds: all}
    }

    // Calculate score delta and update state.
    delta := g.scoreForCmd(recorded)
    scoreAfter := scoreBefore + delta
    if scoreAfter < 0 {
        scoreAfter = 0
    }
    g.state.Score = scoreAfter
    g.state.MoveCount++

    g.history.Push(recorded, scoreBefore, scoreAfter)
    return nil
}
```

**Why auto-flip runs inside Execute (not in TUI)**: The compound wrapping must happen before
`history.Push` so that a single Ctrl+Z reverses both the card move and the auto-flip atomically.
The TUI should never need to issue a separate flip command.

---

### `scoreForCmd(cmd Command) int`  ← unexported helper

Type-switches on the concrete command type to call the appropriate `Scorer` method.

```go
func (g *Game) scoreForCmd(cmd Command) int {
    switch c := cmd.(type) {
    case *MoveCardCmd:
        return g.scorer.OnMove(Move{From: c.From, To: c.To, CardCount: c.CardCount}, g.state)
    case *MoveToFoundationCmd:
        to := PileFoundation0 + PileID(c.FoundationIdx)
        return g.scorer.OnMove(Move{From: c.From, To: to, CardCount: 1}, g.state)
    case *FlipTableauCardCmd:
        return g.scorer.OnFlipTableau()
    case *RecycleStockCmd:
        return g.scorer.OnRecycleStock()
    case *FlipStockCmd:
        return 0
    case *CompoundCmd:
        total := 0
        for _, sub := range c.Cmds {
            total += g.scoreForCmd(sub)
        }
        return total
    }
    return 0
}
```

`StandardScorer.OnMove` ignores the `state` parameter (only inspects `From`/`To`), so calling
it after execution (when state has changed) is safe.

---

### `Undo() error` and `Redo() error`

`History.Undo` and `History.Redo` already handle score restoration via snapshots. `Game` just
delegates. `MoveCount` is intentionally NOT decremented on undo (standard Klondike behavior).

```go
func (g *Game) Undo() error { return g.history.Undo(g.state) }
func (g *Game) Redo() error { return g.history.Redo(g.state) }
```

---

## `engine/game_test.go`

All tests live in `package engine` (same package, no import of engine itself).
They reuse the `buildState`, `faceUpCard`, `faceDownCard` helpers from `rules_test.go`.

### `TestGame_NewGame`

Verify that `NewGame(42, 1)` produces a correctly dealt state:
- Tableau columns 0-6 have 1-7 cards respectively.
- Each column's top card is face-up; all others are face-down.
- Stock has 24 cards.
- Waste, Foundations are empty.
- `g.Score() == 0`, `g.MoveCount() == 0`, `g.Seed() == 42`.
- `g.CanUndo() == false`, `g.CanRedo() == false`.

### `TestGame_IsWon_False`

Freshly dealt game: `g.IsWon()` must return false.

### `TestGame_IsWon_True`

Manually fill all four foundation piles with 13 cards each (via direct state manipulation on a
`buildState()` result, then verify `IsWon` on the `Game` wrapping it). Actually, it's easier to
use `NewGame`, manipulate `g.state` directly, and call `g.IsWon()`.

### `TestGame_IsAutoCompletable_False_FaceDown`

State with a face-down tableau card: `g.IsAutoCompletable()` returns false.

### `TestGame_IsAutoCompletable_False_StockNotEmpty`

State with non-empty stock: `g.IsAutoCompletable()` returns false even if all tableau cards are
face-up.

### `TestGame_IsAutoCompletable_True`

State where stock is empty and all tableau cards are face-up: returns true.

### `TestGame_Execute_SimpleMove`

Setup: T0 = [K♣(up)], T1 = [Q♥(up), J♠(up)]. Execute `MoveCardCmd{PileTableau1, PileTableau0, 1}`.
Assert:
- No error.
- `g.MoveCount() == 1`.
- `g.Score() == 0` (tableau→tableau scores 0).
- T0 has [K♣, J♠], T1 has [Q♥].

### `TestGame_Execute_ScoreWasteToTableau`

Setup: Waste = [9♠(up)], T0 = [10♥(up)]. Execute `MoveCardCmd{PileWaste, PileTableau0, 1}`.
Assert `g.Score() == 5` (Waste→Tableau = +5).

### `TestGame_Execute_ScoreToFoundation`

Setup: Waste = [A♠(up)], F0 (Spades) empty. Execute `MoveToFoundationCmd{PileWaste, 0}`.
Assert `g.Score() == 10`.

### `TestGame_Execute_ScoreFloor`

Setup: `g.state.Score = 0`. Execute `MoveCardCmd{PileFoundation0, PileTableau0, 1}` where
Foundation→Tableau is valid (e.g. F0=[A♠,2♠], T0=[3♥(up)]... actually use a simpler setup).

Actually: set `g.state.Score = 10`, perform a Foundation→Tableau move (−15), assert
`g.Score() == 0` (clamped, not −5).

### `TestGame_Execute_Invalid`

Setup: two empty tableau columns. Execute `MoveCardCmd{PileTableau0, PileTableau1, 1}`.
Assert error returned, `g.MoveCount() == 0`, `g.Score() == 0`.

### `TestGame_Execute_AutoFlip`

Setup: T0 = [K♠(down), Q♥(up)], T1 = [K♣(up)].
Execute `MoveCardCmd{PileTableau0, PileTableau1, 1}` (moves Q♥ to T1).
After execution: T0 = [K♠(up)] — auto-flip triggered.
Assert `g.state.Tableau[0].Cards[0].FaceUp == true`.
Assert `g.Score() == 5` (flip tableau = +5; tableau→tableau move = 0; total = 5).

### `TestGame_Undo_RevertsState`

Setup: T0 = [K♣(up)], T1 = [Q♥(up), J♠(up)].
Execute `MoveCardCmd{PileTableau1, PileTableau0, 1}`.
Call `g.Undo()`.
Assert T0/T1 return to original layout, `g.Score() == 0`, `g.CanUndo() == false`.

### `TestGame_Undo_AutoFlip_RevertsCompound`

Setup: T0 = [K♠(down), Q♥(up)], T1 = [K♣(up)].
Execute move (auto-flip fires). Assert T0 top is face-up.
Call `g.Undo()`. Assert T0 top is face-down again AND Q♥ is back on T0.
This verifies the `CompoundCmd` undoes both operations atomically.

### `TestGame_Redo`

Execute a move, undo it, redo it. Assert state matches the post-execute state.

### `TestGame_Undo_Empty`

Call `g.Undo()` on a fresh game. Assert error returned.

### `TestGame_Redo_Empty`

Call `g.Redo()` on a fresh game. Assert error returned.

### `TestGame_RestartDeal`

Call `NewGame(99, 1)`, snapshot initial state (deep copy top cards of each tableau column).
Execute a few moves. Call `RestartDeal()`. Assert state matches snapshot (same top cards),
`g.CanUndo() == false`, `g.Score() == 0`, `g.MoveCount() == 0`.

### `TestGame_ValidMoves_NotEmpty`

Freshly dealt game should have at least one valid move (stock flip).
Assert `len(g.ValidMoves()) > 0`.

### `TestGame_IsValidMove`

Setup: T0 = [K♣(up)], Waste = [Q♥(up)].
`g.IsValidMove(Move{PileWaste, PileTableau0, 1})` → true.
`g.IsValidMove(Move{PileWaste, PileTableau1, 1})` where T1 is empty → false (Q♥ not a King).

---

## `engine/integration_test.go`

### `TestIntegration_Seed42_Playthrough`

Use `NewGame(42, 1)` to get a deterministic starting board. The test:

1. Records the initial state snapshot (top cards of each tableau column, stock size).
2. Executes a sequence of moves that are guaranteed valid for seed 42:
   - Flip stock (`FlipStockCmd`).
   - Moves waste→tableau if valid.
   - Additional moves as needed to accumulate 3-5 history entries.
3. Verifies score and move count match expected values.
4. Calls `g.Undo()` for each move executed.
5. After all undos: asserts state equals the initial snapshot.

**Implementation note**: Rather than hard-coding a specific seed-42 board (which is fragile),
the test introspects the live state after `NewGame(42, 1)` to find valid moves dynamically
via `g.ValidMoves()`, then executes the first N moves from that list. This makes the test
correct regardless of minor changes to deck/shuffle implementation, while still verifying the
full execute→undo cycle at integration level.

```go
func TestIntegration_Seed42_Playthrough(t *testing.T) {
    g := NewGame(42, 1)

    // Snapshot initial top cards.
    initialTops := snapshotTableauTops(g.state)
    initialStockLen := len(g.state.Stock.Cards)

    // Execute up to 5 valid moves.
    executed := 0
    for executed < 5 {
        moves := g.ValidMoves()
        if len(moves) == 0 {
            break
        }
        if err := g.Execute(moveToCmd(moves[0])); err != nil {
            t.Fatalf("Execute move %d: %v", executed+1, err)
        }
        executed++
    }

    if executed == 0 {
        t.Fatal("no valid moves found — test setup error")
    }
    if g.MoveCount() != executed {
        t.Errorf("MoveCount = %d, want %d", g.MoveCount(), executed)
    }

    // Undo all moves.
    for i := 0; i < executed; i++ {
        if err := g.Undo(); err != nil {
            t.Fatalf("Undo %d: %v", i+1, err)
        }
    }

    // State must match initial snapshot.
    if len(g.state.Stock.Cards) != initialStockLen {
        t.Errorf("stock len = %d, want %d", len(g.state.Stock.Cards), initialStockLen)
    }
    tops := snapshotTableauTops(g.state)
    for i, card := range tops {
        if card != initialTops[i] {
            t.Errorf("tableau[%d] top card = %v, want %v", i, card, initialTops[i])
        }
    }
    if g.Score() != 0 {
        t.Errorf("score after full undo = %d, want 0", g.Score())
    }
    if g.CanUndo() {
        t.Error("CanUndo should be false after undoing all moves")
    }
}
```

Helper functions (unexported, only in integration_test.go):

```go
// snapshotTableauTops returns the top card of each non-empty tableau column, or zero Card.
func snapshotTableauTops(state *GameState) [7]Card {
    var tops [7]Card
    for i, t := range state.Tableau {
        if !t.IsEmpty() {
            tops[i] = t.Cards[len(t.Cards)-1]
        }
    }
    return tops
}

// moveToCmd converts a Move returned by ValidMoves into an executable Command.
// Stock flip/recycle maps to FlipStockCmd or RecycleStockCmd.
// Tableau/waste→foundation uses MoveToFoundationCmd.
// Everything else uses MoveCardCmd.
func moveToCmd(m Move) Command {
    switch {
    case m.From == PileStock:
        if len... // detect recycle vs flip by checking state — this is awkward.
        // Simpler: always use FlipStockCmd; command.Execute handles recycle internally? No.
        // Actually FlipStockCmd errors if stock is empty; RecycleStockCmd errors if stock is non-empty.
        // ValidMoves returns Move{From:PileStock, To:PileWaste} for both flip and recycle.
        // We need state to distinguish. Refine: pass state as parameter.
    case isFoundationPile(m.To):
        return &MoveToFoundationCmd{From: m.From, FoundationIdx: foundationIndex(m.To)}
    default:
        return &MoveCardCmd{From: m.From, To: m.To, CardCount: m.CardCount}
    }
}
```

**Refinement for stock move dispatch**: `moveToCmd` needs access to state to distinguish
flip vs recycle. Pass `state *GameState` as second parameter and check `state.Stock.IsEmpty()`:

```go
func moveToCmd(m Move, state *GameState) Command {
    if m.From == PileStock {
        if state.Stock.IsEmpty() {
            return &RecycleStockCmd{}
        }
        return &FlipStockCmd{}
    }
    if isFoundationPile(m.To) {
        return &MoveToFoundationCmd{From: m.From, FoundationIdx: foundationIndex(m.To)}
    }
    return &MoveCardCmd{From: m.From, To: m.To, CardCount: m.CardCount}
}
```

### `TestIntegration_IsWon`

Manually build a near-won state (all foundations have 12 cards each, one card left in each
tableau that can go to foundation). Execute the final four `MoveToFoundationCmd` commands.
Assert `g.IsWon() == true`.

---

## Score Tracking — Edge Cases

| Scenario | Handling |
|----------|----------|
| Score delta produces negative total | Clamped to 0; `scoreAfter = max(0, scoreBefore+delta)` |
| Undo restores score | `History.Undo` restores `entry.scoreBefore` — no re-calculation |
| Redo restores score | `History.Redo` restores `entry.scoreAfter` — no re-calculation |
| CompoundCmd with flip | `scoreForCmd` recurses; flip sub-cmd contributes +5 |
| MoveCount on Undo | NOT decremented (matches standard Klondike behavior) |

---

## Auto-Flip Edge Cases

| Scenario | Behavior |
|----------|----------|
| Move from tableau exposes face-down card | Auto-flip fires, compound cmd recorded |
| Move from waste to tableau (source is waste, not tableau) | No tableau column loses cards from source; only dest gains. No flip needed on source. |
| MoveToFoundationCmd from tableau | Source tableau loses top card; auto-flip fires if next card is face-down |
| Multiple columns somehow need flip simultaneously | All flips appended to compound (edge case, shouldn't occur in practice since each Execute operates on one move) |
| Move leaves tableau empty | `t.IsEmpty()` check prevents flip attempt on empty column |

---

## Implementation Order

1. **`Game` struct + constructor** in `engine/game.go`  
2. **State queries** (`State`, `Score`, `MoveCount`, `Seed`, `CanUndo`, `CanRedo`)  
3. **`IsWon`** and **`IsAutoCompletable`**  
4. **`NewGame`** and **`RestartDeal`**  
5. **`scoreForCmd`** (needed by Execute)  
6. **`Execute`** (most complex; build after scoreForCmd is ready)  
7. **`Undo`** and **`Redo`**  
8. **`ValidMoves`** and **`IsValidMove`**  
9. **`engine/game_test.go`** — unit tests in order listed above  
10. **`engine/integration_test.go`** — integration tests last  

---

## Verification Gate

```bash
go build ./...           # must compile cleanly
go test ./engine/...     # all existing tests + T12 tests must pass
go vet ./engine/...      # no vet errors
```

The `GameEngine` interface in `engine/interfaces.go` is already complete and must not be changed.
The `Game` struct must satisfy it — the compiler will enforce this via:

```go
var _ GameEngine = (*Game)(nil)  // compile-time interface check
```

Add this line at the top of the `Game` struct declaration block.

---

## Handoff Contract

When T12 is complete, Agent C can use:
- `engine.GameEngine` interface (already in `interfaces.go`)
- `engine.NewGame(seed int64, drawCount int) *Game` — package-level constructor
- `engine.Move` type for `ValidMoves()` / `IsValidMove()` calls
- All concrete command types remain unchanged
