# Agent C — T16: Mouse Input Support Implementation Plan

**Stage 8 | Phase 7 | Estimate: Medium | Dependencies: T11 ✓, T13 ✓**

---

## 1. Overview

T16 wires up full mouse support so players can click piles and cards instead of (or in addition to) using the keyboard. The work spans three concerns:

1. **Input translation** — `tea.MouseMsg` → `ActionSelect` (with the raw `MouseMsg` as payload)
2. **Hit-testing** — map terminal click coordinates `(x, y)` to a `(PileID, cardIndex)` using layout geometry
3. **Board integration** — consume the hit-test result inside `BoardModel.Update` to set cursor position before the existing select logic runs

The design doc also calls for table-driven tests for hit-testing (§TODO) and model-state tests for mouse-driven drag flows (§8.4).

**Scope:**
- **No new files required** — all logic belongs in existing files
- **Modify**: `tui/input.go`, `tui/cursor.go` (or `renderer/layout.go`), `tui/board.go`
- **Add tests**: `renderer/layout_test.go` (hit-test table), `tui/board_test.go` (mouse drag tests)

---

## 2. Audit: What Is Already Implemented

Before writing any code, verify the following (all were implemented as part of T11/T13):

### 2.1 `tui/input.go` — mouse translation ✅

```go
func translateMouse(m tea.MouseMsg) (GameAction, interface{}) {
    if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
        return ActionSelect, m
    }
    return ActionNone, nil
}
```

- Left-press → `ActionSelect` with `tea.MouseMsg` as payload.
- All other mouse events (release, scroll, right-click) → `ActionNone`.

### 2.2 `tui/board.go` — hit-test integration in `ActionSelect` ✅

```go
case ActionSelect:
    if mouse, ok := payload.(tea.MouseMsg); ok {
        pile, cardIdx, hit := renderer.PileHitTestWithWidth(mouse.X, mouse.Y, state, m.width)
        if !hit {
            break
        }
        m.cursor.Pile = pile
        m.cursor.CardIndex = cardIdx
    }
    m = m.handleSelect(state)
```

- Miss-clicks outside all piles are silently ignored (`break` before `handleSelect`).
- Hit-clicks warp the cursor to the target pile/card, then run normal select logic (pick up or place).

### 2.3 `renderer/layout.go` — layout constants and hit-test functions ✅

Exported symbols used by `tui/board.go`:

| Symbol | Value | Purpose |
|--------|-------|---------|
| `CardWidth` | 9 | Full card width including borders |
| `CardHeight` | 7 | Full card height including borders |
| `MinTermWidth` | 78 | Minimum terminal width |
| `MinTermHeight` | 25 | Minimum terminal height |
| `ColGap` | 1 | Gap between tableau columns |
| `PileHitTest(x, y, state)` | — | Hit-test with default `MinTermWidth` |
| `PileHitTestWithWidth(x, y, state, termWidth)` | — | Width-aware hit-test (preferred) |

### 2.4 `tui/input_test.go` — mouse translation tests ✅

| Test | Covers |
|------|--------|
| `TestTranslateInput` (mouse left click row) | Left-press → `ActionSelect` |
| `TestTranslateInput_MousePayload` | Payload preserves `X`, `Y` |
| `TestTranslateInput_MouseRelease` | Release → `ActionNone` |

### 2.5 `tui/board_test.go` — basic mouse integration tests ✅

| Test | Covers |
|------|--------|
| `TestBoardMouseClickMovesAndSelects` | Click stock → cursor warps, card flips |
| `TestBoardMouseClickOutsidePile` | Miss-click → cursor unchanged, no engine effect |

---

## 3. Remaining Work

Two gaps remain. The implementation code is complete; only tests are missing.

### Gap 1 — Hit-test unit tests (`renderer/layout_test.go`)

The design doc calls for "table-driven tests for hit-testing with known layout coordinates from seed 42." No `renderer/layout_test.go` exists yet.

### Gap 2 — Mouse drag model-state tests (`tui/board_test.go`)

The design doc calls for "teatest model state tests for mouse-driven selection and placement." The existing mouse tests only cover stock-click and miss-click. Two scenarios are missing:
- Mouse click on a face-up tableau card → drag pick-up (Dragging becomes true)
- Two mouse clicks completing a valid tableau-to-tableau move via drag flow

---

## 4. Layout Geometry for Seed-42 Tests

All hit-test coordinates below assume `termWidth=78` (the default `MinTermWidth`) and the seed-42 draw-1 deal.

### 4.1 Pile Origins

```
topRow  = 2               (header row 0 + blank spacer row 1)
tabRow  = 2 + 7 + 1 = 10  (topRow + CardHeight + blank spacer)

foundationStartX(78) = 78 - (4×9 + 3×1) = 78 - 39 = 39

Pile         X    Y
──────────────────────
Stock        0    2
Waste        10   2
Foundation0  39   2
Foundation1  49   2
Foundation2  59   2
Foundation3  69   2
Tableau0     0    10
Tableau1     10   10
Tableau2     20   10
Tableau3     30   10
Tableau4     40   10
Tableau5     50   10
Tableau6     60   10
```

### 4.2 Tableau Column Layout (Seed-42 Draw-1)

At deal time each column N has N+1 cards total: N face-down stubs followed by 1 face-up card.

| Column | FD count | FU count | FD stub rows | FU card rows |
|--------|----------|----------|--------------|--------------|
| T0 | 0 | 1 | — | 10..16 |
| T1 | 1 | 1 | 10 | 11..17 |
| T2 | 2 | 1 | 10,11 | 12..18 |
| T3 | 3 | 1 | 10,11,12 | 13..19 |
| T4 | 4 | 1 | 10..13 | 14..20 |
| T5 | 5 | 1 | 10..14 | 15..21 |
| T6 | 6 | 1 | 10..15 | 16..22 |

Face-down stubs: each stub `ci` occupies exactly **row = tabRow + ci**; the hit condition is `y == row`.
Face-up card: the last (and only) face-up card occupies **CardHeight = 7** rows; hit condition is `row ≤ y < row+7`.

### 4.3 Card Index Mapping

`pileHitTestWithWidth` returns `cardIndex` = position in `pile.Cards` slice:

| Pile | Card being hit | Returned cardIndex |
|------|---------------|--------------------|
| T0 | only card (fu) | 0 |
| T1 | fd stub | 0 |
| T1 | fu card | 1 |
| T2 | fd stub 0 | 0 |
| T2 | fd stub 1 | 1 |
| T2 | fu card | 2 |
| T6 | fd stub 0 | 0 |
| T6 | fd stub 5 | 5 |
| T6 | fu card | 6 |

---

## 5. `renderer/layout_test.go` — Implementation

Create a new file `renderer/layout_test.go` in `package renderer`.

### 5.1 Test Helper

```go
package renderer

import (
    "testing"
    "solituire/engine"
)

func newSeed42DrawState() *engine.GameState {
    deck := engine.NewDeck()
    engine.Shuffle(deck, 42)
    state := engine.Deal(deck, 1)
    state.Seed = 42
    return state
}
```

### 5.2 Table-Driven Test: `TestPileHitTestWithWidth`

```go
func TestPileHitTestWithWidth(t *testing.T) {
    state := newSeed42DrawState()

    type wantHit struct {
        pile      engine.PileID
        cardIndex int
    }

    tests := []struct {
        name string
        x, y int
        want *wantHit // nil = expect miss (ok=false)
    }{
        // ── Stock ────────────────────────────────────────────────────────────
        {"stock top-left corner",      0,  2, &wantHit{engine.PileStock, 0}},
        {"stock center",               4,  5, &wantHit{engine.PileStock, 0}},
        {"stock bottom-right corner",  8,  8, &wantHit{engine.PileStock, 0}},

        // ── Waste (empty at deal time, wasteVisCount clamps to 1) ─────────
        {"waste left edge",            10, 2, &wantHit{engine.PileWaste, 0}},
        {"waste center",               14, 5, &wantHit{engine.PileWaste, 0}},
        {"waste right edge",           18, 8, &wantHit{engine.PileWaste, 0}},

        // ── Foundations ──────────────────────────────────────────────────
        {"foundation 0",               39, 2, &wantHit{engine.PileFoundation0, 0}},
        {"foundation 1",               49, 4, &wantHit{engine.PileFoundation1, 0}},
        {"foundation 2",               59, 6, &wantHit{engine.PileFoundation2, 0}},
        {"foundation 3",               69, 2, &wantHit{engine.PileFoundation3, 0}},

        // ── Tableau T0 (0 fd, 1 fu at rows 10..16) ───────────────────────
        {"T0 fu top row",          4, 10, &wantHit{engine.PileTableau0, 0}},
        {"T0 fu bottom row",       4, 16, &wantHit{engine.PileTableau0, 0}},

        // ── Tableau T1 (1 fd at row 10, 1 fu at rows 11..17) ─────────────
        {"T1 fd stub",             14, 10, &wantHit{engine.PileTableau1, 0}},
        {"T1 fu card",             14, 11, &wantHit{engine.PileTableau1, 1}},
        {"T1 fu last row",         14, 17, &wantHit{engine.PileTableau1, 1}},

        // ── Tableau T2 (2 fd, 1 fu) ─────────────────────────────────────
        {"T2 fd stub 0",           24, 10, &wantHit{engine.PileTableau2, 0}},
        {"T2 fd stub 1",           24, 11, &wantHit{engine.PileTableau2, 1}},
        {"T2 fu card",             24, 12, &wantHit{engine.PileTableau2, 2}},

        // ── Tableau T6 (6 fd stubs rows 10..15, 1 fu rows 16..22) ────────
        {"T6 fd stub 0",           64, 10, &wantHit{engine.PileTableau6, 0}},
        {"T6 fd stub 5",           64, 15, &wantHit{engine.PileTableau6, 5}},
        {"T6 fu card top row",     64, 16, &wantHit{engine.PileTableau6, 6}},
        {"T6 fu card bottom row",  64, 22, &wantHit{engine.PileTableau6, 6}},
        {"T6 last valid x",        68, 16, &wantHit{engine.PileTableau6, 6}},

        // ── Misses ───────────────────────────────────────────────────────
        {"above top row",          4,  1,  nil},
        {"stock-waste gap",        9,  2,  nil},
        {"gap before foundation",  38, 2,  nil},
        {"gap below top row",      4,  9,  nil},  // tabRow=10; row 9 is spacer
        {"right of T6",            69, 10, nil},
        {"below T0 fu card",       4,  17, nil},  // T0 fu ends at row 16
        {"far below tableau",      4,  30, nil},
    }

    termWidth := MinTermWidth // 78

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pile, cardIdx, ok := PileHitTestWithWidth(tt.x, tt.y, state, termWidth)
            if tt.want == nil {
                if ok {
                    t.Errorf("expected miss at (%d,%d), got pile=%v cardIndex=%d", tt.x, tt.y, pile, cardIdx)
                }
                return
            }
            if !ok {
                t.Errorf("expected hit at (%d,%d), got miss", tt.x, tt.y)
                return
            }
            if pile != tt.want.pile {
                t.Errorf("(%d,%d): pile = %v, want %v", tt.x, tt.y, pile, tt.want.pile)
            }
            if cardIdx != tt.want.cardIndex {
                t.Errorf("(%d,%d): cardIndex = %d, want %d", tt.x, tt.y, cardIdx, tt.want.cardIndex)
            }
        })
    }
}
```

### 5.3 Test: `TestPileHitTest_DefaultWidth`

Verify that `PileHitTest` (no explicit width) is equivalent to `PileHitTestWithWidth(x, y, state, MinTermWidth)`:

```go
func TestPileHitTest_DefaultWidth(t *testing.T) {
    state := newSeed42DrawState()
    // Stock is at (0,2) regardless of terminal width.
    p1, c1, ok1 := PileHitTest(4, 5, state)
    p2, c2, ok2 := PileHitTestWithWidth(4, 5, state, MinTermWidth)
    if ok1 != ok2 || p1 != p2 || c1 != c2 {
        t.Errorf("PileHitTest and PileHitTestWithWidth(MinTermWidth) disagree: "+
            "got (%v,%d,%v) vs (%v,%d,%v)", p1, c1, ok1, p2, c2, ok2)
    }
}
```

### 5.4 Test: `TestPileHitTestWaste_Draw3Expansion`

In draw-3 mode after flipping stock, three cards are visible side-by-side. The hit region expands to `wasteVisCount * CardWidth`. Verify a click on the rightmost of 3 visible waste cards still registers as PileWaste:

```go
func TestPileHitTestWaste_Draw3Expansion(t *testing.T) {
    // Build a draw-3 game and flip stock three times to get 3 waste cards.
    deck := engine.NewDeck()
    engine.Shuffle(deck, 42)
    state := engine.Deal(deck, 3)
    state.DrawCount = 3

    // Flip stock once (each flip in draw-3 moves 3 cards to waste).
    flipCmd := &engine.FlipStockCmd{}
    _ = flipCmd.Execute(state)

    visCount := len(state.Waste.VisibleCards())
    if visCount < 2 {
        // Not enough visible cards to test expansion — skip gracefully.
        return
    }

    // The rightmost visible card is at X = WasteOriginX + (visCount-1)*CardWidth + 1.
    // With termWidth=MinTermWidth, wasteOriginX = CardWidth + ColGap = 9 + 1 = 10.
    wasteOriginX := CardWidth + ColGap // 10
    rightCardX := wasteOriginX + (visCount-1)*CardWidth + 1

    pile, _, ok := PileHitTestWithWidth(rightCardX, 4, state, MinTermWidth)
    if !ok || pile != engine.PileWaste {
        t.Errorf("rightmost draw-3 waste card at x=%d: got pile=%v ok=%v, want PileWaste ok=true",
            rightCardX, pile, ok)
    }
}
```

---

## 6. `tui/board_test.go` — Mouse Drag Tests

Add the following two tests to `tui/board_test.go`. They use the existing `newBoard()` helper and the `testEngine` already defined in that file.

### 6.1 Helper: `clickPile`

Add a small helper (unexported, test-only) to compute a click coordinate at the natural (last) card of any pile and deliver a mouse event to the board:

```go
// clickPile delivers a left-press mouse click aimed at the natural card of
// pileID in board's current state. It uses the board's actual terminal width
// so that right-justified foundations are positioned correctly.
//
// Coordinate computation mirrors renderer.pileOrigins:
//   topRow  = 2
//   tabRow  = topRow + renderer.CardHeight + 1 = 10
func clickPile(board BoardModel, pileID engine.PileID, cardIdx int) BoardModel {
    const tabRow = 2 + renderer.CardHeight + 1  // 10

    var x, y int
    state := board.eng.State()

    switch {
    case pileID == engine.PileStock:
        x, y = renderer.CardWidth/2, 2+renderer.CardHeight/2

    case pileID == engine.PileWaste:
        x = renderer.CardWidth + renderer.ColGap + renderer.CardWidth/2
        y = 2 + renderer.CardHeight/2

    case isFoundationPile(pileID):
        fi := int(pileID - engine.PileFoundation0)
        fStartX := board.width - (4*renderer.CardWidth + 3*renderer.ColGap)
        x = fStartX + fi*(renderer.CardWidth+renderer.ColGap) + renderer.CardWidth/2
        y = 2 + renderer.CardHeight/2

    case isTableauPile(pileID):
        col := int(pileID - engine.PileTableau0)
        x = col*(renderer.CardWidth+renderer.ColGap) + renderer.CardWidth/2
        pile := state.Tableau[col]
        fdCount := pile.FaceDownCount()
        row := tabRow
        // Advance through face-down stubs (1 row each).
        for ci := 0; ci < fdCount; ci++ {
            if ci == cardIdx {
                y = row
                goto done
            }
            row++
        }
        // Advance through face-up cards.
        fuCards := pile.FaceUpCards()
        for fi := range fuCards {
            height := 2
            if fi == len(fuCards)-1 {
                height = renderer.CardHeight
            }
            ci := fdCount + fi
            if ci == cardIdx {
                y = row + height/2
                goto done
            }
            row += height
        }
        // Default: click at the start of the tableau column.
        y = tabRow
    done:
    }

    updated, _ := board.Update(tea.MouseMsg{
        Action: tea.MouseActionPress,
        Button: tea.MouseButtonLeft,
        X:      x,
        Y:      y,
    })
    return updated.(BoardModel)
}
```

### 6.2 Test: `TestBoardMousePickUp`

Verifies that a left-click on a face-up tableau card starts a drag (Dragging becomes true) and sets DragSource to the clicked pile.

```go
func TestBoardMousePickUp(t *testing.T) {
    board, eng := newBoard()

    // T0 has exactly 1 card which is face-up; click it.
    state := eng.State()
    if state.Tableau[0].FaceDownCount() != 0 || len(state.Tableau[0].Cards) == 0 {
        t.Skip("T0 layout does not match seed-42 expectation")
    }

    board = clickPile(board, engine.PileTableau0, 0)

    if !board.cursor.Dragging {
        t.Fatal("mouse click on face-up card must set Dragging=true")
    }
    if board.cursor.DragSource != engine.PileTableau0 {
        t.Errorf("DragSource: got %v, want PileTableau0", board.cursor.DragSource)
    }
    if board.cursor.DragCardCount < 1 {
        t.Errorf("DragCardCount must be >= 1, got %d", board.cursor.DragCardCount)
    }
}
```

### 6.3 Test: `TestBoardMouseDragPlace_Valid`

Verifies that two sequential mouse clicks (pick up then place) execute a valid tableau-to-tableau move identically to the keyboard flow. Mirrors `TestBoardDragPlace_Valid`.

```go
func TestBoardMouseDragPlace_Valid(t *testing.T) {
    board, eng := newBoard()

    // Find a valid tableau-to-tableau move.
    var move engine.Move
    for _, m := range eng.ValidMoves() {
        if isTableauPile(m.From) && isTableauPile(m.To) {
            move = m
            break
        }
    }
    if move.From == 0 && move.To == 0 {
        t.Skip("no tableau-to-tableau move available with seed 42")
    }

    srcCol := int(move.From - engine.PileTableau0)
    srcLen := len(eng.State().Tableau[srcCol].Cards)
    srcCardIdx := srcLen - move.CardCount

    // Click 1: pick up.
    board = clickPile(board, move.From, srcCardIdx)
    if !board.cursor.Dragging {
        t.Fatal("first mouse click must pick up card (Dragging=true)")
    }

    // Click 2: place on destination.
    destCardIdx := naturalCardIndex(move.To, eng.State())
    board = clickPile(board, move.To, destCardIdx)

    if board.cursor.Dragging {
        t.Error("second mouse click must clear Dragging")
    }
    afterLen := len(eng.State().Tableau[srcCol].Cards)
    if afterLen >= srcLen {
        t.Errorf("source pile must shrink: before=%d after=%d", srcLen, afterLen)
    }
}
```

### 6.4 Test: `TestBoardMouseDragPlace_FaceDownCard`

Verifies that clicking a face-down card does not start a drag (face-down cards are not valid drag sources). Mirrors `TestBoardEnterOnFaceDownTableauCard`.

```go
func TestBoardMouseDragPlace_FaceDownCard(t *testing.T) {
    board, eng := newBoard()

    // Find any column with a face-down card.
    targetCol := -1
    for col := 0; col < 7; col++ {
        if eng.State().Tableau[col].FaceDownCount() > 0 {
            targetCol = col
            break
        }
    }
    if targetCol < 0 {
        t.Skip("no face-down cards at deal time")
    }

    pileID := engine.PileTableau0 + engine.PileID(targetCol)
    board = clickPile(board, pileID, 0) // cardIdx=0 is always face-down

    if board.cursor.Dragging {
        t.Error("mouse click on face-down card must not start a drag")
    }
}
```

---

## 7. Execution Order

1. **Create `renderer/layout_test.go`** — add `newSeed42DrawState` helper, then tests §5.2–5.4.
2. **Run `go test ./renderer/ -run TestPileHitTest`** — verify all hit-test cases pass before touching `tui/`.
3. **Add `clickPile` helper to `tui/board_test.go`** — add §6.1 after the existing `sendRune` helper.
4. **Add §6.2–6.4 tests to `tui/board_test.go`**.
5. **Run `go test ./tui/ -run TestBoardMouse`** — new mouse drag tests must pass.
6. **Run `go test ./...`** — full suite must be green (no regressions).
7. Commit and push on `claude/agent-c-t16-plan-o9aoi`.

---

## 8. Edge Cases and Constraints

| Situation | Handling |
|-----------|----------|
| Click on face-down card | `dragCount` returns 0 → `Dragging` stays false |
| Click on empty foundation | `dragCount` returns 0 → `Dragging` stays false |
| Click on stock while dragging | `ActionSelect` hit-tests to PileStock → `handleSelect` calls `flipStock` (stock special case), cancels drag first |
| Miss-click (outside all piles) | `PileHitTestWithWidth` returns `ok=false` → `break` before `handleSelect`, cursor unchanged |
| Draw-3 waste click on non-top card | Hit-test returns `PileWaste, 0` regardless — waste always has cardIndex 0 since only top card is playable |
| Terminal narrower than MinTermWidth | Renderer shows "too small" message; no game board is rendered, so no valid click targets exist |
| `clickPile` helper coordinates | Use `board.width` (set via `WindowSizeMsg`) for foundation x-offset so tests work at any terminal width |

---

## 9. File Checklist

```
renderer/
└── layout_test.go   ← NEW  (3 hit-test functions, ~100 lines)

tui/
└── board_test.go    ← MODIFY  (add clickPile helper + 3 new test functions)
```

No changes to production code — all T16 implementation was completed during T11/T13.

---

## 10. Pre-existing Test Compatibility

No existing tests are affected by these additions. The new tests are purely additive.

The existing `TestBoardMouseClickMovesAndSelects` and `TestBoardMouseClickOutsidePile` remain as is and continue to cover the stock-click and miss-click paths respectively. The new `TestBoardMousePickUp` and `TestBoardMouseDragPlace_Valid` extend coverage to the drag flow.

---

## 11. T17 Handoff Note

After T16 merges:
- Mouse input is fully tested end-to-end: translation → hit-test → board integration.
- T17 (Auto-Complete) touches `tui/board.go` only — no conflict with this work.
- T18 (Win Celebration) touches `tui/celebration.go` only — no conflict.
- The remaining placeholder is `ScreenWin` → `tui/celebration.go`.
