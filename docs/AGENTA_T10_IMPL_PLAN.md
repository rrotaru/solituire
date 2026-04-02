# Agent A — Phase T10 Implementation Plan
## Hint Engine

**Branch**: `claude/agent-a-t10-plan-NrI4m`
**Dependencies**: T2 complete ✓, T3 complete ✓ (`ValidMoves` is already implemented in `engine/rules.go`)
**Blocks**: T12 (GameEngine Wiring) — `ValidMoves()` delegates to hint engine

---

## Overview

T10 replaces the single `panic("not implemented")` stub in `engine/hint.go` with a full
`FindHints` implementation. The `Hint` struct definition is already correct — do **not** modify
the struct fields.

One new test file is created: `engine/hint_test.go`.

### Files changed

| File | Action |
|------|--------|
| `engine/hint.go` | Replace `FindHints` stub with full implementation |
| `engine/hint_test.go` | Create — known-position tests, ordering, edge cases |

No other files are modified.

---

## Key Observation: `ValidMoves` Already Exists

`engine/rules.go` already exports `ValidMoves(state *GameState) []Move` which enumerates every
legal move. `FindHints` does **not** re-implement move enumeration — it calls `ValidMoves`,
converts each `Move` to a `Hint` by computing `CardIdx` and `Priority`, then sorts descending
by `Priority`.

This keeps T10 thin and free of duplication.

---

## Priority Constants

Define package-level unexported constants (not exported — callers compare hints by ordering only):

```go
const (
    priorityFoundation  = 40 // move any card to foundation
    priorityExposeDown  = 30 // reveal a face-down tableau card
    priorityKingToEmpty = 20 // move a King to an empty tableau column
    priorityBuildLength = 10 // extend a tableau build (non-empty destination)
    priorityStockFlip   =  5 // stock flip / recycle (last resort)
)
```

Using a spread of 10 between levels leaves room for future sub-priorities without changing
the public interface.

---

## `CardIdx` Derivation

`Hint.CardIdx` is the index within the **source pile's Cards slice** of the bottom card in
the moved sequence (i.e., the card the TUI will highlight as the hint source).

| Source pile | CardIdx value |
|---|---|
| `PileWaste` | `len(state.Waste.Cards) - 1` (the playable top) |
| `PileTableau*` | `len(t.Cards) - move.CardCount` (bottom card of moved sequence) |
| `PileFoundation*` | `len(f.Cards) - 1` (top card) |
| `PileStock` | `len(state.Stock.Cards) - 1` (top stock card; 0 if empty for recycle) |

---

## Priority Assignment Logic

Given a `Move`, assign priority by testing conditions in descending priority order (first match wins):

1. **Foundation move** — `isFoundationPile(move.To)` → `priorityFoundation`
2. **Expose face-down** — source is a tableau pile **and** removing `move.CardCount` cards
   would uncover a face-down card directly beneath:
   - `isTableauPile(move.From)` **and**
   - `len(t.Cards) > move.CardCount` (there is a card below) **and**
   - `t.Cards[len(t.Cards)-move.CardCount-1].FaceUp == false`
   → `priorityExposeDown`
3. **King to empty column** — destination tableau is empty and the bottom card of the moved
   sequence is a King:
   - `isTableauPile(move.To)` **and** `state.Tableau[dstCol].IsEmpty()` **and**
   - bottom card of sequence has `Rank == King`
   → `priorityKingToEmpty`
4. **Build length** — source is tableau/waste/foundation, destination is a non-empty tableau:
   - `isTableauPile(move.To)` **and** `!state.Tableau[dstCol].IsEmpty()`
   → `priorityBuildLength`
5. **Stock flip** — `move.From == PileStock` → `priorityStockFlip`

Note: conditions are mutually exclusive by construction (e.g., a foundation move can't also have a tableau destination), except for the "expose face-down" vs "King to empty" case. A move that both exposes a face-down card **and** places a King on an empty column gets `priorityExposeDown` (higher).

---

## Step-by-Step Implementation

### Step 1 — `assignPriority` helper

```go
func assignPriority(state *GameState, move Move) int {
    // 1. Foundation move.
    if isFoundationPile(move.To) {
        return priorityFoundation
    }

    // 2. Expose a face-down tableau card.
    if isTableauPile(move.From) {
        t := state.Tableau[tableauIndex(move.From)]
        belowIdx := len(t.Cards) - move.CardCount - 1
        if belowIdx >= 0 && !t.Cards[belowIdx].FaceUp {
            return priorityExposeDown
        }
    }

    // 3. King to empty tableau column.
    if isTableauPile(move.To) {
        dst := state.Tableau[tableauIndex(move.To)]
        if dst.IsEmpty() {
            return priorityKingToEmpty
        }
    }

    // 4. Regular tableau build (non-empty destination).
    if isTableauPile(move.To) {
        return priorityBuildLength
    }

    // 5. Stock flip / recycle.
    if move.From == PileStock {
        return priorityStockFlip
    }

    return 0 // should not be reached; ValidMoves only returns supported moves
}
```

### Step 2 — `cardIdxForMove` helper

```go
func cardIdxForMove(state *GameState, move Move) int {
    switch {
    case move.From == PileWaste:
        return len(state.Waste.Cards) - 1
    case isTableauPile(move.From):
        t := state.Tableau[tableauIndex(move.From)]
        return len(t.Cards) - move.CardCount
    case isFoundationPile(move.From):
        f := state.Foundations[foundationIndex(move.From)]
        return len(f.Cards) - 1
    case move.From == PileStock:
        if len(state.Stock.Cards) > 0 {
            return len(state.Stock.Cards) - 1
        }
        return 0
    }
    return 0
}
```

### Step 3 — `FindHints` (replace stub)

```go
func FindHints(state *GameState) []Hint {
    moves := ValidMoves(state)
    hints := make([]Hint, 0, len(moves))
    for _, m := range moves {
        hints = append(hints, Hint{
            From:     m.From,
            CardIdx:  cardIdxForMove(state, m),
            To:       m.To,
            Priority: assignPriority(state, m),
        })
    }
    // Sort descending by priority (stable to keep ValidMoves ordering as tie-breaker).
    sort.SliceStable(hints, func(i, j int) bool {
        return hints[i].Priority > hints[j].Priority
    })
    return hints
}
```

Add `"sort"` to the import block in `hint.go`.

---

## Step 4 — `engine/hint_test.go`

Use the same `buildState` / `card` / `faceDown` helpers from `engine/rules_test.go`.
If those helpers are defined in a `_test.go` file (package `engine`) they are automatically
available to all `_test.go` files in the same package — no duplication needed.

### Test structure

Each test:
1. Constructs a minimal `GameState` with a known configuration
2. Calls `FindHints(state)`
3. Asserts the returned slice matches expected hints (pile IDs, ordering, length)

---

### `TestFindHints_FoundationHighestPriority`

**Goal**: Foundation move always outranks all other moves.

```
Setup:
  Waste = [A♠(face-up)]           ← can go to Foundation
  T0    = [K♥(face-up), Q♠(face-up)]   ← Q♠ could go to T1
  T1    = [K♦(face-up)]
  F0    = []  (Spades foundation, empty, accepts A♠)
```

`FindHints` returns at least one hint. `hints[0].To` must equal `PileFoundation0` (foundation move
is first). The hint for Waste→Foundation has `From == PileWaste`, `CardIdx == 0` (only card in waste).

---

### `TestFindHints_ExposeFaceDownSecondPriority`

**Goal**: A move that uncovers a face-down card ranks above King-to-empty and build-length.

```
Setup:
  T0 = [K♠(face-down), Q♥(face-up)]   ← moving Q♥ exposes K♠ (face-down)
  T1 = [K♣(face-up)]                   ← accepts Q♥ (build-length, non-expose)
  T2 = []                               ← empty; K♠ could move here (King-to-empty)
```

Expected: `hints[0]` is the `T0→T1` move (exposes face-down, priority 30).
`hints[0].From == PileTableau0`, `hints[0].CardIdx == 1` (Q♥ at index 1 of T0),
`hints[0].Priority == priorityExposeDown`.

---

### `TestFindHints_KingToEmpty`

**Goal**: King-to-empty ranks above ordinary build-length.

```
Setup:
  T0 = [K♥(face-up)]
  T1 = []             ← empty
  T2 = [A♠(face-up)] ← non-empty but K♥ can't go here anyway (no valid placement)
```

The only valid tableau move is T0→T1 (King to empty column).
`hints[0].Priority == priorityKingToEmpty`.

---

### `TestFindHints_BuildLength`

**Goal**: An ordinary tableau build (non-empty destination) is lower priority than expose-down/King-to-empty.

```
Setup:
  T0 = [Q♥(face-up)]
  T1 = [K♠(face-up)]   ← accepts Q♥
```

`hints[0].From == PileTableau0`, `hints[0].To == PileTableau1`,
`hints[0].Priority == priorityBuildLength`.

---

### `TestFindHints_StockFlip`

**Goal**: Stock flip is lowest priority.

```
Setup:
  Stock = [7♣(face-down)]  ← can flip
  Waste = []
  All tableaux: no valid moves possible.
```

`len(hints) == 1`, `hints[0].From == PileStock`, `hints[0].Priority == priorityStockFlip`.

---

### `TestFindHints_Empty_NoMoves`

**Goal**: Returns empty (non-nil) slice when no moves are available.

```
Setup:
  Stock = [] (empty), Waste = [] (empty)
  All foundation piles: full (13 cards each) — IsWon() scenario has no moves.
```

Actually, a won game has no tableau moves. Simpler: create a state where no pile can be
moved and stock/waste are empty.

```
Setup:
  Stock = [], Waste = []
  T0 = [2♠(face-up)]        ← can't go anywhere
  T1..T6: empty
  F0..F3: empty
```

`len(hints) == 0`.

---

### `TestFindHints_CardIdx_Tableau_MultiCard`

**Goal**: `CardIdx` points to the **bottom** card of a multi-card sequence (not the top).

```
Setup:
  T0 = [A♦(face-down), Q♥(face-up), J♠(face-up)]
  T1 = [K♣(face-up)]
```

Moving Q♥-J♠ (2 cards) to T1 is valid. The corresponding hint has:
`CardIdx == 1` (index of Q♥, the bottom of the 2-card sequence = `len(T0.Cards) - 2 = 3-2 = 1`).

Also: this move exposes A♦ (face-down), so `Priority == priorityExposeDown`.

---

### `TestFindHints_OrderingStability`

**Goal**: When two moves share the same priority, their relative order matches `ValidMoves` output
(stable sort).

```
Setup:
  Waste = [A♥(face-up)]   ← Waste→Foundation1 (Hearts)
  T3    = [A♦(face-up)]   ← T3→Foundation3 (Diamonds)
  F1 = [], F3 = []
```

Both are foundation moves (priority 40). `FindHints` returns both, both with priority 40.
Their relative order matches whatever order `ValidMoves` returns them (ValidMoves checks
Waste→Foundation before Tableau→Foundation).
Assert `len(hints) >= 2`, `hints[0].Priority == 40`, `hints[1].Priority == 40`.

---

## Imports Required

**`engine/hint.go`**:
```go
import "sort"
```

---

## File Checklist

| File | Action |
|------|--------|
| `engine/hint.go` | Replace `FindHints` panic stub; add `assignPriority`, `cardIdxForMove` helpers; add `sort` import; add priority constants |
| `engine/hint_test.go` | Create with ≥8 test functions |

---

## Verification Gate

```bash
go build ./...          # no new exported types — must compile cleanly
go test ./engine/...    # all existing tests + T10 tests must pass
go vet ./engine/...     # no vet errors
```

No other packages are modified.

---

## Handoff Contract

When T10 is complete, T12 (GameEngine Wiring) is unblocked. It calls `FindHints(state)` inside
`Game.ValidMoves()` to return `[]Move` (by extracting the `Move` embedded in each `Hint`).

**Note for T12**: `ValidMoves()` on `GameEngine` returns `[]Move`, not `[]Hint`. T12 should
call `FindHints` and map back to `[]Move` (discarding priority), or call `engine.ValidMoves`
directly if hint ordering is not needed for the interface contract.
