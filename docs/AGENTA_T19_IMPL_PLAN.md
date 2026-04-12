# Agent A — Phase T19 Implementation Plan
## VHS Tapes + CI Pipeline

**Branch**: `claude/agent-a-t19-plan-6xvxW`
**Dependencies**: T13–T18 ✓ (all complete — app shell, board, input, themes, help/pause, win celebration all merged)
**Blocks**: T20 (Final Integration + Smoke Test)

---

## Overview

T19 closes the automated quality loop by adding three things:

1. **CLI flag parsing** in `main.go` — tapes drive the binary with `--seed` and `--draw` flags that don't exist yet.
2. **Five VHS tape files** in `tapes/` — each captures a distinct visual scenario for regression detection.
3. **GitHub Actions CI workflow** at `.github/workflows/test.yml` — runs Go unit tests, then VHS visual regression, then gates on a `git diff` of the committed baselines.

Baseline `.txt` and `.png` outputs in `testdata/vhs/` are generated locally and committed as ground truth. The CI diff step fails the build if any tape output changes without an explicit baseline update.

### Files created / modified

| File | Action |
|------|--------|
| `main.go` | Modify — add `flag` parsing for `--seed` and `--draw`; skip menu when `--draw` is explicitly set |
| `tapes/board-initial.tape` | Create |
| `tapes/card-select.tape` | Create |
| `tapes/theme-cycle.tape` | Create |
| `tapes/screens.tape` | Create |
| `tapes/too-small.tape` | Create |
| `testdata/vhs/*.txt` | Generate locally, commit as baselines |
| `testdata/vhs/*.png` | Generate locally, commit for PR review |
| `.github/workflows/test.yml` | Create |

`tapes/.gitkeep` and `testdata/.gitkeep` are deleted once real files exist in those directories.

---

## Codebase State Entering T19

| Component | Status |
|-----------|--------|
| `engine/` | Complete — all types, rules, commands, scoring, hints, game wiring |
| `tui/app.go` | Complete — `AppModel`, routes messages, `ScreenMenu`/`ScreenPlaying`/`ScreenCelebration` |
| `tui/menu.go` | Complete — draw mode, theme, auto-move settings, start button |
| `tui/board.go` | Complete — full board rendering with cursor, drag state, hint highlighting |
| `tui/input.go` | Complete — keyboard input translator |
| `tui/celebration.go` | Complete — win celebration animation |
| `renderer/` | Complete — card, pile, header/footer rendering |
| `theme/` | Complete — Classic, Dracula, Solarized, Nord, Monokai themes |
| `config/` | Complete — `Config` struct and `DefaultConfig()` |
| `main.go` | Partial — compiles and runs, but **accepts no CLI flags** |
| `tapes/` | Empty (`.gitkeep` only) |
| `testdata/vhs/` | Empty (`.gitkeep` only) |
| `.github/` | Does not exist |

**Critical gap**: `main.go` calls `config.DefaultConfig()` and ignores `os.Args`. All five tape files invoke the binary with `--seed 42` or `--seed 42 --draw 1`. Without flag parsing the binary will exit with an "unknown flag" error and every tape will produce a blank terminal.

---

## Step 1: Add CLI Flag Parsing to `main.go`

### Why two behaviors

| Invocation | App behavior |
|------------|-------------|
| `./klondike --seed 42` | Opens on `ScreenMenu` — seed pre-set, user still picks draw mode and theme |
| `./klondike --seed 42 --draw 1` | Skips menu, opens directly on `ScreenPlaying` |

This distinction is required by the tapes:
- `board-initial.tape` and `card-select.tape` use `--draw 1` and expect the board immediately (no menu Enter needed).
- `screens.tape` and `too-small.tape` use only `--seed 42` and expect the menu first.
- `theme-cycle.tape` uses `--draw 1` and expects the board immediately.

Detect the distinction using Go's `flag.Visit` to check which flags were explicitly provided on the command line.

### Implementation

Replace `main.go` with this:

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"solituire/config"
	"solituire/engine"
	"solituire/renderer"
	"solituire/theme"
	"solituire/tui"
)

func main() {
	cfg := config.DefaultConfig()

	seedFlag := flag.Int64("seed", 0, "RNG seed (0 = random)")
	drawFlag := flag.Int("draw", cfg.DrawCount, "cards to draw per stock flip (1 or 3)")
	flag.Parse()

	// Track whether --draw was explicitly provided.
	drawExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "draw" {
			drawExplicit = true
		}
	})

	if *seedFlag != 0 {
		cfg.Seed = *seedFlag
	}
	cfg.DrawCount = *drawFlag

	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	reg := theme.NewRegistry()
	th := reg.Get(cfg.ThemeName)
	eng := engine.NewGame(seed, cfg.DrawCount)
	rend := renderer.New(th)
	app := tui.NewAppModel(eng, rend, cfg, reg)

	// Skip the menu when --draw is explicitly set (tapes that go straight to board).
	if drawExplicit {
		app = app.WithScreen(tui.ScreenPlaying)
	}

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "klondike: %v\n", err)
		os.Exit(1)
	}
}
```

### Required addition: `WithScreen` on `AppModel`

`tui.AppModel` needs an exported mutator so `main.go` can set the initial screen. Add this to `tui/app.go` (or `tui/tui.go` if there is a public-API surface file):

```go
// WithScreen returns a copy of AppModel with the initial screen overridden.
// Used by main.go to bypass the menu when all config is supplied via CLI flags.
func (m AppModel) WithScreen(s AppScreen) AppModel {
	m.screen = s
	return m
}
```

This is a value receiver returning a copy — safe because `AppModel` is a value type in Bubbletea.

### Verification of Step 1

```bash
go build -o klondike .
./klondike --help                         # prints flag usage, exits 0
./klondike --seed 42 --draw 1 &
sleep 1 && kill %1                        # should not print "unknown flag"
```

---

## Step 2: Five VHS Tape Files

Create each file exactly as specified in DESIGN.md §14.7.2. All tapes share the same
terminal settings: FontSize 14, 900×500, Theme "Charm". Every tape includes `Sleep 1s`
after launching the binary — required by VHS to ensure the TUI renders before sending input.

**Important**: VHS runs tapes in the directory where the tape file lives (`tapes/`), so the
binary path `./klondike` must resolve relative to the repo root (one level up). Build the
binary in the repo root (`go build -o klondike .`) before running tapes locally.

---

### `tapes/board-initial.tape`

Validates: full board layout, card rendering, header/footer bars.

```tape
# tapes/board-initial.tape

Output testdata/vhs/board-initial.png
Output testdata/vhs/board-initial.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1s

Screenshot testdata/vhs/board-initial.png
```

---

### `tapes/card-select.tape`

Validates: cursor highlight, drag styling, card placement rendering.

```tape
# tapes/card-select.tape

Output testdata/vhs/card-select.png
Output testdata/vhs/card-select.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1s

Screenshot testdata/vhs/board-fresh.png

Right
Right
Right
Sleep 300ms
Screenshot testdata/vhs/cursor-on-col3.png

Enter
Sleep 300ms
Screenshot testdata/vhs/card-selected.png

Right
Right
Sleep 300ms
Enter
Sleep 300ms
Screenshot testdata/vhs/card-placed.png
```

---

### `tapes/theme-cycle.tape`

Validates: each theme renders correctly (colors, contrast, borders). Only produces a `.txt`
output (no overall `.png`) because the per-theme screenshots are more useful than a final frame.

```tape
# tapes/theme-cycle.tape

Output testdata/vhs/theme-cycle.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42 --draw 1"
Enter
Sleep 1s

Screenshot testdata/vhs/theme-classic.png

Type "t"
Sleep 500ms
Screenshot testdata/vhs/theme-dracula.png

Type "t"
Sleep 500ms
Screenshot testdata/vhs/theme-solarized.png
```

---

### `tapes/screens.tape`

Validates: menu screen, help overlay, pause screen layout and transitions.

```tape
# tapes/screens.tape

Output testdata/vhs/screens.txt

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Charm"

Type "./klondike --seed 42"
Enter
Sleep 1s
Screenshot testdata/vhs/menu-screen.png

# Start game
Enter
Sleep 1s
Screenshot testdata/vhs/game-started.png

# Help overlay
Type "F1"
Sleep 500ms
Screenshot testdata/vhs/help-overlay.png

# Dismiss help, open pause
Escape
Sleep 300ms
Type "p"
Sleep 500ms
Screenshot testdata/vhs/pause-screen.png
```

---

### `tapes/too-small.tape`

Validates: graceful degradation warning at undersized terminal (300×200).

```tape
# tapes/too-small.tape

Output testdata/vhs/too-small.txt

Set FontSize 14
Set Width 300
Set Height 200
Set Theme "Charm"

Type "./klondike --seed 42"
Enter
Sleep 1s
Screenshot testdata/vhs/too-small-warning.png
```

---

## Step 3: GitHub Actions CI Workflow

Create `.github/workflows/test.yml`. This requires creating the `.github/workflows/`
directory structure first (`mkdir -p .github/workflows`).

The workflow adds one step not in the DESIGN.md spec: **build the binary before running VHS**.
The module name `solituire` means `go build` produces `./solituire`, but all tape files
reference `./klondike`. The explicit `-o klondike` flag bridges this gap.

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: stable

      # Layer 1: Engine unit tests + teatest golden tests
      - name: Run all Go tests
        run: go test ./...

      # Build binary for VHS tapes (module produces 'solituire'; tapes expect 'klondike')
      - name: Build binary
        run: go build -o klondike .

      # Layer 2: VHS visual regression
      - name: Run VHS visual regression
        uses: charmbracelet/vhs-action@v2
        with:
          path: "tapes/"

      - name: Fail on VHS text diff
        run: git diff --exit-code testdata/vhs/*.txt

      - name: Upload screenshots on failure
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: vhs-screenshots
          path: testdata/vhs/*.png
```

---

## Step 4: Generate and Commit VHS Baselines

This step runs locally. VHS, ffmpeg, and ttyd must all be installed (see DESIGN.md §14.7.1).

```bash
# Build the binary first
go build -o klondike .

# Run all five tapes — outputs land in testdata/vhs/
vhs tapes/board-initial.tape
vhs tapes/card-select.tape
vhs tapes/theme-cycle.tape
vhs tapes/screens.tape
vhs tapes/too-small.tape

# Remove placeholder, stage real outputs
git rm tapes/.gitkeep testdata/.gitkeep
git add testdata/vhs/
git commit -m "chore: add VHS baseline outputs"
```

The `.txt` files are the CI regression gate. The `.png` files are committed for human review
on PRs but are not used by `git diff --exit-code`.

### Expected outputs after running all five tapes

| File | Produced by |
|------|-------------|
| `testdata/vhs/board-initial.png` | `board-initial.tape` (Screenshot) |
| `testdata/vhs/board-initial.txt` | `board-initial.tape` (Output) |
| `testdata/vhs/board-fresh.png` | `card-select.tape` (Screenshot) |
| `testdata/vhs/cursor-on-col3.png` | `card-select.tape` (Screenshot) |
| `testdata/vhs/card-selected.png` | `card-select.tape` (Screenshot) |
| `testdata/vhs/card-placed.png` | `card-select.tape` (Screenshot) |
| `testdata/vhs/card-select.txt` | `card-select.tape` (Output) |
| `testdata/vhs/theme-classic.png` | `theme-cycle.tape` (Screenshot) |
| `testdata/vhs/theme-dracula.png` | `theme-cycle.tape` (Screenshot) |
| `testdata/vhs/theme-solarized.png` | `theme-cycle.tape` (Screenshot) |
| `testdata/vhs/theme-cycle.txt` | `theme-cycle.tape` (Output) |
| `testdata/vhs/menu-screen.png` | `screens.tape` (Screenshot) |
| `testdata/vhs/game-started.png` | `screens.tape` (Screenshot) |
| `testdata/vhs/help-overlay.png` | `screens.tape` (Screenshot) |
| `testdata/vhs/pause-screen.png` | `screens.tape` (Screenshot) |
| `testdata/vhs/screens.txt` | `screens.tape` (Output) |
| `testdata/vhs/too-small-warning.png` | `too-small.tape` (Screenshot) |
| `testdata/vhs/too-small.txt` | `too-small.tape` (Output) |

---

## Step 5: Verification Gate

Run locally before pushing:

```bash
# 1. Go unit tests must all pass
go test ./...

# 2. Rebuild binary and re-run all tapes
go build -o klondike .
vhs tapes/board-initial.tape
vhs tapes/card-select.tape
vhs tapes/theme-cycle.tape
vhs tapes/screens.tape
vhs tapes/too-small.tape

# 3. Diff must be clean — no changes to committed .txt baselines
git diff --exit-code testdata/vhs/*.txt

# 4. Vet
go vet ./...
```

A clean run of all four steps means T19 is complete and T20 can begin.

---

## Implementation Order

1. **`main.go`**: Add `flag` parsing and `WithScreen` bypass logic.
2. **`tui/app.go`**: Add `WithScreen(s AppScreen) AppModel` method.
3. **Tape files**: Create all five in `tapes/`.
4. **CI workflow**: Create `.github/workflows/test.yml`.
5. **Build + generate baselines**: Run locally, commit `testdata/vhs/` outputs.
6. **Verification**: Run full gate, fix anything that fails.

Steps 3 and 4 are independent of each other and can be done in parallel. Steps 1 and 2 must
complete before Step 5 (baseline generation) because the tapes invoke the binary with flags.

---

## Edge Cases and Gotchas

| Scenario | Handling |
|----------|----------|
| Tape sends keys before TUI renders | `Sleep 1s` after launch is mandatory in every tape |
| Binary named `solituire`, tapes expect `klondike` | `go build -o klondike .` before running any tape; CI adds explicit build step |
| `--draw` flag value matches default (1) but was explicit | `flag.Visit` detects explicit flags regardless of value |
| `--seed 0` on command line | Treated same as omitting `--seed` — `main.go` falls through to `time.Now().UnixNano()` |
| VHS PNG rendering varies across machines | `.txt` output is the CI gate; `.png` is for human review only |
| `.gitkeep` files become stale | Remove with `git rm` when committing real content into those dirs |
| `screens.tape` `F1` key for help | VHS sends the literal characters `F`, `1` — confirm the TUI binds help to the `F1` key sequence, not the escape sequence; adjust tape if needed |
| `card-select.tape` move validity with seed 42 | Verify manually that the column-3 card (3 rights from initial cursor position) can actually be selected and placed 2 columns right — if the move is invalid under seed 42, adjust the right-arrow counts |

---

## Updating Baselines (Future Reference)

When an intentional TUI change modifies visual output:

```bash
# Rebuild and re-run affected tapes
go build -o klondike .
vhs tapes/<changed>.tape     # run only the affected tape(s)

# Or update all at once
vhs tapes/board-initial.tape
vhs tapes/card-select.tape
vhs tapes/theme-cycle.tape
vhs tapes/screens.tape
vhs tapes/too-small.tape

# Stage and commit the new baselines
git add testdata/vhs/
git commit -m "chore: update VHS baselines for <description of change>"
```

Always run the full test suite after updating to confirm no unintended regressions.

---

## Handoff Contract

When T19 is complete, T20 can begin. T20 requires:
- `go test ./...` passes cleanly.
- `go build -o klondike .` produces a working binary.
- All five tapes run without error and their `.txt` diffs are clean.
- `.github/workflows/test.yml` is present and syntactically valid YAML.
