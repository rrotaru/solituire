# Agent C — TUI Specialist

> Owns: `config/`, `tui/`, and `main.go` (interactive shell)
> Tasks: T4 → T9 → T11 → T13 → T14 → T15 → T16 → T17

---

## Phase 1: Config Struct (T4)

**Stage 3 | Estimate: Small | Dependencies: T1 (Agent A Phase 1)**

Agent A must complete the project scaffold (T1) before this phase can begin.

- [ ] Implement `Config` struct → `config/config.go`
  - `DrawCount int` (1 or 3)
  - `ThemeName string`
  - `AutoMoveEnabled bool`
  - `Seed int64`
- [ ] Implement `DefaultConfig()` function
- [ ] Implement validation method (DrawCount must be 1 or 3, etc.)
- [ ] Write `config/config_test.go`: defaults are sane, validation rejects invalid draw counts

### Outputs
- `config/config.go`

---

## Phase 2: Input Translator (T9)

**Stage 5 | Estimate: Small | Dependencies: T1 (Bubbletea importable)**

Can start as soon as T1 is complete — only needs Bubbletea's `tea.KeyMsg` / `tea.MouseMsg` types.

- [ ] Define `GameAction` enum with all actions from Section 8.1 → `tui/input.go`
- [ ] Implement `TranslateInput(tea.Msg) (GameAction, interface{})` — pure function mapping every key/mouse binding from Section 8.2 → `tui/input.go`
- [ ] Define all custom `Msg` types from Section 7.3 → `tui/messages.go`
  - `NewGameMsg`, `RestartDealMsg`, `GameWonMsg`, `ChangeScreenMsg`
  - `TickMsg`, `CelebrationTickMsg`
  - `ConfigChangedMsg`, `ThemeChangedMsg`, `AutoCompleteStepMsg`
- [ ] Write `tui/input_test.go`: table-driven tests for every binding in Section 8.2
  - Arrow keys, vim keys, number keys, spacebar, enter, escape, ctrl+z, ctrl+y
  - Mouse clicks (basic — extended in T16)
  - Unmapped keys produce `ActionNone`

### Outputs
- `tui/input.go`, `tui/messages.go`

---

## Phase 3: Cursor + Board Model (T11)

**Stage 6 | Estimate: Large | Dependencies: T6 (Agent A Phase 4), T8 (Agent B Phase 2), T9 (Phase 2)**

This is the first major integration point. Requires inputs from all three agents:

### Received Contract: Engine → TUI
```go
// From Agent A (T2 + T6):
engine.GameState    // Read-only for rendering
engine.Card         // Card type with Suit, Rank, FaceUp
engine.Suit, engine.Rank
engine.PileID       // Enum: Stock, Waste, Foundation0-3, Tableau0-6
engine.Command      // Interface: Execute(*GameState) error, Undo(*GameState) error, Description() string
engine.MoveCardCmd          // Construct with source, dest, count
engine.MoveToFoundationCmd  // Construct with source pile
engine.FlipStockCmd         // Construct with draw count
engine.RecycleStockCmd      // No args
engine.FlipTableauCardCmd   // Construct with column index
engine.CompoundCmd          // Wraps multiple commands
engine.History      // Push(), Undo(), Redo(), CanUndo(), CanRedo()
```

### Received Contract: Renderer → TUI
```go
// From Agent B (T8):
renderer.Renderer           // Struct
renderer.New(theme) *Renderer
renderer.Render(state, cursor, config) string
renderer.MinTermWidth       // Constant (78)
renderer.MinTermHeight      // Constant (24)
```

### Work

- [ ] Define `PileID` enum and `CursorState` struct (current pile, card index, drag state) → `tui/cursor.go`
- [ ] Implement cursor navigation logic → `tui/cursor.go`
  - Left/right between piles
  - Up/down within tableau
  - Tab cycling order: stock→waste→foundations→tableau
  - Number key jumps
- [ ] Implement `BoardModel` as `tea.Model` → `tui/board.go`
  - Stores engine reference, cursor, config
- [ ] `BoardModel.Update()`: translate input actions to cursor movements or engine commands → `tui/board.go`
  - Drag-style flow: enter to pick up, move, enter to place
  - Shortcut two-step: number key source, number key or `f` destination
  - Silent rejection on invalid moves
- [ ] `BoardModel.View()`: delegate to renderer → `tui/board.go`
- [ ] Elapsed time tick subscription → `tui/board.go`
- [ ] Write `tui/cursor_test.go`:
  - Left from tableau[0] goes to waste
  - Right from tableau[6] wraps or stops
  - Up/down within columns respects face-up count
  - Tab visits all piles in order
- [ ] Write `tui/board_test.go`:
  - teatest Pattern A (golden) for initial board render with seed 42
  - Pattern C (model state) for drag pick-up/cancel, valid move execution, invalid move rejection, undo/redo

### Outputs
- `tui/cursor.go`, `tui/board.go`

### Handoff → Phase 4 (self)
- The board model is needed by the App Shell (T13).

---

## Phase 4: App Shell + Screen Routing (T13)

**Stage 7 | Estimate: Medium | Dependencies: T4 (Phase 1), T11 (Phase 3), T12 (Agent A Phase 7)**

### Received Contract: GameEngine
```go
// From Agent A (T12):
engine.GameEngine   // Full interface:
  // NewGame(seed, drawCount) error
  // RestartDeal() error
  // Execute(cmd Command) error
  // Undo() error
  // Redo() error
  // State() *GameState
  // ValidMoves() []Hint
  // IsWon() bool
  // IsAutoCompletable() bool
```

### Work

- [ ] Define `AppScreen` enum: Menu, Playing, Paused, Help, QuitConfirm, Win → `tui/app.go`
- [ ] Implement `AppModel` struct → `tui/app.go`
  - Current screen, engine, config, theme, sub-models, window size
- [ ] `AppModel.Update()`: route messages to active sub-model based on screen → `tui/app.go`
  - Handle `ChangeScreenMsg`, `WindowSizeMsg`, `NewGameMsg`
- [ ] `AppModel.View()`: delegate to active sub-model's view, or render "too small" warning → `tui/app.go`
- [ ] Update `main.go`: initialize config, theme, engine, create `AppModel`, run `tea.NewProgram` with mouse support
- [ ] Window size tracking and minimum size check
- [ ] Write `tui/app_test.go`: teatest Pattern C (model state) for screen transitions
  - Verify `ChangeScreenMsg` updates `app.screen` for all transitions in Section 7.1

### Outputs
- `tui/app.go`, `main.go` (updated)

---

## Phase 5: Settings Menu (T14)

**Stage 8 | Estimate: Small | Dependencies: T13 (Phase 4)**

- [ ] Implement menu sub-model → `tui/menu.go`
  - Draw mode toggle (1 or 3)
  - Theme selector (cycle through registry)
  - Auto-move toggle
  - Seed display
  - "Start New Game" action
- [ ] Emit `NewGameMsg` and `ConfigChangedMsg` on changes
- [ ] Write teatest golden for menu layout
- [ ] Write model state test for config changes

### Outputs
- `tui/menu.go`

---

## Phase 6: Help Overlay, Pause Screen, Quit Dialog (T15)

**Stage 8 | Estimate: Small | Dependencies: T13 (Phase 4)**

- [ ] Implement help overlay → `tui/help.go`
  - Render all keybindings from Section 8.2
  - Dismiss on Esc or F1
- [ ] Implement pause screen → `tui/pause.go`
  - Hide all cards, show "Game Paused" message
  - Stop timer, any key resumes
- [ ] Implement quit dialog → `tui/dialog.go`
  - Centered Yes/No dialog
  - Arrow keys or y/n to select
- [ ] Write teatest golden for each screen
- [ ] Write model state test for pause→resume timer behavior

### Outputs
- `tui/help.go`, `tui/pause.go`, `tui/dialog.go`

---

## Phase 7: Mouse Input Support (T16)

**Stage 8 | Estimate: Medium | Dependencies: T11 (Phase 3), T13 (Phase 4)**

### Received Contract: Layout Constants (from Agent B)
```go
// renderer/layout.go — needed for hit-testing
renderer.CardWidth      // Card width in terminal columns
renderer.CardHeight     // Card height in terminal rows
renderer.MinTermWidth
renderer.MinTermHeight
renderer.PileHitTest(x, y int, state *engine.GameState) (engine.PileID, int, bool)
```

### Work

- [ ] Extend `TranslateInput` to handle `tea.MouseMsg` → update `tui/input.go`
  - Map click coordinates to pile and card index using layout geometry
- [ ] Click on stock = flip, click on card = select/place, click on empty area = deselect
- [ ] Implement hit-testing function: given (x, y) and current layout, return `(PileID, cardIndex)` or nil → update `tui/cursor.go`
- [ ] Write table-driven tests for hit-testing with known layout coordinates from seed 42
- [ ] Write teatest model state tests for mouse-driven selection and placement

### Outputs
- Updated `tui/input.go`, updated `tui/cursor.go`

---

## Phase 8: Auto-Complete + Auto-Move (T17)

**Stage 8 | Estimate: Medium | Dependencies: T12 (Agent A Phase 7), T13 (Phase 4)**

- [ ] Implement auto-move → update `tui/board.go`
  - After each player action, if `Config.AutoMoveEnabled`, check for safe-to-move cards
  - "Safe" = both colors of rank-1 already on foundations
  - Execute `MoveToFoundationCmd` via `CompoundCmd`
- [ ] Implement auto-complete → update `tui/board.go`
  - When `IsAutoCompletable()` returns true, enter animation loop
  - Each `AutoCompleteStepMsg` tick moves lowest-rank eligible card to foundation
  - Any keypress interrupts
- [ ] `tea.Tick` subscription for animation timing (~100ms per step)
- [ ] Write model state tests for auto-complete progression
- [ ] Verify interrupt-on-keypress stops the loop

### Outputs
- Updated `tui/board.go`

---

## Dependency Summary for Agent C

```
Phase 1 (T4: Config)           ← waits for Agent A Phase 1 (T1)
Phase 2 (T9: Input)            ← waits for Agent A Phase 1 (T1)
Phase 3 (T11: Board Model)     ← waits for Agent A Phase 4 (T6) + Agent B Phase 2 (T8) + Phase 2 (T9)
Phase 4 (T13: App Shell)       ← waits for Agent A Phase 7 (T12) + Phase 1 (T4) + Phase 3 (T11)
Phase 5 (T14: Menu)            ← waits for Phase 4 (T13)
Phase 6 (T15: Help/Pause)      ← waits for Phase 4 (T13)
Phase 7 (T16: Mouse)           ← waits for Phase 3 (T11) + Phase 4 (T13)
Phase 8 (T17: Auto-Complete)   ← waits for Agent A Phase 7 (T12) + Phase 4 (T13)
```

Phases 1 and 2 can run in parallel. Phases 5, 6, 7, and 8 can all run in parallel once Phase 4 completes.
