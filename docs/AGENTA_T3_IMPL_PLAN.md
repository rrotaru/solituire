# Agent A — Phase T3 Implementation Plan
## Move Validation Rules

**Branch**: `claude/agent-a-t3-plan-uDcEC`
**Dependencies**: T2 complete ✓
**Blocks**: T6 (Commands) — commands call `ValidateMove` before executing

---

## Overview

T3 fully implements `engine/rules.go`, replacing both `panic("not implemented")` stubs with correct logic. The `Move` type already exists from T2. Nothing else in the codebase changes. One new test file is created: `engine/rules_test.go`.

The two exported functions to implement are:

```go
func ValidateMove(state *GameState, move Move) error
func ValidMoves(state *GameState) []Move
```

`ValidateMove` is the single gating function for all game moves. `ValidMoves` enumerates every legal move in a given state — used by the hint engine (T10) and the `GameEngine.ValidMoves()` method (T12).

---

## Data Model Recap (from T2)

Relevant types already in scope:

| Type | File | Key methods used |
|------|------|-----------------|
| `Card{Suit, Rank, FaceUp}` | `card.go` | `.Color()`, `.Rank`, `.Suit` |
| `Rank` uint8, Ace=1…King=13 | `card.go` | arithmetic comparison |
| `Color` Black/Red | `card.go` | equality comparison |
| `TableauPile` | `tableau.go` | `.FaceUpCards()`, `.TopCard()`, `.IsEmpty()` |
| `FoundationPile` | `foundation.go` | `.AcceptsCard()`, `.TopCard()`, `.IsComplete()` |
| `StockPile` | `stock.go` | `.IsEmpty()` |
| `WastePile` | `stock.go` | `.TopCard()`, `.IsEmpty()` |
| `GameState` | `game.go` | `.Tableau[7]`, `.Foundations[4]`, `.Stock`, `.Waste`, `.DrawCount` |
| `PileID` consts | `game.go` | `PileStock`, `PileWaste`, `PileFoundation0-3`, `PileTableau0-6` |
| `Move{From, To, CardCount}` | `rules.go` | already defined in stub |

---

## Step-by-Step Implementation

### Step 1 — Private helper functions

These helpers keep `ValidateMove` readable and are not exported. Add them at the top of the implementation section in `engine/rules.go`.

```go
// isTableauPile returns true if id is one of PileTableau0..PileTableau6.
func isTableauPile(id PileID) bool {
    return id >= PileTableau0 && id <= PileTableau6
}

// isFoundationPile returns true if id is one of PileFoundation0..PileFoundation3.
func isFoundationPile(id PileID) bool {
    return id >= PileFoundation0 && id <= PileFoundation3
}

// tableauIndex returns the 0-based column index for a tableau PileID.
// Caller must ensure isTableauPile(id) is true.
func tableauIndex(id PileID) int {
    return int(id - PileTableau0)
}

// foundationIndex returns the 0-based index for a foundation PileID.
// Caller must ensure isFoundationPile(id) is true.
func foundationIndex(id PileID) int {
    return int(id - PileFoundation0)
}
```

**Key decision**: Helper bounds use the `PileID` constants defined in `game.go`. If constants ever shift, these helpers automatically stay correct.

---

### Step 2 — `isValidTableauPlacement`

Encapsulates the tableau placement rule (alternating color, descending rank, King-to-empty) used by multiple move types.

```go
// isValidTableauPlacement reports whether card can be placed onto dest.
// Empty dest accepts only Kings. Non-empty dest requires opposite color
// and exactly one rank lower than the destination top card.
func isValidTableauPlacement(card Card, dest *TableauPile) bool {
    if dest.IsEmpty() {
        return card.Rank == King
    }
    top := dest.TopCard()
    return card.Color() != top.Color() && card.Rank == top.Rank-1
}
```

---

### Step 3 — `isValidFaceUpSequence`

Validates that a slice of cards forms a correctly-built sequence (descending rank, alternating color). Used for multi-card Tableau→Tableau moves.

```go
// isValidFaceUpSequence returns true if cards form a valid built sequence:
// each card is one rank lower and the opposite color from the card above it.
// A single card or nil slice is always valid.
func isValidFaceUpSequence(cards []Card) bool {
    for i := 1; i < len(cards); i++ {
        prev, curr := cards[i-1], cards[i]
        if curr.Color() == prev.Color() || curr.Rank != prev.Rank-1 {
            return false
        }
    }
    return true
}
```

---

### Step 4 — Move-type validators (private)

One function per Section 13 rule set. Each returns a typed `error` on failure.

#### `validateTableauToTableau` (Section 13.1)

```go
func validateTableauToTableau(state *GameState, move Move) error {
    src := state.Tableau[tableauIndex(move.From)]
    dst := state.Tableau[tableauIndex(move.To)]

    faceUp := src.FaceUpCards()
    if len(faceUp) == 0 {
        return errors.New("source tableau has no face-up cards")
    }
    if move.CardCount < 1 || move.CardCount > len(faceUp) {
        return errors.New("card count out of range for source face-up cards")
    }

    // The moved sub-sequence starts at index (len(faceUp) - CardCount).
    seq := faceUp[len(faceUp)-move.CardCount:]

    if !isValidFaceUpSequence(seq) {
        return errors.New("moved cards do not form a valid sequence")
    }
    if !isValidTableauPlacement(seq[0], dst) {
        return errors.New("destination tableau does not accept this card")
    }
    return nil
}
```

**Notes**:
- `seq[0]` is the **bottom** card of the moved sequence (lowest rank, highest in the stack visually).
- A `CardCount` of 1 moves only the top card of the source.
- Moving a face-down card is implicitly rejected because `FaceUpCards()` never includes them.

#### `validateWasteToTableau` (Section 13.2)

```go
func validateWasteToTableau(state *GameState, move Move) error {
    top := state.Waste.TopCard()
    if top == nil {
        return errors.New("waste pile is empty")
    }
    // Waste→Tableau is always a single card.
    if move.CardCount != 1 {
        return errors.New("can only move one card from waste")
    }
    dst := state.Tableau[tableauIndex(move.To)]
    if !isValidTableauPlacement(*top, dst) {
        return errors.New("destination tableau does not accept this card")
    }
    return nil
}
```

#### `validateToFoundation` (Section 13.3) — covers Waste→Foundation and Tableau→Foundation

```go
func validateToFoundation(state *GameState, move Move) error {
    var card Card
    switch {
    case move.From == PileWaste:
        top := state.Waste.TopCard()
        if top == nil {
            return errors.New("waste pile is empty")
        }
        card = *top
    case isTableauPile(move.From):
        src := state.Tableau[tableauIndex(move.From)]
        top := src.TopCard()
        if top == nil {
            return errors.New("source tableau is empty")
        }
        if !top.FaceUp {
            return errors.New("top card of source tableau is face-down")
        }
        // Only the top card may move to the foundation; CardCount must be 1.
        if move.CardCount != 1 {
            return errors.New("can only move one card to foundation")
        }
        card = *top
    default:
        return errors.New("invalid source for foundation move")
    }

    // Foundation destination: move.To identifies which of the four foundations.
    // If move.To is PileFoundation0..3, use that specific pile.
    // The design allows targeting any foundation; AcceptsCard enforces suit/rank.
    foundation := state.Foundations[foundationIndex(move.To)]
    if !foundation.AcceptsCard(card) {
        return errors.New("foundation does not accept this card")
    }
    return nil
}
```

**Note**: `FoundationPile.AcceptsCard` already enforces the Ace-first, ascending-same-suit rule, so no duplication needed.

#### `validateFoundationToTableau` (Section 13.4)

```go
func validateFoundationToTableau(state *GameState, move Move) error {
    foundation := state.Foundations[foundationIndex(move.From)]
    top := foundation.TopCard()
    if top == nil {
        return errors.New("source foundation is empty")
    }
    dst := state.Tableau[tableauIndex(move.To)]
    if !isValidTableauPlacement(*top, dst) {
        return errors.New("destination tableau does not accept this card")
    }
    return nil
    // Note: the −15 scoring penalty is applied by StandardScorer, not here.
}
```

#### `validateStockFlip` (Section 13.5)

```go
func validateStockFlip(state *GameState, move Move) error {
    // move.From must be PileStock; move.To must be PileWaste.
    if move.To != PileWaste {
        return errors.New("stock flip destination must be waste")
    }
    if !state.Stock.IsEmpty() {
        return nil // Normal flip: draw DrawCount cards (or fewer if stock nearly empty).
    }
    // Stock empty: recycle only if waste is non-empty.
    if state.Waste.IsEmpty() {
        return errors.New("stock and waste are both empty")
    }
    return nil // Recycle: all waste cards go back to stock reversed.
}
```

---

### Step 5 — `ValidateMove` dispatcher

```go
func ValidateMove(state *GameState, move Move) error {
    switch {
    case move.From == PileStock:
        return validateStockFlip(state, move)
    case isTableauPile(move.From) && isTableauPile(move.To):
        return validateTableauToTableau(state, move)
    case move.From == PileWaste && isTableauPile(move.To):
        return validateWasteToTableau(state, move)
    case (move.From == PileWaste || isTableauPile(move.From)) && isFoundationPile(move.To):
        return validateToFoundation(state, move)
    case isFoundationPile(move.From) && isTableauPile(move.To):
        return validateFoundationToTableau(state, move)
    default:
        return fmt.Errorf("unsupported move: from %d to %d", move.From, move.To)
    }
}
```

**Dispatch order matters**: `PileStock` is checked first to avoid ambiguity (it is neither a tableau nor a foundation pile).

---

### Step 6 — `ValidMoves`

Enumerate every legal move by probing all source/destination combinations through `ValidateMove`. This is O(n) in the number of possible moves (bounded constant for Klondike), so no optimization is needed.

```go
func ValidMoves(state *GameState) []Move {
    var moves []Move

    // 1. Stock flip or recycle.
    stockMove := Move{From: PileStock, To: PileWaste}
    if ValidateMove(state, stockMove) == nil {
        moves = append(moves, stockMove)
    }

    // 2. Waste → Tableau (single card).
    for col := 0; col < 7; col++ {
        m := Move{From: PileWaste, To: PileTableau0 + PileID(col), CardCount: 1}
        if ValidateMove(state, m) == nil {
            moves = append(moves, m)
        }
    }

    // 3. Waste → Foundation.
    for f := 0; f < 4; f++ {
        m := Move{From: PileWaste, To: PileFoundation0 + PileID(f), CardCount: 1}
        if ValidateMove(state, m) == nil {
            moves = append(moves, m)
        }
    }

    // 4. Tableau → Foundation (top card only).
    for col := 0; col < 7; col++ {
        for f := 0; f < 4; f++ {
            m := Move{
                From:      PileTableau0 + PileID(col),
                To:        PileFoundation0 + PileID(f),
                CardCount: 1,
            }
            if ValidateMove(state, m) == nil {
                moves = append(moves, m)
            }
        }
    }

    // 5. Foundation → Tableau (top card only).
    for f := 0; f < 4; f++ {
        for col := 0; col < 7; col++ {
            m := Move{
                From:      PileFoundation0 + PileID(f),
                To:        PileTableau0 + PileID(col),
                CardCount: 1,
            }
            if ValidateMove(state, m) == nil {
                moves = append(moves, m)
            }
        }
    }

    // 6. Tableau → Tableau (all valid sequence lengths).
    for srcCol := 0; srcCol < 7; srcCol++ {
        faceUp := state.Tableau[srcCol].FaceUpCards()
        for count := 1; count <= len(faceUp); count++ {
            for dstCol := 0; dstCol < 7; dstCol++ {
                if srcCol == dstCol {
                    continue
                }
                m := Move{
                    From:      PileTableau0 + PileID(srcCol),
                    To:        PileTableau0 + PileID(dstCol),
                    CardCount: count,
                }
                if ValidateMove(state, m) == nil {
                    moves = append(moves, m)
                }
            }
        }
    }

    return moves
}
```

**Key decision**: Iterating `count` from 1 to `len(faceUp)` is correct because a multi-card sequence move is only legal if the entire sub-sequence from `faceUp[len-count]` down is well-formed **and** placeable. `ValidateMove` rejects malformed sub-sequences, so `ValidMoves` never returns illegal moves.

---

### Step 7 — `engine/rules_test.go`

Use table-driven tests. A set of small board-construction helpers (`buildState`) creates the precise game state needed for each case without going through `Deal` — this keeps tests hermetic and fast.

#### Board construction helper pattern

```go
// buildState returns a zeroed GameState with all piles initialized (empty).
func buildState() *GameState {
    s := &GameState{
        Stock: &StockPile{},
        Waste: &WastePile{DrawCount: 1},
    }
    for i := range s.Tableau {
        s.Tableau[i] = &TableauPile{}
    }
    for i := range s.Foundations {
        s.Foundations[i] = &FoundationPile{}
    }
    return s
}

// card constructs a face-up Card.
func card(r Rank, s Suit) Card { return Card{Rank: r, Suit: s, FaceUp: true} }

// faceDown constructs a face-down Card.
func faceDown(r Rank, s Suit) Card { return Card{Rank: r, Suit: s, FaceUp: false} }
```

#### Test functions

**`TestValidateMove_TableauToTableau`** — table-driven:

| Case | Setup | Move | wantErr |
|------|-------|------|---------|
| red 6 onto black 7 | T2=[…,7♠ face-up], T4=[…,6♥ face-up] | T4→T2, count=1 | false |
| red 6 onto red 7 | T2=[7♥], T4=[6♥] | T4→T2, count=1 | true |
| black 6 onto black 7 | T2=[7♠], T4=[6♠] | T4→T2, count=1 | true |
| wrong rank (5 onto 7) | T2=[7♠], T4=[5♥] | T4→T2, count=1 | true |
| King to empty column | T2=[], T4=[K♥] | T4→T2, count=1 | false |
| non-King to empty column | T2=[], T4=[Q♠] | T4→T2, count=1 | true |
| move face-down card | T2=[7♠], T4=[6♥ face-down] | T4→T2, count=1 | true |
| valid 2-card sequence (6♥,5♠) onto 7♣ | T2=[7♣], T4=[6♥,5♠ face-up] | T4→T2, count=2 | false |
| invalid sub-sequence (6♥,5♥ same color) | T2=[7♣], T4=[6♥,5♥ face-up] | T4→T2, count=2 | true |
| CardCount exceeds face-up cards | T2=[7♠], T4=[6♥] | T4→T2, count=5 | true |
| same source and destination | T2=[7♠,6♥ face-up] | T2→T2, count=1 | true |

**`TestValidateMove_WasteToTableau`**:

| Case | Waste top | Dest | wantErr |
|------|-----------|------|---------|
| valid placement (6♥ onto 7♠) | 6♥ | [7♠] | false |
| wrong color (6♥ onto 7♥) | 6♥ | [7♥] | true |
| waste empty | (empty) | [7♠] | true |
| King to empty tableau | K♠ | [] | false |
| non-King to empty tableau | Q♥ | [] | true |

**`TestValidateMove_ToFoundation`** — covers both Waste→Foundation and Tableau→Foundation:

| Case | Source | Foundation | wantErr |
|------|--------|-----------|---------|
| Ace to empty foundation | Waste: A♠ | F0=[] | false |
| 2 of same suit onto Ace | Waste: 2♠ | F0=[A♠] | false |
| 2 of wrong suit | Waste: 2♥ | F0=[A♠] | true |
| wrong rank (3 onto Ace) | Waste: 3♠ | F0=[A♠] | true |
| Tableau top card to foundation | T0=[A♥] | F1=[] | false |
| Tableau face-down top blocked | T0=[A♥ face-down] | F1=[] | true |
| CardCount > 1 from tableau rejected | T0=[A♥] | F1=[] | true |

**`TestValidateMove_FoundationToTableau`**:

| Case | Foundation top | Dest | wantErr |
|------|----------------|------|---------|
| valid (K♠ to empty tableau) | F0=[…,K♠] | T3=[] | false |
| valid (Q♥ onto K♠) | F0=[…,Q♥] | T3=[K♠] | false |
| wrong color | F0=[…,Q♠] | T3=[K♠] | true |
| empty foundation | F0=[] | T3=[K♠] | true |

**`TestValidateMove_StockFlip`**:

| Case | Stock | Waste | wantErr |
|------|-------|-------|---------|
| stock non-empty flip | [A♠, 2♠] | [] | false |
| stock empty, waste non-empty (recycle) | [] | [5♥] | false |
| both empty | [] | [] | true |
| wrong destination (To != PileWaste) | [A♠] | [] | true |

**`TestValidMoves_NoDuplicates`**: Call `ValidMoves` on a freshly dealt state (seed 42), verify no two moves are identical `(From, To, CardCount)` tuples.

**`TestValidMoves_EmptyBoard`**: All piles empty, `ValidMoves` returns empty slice.

**`TestValidMoves_StockOnlyBoard`**: Only stock has cards; `ValidMoves` returns exactly one move (the stock flip).

---

## File Checklist

| File | Action |
|------|--------|
| `engine/rules.go` | Replace both `panic` stubs with full implementation |
| `engine/rules_test.go` | Create — full table-driven test suite |

No other files change.

---

## Imports Required

`engine/rules.go` will need:
```go
import (
    "errors"
    "fmt"
)
```

`engine/rules_test.go` will need:
```go
import "testing"
```

---

## Verification Gate

```bash
go build ./...              # must succeed — no new types introduced
go test ./engine/...        # all existing tests + rules_test.go must pass
```

The `panic("not implemented")` stubs in `command.go`, `history.go`, `scoring.go`, `hint.go` are **not called** during T3 tests, so no panics will occur.

---

## Handoff Contract

When T3 is complete, signal to:

**T6 (Commands)** — can start immediately. Exports available:
- `engine.ValidateMove(state, move) error` — call this inside each command's `Execute` before mutating state
- `engine.Move{From, To, CardCount}` type
- `engine.ValidMoves(state) []Move` — available for T12 wiring later

**T10 (Hint Engine)** — can also start in parallel with T6; it calls `ValidMoves` and `ValidateMove` to enumerate and prioritize hints.
