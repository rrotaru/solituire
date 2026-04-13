# BUGFIX B5 — Tableau Column Gaps and Trailing Area Lack Board Background

## 1. Issue

The 1-character gaps between tableau columns, and the space between the last tableau
column and the right edge of the terminal, are rendered in the **terminal's default
background colour** rather than the board's green (or theme-appropriate) background.
This produces visible vertical stripes of wrong colour between every column, and a
mismatched strip to the right of column 7.

The same problem exists in `renderTopRow`: the gap between the waste pile and the
foundation piles, and the 1-character gaps between foundation piles, are also unstyled
plain spaces.

Observed symptom: "The tableau background color also stops in the empty space after the
final column is printed AND in between each column."

Affected functions:
- `renderer/renderer.go` — `renderTableau` (line 136)
- `renderer/renderer.go` — `renderTopRow` (lines 98, 110, 113, 115, 117)

---

## 2. How to Reproduce

1. `go run .`
2. Set the terminal background to any colour that differs visibly from the board
   background (e.g. default black terminal background vs. Classic theme's `#35654d`
   green).
3. Look at the tableau area. Vertical stripes of the wrong background colour will be
   visible between every tableau column, and to the right of the final column.

Even without changing terminal settings the issue is visible if the terminal background
differs at all from `BoardBackground` — which is true for every user whose terminal is
not configured to exactly match the theme.

---

## 3. Root Cause

In `renderTableau`, column gaps are constructed as unstyled Go strings:

```go
parts = append(parts, strings.Repeat(" ", ColGap))
```

`lipgloss.JoinHorizontal` assembles these strings alongside card renders. Each card
render ends with a Lipgloss reset (`\x1b[0m`) that clears all active colours. The plain
space that follows has no colour information and therefore inherits the terminal default
background.

Similarly in `renderTopRow`:

```go
leftSection := lipgloss.JoinHorizontal(lipgloss.Top,
    stock,
    strings.Repeat(" ", ColGap),   // ← plain, no background
    waste,
)

gap := strings.Repeat(" ", gapWidth)  // ← plain, no background

rightSection := lipgloss.JoinHorizontal(lipgloss.Top,
    f0,
    strings.Repeat(" ", ColGap),   // ← plain, no background
    f1,
    strings.Repeat(" ", ColGap),   // ← plain, no background
    f2,
    strings.Repeat(" ", ColGap),   // ← plain, no background
    f3,
)
```

The outer `Background(r.theme.BoardBackground).Width(r.width)` style applied in
`Render()` is intended to fill the board, but Lipgloss cannot retroactively recolor
characters that have already been given explicit resets; it can only pad lines to the
correct width.

---

## 4. Fix

Add a helper method on `Renderer` that produces board-background-coloured space strings,
and replace every plain `strings.Repeat(" ", ...)` gap in `renderTableau` and
`renderTopRow` with it.

### 4.1 Add `boardGap` helper to `Renderer` (`renderer/renderer.go`)

Place this after the existing `centerString` helper:

```go
// boardGap returns n spaces explicitly styled with the board background colour.
// Use this instead of plain strings.Repeat(" ", n) for all gap/padding between
// rendered card cells so the board background shows through after ANSI resets.
func (r *Renderer) boardGap(n int) string {
    return lipgloss.NewStyle().Background(r.theme.BoardBackground).Render(strings.Repeat(" ", n))
}
```

### 4.2 Update `renderTableau` (`renderer/renderer.go`)

```go
// Before:
parts = append(parts, strings.Repeat(" ", ColGap))

// After:
parts = append(parts, r.boardGap(ColGap))
```

### 4.3 Update `renderTopRow` (`renderer/renderer.go`)

Replace all four plain gap strings:

```go
leftSection := lipgloss.JoinHorizontal(lipgloss.Top,
    stock,
    r.boardGap(ColGap),   // ← was strings.Repeat(" ", ColGap)
    waste,
)

gap := r.boardGap(gapWidth)  // ← was strings.Repeat(" ", gapWidth)

rightSection := lipgloss.JoinHorizontal(lipgloss.Top,
    f0,
    r.boardGap(ColGap),   // ← was strings.Repeat(" ", ColGap)
    f1,
    r.boardGap(ColGap),   // ← was strings.Repeat(" ", ColGap)
    f2,
    r.boardGap(ColGap),   // ← was strings.Repeat(" ", ColGap)
    f3,
)
```

---

## 5. Verification

1. `go build ./...` — must compile without errors.
2. `go run .` on a terminal whose default background differs from the board background
   (virtually any standard terminal). The 1-character gaps between every tableau column
   must show the board's theme colour, not a stripe of a different colour.
3. The large gap in the top row (between waste pile and foundations) must also be
   board-coloured.
4. Press `t` to cycle through all five themes. Gaps must match `BoardBackground` on each.
5. Verify the board still fills to the terminal width correctly (no regression in the
   outer `Width(r.width)` padding).
