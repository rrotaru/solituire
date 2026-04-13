# BUGFIX B1 — Rank Text Has No Explicit Foreground Color

## 1. Issue

Rank text on face-up cards (`K`, `A`, `2`–`10`, `J`, `Q`) has **no explicit foreground
color**. The color falls back to the terminal's default foreground. On any terminal
configured with a light default foreground (white, cream, `#f8f8f2`, etc. — common in
dark-theme profiles), rank characters become **invisible** against the white/light card
background used by every theme.

Affected functions:
- `renderer/card.go` — `renderFaceUp` (lines 138–139, 145–146)
- `renderer/card.go` — `cardPeekLines` (lines 197–198)

---

## 2. How to Reproduce

1. Open a terminal emulator. Set the default foreground color to a light value
   (e.g. `#ffffff` or `#f0f0f0`) in the emulator's colour preferences.
2. `go run .`
3. Look at any face-up tableau card. The rank glyph ("K", "9", "A", etc.) will be
   invisible against the white card background.

The screenshots in this review happen to look fine only because the test terminal has a
dark default foreground.

---

## 3. Root Cause

In `renderFaceUp` the rank strings are plain unstyled Go strings:

```go
rankPad  := fmt.Sprintf("%-2s", rank)  // e.g. "K "
rankPadR := fmt.Sprintf("%2s", rank)   // e.g. " K"
```

No Lipgloss `.Foreground(...)` is applied to them. They are embedded in `line0`/`line4`
and wrapped only in `bgStyle` which sets `Background(t.CardBackground)`. The text color
is therefore whatever the terminal inherits as its default foreground.

The `Theme` struct (defined in `theme/theme.go`) has no `CardForeground` field, so there
is no mechanism for the renderer to supply a contrasting colour at all.

The same problem exists in `cardPeekLines`.

---

## 4. Fix

### 4.1 Add `CardForeground` to `Theme` (`theme/theme.go`)

Insert one field directly after `CardBackground` in the "Card colors" block:

```go
// Card colors
CardBackground  lipgloss.Color
CardForeground  lipgloss.Color // rank text on face-up cards  ← ADD
CardBorder      lipgloss.Color
CardFaceDown    lipgloss.Color
RedSuit         lipgloss.Color
BlackSuit       lipgloss.Color
```

### 4.2 Populate `CardForeground` in every theme file

| File | `CardBackground` | `CardForeground` to add |
|------|-----------------|------------------------|
| `theme/classic.go` | `#ffffff` | `#1a1a1a` |
| `theme/dracula.go` | `#f8f8f2` | `#282a36` |
| `theme/nord.go` | `#eceff4` | `#2e3440` |
| `theme/solarized.go` (Dark) | `#fdf6e3` | `#073642` |
| `theme/solarized.go` (Light) | `#073642` | `#eee8d5` |

### 4.3 Apply `rankStyle` in `renderFaceUp` (`renderer/card.go`)

After the existing `suitStyle` declaration, add:

```go
rankStyle := lipgloss.NewStyle().Foreground(t.CardForeground)
```

Then wrap both rank strings in `line0` and `line4` using `.Inline(true)` (the inline
flag prevents the inner style from emitting a reset that would strip the outer card
background — see BUGFIX_B4):

```go
line0 := rankStyle.Inline(true).Render(rankPad) + strings.Repeat(" ", innerWidth-2-1) + suitStyle.Inline(true).Render(suit)
line4 := suitStyle.Inline(true).Render(suit) + strings.Repeat(" ", innerWidth-1-2) + rankStyle.Inline(true).Render(rankPadR)
```

### 4.4 Same change in `cardPeekLines` (`renderer/card.go`)

Add `rankStyle` after `suitStyle` and update `line0`:

```go
rankStyle := lipgloss.NewStyle().Foreground(t.CardForeground)
...
line0 := rankStyle.Inline(true).Render(rankPad) + strings.Repeat(" ", innerWidth-2-1) + suitStyle.Inline(true).Render(suit)
```

---

## 5. Verification

1. `go build ./...` — must compile without errors.
2. Configure the test terminal with a **light default foreground** (`#ffffff`).
3. `go run .` — ranks must be clearly readable on every face-up card.
4. Press `t` to cycle through all five themes. Confirm ranks are readable on each.
5. Confirm suit symbols (red and black) retain their colours — the fix must not change `suitStyle`.
