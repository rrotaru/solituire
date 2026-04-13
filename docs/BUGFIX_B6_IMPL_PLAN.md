# BUGFIX B6 — Foundation Piles Positioned Relative to Terminal Width Instead of Tableau Width

## 1. Issue

The four foundation piles in the top row are right-justified to the **full terminal
width** rather than to the width of the tableau below them. On any terminal wider than
the 7-column tableau (69 characters), the foundations shift far to the right — beyond
the last tableau column — and appear beside or after the tableau rather than above it.

Observed symptom: "The foundation piles are far off to the right after the tableau
rather than above the tableau."

Affected function:
- `renderer/renderer.go` — `renderTopRow` (line 106)

---

## 2. How to Reproduce

1. `go run .` in any terminal wider than ~78 characters (e.g. a maximised window at
   120 or 150 characters wide).
2. Observe the top row. The stock and waste piles appear on the left; the foundation
   piles appear at the far right edge of the terminal — well past the right boundary
   of the 7 tableau columns below them.

At exactly 78 characters the layout appears correct by coincidence (the default
`MinTermWidth` happens to align the foundations roughly over the tableau). At any wider
width the misalignment becomes apparent.

---

## 3. Root Cause

In `renderTopRow`:

```go
foundationsWidth := 4*CardWidth + 3*ColGap
gapWidth := r.width - lipgloss.Width(leftSection) - foundationsWidth
```

`r.width` is the actual terminal width (e.g. 150). This means the gap between the
waste pile and the foundations is scaled to push the foundations to the terminal's
absolute right edge, regardless of how wide the tableau is.

The tableau is always exactly `7*CardWidth + 6*ColGap = 7*9 + 6*1 = 69` characters
wide. The stock+waste section is `9 + 1 + 9 = 19` characters wide. The four foundations
are `4*9 + 3*1 = 39` characters wide. Together that is exactly 69 characters — meaning
the top row and tableau should always be the same width, with no dependency on terminal
width.

---

## 4. Fix

Replace the `r.width`-based gap calculation with a fixed tableau-width-based one in
`renderTopRow` (`renderer/renderer.go`):

```go
// Before:
foundationsWidth := 4*CardWidth + 3*ColGap
gapWidth := r.width - lipgloss.Width(leftSection) - foundationsWidth
if gapWidth < 1 {
    gapWidth = 1
}

// After:
tableauWidth    := 7*CardWidth + 6*ColGap        // = 69
foundationsWidth := 4*CardWidth + 3*ColGap        // = 39
gapWidth := tableauWidth - lipgloss.Width(leftSection) - foundationsWidth
if gapWidth < 1 {
    gapWidth = 1
}
```

With `leftSection` visual width of 19 (stock 9 + gap 1 + waste 9):
```
gapWidth = 69 - 19 - 39 = 11
```

The top row is now always 69 characters wide, exactly matching the tableau, independent
of terminal width. The outer `Background(r.theme.BoardBackground).Width(r.width)` style
in `Render()` continues to fill any remaining terminal width with the board background
colour to the right.

Note: if draw-3 mode is active, `leftSection` can be up to 3 cards wide (27 chars),
which would give `gapWidth = 69 - 27 - 39 = 3`. The `if gapWidth < 1` guard handles
any extreme edge case. The draw-3 waste pile rendering is the same as before — only
the gap reference width changes.

---

## 5. Verification

1. `go build ./...` — must compile without errors.
2. `go run .` at several terminal widths: 80, 100, 120, 150, 200 characters.
   On every width, the rightmost foundation pile must be **directly above** the
   rightmost tableau column (column 7), not shifted further right.
3. Verify the board background still fills the full terminal width to the right of the
   game content (handled by the outer `Width(r.width)` render, unaffected by this fix).
4. Test draw-3 mode: start with `--draw 3` or press the relevant key. The waste pile
   expands; confirm the gap shrinks accordingly and foundations remain above the
   rightmost tableau columns.
5. Press `t` to cycle themes — layout must be consistent on each.
6. Verify the `PileHitTestWithWidth` function in `renderer/layout.go` is **not**
   affected: hit-testing uses `foundationStartX(termWidth)` which correctly computes
   the on-screen pixel position of foundations based on `termWidth`. That function
   should remain unchanged. The layout fix only changes the *rendered gap* in the
   visual output; the hit-test geometry may need a matching fix if it also uses
   `r.width` as the reference — verify both produce consistent results.
