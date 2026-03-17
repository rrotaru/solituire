# Agent B — Renderer Specialist

> Owns: `theme/` and `renderer/` packages, plus win celebration animation
> Tasks: T5 → T8 → T18

---

## Phase 1: Theme System (T5)

**Stage 3 | Estimate: Small | Dependencies: T1 (Agent A Phase 1)**

Agent A must complete the project scaffold (T1) before this phase can begin. Lipgloss must be importable.

- [ ] Define `Theme` struct with all color fields from Section 10.1 → `theme/theme.go`
- [ ] Implement Classic (green felt) theme → `theme/classic.go`
- [ ] Implement Dracula theme → `theme/dracula.go`
- [ ] Implement Solarized Dark theme → `theme/solarized.go`
- [ ] Implement Solarized Light theme → `theme/solarized.go`
- [ ] Implement Nord theme → `theme/nord.go`
- [ ] Implement `ThemeRegistry` with `List()`, `Get(name)`, `Next(current)` for cycling → `theme/registry.go`
- [ ] Write `theme/theme_test.go`:
  - Registry returns all themes
  - `Get()` by name works
  - `Next()` cycles correctly
  - All themes have non-zero values for all required fields

### Outputs
- `theme/theme.go`, `theme/classic.go`, `theme/dracula.go`, `theme/solarized.go`, `theme/nord.go`, `theme/registry.go`

### Handoff → Phase 2 (self) + Agent C
- The `Theme` struct and `ThemeRegistry` are consumed by the renderer (Phase 2) and by Agent C's App Shell (T13).

### Handoff Contract: Theme → Renderer
```go
// theme/theme.go
type Theme struct {
    Name            string
    CardFaceUp      lipgloss.Style  // or color fields — per Section 10.1
    CardFaceDown    lipgloss.Style
    CardSelected    lipgloss.Style
    CardHint        lipgloss.Style
    EmptySlot       lipgloss.Style
    Background      lipgloss.Style
    HeaderText      lipgloss.Style
    FooterText      lipgloss.Style
    CursorHighlight lipgloss.Style
    SuitRed         lipgloss.Color
    SuitBlack       lipgloss.Color
    // ... all fields per Section 10.1
}

// theme/registry.go
type ThemeRegistry struct { ... }
func NewRegistry() *ThemeRegistry
func (r *ThemeRegistry) List() []string
func (r *ThemeRegistry) Get(name string) Theme
func (r *ThemeRegistry) Next(current string) Theme
```

---

## Phase 2: Renderer — Static Board (T8)

**Stage 5 | Estimate: Large | Dependencies: T2 (Agent A Phase 2), T5 (Phase 1)**

This phase requires:
- Engine types from Agent A Phase 2: `engine.Card`, `engine.Suit`, `engine.Rank`, `engine.GameState`, all pile types, `engine.PileID`
- Theme types from Phase 1: `theme.Theme`

### Received Contract: Engine → Renderer
```go
// Types Agent B reads (never mutates):
engine.Card       // Has Suit, Rank, FaceUp fields; Color(), Symbol(), String() methods
engine.Suit       // Spades, Hearts, Diamonds, Clubs
engine.Rank       // Ace through King
engine.GameState  // Contains Tableau [7]TableauPile, Foundations [4]FoundationPile,
                  //   Stock StockPile, Waste WastePile, Score int, Moves int, Elapsed time.Duration
engine.PileID     // Enum: Stock, Waste, Foundation0-3, Tableau0-6
```

### Work

- [ ] Single card rendering: face-up (with suit color), face-down (hatched), empty slot (dashed border) → `renderer/card.go`
- [ ] All visual states from Section 9.5: cursor hover, selected, hint target → `renderer/card.go`
- [ ] Pile rendering: fanned tableau column (overlapping, only top card full-height), stacked foundation (top card only), stock/waste → `renderer/pile.go`
- [ ] Layout engine: calculate positions for 7 tableau columns + stock/waste/foundations → `renderer/layout.go`
- [ ] Minimum terminal size check (78×24) → `renderer/layout.go`
- [ ] Header bar: score, moves, time, seed, draw mode → `renderer/header.go`
- [ ] Footer bar: context-sensitive keybinding hints → `renderer/footer.go`
- [ ] `Renderer.Render(state, cursor, config) string` — full board composition → `renderer/renderer.go`
- [ ] "Terminal too small" fallback message → `renderer/renderer.go`
- [ ] Add `init()` color profile lock to all renderer test files
- [ ] Write `renderer/card_test.go`: golden file tests for each card state (face-up red, face-up black, face-down, empty, selected, hint-highlighted) using `t.Run()` subtests
- [ ] Write `renderer/renderer_test.go`: golden file test for full board render using seed 42 deal, separate golden for "terminal too small"

### Outputs
- `renderer/card.go`, `renderer/pile.go`, `renderer/layout.go`, `renderer/header.go`, `renderer/footer.go`, `renderer/renderer.go`
- Golden test files in `testdata/`

### Handoff → Agent C
- **Agent C is blocked on T11 (Board Model) until this phase completes.** Agent C needs:
  - `renderer.Renderer` struct and its `Render(state, cursor, config) string` method
  - Layout constants (card width, minimum terminal size) — exported for hit-testing in T16

### Handoff Contract: Renderer → TUI
```go
// renderer/renderer.go
type Renderer struct { ... }
func New(theme theme.Theme) *Renderer
func (r *Renderer) Render(state *engine.GameState, cursor CursorState, config *config.Config) string
func (r *Renderer) SetTheme(theme theme.Theme)

// renderer/layout.go — exported constants
const (
    CardWidth       = 9   // or per Section 9.2
    CardHeight      = 7
    MinTermWidth    = 78
    MinTermHeight   = 24
)

// For hit-testing (T16):
func PileHitTest(x, y int, state *engine.GameState) (engine.PileID, int, bool)
// Returns (pileID, cardIndex, hit) given terminal coordinates
```

---

## Phase 3: Win Celebration (T18)

**Stage 8 | Estimate: Small | Dependencies: T12 (Agent A Phase 7), T13 (Agent C Phase 4)**

This phase requires:
- `engine.GameEngine.IsWon()` from Agent A
- App shell screen routing from Agent C

### Work

- [ ] Implement `CelebrationModel` → `tui/celebration.go`
  - Renders congratulations message with final score, move count, time
  - Cascading card animation via `tea.Tick` and frame-by-frame rendering
  - "New Game" and "Quit" options
- [ ] Write teatest model state test for win detection trigger
- [ ] Write golden test for static congratulations message

### Outputs
- `tui/celebration.go`

### Note
Animation frames are non-deterministic and tested via VHS (T19) rather than golden files.
