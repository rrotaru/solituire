# BUGFIX B4 — Card Background Drops Out After Suit Symbol on Bottom Row

## 1. Issue

On face-up cards, the white card background colour disappears **after the suit symbol**
on the bottom content row (`line4`). The trailing spaces and the rank character that
follow the suit symbol are rendered against the terminal's default background instead of
the card's white background. This produces a visible colour break at the bottom of every
face-up card.

Observed symptom: "The card face colors stop on the bottom line of the card after the
suit is printed."

Affected functions:
- `renderer/card.go` — `renderFaceUp` (`line4` construction, line 146)
- `renderer/card.go` — `cardPeekLines` (`line0` construction, line 198) — milder, but
  the suit is at the end of `line0`, so it affects the trailing reset there too

---

## 2. How to Reproduce

1. `go run .`
2. Look at any face-up tableau card where the suit is a visible colour (red suits are
   easiest to spot). The bottom inner row of the card should be fully white. Instead,
   the white background ends immediately after the suit glyph, and the remainder of
   the row (trailing spaces + rank) shows the board's green background.

---

## 3. Root Cause

`suitStyle.Render(suit)` produces a complete ANSI sequence that **includes a trailing
reset** (`\x1b[0m`):

```
\x1b[38;2;r;g;bm  ♠  \x1b[0m
```

This reset clears **all** active styling, including the `Background(t.CardBackground)`
applied by the outer `bgStyle`. In `line4`, the suit appears first:

```go
line4 := suitStyle.Render(suit) + strings.Repeat(" ", innerWidth-1-2) + rankPadR
//        ^--- \x1b[0m here resets background for everything after this
```

When `bgStyle.Render(line4)` wraps `line4`, Lipgloss sets the background at the start
of the string, but the embedded `\x1b[0m` from `suitStyle` cancels it mid-line. The
four trailing spaces and `rankPadR` are therefore rendered without any background.

The same mechanism affects `line0` in `renderFaceUp` and `line0` in `cardPeekLines`
(suit is at the end of those lines, so the truncated background only affects the
newline/reset boundary, which is less visually obvious but still incorrect).

---

## 4. Fix

Use Lipgloss's `.Inline(true)` modifier on `suitStyle` (and `rankStyle` from BUGFIX_B1)
when the rendered result will be **embedded inside another styled string**. The inline
flag suppresses the trailing `\x1b[0m` reset, allowing the outer `bgStyle` background
to remain active for the rest of the line.

### 4.1 `renderFaceUp` (`renderer/card.go`)

Change every embedded `suitStyle.Render(...)` and `rankStyle.Render(...)` call within
line construction to use `.Inline(true)`:

```go
// Before:
line0 := rankPad + strings.Repeat(" ", innerWidth-2-1) + suitStyle.Render(suit)
line4 := suitStyle.Render(suit) + strings.Repeat(" ", innerWidth-1-2) + rankPadR

// After:
line0 := rankStyle.Inline(true).Render(rankPad) +
         suitStyle.Inline(true).Render(suit) +
         strings.Repeat(" ", innerWidth-3)

line4 := strings.Repeat(" ", innerWidth-3) +
         suitStyle.Inline(true).Render(suit) +
         rankStyle.Inline(true).Render(rankPadR)
```

(The line0/line4 layout changes are also part of BUGFIX_B2; applying both fixes
together is the recommended approach.)

The key invariant: **every** `Render()` call whose output will be concatenated with
other strings before being passed to another `Render()` must use `.Inline(true)`.

### 4.2 `cardPeekLines` (`renderer/card.go`)

Same rule applies:

```go
// Before:
line0 := rankPad + strings.Repeat(" ", innerWidth-2-1) + suitStyle.Render(suit)

// After:
line0 := rankStyle.Inline(true).Render(rankPad) +
         suitStyle.Inline(true).Render(suit) +
         strings.Repeat(" ", innerWidth-3)
```

### 4.3 `renderEmptyWithSuit` — no change needed

The suit in `renderEmptyWithSuit` is rendered via `centerInWidth(textStyle.Render(suit), innerWidth)`. This style is the last (and only) styled item in the `mid` line, and the outer `borderStyle` wraps only the border characters, not the full line content. This path does not exhibit the same truncation symptom.

---

## 5. Verification

1. `go build ./...` — must compile without errors.
2. `go run .` — look at face-up cards with red suits (hearts/diamonds). The bottom row
   of every card must be **entirely white** from border to border, with no background
   colour break after the suit symbol.
3. Verify for both red and black suits across all five themes (`t` to cycle).
4. Verify peek-mode cards (stacked face-up cards showing only 2 lines) also have a
   consistent white background on their content row.
