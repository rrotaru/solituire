# Agent A — Phase T20 Implementation Plan
## Final Integration + Smoke Test

**Branch**: `claude/agent-a-t20-plan-b8xZd`
**Dependencies**: T19 ✓ (VHS tapes, baselines, and CI workflow all committed)
**Blocks**: nothing — this is the last task

---

## Overview

T20 closes the project. All features are implemented; this phase produces the
release-ready binary by:

1. **Auditing the integration layer** — code review of every playthrough item
   against the actual source to confirm all paths are wired end-to-end.
2. **Running the automated verification gate** — unit tests, vet, binary build,
   and VHS regression.
3. **Fixing any integration issues** discovered during the audit.
4. **Declaring the binary release-ready** once all checks pass.

T20 introduces no new features. Every change made here is a bug fix.

### Files that may be modified

| File | Reason |
|------|--------|
| Any file in `tui/`, `renderer/`, `engine/`, `theme/`, `config/`, `main.go` | Integration bug fixes only |
| `testdata/vhs/*.txt` | Update baselines if a bug fix legitimately changes output |
| `docs/AGENTA_T20_IMPL_PLAN.md` | This file |

---

## Codebase State Entering T20

| Component | Status |
|-----------|--------|
| `engine/` | Complete — card/deck/rules/commands/scoring/hints/game all pass `go test ./...` |
| `main.go` | Complete — `flag` parsing (`--seed`, `--draw`), `WithScreen` bypass |
| `tui/app.go` | Complete — full screen routing, `tooSmall` guard, all message handlers |
| `tui/board.go` | Complete — cursor nav, drag flow, auto-move, auto-complete, undo/redo |
| `tui/menu.go` | Complete — draw mode, theme, auto-move settings, Start button |
| `tui/input.go` | Complete — keyboard + mouse → `GameAction`; 'H' and F1 both map to `ActionHelp` |
| `tui/celebration.go` | Complete — cascading animation + stats box |
| `renderer/` | Complete — card, pile, header, footer, layout, hit-testing |
| `theme/` | Complete — Classic, Dracula, SolarizedDark, SolarizedLight, Nord (5 themes) |
| `config/` | Complete — `Config` struct, `DefaultConfig()` |
| `tapes/` | Complete — all 5 tapes committed |
| `testdata/vhs/` | Complete — all 5 `.txt` baselines committed (non-empty) |
| `.github/workflows/test.yml` | Complete — Docker-based VHS regression + Go unit tests |

Pre-T20 gate (run before starting):

```
go test ./...     → all packages PASS
go vet ./...      → clean (zero diagnostics)
go build -o klondike .  → succeeds
```

---

## Step 1: Integration Audit (Code Review)

Walk every item on the playthrough checklist and confirm the code path
is complete end-to-end. Mark any broken path as an integration bug to fix
in Step 2.

### 1.1 Start a game

**Path**: `--draw 1 --seed 42` → `main.go` skips menu via `WithScreen(ScreenPlaying)` →
`NewBoardModel` → `board.Init()` → `tickCmd()` starts elapsed-time ticker.

**Verify**:
- `main.go` calls `engine.NewGame(seed, cfg.DrawCount)` ✓
- `tui.NewAppModel` sets `screen: ScreenMenu`; `WithScreen` overrides to `ScreenPlaying` ✓
- `NewBoardModel` positions cursor at `PileTableau0`, bottom face-up card via
  `naturalCardIndex` ✓

**Known gap**: `NewBoardModel` creates its own `theme.NewRegistry()` rather than
accepting the shared registry from `AppModel`. This means the board always starts
on Classic regardless of any pre-selected theme. Verify whether `ActionCycleTheme`
(which calls `m.rend.SetTheme`) correctly propagates back to `AppModel.cfg.ThemeName`
via `ThemeChangedMsg`, so the board's independent registry does not drift out of sync
with the renderer's actual theme.

### 1.2 Make moves (cursor navigation + drag-and-drop)

**Path**: arrow keys → `TranslateInput` → `ActionCursorLeft/Right/Up/Down` →
`cursor.MoveLeft/Right/Up/Down` → `board.View()` highlights new cursor position.

Enter on a face-up card → `handleSelect` sets `cursor.Dragging = true`.  
Second Enter → `handleSelect` calls `buildMoveCmd` → `eng.Execute(cmd)` → renderer
reflects new state.

**Verify**:
- `translateKey` maps all arrow keys ✓
- `handleSelect` distinguishes stock (flip) from tableau/waste (drag) ✓
- `dragCount` returns the correct sequence length for multi-card moves ✓
- `buildMoveCmd` constructs the right command type per source/dest pair ✓

### 1.3 Undo / Redo

**Path**: Ctrl+Z → `ActionUndo` → `eng.Undo()` → `m.clampCursor()` → `m.winCmd()`.
Ctrl+Y → `ActionRedo` → same.

**Verify**:
- Both actions clear the drag state before calling undo/redo (drag cleared at top
  of each case in `handleAction`) ✓
- `clampCursor` prevents the cursor from pointing at a card index that no longer
  exists after a pile shrinks ✓
- `winCmd()` is used (not `autoCompleteTickCmd`) so undo/redo cannot accidentally
  trigger an auto-complete loop ✓

### 1.4 Hints

**Path**: '?' → `ActionHint` → `toggleHint(state)` → sets `cursor.ShowHint = true`
and stores hint details in `cursor.HintSource` / `cursor.HintDest`.  
`renderer.Render` applies hint highlight styling to hinted cards.

**Verify**:
- `toggleHint` calls `engine.FindHints(state)` and picks the first hint ✓
- A second '?' press turns the hint off (toggle behaviour) ✓
- `ShowHint` is cleared on any non-hint action (line 103 of `board.go`) ✓
- Renderer applies distinct highlight when `cursor.ShowHint` is set ✓

### 1.5 Cycle themes (in-game 't' key)

**Path**: 't' → `ActionCycleTheme` → `m.themes.Next(m.cfg.ThemeName)` →
`m.rend.SetTheme(next)` → emits `ThemeChangedMsg`.

`AppModel.Update` handles `ThemeChangedMsg` → updates `m.cfg.ThemeName`.

**Verify**:
- `themes` field on `BoardModel` must be the shared registry (same pointer as
  `AppModel.themes`) so `Next()` cycles through the same list ✓/⚠
- After cycling, `m.rend.SetTheme` updates the renderer immediately ✓
- `ThemeChangedMsg` propagates upward so `AppModel.cfg.ThemeName` stays in sync ✓
- The 5-theme cycle order (Classic → Dracula → SolarizedDark → SolarizedLight →
  Nord → Classic) wraps correctly in `registry.Next()` ✓

**Integration risk**: `NewBoardModel` creates `themes: theme.NewRegistry()` — a
fresh registry unrelated to the one stored in `AppModel`. The `Next()` call uses the
board's private registry which is fine (same order), but it means the board's
`m.cfg.ThemeName` is the one propagated. The `ThemeChangedMsg` handler in `AppModel`
sets `m.cfg.ThemeName = msg.Theme.Name`, so they stay in sync after the first cycle.

### 1.6 Pause / Resume

**Path**: 'p' → `ActionPause` → emits `ChangeScreenMsg{ScreenPaused}`.
`AppModel.Update` sets `m.screen = ScreenPaused`.
`AppModel.View` returns "Game Paused — press any key to resume."
Any keypress → `ChangeScreenMsg{ScreenPlaying}`.

**Verify**:
- The `TickMsg` handler in `AppModel.Update` always forwards to `m.board` regardless
  of screen, so the elapsed timer continues to increment while paused ✓/⚠
- **Integration issue**: According to §12.5 of the design doc, the pause screen
  should *stop* the elapsed timer. The current implementation forwards `TickMsg`
  to `BoardModel` on all screens, causing the timer to advance even while paused.
  Fix: stop forwarding `TickMsg` to `BoardModel` when `m.screen == ScreenPaused`.

**Fix for timer-during-pause** (apply in Step 2 if this gap is confirmed critical):

```go
// In AppModel.Update, TickMsg case — guard against ScreenPaused:
case TickMsg:
    if m.screen != ScreenPaused {
        updated, cmd := m.board.Update(msg)
        m.board = updated.(BoardModel)
        return m, cmd
    }
    // While paused, re-queue the tick to keep the chain alive but do not
    // advance the elapsed time.
    return m, tickCmd()
```

Note: the `tickCmd()` function is defined in `tui/board.go` and is unexported.
If the fix is applied at the `AppModel` level, either export `tickCmd` or move
the re-queue logic into `BoardModel.Update` (where `TickMsg` would become a
no-op when the board knows it is paused). The simplest fix is the guard above,
exporting `TickCmd() tea.Cmd` from the `tui` package.

### 1.7 Auto-complete

**Path**: `eng.IsAutoCompletable()` returns true (all cards face-up) →
`m.autoCompleting = true` → `autoCompleteTickCmd()` is returned → after
`AutoCompleteTickDelay` (100 ms) → `AutoCompleteStepMsg` fires →
`handleAutoCompleteStep()` moves the lowest eligible card to a foundation →
loop continues until `IsWon()`.

Any keypress or mouse press during loop → `m.autoCompleting = false` (interrupt).

**Verify**:
- `applyAutoMove` uses the standard safety check (both colors of rank-1 on
  foundations) before auto-moving ✓
- `handleAutoCompleteStep` enumerates foundation moves directly (no safety
  check needed once auto-complete mode is entered — all cards are face-up) ✓
- Interrupt-on-keypress guard precedes `TranslateInput` in `Update` ✓
- `winCmd()` is called at the end of `handleAutoCompleteStep` so the win screen
  fires when the last card lands ✓

### 1.8 Win celebration

**Path**: `eng.IsWon()` → `winCmd()` → `GameWonMsg` →
`AppModel.Update` creates `CelebrationModel` with score/moves/elapsed →
synthesizes `WindowSizeMsg` to give correct dimensions →
sets `m.screen = ScreenWin` → calls `m.celebration.Init()` →
`CelebrationTickMsg` loop drives cascading card animation.

`ScreenWin` routes key/mouse events to `CelebrationModel.Update` which handles
'n' (new game), 'r' (restart), 'q' (quit confirm).

**Verify**:
- `AppModel.Update` handles `GameWonMsg` ✓
- `CelebrationModel.Init()` starts the tick chain ✓
- `CelebrationTickMsg` handler in `AppModel.Update` guards `inWinFlow` so the
  chain terminates cleanly when a new game starts ✓
- Celebration view renders congratulations box with score, moves, and elapsed time ✓

### 1.9 Minimum terminal size warning

**Path**: `tea.WindowSizeMsg` → `AppModel.Update` sets `m.tooSmall` if
`Width < renderer.MinTermWidth || Height < renderer.MinTermHeight` →
`AppModel.View()` renders the warning string instead of the board.

Constants: `MinTermWidth = 78`, `MinTermHeight = 25`.

**Verify**:
- Warning fires on any screen, not just ScreenPlaying ✓
- Warning message includes current and required dimensions ✓
- `too-small.tape` uses `Set Width 300 Set Height 200` — at FontSize 14 this
  renders as approximately 37 cols × 14 rows, both below thresholds ✓
- `testdata/vhs/too-small.txt` baseline is non-empty (2561 bytes) ✓

### 1.10 All five themes render correctly

**Theme registry order**: Classic → Dracula → SolarizedDark → SolarizedLight → Nord

**VHS coverage**:
- `theme-cycle.tape` screenshots: Classic, Dracula, SolarizedDark (3 of 5)
- SolarizedLight and Nord are **not** in any VHS tape

**Unit test coverage**: `theme/theme_test.go` validates all 5 themes compile and
return non-zero color values, but does not assert visual output.

**Gap**: SolarizedLight and Nord have no rendered baseline. For the smoke-test
definition in T20, unit-test coverage is sufficient since visual regressions for
those two themes can only be caught by human review of the committed `.png` files.
Document this as a known coverage gap for a future tape update.

---

## Step 2: Fix Integration Issues

Apply fixes for confirmed bugs found in Step 1. After each fix:

1. `go build -o klondike .` — confirm it still compiles
2. `go test ./...` — confirm no regressions
3. If the fix changes visual output, regenerate the affected VHS baseline locally:
   ```bash
   vhs tapes/<affected>.tape
   git add testdata/vhs/<affected>.txt
   git commit -m "fix: <description>"
   ```

### Confirmed fix: timer advances during pause

If `app.go` line 96–103 is confirmed to not guard against `ScreenPaused`, apply
the following minimal fix. The goal is to keep the tick chain alive (so the timer
resumes correctly on unpause) while not advancing `ElapsedTime`.

In `tui/board.go`, add a paused check to the `TickMsg` case, OR expose a
`TickCmd()` function from the `tui` package and gate in `AppModel`.

**Preferred approach** (minimal blast radius — touches only `board.go`):

```go
// board.go — BoardModel does not know about app screens,
// so this fix belongs in AppModel instead (see below).
```

**AppModel fix** (`tui/app.go`, TickMsg case):

```go
case TickMsg:
    // Keep the tick chain alive on all screens. Only advance the elapsed
    // timer when the game is actively playing (not paused).
    if m.screen == ScreenPaused {
        return m, tickCmd()   // re-queue without advancing time
    }
    updated, cmd := m.board.Update(msg)
    m.board = updated.(BoardModel)
    return m, cmd
```

`tickCmd` is currently defined unexported in `tui/board.go`. Expose it:

```go
// tui/board.go — rename to exported form
func TickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg{}
    })
}
```

Update all internal call sites from `tickCmd()` to `TickCmd()`.

### Other bugs (if found during audit)

Document each additional issue here as it is discovered during the hands-on
playthrough. The plan does not pre-enumerate hypothetical bugs.

---

## Step 3: Full Automated Verification Suite

Run in this order. Every command must exit 0 before proceeding to the next.

```bash
# 1 — Static analysis
go vet ./...

# 2 — Unit + integration tests (all packages)
go test ./...

# 3 — Build release binary
go build -o klondike .

# 4 — VHS visual regression (requires VHS + Docker locally or via CI)
# Run tapes from repo root so ./klondike resolves correctly.
vhs tapes/board-initial.tape
vhs tapes/card-select.tape
vhs tapes/theme-cycle.tape
vhs tapes/screens.tape
vhs tapes/too-small.tape

# 5 — Regression gate: committed .txt baselines must not have changed
git diff --exit-code testdata/vhs/*.txt
```

If Step 4 cannot be run locally (VHS not installed), push to the branch and let
the GitHub Actions workflow execute Steps 4–5 in Docker. The CI workflow will
auto-commit any brand-new baselines and fail on any regression.

---

## Step 4: Verify Playthrough Items

This step is performed interactively against the running binary. In an agent
context it is satisfied by the VHS tape outputs + the code audit in Step 1.

| Playthrough item | Verified by |
|------------------|-------------|
| Start game (menu → board) | `screens.tape` screenshots `menu-screen.png`, `game-started.png` |
| Make moves (cursor + drag) | `card-select.tape` screenshots `card-selected.png`, `card-placed.png` |
| Undo / Redo | Unit test: `engine/history_test.go` + `engine/command_test.go` |
| Hints ('?') | Unit test: `engine/hint_test.go`; cursor `ShowHint` path in `board_test.go` |
| Cycle themes ('t') | `theme-cycle.tape` screenshots + `theme/theme_test.go` |
| Pause / Resume ('p') | `screens.tape` screenshot `pause-screen.png` |
| Minimum terminal size | `too-small.tape` screenshot `too-small-warning.png` |
| Auto-complete | `tui/board_test.go` model state tests |
| Win celebration | `tui/celebration_test.go` + `tui/app_test.go` |
| All 5 themes render | Classic/Dracula/SolarizedDark via `theme-cycle.tape`; SolarizedLight/Nord via unit tests |

---

## Step 5: Tag the Release

Once all verification gates pass, tag the commit as the release-ready binary:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The GitHub Actions workflow will re-run on the tagged commit. A clean CI run on
the tag is the final gate.

---

## Implementation Order

1. **Audit** (Step 1) — code review of all playthrough paths; no file changes
2. **Fixes** (Step 2) — apply confirmed bugs, one commit per fix
3. **Verify** (Step 3) — `go vet`, `go test`, `go build`, VHS, diff
4. **Playthrough table** (Step 4) — confirm each item is covered
5. **Tag** (Step 5) — `v1.0.0`

Steps 1 and 2 are sequential. Steps 3 and 4 can be run in parallel once Step 2
fixes are applied.

---

## Known Coverage Gaps (not blocking release)

| Gap | Risk | Mitigation |
|-----|------|------------|
| SolarizedLight and Nord not in any VHS tape | Low — both are structurally identical to Classic; unit tests confirm they compile | Add a `tapes/theme-cycle-extended.tape` in a follow-up |
| Pause screen shows placeholder text ("Game Paused — press any key to resume.") | Low — functional, baseline captures it | Replace with a styled pause screen in a follow-up |
| Help screen shows placeholder text ("Help — press Esc or F1 to close.") | Low — functional | Replace with a real keybinding reference table in a follow-up |
| Timer may advance during pause | Medium — user-visible quality issue | Fix in Step 2 (confirmed in audit above) |

---

## Handoff Contract

T20 is complete when:
- `go test ./...` passes with zero failures
- `go vet ./...` is clean
- `go build -o klondike .` succeeds
- `git diff --exit-code testdata/vhs/*.txt` is clean after all five tapes re-run
- `v1.0.0` tag pushed and CI passes on the tag
