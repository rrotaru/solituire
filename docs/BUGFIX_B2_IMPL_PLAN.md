# BUGFIX B2 — Non-Standard Card Corner Layout and Missing Center Suit

## 1. Issue

Face-up cards display their rank and suit symbols at **opposite corners** of the card:
rank at top-left, suit at top-right, suit at bottom-left, rank at bottom-right.
Standard playing cards place **rank and suit together** at each corner (e.g., `K♠` at
top-left, `♠K` at bottom-right). Additionally, the three center rows of every face-up
card are **completely blank** — real solitaire cards show the suit symbol prominently in
the middle, making the card face easier to read at a glance.

Current card layout:
```
┌───────┐
│K     ♠│   rank at 0-1, suit at 6
│       │
│       │   (blank)
│       │
│♠     K│   suit at 0, rank at 5-6
└───────┘
```

Target layout:
```
┌───────┐
│K♠     │   rank+suit together at 0-2
│       │
│   ♠   │   center suit
│       │
│     ♠K│   suit+rank together at 4-6
└───────┘
```

Affected functions:
- `renderer/card.go` — `renderFaceUp` (lines 141–157)
- `renderer/card.go` — `cardPeekLines` (lines 197–203)

---

## 2. How to Reproduce

Run `go run .` and observe any face-up tableau card. The suit symbol is isolated at the
far opposite corner from the rank, and the card center is blank.

---

## 3. Root Cause

In `renderFaceUp` the inner content is constructed as:

```go
// line0: rank at positions 0-1, suit at position 6
line0 := rankPad + strings.Repeat(" ", innerWidth-2-1) + suitStyle.Render(suit)
// line4: suit at position 0, rank at positions 5-6
line4 := suitStyle.Render(suit) + strings.Repeat(" ", innerWidth-1-2) + rankPadR
blank := strings.Repeat(" ", innerWidth)

// r1, r2, r3 all render `blank` — no center symbol
r1 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
r2 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
r3 := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
```

The rank and suit are placed at opposite ends of each content line, and none of the
rows contains a centered suit symbol.

---

## 4. Fix

### 4.1 Rewrite inner content lines in `renderFaceUp` (`renderer/card.go`)

Requires the `rankStyle` added by BUGFIX_B1. Use `.Inline(true)` on every embedded
style to prevent inner resets from stripping the card background (see BUGFIX_B4).

Width arithmetic (innerWidth = 7):
- `line0`: `rankPad` (2) + `suit` (1) + spaces (4) = 7
- `line2`: spaces (3) + `suit` (1) + spaces (3) = 7
- `line4`: spaces (4) + `suit` (1) + `rankPadR` (2) = 7

```go
// line0: "K♠     " — rank+suit at top-left corner
line0 := rankStyle.Inline(true).Render(rankPad) +
         suitStyle.Inline(true).Render(suit) +
         strings.Repeat(" ", innerWidth-3)

// line2: "   ♠   " — suit centered
line2 := strings.Repeat(" ", 3) +
         suitStyle.Inline(true).Render(suit) +
         strings.Repeat(" ", 3)

// line4: "     ♠K" — suit+rank at bottom-right corner
line4 := strings.Repeat(" ", innerWidth-3) +
         suitStyle.Inline(true).Render(suit) +
         rankStyle.Inline(true).Render(rankPadR)
```

Then update the six row builds, replacing the old blank `r2` with the center line:

```go
top := borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
r0  := borderStyle.Render("│") + bgStyle.Render(line0) + borderStyle.Render("│")
r1  := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
r2  := borderStyle.Render("│") + bgStyle.Render(line2) + borderStyle.Render("│")   // ← was blank
r3  := borderStyle.Render("│") + bgStyle.Render(blank) + borderStyle.Render("│")
r4  := borderStyle.Render("│") + bgStyle.Render(line4) + borderStyle.Render("│")
bot := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")
```

### 4.2 Update `cardPeekLines` (`renderer/card.go`)

Peek lines show only the top two rows of a card, so update `line0` to put rank+suit
together at the top-left:

```go
line0 := rankStyle.Inline(true).Render(rankPad) +
         suitStyle.Inline(true).Render(suit) +
         strings.Repeat(" ", innerWidth-3)
```

---

## 5. Verification

1. `go build ./...` — must compile without errors.
2. `go run .` — every face-up card must show:
   - Rank and suit symbol together at the **top-left** corner (`K♠`, `A♥`, `10♦`, etc.)
   - A suit symbol **centered** in the middle row
   - Suit and rank together at the **bottom-right** corner (`♠K`, `♥A`, etc.)
3. Press `t` to cycle themes — layout must be consistent on all five themes.
4. Check peek-mode cards (all non-bottom face-up cards in a tableau stack) — they must
   show `K♠` (rank+suit) at top-left, not `K` alone.
5. Verify the suit colors (red/black) are still applied correctly.
