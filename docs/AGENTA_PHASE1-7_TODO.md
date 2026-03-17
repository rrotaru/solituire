# Agent A — Engine Specialist

> Owns: `engine/` package (pure Go, zero TUI dependencies)
> Tasks: T1 → T2 → T3 → T6 → T7 → T10 → T12 → T19 → T20

---

## Phase 1: Project Scaffold (T1)

**Stage 1 | Estimate: Small | Dependencies: None**

- [ ] `go mod init` with all dependencies from Section 16
- [ ] Create directory structure: `engine/`, `tui/`, `renderer/`, `theme/`, `config/`, `testdata/`, `tapes/`
- [ ] Create `.gitattributes` with `testdata/*.golden -text`
- [ ] Create stub `main.go` that compiles and exits
- [ ] Verify: `go build ./...` succeeds

### Outputs
- `go.mod`, `go.sum`, directory tree, `.gitattributes`, `main.go`

### Handoff → Agents B & C
- **Agents B and C are blocked until Phase 1 is complete.** The project must compile and all dependencies must be importable (Bubbletea, Lipgloss, etc.).

---

## Phase 2: Card, Deck, and Pile Types (T2)

**Stage 2 | Estimate: Medium | Dependencies: T1**

- [ ] Implement `Card`, `Suit`, `Rank` types with helpers: `Color()`, `Symbol()`, `String()` → `engine/card.go`
- [ ] Implement `TableauPile`, `FoundationPile`, `StockPile`, `WastePile` structs → `engine/tableau.go`, `engine/foundation.go`, `engine/stock.go`
- [ ] Implement `GameState` struct → `engine/game.go`
- [ ] Implement `NewDeck()`, `Shuffle(deck, seed)`, `Deal(deck)` → `engine/deck.go`
- [ ] Implement `deepCopyState()` helper
- [ ] **Create stub files for all engine interfaces and types** with `panic("not implemented")` bodies so Agents B and C can compile against real types immediately
- [ ] Write `engine/card_test.go`: suit colors, rank strings, all 52 cards unique
- [ ] Write `engine/deck_test.go`: deterministic shuffle, `Deal()` layout correctness
- [ ] Write `engine/pile_test.go`: `FaceUpCards()`, `AcceptsCard()`, `TopCard()`, `VisibleCards()` for draw-1 and draw-3

### Outputs
- `engine/card.go`, `engine/deck.go`, `engine/tableau.go`, `engine/foundation.go`, `engine/stock.go`, `engine/game.go`
- Stub `engine/interfaces.go` with `GameEngine`, `Command`, `Scorer` interface signatures
- Stub `engine/rules.go`, `engine/command.go`, `engine/history.go`, `engine/scoring.go`, `engine/hint.go`

### Handoff → Agent B
- Agent B can start **T8 (Renderer)** once this phase completes. Agent B needs read-only access to:
  - `engine.Card`, `engine.Suit`, `engine.Rank` types and their methods
  - `engine.GameState` struct and all pile types
  - `engine.PileID` type and constants

### Handoff → Agent C
- Agent C can start **T9 (Input Translator)** in parallel (only needs Bubbletea types, not engine types)
- Agent C will need `engine.Command` interface and concrete command types later (from Phase 4)

---

## Phase 3: Move Validation Rules (T3)

**Stage 3 | Estimate: Medium | Dependencies: T2**

- [ ] Define `Move` type (source pile, destination pile, card count) → `engine/rules.go`
- [ ] Implement `ValidateMove(state, move) error`
- [ ] Tableau→Tableau: alternating color, descending rank, Kings on empty
- [ ] Waste→Tableau: same rules, single card only
- [ ] Waste/Tableau→Foundation: next rank of matching suit
- [ ] Foundation→Tableau: single card, normal tableau placement rules
- [ ] Stock flip: draw N cards or recycle waste to stock
- [ ] Write `engine/rules_test.go`: table-driven tests for all valid/invalid cases per Section 13
  - Valid: red-on-black, King-to-empty, Ace-to-foundation, sequence move, stock flip, stock recycle
  - Invalid: red-on-red, non-King-to-empty, wrong-rank-to-foundation, wrong-suit-to-foundation, flip empty stock, move face-down card

### Outputs
- `engine/rules.go` (full implementation replacing stub)

### Handoff → Phase 4
- T6 (Commands) depends on T3 for validation within command execution.

---

## Phase 4: Commands and History (T6)

**Stage 4 | Estimate: Large | Dependencies: T2, T3**

This is the most complex single task in the project.

- [ ] Define `Command` interface: `Execute(*GameState) error`, `Undo(*GameState) error`, `Description() string` → `engine/interfaces.go`
- [ ] Define `Scorer` interface (implementation deferred to Phase 5) → `engine/interfaces.go`
- [ ] Implement `MoveCardCmd` → `engine/command.go`
- [ ] Implement `MoveToFoundationCmd` → `engine/command.go`
- [ ] Implement `FlipStockCmd` → `engine/command.go`
- [ ] Implement `RecycleStockCmd` → `engine/command.go`
- [ ] Implement `FlipTableauCardCmd` → `engine/command.go`
- [ ] Implement `CompoundCmd` → `engine/command.go`
- [ ] Each command stores enough internal state on Execute to reverse itself on Undo
- [ ] Implement `History` struct with `undoStack`/`redoStack`, `Push()`, `Undo()`, `Redo()`, `CanUndo()`, `CanRedo()`, `Clear()` → `engine/history.go`
- [ ] Write `engine/command_test.go`: execute/undo round-trip for each command type, `CompoundCmd` with move+auto-flip, execute with invalid state returns error
- [ ] Write `engine/history_test.go`: push/undo/redo ordering, undo on empty returns error, push clears redo stack, multiple cycles

### Outputs
- `engine/command.go`, `engine/history.go`, `engine/interfaces.go` (Command + Scorer interfaces)

### Handoff → Agent C
- **Agent C is blocked on T11 (Board Model) until this phase completes.** The board model needs:
  - `engine.Command` interface
  - All concrete command types (`MoveCardCmd`, `MoveToFoundationCmd`, `FlipStockCmd`, `RecycleStockCmd`, `FlipTableauCardCmd`, `CompoundCmd`)
  - `engine.History` for undo/redo

---

## Phase 5: Scoring Engine (T7)

**Stage 5 | Estimate: Small | Dependencies: T6**

- [ ] Implement `StandardScorer` implementing `Scorer` interface → `engine/scoring.go`
- [ ] Point deltas: waste→tableau +5, waste→foundation +10, tableau→foundation +10, foundation→tableau −15, flip tableau card +5, recycle stock −100
- [ ] Score floor at 0
- [ ] Write `engine/scoring_test.go`: every action type correct delta, score floor tests

### Outputs
- `engine/scoring.go`

---

## Phase 6: Hint Engine (T10)

**Stage 6 | Estimate: Medium | Dependencies: T2, T3**

- [ ] Define `Hint` struct: source pile, card index, destination pile, priority → `engine/hint.go`
- [ ] Implement `FindHints(state) []Hint`: enumerate all valid moves, assign priorities per Section 12.4
  - Priority order: foundation > expose face-down > King to empty > build length > stock flip
- [ ] Sort hints by priority descending
- [ ] Write `engine/hint_test.go`: known board positions with expected hints, correct ordering, empty list when no moves, foundation move highest priority

### Outputs
- `engine/hint.go`

---

## Phase 7: GameEngine Wiring (T12)

**Stage 7 | Estimate: Medium | Dependencies: T6, T7, T10**

- [ ] Define full `GameEngine` interface in `engine/interfaces.go` (Section 4.1)
- [ ] Implement `Game` struct implementing `GameEngine` → update `engine/game.go`
  - Wires together `GameState`, `History`, `StandardScorer`
- [ ] `Execute(cmd)`: validate, execute, record in history, update score
- [ ] `Undo()` / `Redo()`: delegate to history, reverse/reapply score
- [ ] `ValidMoves()`: delegate to hint engine
- [ ] `IsWon()`: all 4 foundations have 13 cards
- [ ] `IsAutoCompletable()`: all remaining cards face-up
- [ ] `NewGame(seed, drawCount)`: create deck, shuffle, deal
- [ ] `RestartDeal()`: re-deal using stored seed
- [ ] Auto-flip logic: after any move exposing face-down tableau card, append `FlipTableauCardCmd` via `CompoundCmd`
- [ ] Write `engine/game_test.go`: full integration — create engine, execute moves, verify score, undo all, verify state matches initial deal, test `IsWon()` and `IsAutoCompletable()`
- [ ] Write `engine/integration_test.go`: scripted playthrough with seed 42, play known sequence, assert final score/state, undo all, assert return to initial

### Outputs
- `engine/game.go` (full implementation), `engine/interfaces.go` (full `GameEngine` interface)

### Handoff → Agent C
- **Agent C needs the completed `GameEngine` interface for T13 (App Shell).** Provide:
  - `engine.GameEngine` interface with all methods
  - `engine.Game` constructor (`NewGame(seed, drawCount)`)

---

## Phase 8: VHS Tapes + CI Pipeline (T19)

**Stage 8 | Estimate: Medium | Dependencies: All T14–T18 complete**

- [ ] Write all five tape files from Section 14.7.2 → `tapes/`
- [ ] Generate and commit baseline `.txt` and `.png` outputs → `testdata/vhs/`
- [ ] Create GitHub Actions workflow from Section 14.8 → `.github/workflows/test.yml`
- [ ] Verify: `go test ./...` + VHS tapes + `git diff --exit-code`

---

## Phase 9: Final Integration + Smoke Test (T20)

**Stage 8 | Estimate: Small | Dependencies: All tasks complete**

- [ ] Manual playthrough: start game, make moves, undo/redo, hints, cycle themes, pause/resume, auto-complete, win celebration
- [ ] Fix any integration issues
- [ ] Run full test suite: `go test ./... && vhs tapes/*.tape && git diff --exit-code testdata/vhs/*.txt`
- [ ] Verify minimum terminal size warning
- [ ] Verify all five themes render correctly

### Outputs
- Release-ready binary
