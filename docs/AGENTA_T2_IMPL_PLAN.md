# Agent A — Phase T2 Implementation Plan
## Card, Deck, and Pile Types

**Branch**: `claude/agent-a-t2-plan-MSQ9l`
**Dependencies**: T1 complete ✓
**Blocks**: Agent B (T8 Renderer), Agent C (T9 Input Translator can start; T11 Board Model blocked until T6)

---

## Overview

T2 delivers the complete data model for the engine package: all card/pile/state types, deck operations, deep-copy, and stub files that let Agents B and C compile immediately. No game logic (rules, commands, scoring) is implemented here — only the types and their structural methods.

---

## Step-by-Step Implementation Order

### Step 1 — `engine/card.go`: Card, Suit, Rank types

Define the core value types. These have no dependencies and everything else builds on them.

```go
package engine

type Color uint8
const (
    Black Color = iota
    Red
)

type Suit uint8
const (
    Spades   Suit = iota // Black
    Hearts               // Red
    Diamonds             // Red
    Clubs                // Black
)

func (s Suit) Color() Color  // Spades/Clubs → Black; Hearts/Diamonds → Red
func (s Suit) Symbol() string // "♠" "♥" "♦" "♣"
func (s Suit) String() string // "Spades" "Hearts" "Diamonds" "Clubs"

type Rank uint8
const (
    Ace   Rank = iota + 1
    Two
    Three
    Four
    Five
    Six
    Seven
    Eight
    Nine
    Ten
    Jack
    Queen
    King  // = 13
)

func (r Rank) String() string // "A" "2"…"10" "J" "Q" "K"

type Card struct {
    Suit   Suit
    Rank   Rank
    FaceUp bool
}

func (c Card) String() string        // e.g. "K♠" (face-up) or "??" (face-down)
func (c Card) Color() Color          // delegates to c.Suit.Color()
```

**Key decisions**:
- `Color` is a first-class type (not `bool`) so renderer theme can switch on it cleanly.
- `Card` is a value type (not pointer) — piles hold `[]Card`, copies are cheap and safe for deep-copy.
- `FaceUp` lives on `Card` so piles don't need parallel bool slices.

---

### Step 2 — `engine/deck.go`: NewDeck, Shuffle, Deal

```go
package engine

import "math/rand"

// NewDeck returns an ordered 52-card deck (Spades A-K, Hearts A-K, Diamonds A-K, Clubs A-K).
// All cards are face-down initially; Deal() flips as needed.
func NewDeck() []Card

// Shuffle Fisher-Yates shuffles deck in-place using a seeded PRNG and returns it.
func Shuffle(deck []Card, seed int64) []Card {
    r := rand.New(rand.NewSource(seed))
    r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
    return deck
}

// Deal distributes a shuffled deck into a new GameState.
// Tableau layout: column i gets (i+1) cards, bottom (i) face-down, top 1 face-up.
// Remaining 24 cards go to Stock face-down.
// DrawCount defaults to 1; caller sets GameState.DrawCount after Deal.
func Deal(deck []Card) *GameState
```

**Deal layout** (standard Klondike):
- Column 0: 1 card (face-up)
- Column 1: 2 cards (1 face-down, 1 face-up)
- …
- Column 6: 7 cards (6 face-down, 1 face-up)
- Stock: remaining 24 cards (all face-down)
- Waste: empty
- Foundations: all empty

---

### Step 3 — `engine/tableau.go`: TableauPile

```go
package engine

type TableauPile struct {
    Cards []Card
}

func (t *TableauPile) FaceDownCount() int       // count of face-down cards at bottom
func (t *TableauPile) FaceUpCards() []Card      // slice of face-up cards at top (may be empty)
func (t *TableauPile) TopCard() *Card           // nil if empty; last element of Cards
func (t *TableauPile) IsEmpty() bool
```

**Note**: `FaceUpCards()` returns a slice of the underlying array (not a copy) — callers must not mutate. Rules and commands work with indices into `Cards`.

---

### Step 4 — `engine/foundation.go`: FoundationPile

```go
package engine

type FoundationPile struct {
    Cards []Card  // all face-up, Ace at [0], King at [12] when complete
}

func (f *FoundationPile) TopCard() *Card            // nil if empty
func (f *FoundationPile) AcceptsCard(card Card) bool // Ace on empty; next rank same suit
func (f *FoundationPile) IsComplete() bool           // len == 13
func (f *FoundationPile) Suit() *Suit                // nil if empty, else suit of first card
```

---

### Step 5 — `engine/stock.go`: StockPile, WastePile

```go
package engine

type StockPile struct {
    Cards []Card  // all face-down; top of draw pile is last element
}

func (s *StockPile) IsEmpty() bool
func (s *StockPile) Count() int

type WastePile struct {
    Cards     []Card  // face-up; top playable card is last element
    DrawCount int     // 1 or 3
}

func (w *WastePile) TopCard() *Card          // nil if empty; the one playable card (last)
func (w *WastePile) VisibleCards() []Card    // top min(DrawCount, len) cards for rendering
func (w *WastePile) IsEmpty() bool
```

**Draw-3 visibility**: `VisibleCards()` returns `Cards[max(0, len-3):]` — up to 3 cards rendered, only the last is playable. This is read-only for the renderer; `FlipStockCmd` handles the actual transfer.

---

### Step 6 — `engine/game.go`: GameState + deepCopyState

```go
package engine

import "time"

// PileID is used by Move and commands to reference piles without pointer equality.
type PileID uint8
const (
    PileStock      PileID = iota
    PileWaste
    PileFoundation0 // +0..+3 for foundations
    PileFoundation1
    PileFoundation2
    PileFoundation3
    PileTableau0    // +0..+6 for tableau columns
    PileTableau1
    PileTableau2
    PileTableau3
    PileTableau4
    PileTableau5
    PileTableau6
)

type GameState struct {
    Tableau     [7]*TableauPile
    Foundations [4]*FoundationPile
    Stock       *StockPile
    Waste       *WastePile
    Score       int
    MoveCount   int
    ElapsedTime time.Duration
    DrawCount   int    // 1 or 3
    Seed        int64
}

// deepCopyState returns a fully independent copy of state.
// Used by commands to snapshot state for Undo and by RestartDeal.
func deepCopyState(s *GameState) *GameState
```

**deepCopyState implementation notes**:
- Allocate new `TableauPile`, `FoundationPile`, `StockPile`, `WastePile` structs.
- Copy `[]Card` slices with `make` + `copy` (not `append`) to avoid shared backing arrays.
- Scalar fields copy by value.

---

### Step 7 — Stub files (compile targets for Agents B and C)

Create these files with package declaration and `panic("not implemented")` bodies. They must compile but not be correct. Agents B and C import `engine` types; stubs let `go build ./...` succeed.

**`engine/interfaces.go`** — stubs for all three interfaces:
```go
package engine

type GameEngine interface {
    State() *GameState
    IsWon() bool
    IsAutoCompletable() bool
    Score() int
    MoveCount() int
    Seed() int64
    Execute(cmd Command) error
    Undo() error
    Redo() error
    CanUndo() bool
    CanRedo() bool
    ValidMoves() []Move
    IsValidMove(move Move) bool
    NewGame(seed int64, drawCount int)
    RestartDeal()
}

type Command interface {
    Execute(state *GameState) error
    Undo(state *GameState) error
    Description() string
}

type Scorer interface {
    OnMove(move Move, state *GameState) int
    OnUndo(move Move, state *GameState) int
    OnRecycleStock() int
}
```

**`engine/rules.go`** — stub:
```go
package engine

type Move struct {
    From      PileID
    To        PileID
    CardCount int
}

func ValidateMove(state *GameState, move Move) error { panic("not implemented") }
func ValidMoves(state *GameState) []Move             { panic("not implemented") }
```

**`engine/command.go`** — stub command types with panic bodies:
```go
package engine

type MoveCardCmd        struct { From PileID; To PileID; CardCount int }
type MoveToFoundationCmd struct { From PileID; FoundationIdx int }
type FlipStockCmd       struct{}
type RecycleStockCmd    struct{}
type FlipTableauCardCmd struct { ColumnIdx int }
type CompoundCmd        struct { Cmds []Command }

// Each type implements Command interface with panic("not implemented")
```

**`engine/history.go`** — stub:
```go
package engine

type History struct{}

func (h *History) Push(cmd Command)       { panic("not implemented") }
func (h *History) Undo(s *GameState) error { panic("not implemented") }
func (h *History) Redo(s *GameState) error { panic("not implemented") }
func (h *History) CanUndo() bool           { panic("not implemented") }
func (h *History) CanRedo() bool           { panic("not implemented") }
func (h *History) Clear()                  { panic("not implemented") }
```

**`engine/scoring.go`** — stub:
```go
package engine

type StandardScorer struct{}

func (s StandardScorer) OnMove(move Move, state *GameState) int  { panic("not implemented") }
func (s StandardScorer) OnUndo(move Move, state *GameState) int  { panic("not implemented") }
func (s StandardScorer) OnRecycleStock() int                      { panic("not implemented") }
```

**`engine/hint.go`** — stub:
```go
package engine

type Hint struct {
    From     PileID
    CardIdx  int
    To       PileID
    Priority int
}

func FindHints(state *GameState) []Hint { panic("not implemented") }
```

---

### Step 8 — Tests

#### `engine/card_test.go`
- `TestSuitColors`: Hearts/Diamonds → Red; Spades/Clubs → Black
- `TestSuitSymbols`: all four Unicode symbols correct
- `TestRankStrings`: Ace→"A", 2-10→"2"…"10", Jack→"J", Queen→"Q", King→"K"
- `TestAllCardsUnique`: `NewDeck()` returns exactly 52 unique (Suit, Rank) pairs

#### `engine/deck_test.go`
- `TestDeterministicShuffle`: same seed → same order; different seed → different order
- `TestDealLayout`: after `Deal()`:
  - Column 0 has 1 card (face-up), column 6 has 7 cards (top face-up, rest face-down)
  - `FaceDownCount()` for column i == i
  - Stock has 24 cards
  - Waste and all foundations are empty
  - No duplicate cards across all piles (total = 52)

#### `engine/pile_test.go`
- `TestTableauFaceUpCards`: pile with mixed face-down/up returns correct slice
- `TestFoundationAcceptsCard`: nil pile accepts Ace only; accepts next rank same suit; rejects wrong suit/rank
- `TestFoundationIsComplete`: returns true only when 13 cards
- `TestWastePileTopCard`: nil when empty; correct card when populated
- `TestWastePileVisibleCards_Draw1`: always returns at most 1 card
- `TestWastePileVisibleCards_Draw3`: returns up to 3, correct slice when fewer available

---

## File Checklist

| File | Status |
|------|--------|
| `engine/card.go` | implement fully |
| `engine/deck.go` | implement fully |
| `engine/tableau.go` | implement fully |
| `engine/foundation.go` | implement fully |
| `engine/stock.go` | implement fully |
| `engine/game.go` | implement GameState + deepCopyState |
| `engine/interfaces.go` | stub (real signatures, panic bodies) |
| `engine/rules.go` | stub |
| `engine/command.go` | stub |
| `engine/history.go` | stub |
| `engine/scoring.go` | stub |
| `engine/hint.go` | stub |
| `engine/card_test.go` | full tests |
| `engine/deck_test.go` | full tests |
| `engine/pile_test.go` | full tests |

---

## Verification Gate

```bash
go build ./...          # must succeed — stubs satisfy compilation
go test ./engine/...    # card_test, deck_test, pile_test must pass
```

No panics are acceptable in the test run (stubs are not called by tests).

---

## Handoff Contract

When T2 is complete, signal to:

**Agent B (T8 Renderer)** — can start immediately. Exports available:
- `engine.Card`, `engine.Suit`, `engine.Rank` and all methods
- `engine.Color` (Black/Red)
- `engine.GameState` with all pile pointer fields
- `engine.TableauPile`, `engine.FoundationPile`, `engine.StockPile`, `engine.WastePile`
- `engine.PileID` constants

**Agent C (T9 Input Translator)** — can start (only needs Bubbletea types).
Agent C's T11 (Board Model) remains blocked until T6 (Commands) completes.
