# BUGFIX B3 — Face-Down Stub Uses Inconsistent Fill Style

## 1. Issue

The single-line stub used to represent face-down cards in the tableau (`cardStubTop`)
renders as `┌░░░░░░░┐` — a top border with **block characters** (`░`) between the
corners. The rest of the border system (face-up cards, face-down full cards, empty
slots) uses clean horizontal-dash borders: `┌───────┐`. This inconsistency makes the
stacked stubs in the tableau look visually different from all other border lines in the
game.

Current:
```
┌░░░░░░░┐   ← stub (block chars inside top border)
┌░░░░░░░┐   ← stub
┌───────┐   ← top border of a full face-down card
│░░░░░░░│
│░░░░░░░│
...
```

Target:
```
┌───────┐   ← stub (dashes, consistent with all other top borders)
┌───────┐   ← stub
┌───────┐   ← top border of a full face-down card
│░░░░░░░│
│░░░░░░░│
...
```

Affected function:
- `renderer/card.go` — `cardStubTop` (lines 162–166)

---

## 2. How to Reproduce

1. `go run .`
2. Look at any tableau column with two or more face-down cards (columns 2–7 at game
   start). The stacked stubs show a dark dotted fill (`░`) between the corner characters,
   while all other card border lines use `─`.

---

## 3. Root Cause

`cardStubTop` was implemented to visually communicate "face-down card here" by filling
the single stub line with the `CardFaceDown` colour's block pattern:

```go
func cardStubTop(t theme.Theme) string {
    borderStyle := lipgloss.NewStyle().Foreground(t.CardBorder)
    fillStyle := lipgloss.NewStyle().Foreground(t.CardFaceDown)
    return borderStyle.Render("┌") + fillStyle.Render(strings.Repeat("░", innerWidth)) + borderStyle.Render("┐")
}
```

This was an intentional shortcut but produces a line that clashes with all other
`┌───────┐` style borders throughout the renderer.

---

## 4. Fix

Simplify `cardStubTop` to render a plain `┌───────┐` line using the card border color,
identical in style to the top line of `renderFaceDown`:

```go
func cardStubTop(t theme.Theme) string {
    borderStyle := lipgloss.NewStyle().Foreground(t.CardBorder)
    return borderStyle.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
}
```

The unused `fillStyle` variable can be removed entirely.

---

## 5. Verification

1. `go build ./...` — must compile without errors.
2. `go run .` — all stacked stub lines in the tableau must display as clean `┌───────┐`
   horizontal lines, visually consistent with the top border of every other card.
3. Press `t` to cycle themes — stubs must use `t.CardBorder` colour on each theme.
4. Confirm that face-down cards in the **stock pile** are unaffected (stock uses
   `renderFaceDownWithState`, not `cardStubTop`).
