# Klondike Solitaire TUI — Architecture & Design Document

## 1. Project Overview

A fully-featured Klondike Solitaire game built as a terminal user interface (TUI) application in Go, using the [Bubbletea](https://github.com/charmbracelet/bubbletea) framework and [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling. The application targets a premium, polished terminal experience with multiple input methods, themeable card rendering, and complete game features including undo/redo, hints, and auto-complete.

### 1.1 Key Technical Decisions Summary

|Decision         |Choice                                                   |
|-----------------|---------------------------------------------------------|
|Framework        |Bubbletea + Lipgloss                                     |
|Architecture     |Game engine (pure library) + thin TUI shell              |
|Package structure|Multi-package with interfaces for testability            |
|Undo/redo system |Command pattern (Execute/Undo per command)               |
|Card rendering   |Lipgloss-styled cards with borders, padding, color themes|
|Input model      |Hybrid: cursor + shortcuts + tab cycling + mouse         |
|Scoring          |Standard Klondike scoring                                |
|Shuffle          |Deterministic seeded PRNG                                |
|Configuration    |In-game settings menu                                    |

-----

## 2. Game Rules Specification

### 2.1 Klondike Variant

- **Draw mode**: Player chooses Draw-1 or Draw-3 at game start via settings menu.
- **Stock recycling**: Unlimited passes through the stock pile.
- **Foundation building**: Aces up through Kings, by suit.
- **Tableau building**: Descending rank, alternating color (red on black, black on red).
- **Tableau moves**: Any face-up sequence correctly built (descending, alternating color) may be moved as a unit. Only Kings (or King-led sequences) may be placed on empty tableau slots.

### 2.2 Standard Scoring

|Action                       |Points|
|-----------------------------|------|
|Waste → Tableau              |+5    |
|Waste → Foundation           |+10   |
|Tableau → Foundation         |+10   |
|Foundation → Tableau         |−15   |
|Flip tableau face-down card  |+5    |
|Recycle stock (waste → stock)|−100  |

- Score floor: 0 (score cannot go negative).
- Timer: Tracked and displayed, but not used for scoring.
- Move counter: Tracked and displayed.

-----

## 3. Package Architecture

```
klondike/
├── main.go                     # Entry point, initializes Bubbletea program
├── go.mod
├── go.sum
│
├── engine/                     # Pure game logic — zero TUI dependency
│   ├── interfaces.go           # GameEngine, Scorer, Command interfaces
│   ├── game.go                 # GameState struct, implements GameEngine
│   ├── card.go                 # Card, Suit, Rank types and helpers
│   ├── deck.go                 # Deck creation, deterministic shuffle
│   ├── tableau.go              # Tableau pile logic (columns)
│   ├── foundation.go           # Foundation pile logic
│   ├── stock.go                # Stock + waste pile logic
│   ├── rules.go                # Move validation (is this move legal?)
│   ├── scoring.go              # StandardScorer implementation
│   ├── command.go              # Command interface + all command types
│   ├── history.go              # Undo/redo stack management
│   └── hint.go                 # Hint engine (find valid moves)
│
├── tui/                        # Bubbletea UI shell
│   ├── app.go                  # Root Bubbletea model, orchestrates sub-models
│   ├── board.go                # Board sub-model (gameplay screen)
│   ├── menu.go                 # Settings/start menu sub-model
│   ├── help.go                 # Help overlay sub-model
│   ├── pause.go                # Pause screen sub-model
│   ├── dialog.go               # Confirmation dialog sub-model (quit, restart)
│   ├── celebration.go          # Win animation sub-model
│   ├── input.go                # Input translator (keypress/mouse → game commands)
│   ├── cursor.go               # Cursor state and navigation logic
│   └── messages.go             # Custom Bubbletea Msg types
│
├── renderer/                   # All Lipgloss view rendering
│   ├── renderer.go             # Main Render() function, composes layout
│   ├── card.go                 # Single card rendering (face-up, face-down, empty)
│   ├── pile.go                 # Pile rendering (fanned tableau, stacked foundation)
│   ├── header.go               # Top bar (score, time, moves)
│   ├── footer.go               # Bottom bar (context-sensitive keybindings)
│   └── layout.go               # Spatial layout calculations, minimum size check
│
├── theme/                      # Color theme definitions
│   ├── theme.go                # Theme struct and interface
│   ├── classic.go              # Green felt theme
│   ├── dracula.go              # Dracula color scheme
│   ├── solarized.go            # Solarized color scheme
│   └── registry.go             # Theme registry (list, get by name)
│
└── config/                     # Game configuration
    └── config.go               # Config struct (draw mode, theme, auto-move toggle)
```

### 3.1 Dependency Graph

```
main → tui → engine (via interfaces)
          → renderer → theme
          → config

engine has ZERO external dependencies (stdlib only + math/rand)
tui depends on bubbletea, lipgloss (via renderer)
renderer depends on lipgloss, theme, engine (read-only, for GameState)
```

**Critical rule**: The `engine` package must never import `bubbletea`, `lipgloss`, or any other TUI package. All communication between `tui` and `engine` flows through interfaces defined in `engine/interfaces.go`.

-----

## 4. Core Interfaces

### 4.1 GameEngine Interface

```go
// engine/interfaces.go

type GameEngine interface {
    // State queries
    State() *GameState
    IsWon() bool
    IsAutoCompletable() bool
    Score() int
    MoveCount() int
    Seed() int64

    // Command execution
    Execute(cmd Command) error
    Undo() error
    Redo() error
    CanUndo() bool
    CanRedo() bool

    // Query helpers
    ValidMoves() []Move
    IsValidMove(move Move) bool

    // Game lifecycle
    NewGame(seed int64, drawCount int)
    RestartDeal()
}
```

### 4.2 Command Interface

```go
// engine/command.go

type Command interface {
    Execute(state *GameState) error
    Undo(state *GameState) error
    Description() string  // For debugging/logging: "Move K♠ from tableau[3] to tableau[0]"
}
```

**Command types to implement:**

|Command              |Fields                            |Notes                                          |
|---------------------|----------------------------------|-----------------------------------------------|
|`MoveCardCmd`        |source pile, dest pile, card count|Covers tableau↔tableau, waste→tableau, etc.    |
|`MoveToFoundationCmd`|source pile, foundation index     |Specialized for auto-move logic                |
|`FlipStockCmd`       |(none — operates on stock/waste)  |Flips 1 or 3 from stock to waste               |
|`RecycleStockCmd`    |(none)                            |Moves waste back to stock                      |
|`FlipTableauCardCmd` |tableau column index              |Auto-triggered, but still a command            |
|`CompoundCmd`        |[]Command                         |Groups atomic commands (e.g., move + auto-flip)|

**CompoundCmd is essential.** When a player moves a card off a tableau column and the newly exposed card is auto-flipped, that’s two atomic operations that must undo as one. The `CompoundCmd` wraps them so a single Ctrl+Z undoes the whole logical action.

### 4.3 Scorer Interface

```go
// engine/interfaces.go

type Scorer interface {
    OnMove(move Move, state *GameState) int   // Returns point delta
    OnUndo(move Move, state *GameState) int   // Returns point delta (negative of original)
    OnRecycleStock() int                       // Returns -100 for standard
}
```

Even though only Standard scoring is implemented initially, the interface exists so Vegas scoring can be added later without touching existing code.

-----

## 5. Game State Model

### 5.1 Card Representation

```go
// engine/card.go

type Suit uint8
const (
    Spades Suit = iota
    Hearts
    Diamonds
    Clubs
)

type Rank uint8
const (
    Ace Rank = iota + 1
    Two
    Three
    // ...
    King = 13
)

type Card struct {
    Suit   Suit
    Rank   Rank
    FaceUp bool
}

func (s Suit) Color() Color  // Red or Black
func (s Suit) Symbol() string // "♠", "♥", "♦", "♣"
func (r Rank) String() string // "A", "2"..."10", "J", "Q", "K"
```

### 5.2 Pile Types

```go
// Tableau column: mix of face-down and face-up cards
type TableauPile struct {
    Cards []Card
}
func (t *TableauPile) FaceDownCount() int
func (t *TableauPile) FaceUpCards() []Card

// Foundation pile: single suit, Ace through King
type FoundationPile struct {
    Cards []Card
}
func (f *FoundationPile) TopCard() *Card
func (f *FoundationPile) AcceptsCard(card Card) bool

// Stock pile: face-down draw pile
type StockPile struct {
    Cards []Card
}

// Waste pile: drawn cards from stock
type WastePile struct {
    Cards     []Card
    DrawCount int   // 1 or 3 — affects which cards are accessible
}
func (w *WastePile) VisibleCards() []Card  // Top 1 or 3 for rendering
func (w *WastePile) TopCard() *Card         // The one playable card
```

### 5.3 Full Game State

```go
// engine/game.go

type GameState struct {
    Tableau     [7]*TableauPile
    Foundations [4]*FoundationPile
    Stock       *StockPile
    Waste       *WastePile
    Score       int
    MoveCount   int
    ElapsedTime time.Duration
    DrawCount   int    // 1 or 3
    Seed        int64
}
```

### 5.4 Deterministic Shuffle

```go
// engine/deck.go

func NewDeck() []Card           // Returns ordered 52-card deck
func Shuffle(deck []Card, seed int64) []Card {
    r := rand.New(rand.NewSource(seed))
    r.Shuffle(len(deck), func(i, j int) {
        deck[i], deck[j] = deck[j], deck[i]
    })
    return deck
}
func Deal(deck []Card) *GameState  // Deals into 7 tableau columns + stock
```

The seed is stored in `GameState.Seed` so `RestartDeal()` can reproduce the same deal.

-----

## 6. Undo/Redo System

### 6.1 History Manager

```go
// engine/history.go

type History struct {
    undoStack []Command
    redoStack []Command
}

func (h *History) Push(cmd Command)    // Pushes to undo stack, clears redo stack
func (h *History) Undo() (Command, bool)
func (h *History) Redo() (Command, bool)
func (h *History) CanUndo() bool
func (h *History) CanRedo() bool
func (h *History) Clear()
```

### 6.2 Undo/Redo Flow

**Execute flow:**

1. Player action is translated to a `Command` (or `CompoundCmd` if auto-flip triggers).
1. `Command.Execute(state)` is called. If it returns an error, the move is silently rejected.
1. On success, the command is pushed to `History.undoStack`. The `redoStack` is cleared.
1. Scorer calculates point delta and updates `state.Score`.

**Undo flow:**

1. Pop from `undoStack`, call `Command.Undo(state)`.
1. Push the command onto `redoStack`.
1. Reverse the score delta.

**Redo flow:**

1. Pop from `redoStack`, call `Command.Execute(state)`.
1. Push back onto `undoStack`.
1. Re-apply the score delta.

### 6.3 CompoundCmd Behavior

```go
type CompoundCmd struct {
    Commands []Command
}

func (c *CompoundCmd) Execute(state *GameState) error {
    for _, cmd := range c.Commands {
        if err := cmd.Execute(state); err != nil {
            // Rollback previously executed commands in this compound
            // (reverse order undo)
            return err
        }
    }
    return nil
}

func (c *CompoundCmd) Undo(state *GameState) error {
    // Undo in reverse order
    for i := len(c.Commands) - 1; i >= 0; i-- {
        if err := c.Commands[i].Undo(state); err != nil {
            return err
        }
    }
    return nil
}
```

-----

## 7. TUI Architecture

### 7.1 Application States (Screen Flow)

```
┌─────────┐     ┌───────────┐     ┌──────────┐
│  Menu /  │────▶│  Playing   │────▶│   Win    │
│ Settings │     │  (Board)   │     │Celebrate │
└─────────┘     └───────────┘     └──────────┘
                  │   ▲   │
              ┌───┘   │   └───┐
              ▼       │       ▼
          ┌───────┐   │   ┌────────┐
          │ Pause │   │   │  Help  │
          └───────┘   │   │Overlay │
              │       │   └────────┘
              └───────┘       │
                  ▲           │
                  └───────────┘
                        │
                  ┌─────────┐
                  │  Quit   │
                  │ Confirm │
                  └─────────┘
```

### 7.2 Root Model

```go
// tui/app.go

type AppScreen int
const (
    ScreenMenu AppScreen = iota
    ScreenPlaying
    ScreenPaused
    ScreenHelp
    ScreenQuitConfirm
    ScreenWin
)

type AppModel struct {
    screen       AppScreen
    engine       engine.GameEngine
    config       *config.Config
    theme        *theme.Theme

    // Sub-models
    menu         MenuModel
    board        BoardModel
    pause        PauseModel
    help         HelpModel
    dialog       DialogModel
    celebration  CelebrationModel

    // Global state
    windowWidth  int
    windowHeight int
    tooSmall     bool
}
```

### 7.3 Message Types

```go
// tui/messages.go

// Window
type WindowSizeMsg struct{ Width, Height int }  // (Built-in from Bubbletea)

// Game lifecycle
type NewGameMsg struct{ Seed int64; DrawCount int }
type RestartDealMsg struct{}
type GameWonMsg struct{}

// Navigation
type ChangeScreenMsg struct{ Screen AppScreen }

// Tick
type TickMsg time.Time          // For elapsed timer updates
type CelebrationTickMsg struct{} // For win animation frames

// Config
type ConfigChangedMsg struct{ Config *config.Config }
type ThemeChangedMsg struct{ Theme *theme.Theme }

// Auto-complete
type AutoCompleteStepMsg struct{} // Triggers one auto-complete move per tick
```

### 7.4 Update Flow

The root `AppModel.Update()` routes messages based on `screen`:

```
KeyMsg / MouseMsg
    │
    ▼
AppModel.Update()
    │
    ├── If ScreenMenu  → MenuModel.Update() → may emit NewGameMsg
    ├── If ScreenPlaying → BoardModel.Update()
    │       │
    │       ├── input.Translate(msg) → Command or CursorAction
    │       │       │
    │       │       ├── CursorAction → update cursor position
    │       │       └── Command → engine.Execute(cmd)
    │       │               │
    │       │               ├── success → check IsWon(), IsAutoCompletable()
    │       │               └── error → silently ignore (card snaps back)
    │       │
    │       ├── TickMsg → update elapsed time display
    │       └── AutoCompleteStepMsg → execute one foundation move, emit next tick
    │
    ├── If ScreenPaused → PauseModel.Update() → any key returns to ScreenPlaying
    ├── If ScreenHelp   → HelpModel.Update() → Esc returns to previous screen
    └── If ScreenWin    → CelebrationModel.Update() → animation ticks
```

-----

## 8. Input System

### 8.1 Input Translator

The input translator is a pure function that maps raw Bubbletea messages to game-level actions. This keeps input handling decoupled from game logic.

```go
// tui/input.go

type GameAction int
const (
    ActionNone GameAction = iota

    // Cursor movement
    ActionCursorUp
    ActionCursorDown
    ActionCursorLeft
    ActionCursorRight

    // Selection
    ActionSelect       // Enter or click — pick up or place card(s)
    ActionCancel       // Esc — deselect current selection

    // Shortcuts
    ActionFlipStock    // Spacebar
    ActionJumpToColumn // 1-7 number keys
    ActionMoveToFoundation // 'f' — auto-move selected to foundation

    // Meta
    ActionUndo         // Ctrl+Z
    ActionRedo         // Ctrl+Y or Ctrl+Shift+Z
    ActionHint         // 'h' or '?'
    ActionNewGame      // Ctrl+N
    ActionRestartDeal  // Ctrl+R
    ActionPause        // 'p'
    ActionHelp         // F1
    ActionQuit         // 'q' or Ctrl+C
    ActionToggleAutoMove // Ctrl+A — toggle auto-foundation
    ActionCycleTheme   // 't'
)

func TranslateInput(msg tea.Msg) (GameAction, ...payload)
```

### 8.2 Input Binding Table

|Key            |Action                  |Context              |
|---------------|------------------------|---------------------|
|←/→ or h/l     |Move cursor left/right  |Between piles        |
|↑/↓ or j/k     |Move cursor up/down     |Within tableau column|
|Tab / Shift+Tab|Cycle to next/prev pile |All piles            |
|1-7            |Jump to tableau column  |Direct jump          |
|Enter          |Select / Place card     |Toggle drag state    |
|Space          |Flip stock              |Always               |
|Esc            |Cancel selection        |When card selected   |
|f              |Move to foundation      |When card selected   |
|Ctrl+Z         |Undo                    |Always               |
|Ctrl+Y         |Redo                    |Always               |
|?              |Hint                    |Always               |
|F1             |Help overlay            |Always               |
|p              |Pause                   |During play          |
|Ctrl+N         |New game                |Always               |
|Ctrl+R         |Restart deal            |Always               |
|Ctrl+A         |Toggle auto-foundation  |Always               |
|t              |Cycle theme             |Always               |
|q              |Quit (with confirm)     |Always               |
|Mouse click    |Select/place at position|Always               |

### 8.3 Cursor Model

```go
// tui/cursor.go

type PileID int
const (
    PileStock PileID = iota
    PileWaste
    PileFoundation0
    PileFoundation1
    PileFoundation2
    PileFoundation3
    PileTableau0
    PileTableau1
    // ...
    PileTableau6
)

type CursorState struct {
    CurrentPile  PileID
    CardIndex    int     // Position within a fanned tableau column (0 = deepest face-up)

    // Drag state
    Dragging     bool
    DragSource   PileID
    DragIndex    int     // How many cards are being dragged (from CardIndex to top)
}
```

### 8.4 Move Execution Modes

**Cursor drag-style (Enter to pick up, move cursor, Enter to place):**

1. Cursor is on a card. Player presses Enter → `Dragging = true`, source is recorded.
1. Player navigates with arrow keys. The “held” cards render with a visual indicator (highlight, different border).
1. Player presses Enter on destination → `engine.Execute(MoveCardCmd{...})`. On success, `Dragging = false`. On failure, nothing happens (card stays held).
1. Player presses Esc → `Dragging = false`, card returns to source (no command executed).

**Shortcut two-step (number key to select source, then destination):**

1. Player presses `3` → cursor jumps to tableau column 3, selects the top card.
1. Player presses `5` → attempts `MoveCardCmd` from column 3 to column 5.
1. Player presses `f` → attempts `MoveToFoundationCmd` from column 3.

**Mouse click:**

1. Click on a card → same as cursor move + Enter (select or place).
1. Click on stock → same as Spacebar (flip).
1. Click on empty area → deselect.

-----

## 9. Renderer Design

### 9.1 Layout Geometry

```
┌──────────────────────────────────────────────────────────────────────┐
│  Score: 150    Moves: 23    Time: 04:32    Seed: 12345    Draw: 1   │  ← Header bar
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  [STK] [WST]          [♠ F] [♥ F] [♦ F] [♣ F]                      │  ← Stock/Waste + Foundations
│                                                                      │
│  [ 1 ] [ 2 ] [ 3 ] [ 4 ] [ 5 ] [ 6 ] [ 7 ]                        │  ← Tableau columns
│  ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐                        │
│  │░░░│ │░░░│ │░░░│ │░░░│ │░░░│ │░░░│ │ K♠│                        │
│  │ A♠│ │░░░│ │░░░│ │░░░│ │░░░│ │ Q♥│ │ Q♦│                        │
│         │ 5♥│ │░░░│ │░░░│ │ J♣│ │10♠│ │ J♠│                        │
│                │ 8♦│ │░░░│ │10♦│       │ 9♥│                        │
│                      │ 2♣│                                           │
│                                                                      │
├──────────────────────────────────────────────────────────────────────┤
│  ←/→: move  Enter: select  Space: draw  ?: hint  F1: help  q: quit │  ← Footer bar
└──────────────────────────────────────────────────────────────────────┘
```

### 9.2 Card Dimensions

Each Lipgloss card occupies a fixed cell size:

```
┌─────┐
│ K ♠ │    Width:  7 characters (1 border + 1 pad + 3 content + 1 pad + 1 border)
│     │    Height: 3-4 lines for full card (top pile cards)
└─────┘           1-2 lines for overlapping/fanned cards in tableau

Face-down:
┌─────┐
│░░░░░│
└─────┘
```

### 9.3 Minimum Terminal Size

- **Width**: 7 tableau columns × 7 chars + 6 gaps × 2 chars = 61 chars minimum for tableau. Add stock/waste area → ~75 columns minimum.
- **Height**: Header (1) + stock row (3) + gap (1) + max tableau depth (~15 fanned) + footer (1) = ~22 rows minimum.
- **Minimum**: 78 × 24 (with some breathing room).

If terminal is smaller, render a centered message: “Terminal too small. Need at least 78×24. Current: {w}×{h}”.

### 9.4 Render Pipeline

```go
// renderer/renderer.go

type Renderer struct {
    theme  *theme.Theme
    width  int
    height int
}

func (r *Renderer) Render(state *engine.GameState, cursor *tui.CursorState, config *config.Config) string {
    header := r.renderHeader(state)
    stockArea := r.renderStockWasteFoundations(state, cursor)
    tableau := r.renderTableau(state, cursor)
    footer := r.renderFooter(config)

    return lipgloss.JoinVertical(lipgloss.Left,
        header,
        stockArea,
        tableau,
        footer,
    )
}
```

### 9.5 Card Rendering States

A card can be in several visual states. The renderer must handle each:

|State              |Visual Treatment                                 |
|-------------------|-------------------------------------------------|
|Face-down          |Filled/hatched pattern, muted border color       |
|Face-up (normal)   |White/light background, rank + suit, suit-colored|
|Cursor hover       |Highlighted border (bright or accent color)      |
|Selected (dragging)|Inverted colors or distinct accent border        |
|Hint target        |Pulsing or distinct color (e.g., yellow border)  |
|Empty pile slot    |Dashed or dim border, placeholder text           |

-----

## 10. Theme System

### 10.1 Theme Struct

```go
// theme/theme.go

type Theme struct {
    Name string

    // Card colors
    CardBackground     lipgloss.Color
    CardBorder         lipgloss.Color
    CardFaceDown       lipgloss.Color
    RedSuit            lipgloss.Color   // Hearts, Diamonds
    BlackSuit          lipgloss.Color   // Spades, Clubs

    // UI chrome
    HeaderBackground   lipgloss.Color
    HeaderForeground   lipgloss.Color
    FooterBackground   lipgloss.Color
    FooterForeground   lipgloss.Color
    BoardBackground    lipgloss.Color

    // Interactive states
    CursorBorder       lipgloss.Color
    SelectedBorder     lipgloss.Color
    HintBorder         lipgloss.Color

    // Empty slot
    EmptySlotBorder    lipgloss.Color
    EmptySlotText      lipgloss.Color
}
```

### 10.2 Bundled Themes

- **Classic**: Green felt background, white cards, red/black suits, gold cursor.
- **Dracula**: Dark purple background (#282a36), light card faces, pink/cyan accents.
- **Solarized Dark**: Base03 background, Base0 text, orange/blue accents.
- **Solarized Light**: Base3 background, Base00 text, same accents.
- **Nord**: Arctic blue palette, frost accents.

Themes are registered in a `ThemeRegistry` and cycled with the `t` key.

-----

## 11. Settings Menu

### 11.1 Menu Screen

Shown at application launch and accessible in-game. Settings are persisted only for the current session (no file I/O).

**Menu options:**

```
╔══════════════════════════════╗
║     KLONDIKE SOLITAIRE       ║
║                              ║
║  Draw Mode:   [ 1 ] [ 3 ]   ║
║  Theme:       ◀ Classic ▶    ║
║  Auto-Move:   [ON] [OFF]    ║
║                              ║
║      [ Start New Game ]      ║
║                              ║
║         Seed: 12345          ║
╚══════════════════════════════╝
```

**Settings fields:**

|Setting  |Type   |Options              |Default|
|---------|-------|---------------------|-------|
|Draw Mode|Toggle |1 or 3               |1      |
|Theme    |Cycle  |All registered themes|Classic|
|Auto-Move|Toggle |ON or OFF            |OFF    |
|Seed     |Display|Current seed shown   |Random |

-----

## 12. Game Features Detail

### 12.1 Auto-Flip

When a move exposes a face-down card at the bottom of a tableau column, it is automatically flipped face-up. This is implemented as a `FlipTableauCardCmd` appended to the move via `CompoundCmd`, so it undoes together with the move.

### 12.2 Auto-Move to Foundation

When enabled, after every player action, the engine checks if any tableau top card or waste top card can go to a foundation and is “safe” to auto-move (both colors of the rank below are already on foundations — the standard safety check). Safe cards are moved automatically via `CompoundCmd`.

### 12.3 Auto-Complete

Triggered when `GameEngine.IsAutoCompletable()` returns true (all remaining cards are face-up). The TUI enters an animation loop:

1. Emit `AutoCompleteStepMsg` on a `tea.Tick` (e.g., every 100ms).
1. Each tick, the engine finds the lowest-rank card that can go to a foundation and executes the move.
1. Repeat until `IsWon()` returns true.
1. Emit `GameWonMsg`.

The player can interrupt auto-complete by pressing any key.

### 12.4 Hint System

```go
// engine/hint.go

type Hint struct {
    Source      PileID
    CardIndex   int
    Destination PileID
    Priority    int  // Higher = better hint
}

func FindHints(state *GameState) []Hint
```

Hint priority (highest first):

1. Move to foundation.
1. Expose a face-down tableau card.
1. Move a King to an empty column.
1. Tableau-to-tableau moves that increase build length.
1. Stock flip (if no other moves).

When the player presses `?`, the top-priority hint is highlighted in the UI (source and destination cards get `HintBorder` styling). The hint clears on the next player action.

### 12.5 Pause Screen

Pressing `p` switches to `ScreenPaused`. The pause screen:

- Hides all cards (renders a message like “Game Paused — press any key to resume”).
- Stops the elapsed timer.
- Any keypress returns to `ScreenPlaying` and resumes the timer.

### 12.6 Win Celebration

When `IsWon()` is detected, the TUI transitions to `ScreenWin`:

- A congratulatory message with final score, move count, and time.
- An animation (e.g., cards cascading/bouncing — classic Windows Solitaire style). Implemented via `tea.Tick` and frame-by-frame Lipgloss rendering.
- Options: “New Game” or “Quit”.

### 12.7 Quit Confirmation

Pressing `q` during gameplay shows a centered dialog:

```
╔═════════════════════════╗
║   Quit current game?    ║
║                         ║
║    [Yes]      [No]      ║
╚═════════════════════════╝
```

Arrow keys or `y`/`n` to select. Confirming exits the Bubbletea program via `tea.Quit`.

-----

## 13. Move Validation Rules

All validation lives in `engine/rules.go`. The engine validates moves before executing commands.

### 13.1 Tableau → Tableau

- Source cards must be a valid face-up sequence (descending rank, alternating color).
- Destination top card must be opposite color and exactly one rank higher than the bottom card of the moved sequence.
- If destination is empty, only a King (or King-led sequence) may be placed.

### 13.2 Waste → Tableau

- Only the top card of the waste pile.
- Same placement rules as tableau → tableau (alternating color, descending rank).

### 13.3 Waste/Tableau → Foundation

- Card must be the next rank of the matching suit.
- Ace goes on empty foundation. Then 2, 3, …, King.

### 13.4 Foundation → Tableau

- Only the top card of a foundation pile.
- Same placement rules as any card → tableau.
- Incurs −15 point penalty.

### 13.5 Stock Flip

- If stock is non-empty, flip `drawCount` cards (or remaining cards if fewer) to waste.
- If stock is empty and waste is non-empty, recycle (move all waste back to stock, reversed). Incurs −100 penalty. Waste is cleared.

-----

## 14. Testing Strategy

Testing spans three layers: pure engine unit tests (no TUI dependency), teatest-based golden file tests for TUI output and interaction, and VHS visual regression tests for pixel-level rendering verification. These layers are complementary — do not substitute one for another.

|Layer             |Tool              |What It Catches                                                     |
|------------------|------------------|--------------------------------------------------------------------|
|Unit              |`go test`         |Game logic, rules, command execute/undo, scoring                    |
|Integration/Golden|`teatest`+`golden`|ANSI output regressions, interaction state, screen flow             |
|Visual regression |VHS               |Font rendering, Lipgloss layout, box-drawing alignment, theme colors|

### 14.1 Test Dependencies

Add these to `go.mod` alongside the main application dependencies:

```go
require (
    github.com/charmbracelet/x/exp/teatest latest
    github.com/charmbracelet/x/exp/golden  latest
    github.com/muesli/termenv              latest
)
```

VHS is an external binary, not a Go dependency. Install separately (see Section 14.7).

### 14.2 Critical Setup (Do Not Skip)

Two setup steps are required before writing any teatest golden tests. Omitting either one causes tests that pass locally but fail on CI.

**Lock the color profile.** Golden files capture raw ANSI output. Different terminals report different color profiles, producing different escape sequences for the same visual output. Every `_test.go` file that uses golden assertions must include this `init()` block:

```go
import (
    "github.com/charmbracelet/lipgloss"
    "github.com/muesli/termenv"
)

func init() {
    lipgloss.SetColorProfile(termenv.Ascii)
}
```

This strips all color codes, making golden files environment-agnostic. Theme-specific color testing is handled by the VHS layer instead.

**Protect golden files from Git line-ending normalization.** Golden files contain raw ANSI escape sequences that Git will corrupt if it treats them as text. Add this to `.gitattributes` at the repo root:

```gitattributes
testdata/*.golden  -text
```

### 14.3 Engine Unit Tests

The `engine` package is pure logic with zero TUI dependencies. It is tested with standard `go test` assertions — no teatest or golden files needed.

**Card & Deck tests** (`engine/deck_test.go`):

- `TestNewDeck`: Returns exactly 52 cards, no duplicates, all suits/ranks present.
- `TestShuffle_Deterministic`: Same seed produces identical card order across runs.
- `TestShuffle_DifferentSeeds`: Different seeds produce different orders.
- `TestDeal`: Correct tableau layout — column `i` has `i+1` cards, only the top card face-up. Remaining 24 cards in stock.

**Rules tests** (`engine/rules_test.go`): Cover every valid and invalid move combination from Section 13. Use table-driven tests with a helper that constructs specific board states.

```go
func TestTableauToTableau(t *testing.T) {
    tests := []struct {
        name    string
        setup   func() *GameState   // Build a specific board position
        move    Move
        wantErr bool
    }{
        {
            name:    "red 6 onto black 7",
            setup:   setupRedSixOnBlackSeven,
            move:    Move{From: PileTableau2, To: PileTableau4, Count: 1},
            wantErr: false,
        },
        {
            name:    "red 6 onto red 7 rejected",
            setup:   setupRedSixOnRedSeven,
            move:    Move{From: PileTableau2, To: PileTableau4, Count: 1},
            wantErr: true,
        },
        {
            name:    "non-king to empty column rejected",
            setup:   setupNonKingToEmptyColumn,
            move:    Move{From: PileTableau0, To: PileTableau3, Count: 1},
            wantErr: true,
        },
        // ... exhaustive cases for all rules in Section 13
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            state := tt.setup()
            err := ValidateMove(state, tt.move)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateMove() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Command tests** (`engine/command_test.go`): Each command type gets Execute and Undo coverage. The pattern: set up state, execute, assert post-state, undo, assert state equals pre-state.

```go
func TestMoveCardCmd_ExecuteUndo(t *testing.T) {
    state := dealWithSeed(42)
    before := deepCopyState(state)

    cmd := &MoveCardCmd{From: PileTableau2, To: PileTableau4, Count: 1}
    if err := cmd.Execute(state); err != nil {
        t.Fatalf("Execute: %v", err)
    }

    // Assert card actually moved
    // ...

    if err := cmd.Undo(state); err != nil {
        t.Fatalf("Undo: %v", err)
    }

    assertStatesEqual(t, before, state)
}
```

**CompoundCmd tests**: Verify that undo reverses all sub-commands in correct (reverse) order. Specifically test the move + auto-flip compound — undo must un-flip the tableau card and return the moved card(s).

**History tests** (`engine/history_test.go`):

- Undo on empty stack returns error.
- Redo on empty stack returns error.
- Push clears the redo stack.
- Undo→Redo round-trips to the same state.
- Multiple undo/redo cycles maintain correct ordering.

**Scoring tests** (`engine/scoring_test.go`): Every scoring event from Section 2.2 produces the correct delta. Verify the score floor — score must never go below zero, even after foundation→tableau penalties or stock recycling.

**Hint tests** (`engine/hint_test.go`): Construct known board positions with specific available moves. Assert that `FindHints()` returns them in the correct priority order (foundation move > expose face-down > king to empty > etc.).

**Win detection tests**: `IsWon()` returns true only when all 4 foundations have 13 cards. `IsAutoCompletable()` returns true when all remaining cards (not yet on foundations) are face-up.

### 14.4 Integration Tests (Engine, Scripted Games)

Use deterministic seeds to play scripted move sequences through the full engine and assert final state. These tests exercise the engine end-to-end without any TUI.

```go
func TestScriptedGame_Seed42(t *testing.T) {
    eng := NewGameEngine()
    eng.NewGame(42, 1) // seed 42, draw-1

    // Scripted moves discovered by manual play or solver
    moves := []Command{
        &FlipStockCmd{},
        &MoveCardCmd{From: PileWaste, To: PileTableau3, Count: 1},
        &MoveToFoundationCmd{From: PileTableau0, Foundation: 0},
        // ...
    }
    for i, cmd := range moves {
        if err := eng.Execute(cmd); err != nil {
            t.Fatalf("move %d: %v", i, err)
        }
    }

    if eng.Score() != expectedScore {
        t.Errorf("score = %d, want %d", eng.Score(), expectedScore)
    }

    // Undo all moves, verify we're back to initial deal
    for eng.CanUndo() {
        eng.Undo()
    }
    assertStatesEqual(t, dealWithSeed(42), eng.State())
}
```

### 14.5 TUI Tests with teatest + Golden Files

These tests exercise the Bubbletea models — screen rendering, input handling, screen transitions — using teatest’s `TestModel` and golden file assertions. All tests in this section require the `init()` color profile lock from Section 14.2.

#### 14.5.1 Test Patterns

**Pattern A: Full output golden (screen rendering).** Assert the complete rendered output of a screen. Use for the initial board layout, menu screen, help overlay, pause screen, and win screen.

```go
// tui/board_test.go

func TestBoardRender_InitialDeal(t *testing.T) {
    cfg := &config.Config{DrawCount: 1, Theme: "classic"}
    eng := engine.NewGameEngine()
    eng.NewGame(42, cfg.DrawCount)

    m := NewBoardModel(eng, cfg)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Send quit to terminate
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}) // confirm quit

    out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(3*time.Second)))
    if err != nil {
        t.Fatal(err)
    }
    golden.RequireEqual(t, out)
}
```

Each test using `golden.RequireEqual` generates a `.golden` file named after the test function. Use `t.Run()` subtests when a single test function needs multiple golden snapshots.

**Pattern B: Mid-interaction snapshot.** Assert output at intermediate points during user interaction. Use for verifying cursor movement, card selection highlighting, and drag state rendering.

```go
func TestBoardRender_CursorNavigation(t *testing.T) {
    cfg := &config.Config{DrawCount: 1, Theme: "classic"}
    eng := engine.NewGameEngine()
    eng.NewGame(42, cfg.DrawCount)

    m := NewBoardModel(eng, cfg)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Wait for initial render
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Score:"))
    }, teatest.WithDuration(3*time.Second))

    // Move cursor right twice (to tableau column 3)
    tm.Send(tea.KeyMsg{Type: tea.KeyRight})
    tm.Send(tea.KeyMsg{Type: tea.KeyRight})

    // Verify cursor position reflected in output
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        // Check for cursor highlight on column 3
        return bytes.Contains(bts, []byte("expected-cursor-indicator"))
    }, teatest.WithDuration(2*time.Second))

    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
```

Always provide explicit `WithDuration` timeouts — the defaults are too short for predictable CI.

**Pattern C: Internal model state assertion.** Assert the Bubbletea model’s internal struct values. Use for verifying cursor state, drag state, screen transitions, and config changes — anything where the rendered output is secondary to the state.

```go
func TestBoardModel_DragState(t *testing.T) {
    cfg := &config.Config{DrawCount: 1, Theme: "classic"}
    eng := engine.NewGameEngine()
    eng.NewGame(42, cfg.DrawCount)

    m := NewBoardModel(eng, cfg)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Navigate to a card and select it
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}) // jump to column 1
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})                       // pick up

    // Quit to extract final model
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

    final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(BoardModel)
    if !final.cursor.Dragging {
        t.Error("expected Dragging to be true after Enter on a card")
    }
    if final.cursor.DragSource != PileTableau0 {
        t.Errorf("DragSource = %v, want PileTableau0", final.cursor.DragSource)
    }
}
```

The type assertion `.(BoardModel)` must match the concrete type returned by the constructor.

#### 14.5.2 teatest Coverage Map

This table maps every testable TUI behavior to the appropriate teatest pattern. Implement all of these.

|Test                                     |Pattern     |Seed|Notes                                             |
|-----------------------------------------|------------|----|--------------------------------------------------|
|Initial board render (draw-1)            |A (golden)  |42  |Baseline board layout                             |
|Initial board render (draw-3)            |A (golden)  |42  |Verify waste pile shows 3 cards                   |
|Menu screen render                       |A (golden)  |—   |Settings menu layout                              |
|Help overlay render                      |A (golden)  |—   |Full keybinding list                              |
|Pause screen render                      |A (golden)  |—   |Cards hidden, message shown                       |
|Quit confirmation dialog                 |A (golden)  |42  |Dialog overlaid on board                          |
|Cursor left/right between piles          |B (snapshot)|42  |Cursor highlight moves correctly                  |
|Cursor up/down within tableau column     |B (snapshot)|42  |Card selection depth changes                      |
|Tab cycling through all piles            |B (snapshot)|42  |Cursor visits stock→waste→foundations→tableau     |
|Number key jump to column                |C (state)   |42  |`cursor.CurrentPile` matches target column        |
|Enter to pick up card (drag start)       |C (state)   |42  |`cursor.Dragging == true`                         |
|Esc to cancel drag                       |C (state)   |42  |`cursor.Dragging == false` after Esc              |
|Spacebar flips stock                     |B (snapshot)|42  |Waste pile content changes                        |
|Valid move completes (card moves)        |C (state)   |42  |Engine state reflects the move                    |
|Invalid move silently rejected           |C (state)   |42  |Engine state unchanged, no error display          |
|Undo restores previous state             |C (state)   |42  |Execute move, undo, compare to original           |
|Redo re-applies undone move              |C (state)   |42  |Undo then redo, compare to post-move state        |
|Hint highlights source and destination   |B (snapshot)|42  |Hint border styling visible in output             |
|Screen transition: menu → playing        |C (state)   |—   |`app.screen == ScreenPlaying` after start         |
|Screen transition: playing → pause → back|C (state)   |42  |Screen toggles correctly, timer paused            |
|Screen transition: playing → help → back |C (state)   |42  |Help overlay shown and dismissed                  |
|Auto-flip after move exposes face-down   |C (state)   |42  |Newly exposed card is face-up                     |
|Theme cycle changes rendered output      |B (snapshot)|42  |`t` key produces different output                 |
|Terminal too small warning               |A (golden)  |—   |Pass `WithInitialTermSize(40, 12)`, assert warning|
|Win detection triggers celebration       |C (state)   |*   |Use a seed/setup where win is reachable quickly   |

#### 14.5.3 Input Translator Unit Tests

The input translator (`tui/input.go`) is a pure function and does not need teatest. Test it with standard table-driven tests:

```go
func TestTranslateInput(t *testing.T) {
    tests := []struct {
        name   string
        msg    tea.Msg
        want   GameAction
    }{
        {"arrow right", tea.KeyMsg{Type: tea.KeyRight}, ActionCursorRight},
        {"vim l", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}, ActionCursorRight},
        {"space", tea.KeyMsg{Type: tea.KeySpace}, ActionFlipStock},
        {"number 3", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}, ActionJumpToColumn},
        {"ctrl+z", tea.KeyMsg{Type: tea.KeyCtrlZ}, ActionUndo},
        {"mouse click", tea.MouseMsg{Type: tea.MouseLeft, X: 15, Y: 10}, ActionSelect},
        // ... all bindings from Section 8.2
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, _ := TranslateInput(tt.msg)
            if got != tt.want {
                t.Errorf("TranslateInput(%v) = %v, want %v", tt.msg, got, tt.want)
            }
        })
    }
}
```

### 14.6 Golden File Workflow

```bash
# First run, or after intentional output changes — generate/overwrite golden files
go test ./... -update

# All normal runs — compare against golden files
go test ./...

# Inspect a golden file visually (requires charmbracelet/sequin)
cat testdata/TestBoardRender_InitialDeal.golden | sequin
```

When `golden.RequireEqual` fails, it prints a diff using the system `diff` tool. The diff shows escaped ANSI sequences. Use `sequin` to render them visually if the diff is hard to read.

### 14.7 VHS Visual Regression Tests

VHS runs the compiled binary in a real headless terminal with a real font renderer. It catches issues that teatest cannot: Lipgloss border alignment, Unicode card suit rendering, theme color accuracy, and layout under exact terminal dimensions.

#### 14.7.1 VHS Installation

```bash
# macOS
brew install charmbracelet/tap/vhs ffmpeg ttyd

# Docker (recommended for CI parity)
docker run --rm -v $PWD:/vhs ghcr.io/charmbracelet/vhs tapes/board.tape
```

All three dependencies (`vhs`, `ffmpeg`, `ttyd`) are required.

#### 14.7.2 Tape Files

Create one tape file per significant view. Each tape produces a `.txt` (for CI diffing) and a `.png` (for human review).

**Initial board layout:**

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

The `Sleep 1s` after launch is mandatory. Without it, VHS sends keystrokes before the TUI renders.

**Cursor navigation and card selection:**

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

**Theme cycling:**

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

**Menu and overlay screens:**

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

**Terminal too small:**

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

#### 14.7.3 VHS Coverage Map

|Tape file           |What it validates                                       |
|--------------------|--------------------------------------------------------|
|`board-initial.tape`|Full board layout, card rendering, header/footer bars   |
|`card-select.tape`  |Cursor highlight, drag styling, card placement rendering|
|`theme-cycle.tape`  |Each theme renders correctly (colors, contrast, borders)|
|`screens.tape`      |Menu, help overlay, pause screen layout and transitions |
|`too-small.tape`    |Graceful degradation warning at undersized terminal     |

#### 14.7.4 Committing VHS Baselines

Generate baselines locally and commit them:

```bash
vhs tapes/board-initial.tape
vhs tapes/card-select.tape
vhs tapes/theme-cycle.tape
vhs tapes/screens.tape
vhs tapes/too-small.tape
git add testdata/vhs/
git commit -m "chore: add VHS baseline outputs"
```

The `.txt` files are the ground truth for CI. The `.png` files are committed for PR review only.

### 14.8 CI Pipeline

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

The `git diff --exit-code` step is the regression gate for visual tests. If any VHS `.txt` output changed, the build fails. The artifact upload gives reviewers `.png` screenshots to inspect what actually changed visually.

### 14.9 Updating Baselines

When an intentional change is made to the TUI output (new layout, changed styling, added UI element):

```bash
# Update teatest golden files
go test ./... -update

# Update VHS baselines
vhs tapes/board-initial.tape
vhs tapes/card-select.tape
vhs tapes/theme-cycle.tape
vhs tapes/screens.tape
vhs tapes/too-small.tape

# Commit all baselines together
git add testdata/
git commit -m "chore: update golden and VHS baselines"
```

Always run the full test suite after updating to confirm no unintended regressions.

### 14.10 Gotcha Reference

|Symptom                                 |Cause                                           |Fix                                                                         |
|----------------------------------------|------------------------------------------------|----------------------------------------------------------------------------|
|Golden tests pass locally, fail on CI   |Color profile mismatch                          |Add `lipgloss.SetColorProfile(termenv.Ascii)` in `init()` (Section 14.2)    |
|Golden files show garbled diffs in Git  |Git normalizing line endings                    |Add `testdata/*.golden -text` to `.gitattributes` (Section 14.2)            |
|VHS screenshots are blank or mid-render |Keystrokes sent before TUI initialized          |Add `Sleep 1s` after `Type "./klondike ..."\nEnter` in every tape           |
|`FinalOutput` blocks forever            |Program never exits                             |Ensure the test sends `q` then `y` (quit + confirm) before `FinalOutput`    |
|Stale golden files after renaming a test|Old `.golden` file not deleted automatically    |Periodically run `find testdata -name '*.golden'` and delete orphans        |
|VHS PNG diffs are noisy in CI           |Font/GPU rendering variance between environments|Use `.txt` output for the CI pass/fail gate; reserve `.png` for human review|
|Multiple golden assertions in one test  |`golden.RequireEqual` uses test name as filename|Split into subtests with `t.Run(name, ...)` — each gets its own golden file |
|Mouse input tests are flaky             |`tea.MouseMsg` coordinates depend on layout     |Use seed 42 and hardcode coordinates from the known layout geometry         |

### 14.11 Directory Layout (Test Artifacts)

```
klondike/
├── engine/
│   ├── card_test.go
│   ├── deck_test.go
│   ├── rules_test.go
│   ├── command_test.go
│   ├── history_test.go
│   ├── scoring_test.go
│   ├── hint_test.go
│   └── integration_test.go       ← scripted game playthroughs
│
├── tui/
│   ├── board_test.go             ← contains init() color profile lock
│   ├── input_test.go             ← pure function tests, no teatest needed
│   ├── app_test.go               ← screen transition tests
│   └── testdata/
│       ├── TestBoardRender_InitialDeal.golden
│       ├── TestBoardRender_Draw3.golden
│       ├── TestMenuRender.golden
│       ├── TestHelpOverlay.golden
│       ├── TestPauseScreen.golden
│       ├── TestQuitDialog.golden
│       └── TestTerminalTooSmall.golden
│
├── renderer/
│   ├── card_test.go              ← individual card rendering assertions
│   ├── renderer_test.go          ← full board composition golden tests
│   └── testdata/
│       ├── TestRenderCard_FaceUp.golden
│       ├── TestRenderCard_FaceDown.golden
│       ├── TestRenderCard_Selected.golden
│       ├── TestRenderCard_Empty.golden
│       └── TestRenderFullBoard.golden
│
├── tapes/                         ← VHS tape files
│   ├── board-initial.tape
│   ├── card-select.tape
│   ├── theme-cycle.tape
│   ├── screens.tape
│   └── too-small.tape
│
├── testdata/
│   └── vhs/                      ← VHS outputs (committed)
│       ├── board-initial.png
│       ├── board-initial.txt
│       ├── cursor-on-col3.png
│       ├── card-selected.png
│       ├── card-placed.png
│       ├── theme-classic.png
│       ├── theme-dracula.png
│       ├── theme-solarized.png
│       ├── menu-screen.png
│       ├── help-overlay.png
│       ├── pause-screen.png
│       └── too-small-warning.png
│
└── .gitattributes                 ← testdata/*.golden  -text
```

-----

## 15. Implementation Plan

This section breaks the project into atomic implementation tasks, maps their dependencies, and identifies which tasks can be worked on in parallel. The project decomposes into 20 tasks organized across 8 time stages. At peak parallelism (stages 3 and 5), three independent work streams can execute simultaneously.

### 15.1 Dependency Graph

```
Stage 1 ──── T1: Project Scaffold ─────────────────────────────────────────────────────────
              │
Stage 2 ──── T2: Card/Deck/Pile Types ─────────────────────────────────────────────────────
              │
              ├──────────────────────┬────────────────────────┐
Stage 3 ──── T3: Rules + Validation  │  T4: Config Struct     │  T5: Theme System
              │                      │                        │
              │                      │                        │
Stage 4 ──── T6: Commands + History ─┤                        │
              │                      │                        │
              ├──────────────────────┤                        │
Stage 5 ──── T7: Scoring            T8: Renderer             T9: Input Translator
              │                      │  (needs T2, T5)        │
              │                      │                        │
Stage 6 ──── T10: Hint Engine       T11: Cursor + Board Model ────────────────────────────
              │                      │  (needs T6, T8, T9)
              │                      │
Stage 7 ──── T12: GameEngine Wire   T13: App Shell + Screen Routing
              │  (needs T6,T7,T10)   │  (needs T4, T11)
              │                      │
              ├──────────────────────┘
              │
Stage 8 ──── T14: Menu ──── T15: Help/Pause/Dialog ──── T16: Mouse Input
              │
              T17: Auto-Complete + Auto-Move
              │
              T18: Win Celebration
              │
              T19: VHS Tapes + CI Pipeline
              │
              T20: Final Integration
```

### 15.2 Task Breakdown

Each task below specifies its inputs (what must exist before work begins), its outputs (what it produces), which design document sections it implements, and its test deliverables. Tasks at the same stage can be assigned to different agents working in parallel.

-----

#### Stage 1: Foundation

**T1: Project Scaffold**

- Inputs: None
- Outputs: `go.mod`, directory structure, `.gitattributes`, empty `main.go`
- Work:
  - `go mod init` with all application and test dependencies from Section 16
  - Create all directories: `engine/`, `tui/`, `renderer/`, `theme/`, `config/`, `testdata/`, `tapes/`
  - Create `.gitattributes` with `testdata/*.golden -text`
  - Create stub `main.go` that compiles and exits
- Tests: `go build ./...` succeeds
- Estimate: Small

-----

#### Stage 2: Core Types

**T2: Card, Deck, and Pile Types**

- Inputs: T1 (project compiles)
- Outputs: `engine/card.go`, `engine/deck.go`, `engine/tableau.go`, `engine/foundation.go`, `engine/stock.go`, `engine/game.go`
- Sections: 5.1 (Card Representation), 5.2 (Pile Types), 5.3 (Full Game State), 5.4 (Deterministic Shuffle)
- Work:
  - `Card`, `Suit`, `Rank` types with all helper methods (`Color()`, `Symbol()`, `String()`)
  - `TableauPile`, `FoundationPile`, `StockPile`, `WastePile` structs with their methods
  - `GameState` struct
  - `NewDeck()`, `Shuffle(deck, seed)`, `Deal(deck)` functions
  - `deepCopyState()` helper (needed for test assertions later)
- Tests:
  - `engine/card_test.go`: Suit colors, rank strings, all 52 cards unique
  - `engine/deck_test.go`: Deterministic shuffle (same seed = same order), `Deal()` produces correct tableau layout (column `i` has `i+1` cards, top card face-up, 24 remaining in stock)
  - `engine/pile_test.go`: `TableauPile.FaceUpCards()`, `FoundationPile.AcceptsCard()`, `WastePile.TopCard()` and `VisibleCards()` for draw-1 and draw-3
- Estimate: Medium

-----

#### Stage 3: Three Parallel Streams Begin

**T3: Move Validation Rules** ‖ **T4: Config Struct** ‖ **T5: Theme System**

These three tasks have no dependencies on each other. They all depend only on T2.

**T3: Move Validation Rules**

- Inputs: T2 (pile types and GameState exist)
- Outputs: `engine/rules.go`
- Sections: 13.1–13.5 (all move validation rules)
- Work:
  - `Move` type definition (source pile, destination pile, card count)
  - `ValidateMove(state, move) error` — the central validation function
  - Tableau→Tableau: alternating color, descending rank, Kings on empty
  - Waste→Tableau: same rules, single card only
  - Waste/Tableau→Foundation: next rank of matching suit
  - Foundation→Tableau: single card, normal tableau placement rules
  - Stock flip: draw N cards or recycle waste to stock
- Tests:
  - `engine/rules_test.go`: Table-driven tests for every valid/invalid case per Section 13. Use helper functions that construct specific board positions. Minimum test cases:
    - Valid: red-on-black, King-to-empty, Ace-to-foundation, sequence move, stock flip, stock recycle
    - Invalid: red-on-red, non-King-to-empty, wrong-rank-to-foundation, wrong-suit-to-foundation, flip empty stock, move face-down card
- Estimate: Medium

**T4: Config Struct**

- Inputs: T1 (project compiles)
- Outputs: `config/config.go`
- Sections: 11.1 (Settings fields)
- Work:
  - `Config` struct: `DrawCount int`, `ThemeName string`, `AutoMoveEnabled bool`, `Seed int64`
  - `DefaultConfig()` function
  - Validation method (DrawCount must be 1 or 3, etc.)
- Tests:
  - `config/config_test.go`: Defaults are sane, validation rejects invalid draw counts
- Estimate: Small

**T5: Theme System**

- Inputs: T1 (project compiles, Lipgloss importable)
- Outputs: `theme/theme.go`, `theme/classic.go`, `theme/dracula.go`, `theme/solarized.go`, `theme/registry.go`
- Sections: 10.1 (Theme Struct), 10.2 (Bundled Themes)
- Work:
  - `Theme` struct with all color fields from Section 10.1
  - Five theme definitions: Classic, Dracula, Solarized Dark, Solarized Light, Nord
  - `ThemeRegistry` with `List()`, `Get(name)`, `Next(current)` for cycling
- Tests:
  - `theme/theme_test.go`: Registry returns all themes, `Get()` by name works, `Next()` cycles correctly, all themes have non-zero values for all required fields
- Estimate: Small

-----

#### Stage 4: Command System

**T6: Commands and History**

- Inputs: T2 (GameState, pile types), T3 (rules for validation within commands)
- Outputs: `engine/command.go`, `engine/history.go`, `engine/interfaces.go` (Command and Scorer interfaces only — GameEngine interface comes later)
- Sections: 4.2 (Command Interface), 4.3 (Scorer Interface), 6.1 (History Manager), 6.2 (Undo/Redo Flow), 6.3 (CompoundCmd)
- Work:
  - `Command` interface: `Execute(*GameState) error`, `Undo(*GameState) error`, `Description() string`
  - All six command types: `MoveCardCmd`, `MoveToFoundationCmd`, `FlipStockCmd`, `RecycleStockCmd`, `FlipTableauCardCmd`, `CompoundCmd`
  - Each command stores enough internal state on Execute to reverse itself on Undo (e.g., `MoveCardCmd` records which cards it moved, `FlipStockCmd` records how many cards were flipped)
  - `History` struct with `undoStack`/`redoStack`, `Push()`, `Undo()`, `Redo()`, `CanUndo()`, `CanRedo()`, `Clear()`
  - `Scorer` interface definition (implementation in T7)
- Tests:
  - `engine/command_test.go`: For each command type — execute, assert state changed, undo, assert state equals pre-execute. `CompoundCmd` with move + auto-flip: undo must reverse both. Execute with invalid state returns error.
  - `engine/history_test.go`: Push/Undo/Redo ordering, undo on empty returns error, push clears redo stack, multiple undo-redo cycles
- Estimate: Large (this is the most complex single task)

-----

#### Stage 5: Three Parallel Streams

**T7: Scoring** ‖ **T8: Renderer** ‖ **T9: Input Translator**

**T7: Scoring Engine**

- Inputs: T6 (Command interface, move types)
- Outputs: `engine/scoring.go`
- Sections: 2.2 (Standard Scoring table)
- Work:
  - `StandardScorer` implementing the `Scorer` interface
  - Point deltas for all move types: waste→tableau +5, waste→foundation +10, tableau→foundation +10, foundation→tableau −15, flip tableau card +5, recycle stock −100
  - Score floor at 0
- Tests:
  - `engine/scoring_test.go`: Every action type produces correct delta. Score floor: start at 0, execute foundation→tableau (−15), assert score stays 0. Recycle stock from score 50 produces 0 (not −50).
- Estimate: Small

**T8: Renderer (Static)**

- Inputs: T2 (Card/GameState types for read-only access), T5 (Theme struct)
- Outputs: `renderer/card.go`, `renderer/pile.go`, `renderer/layout.go`, `renderer/header.go`, `renderer/footer.go`, `renderer/renderer.go`
- Sections: 9.1 (Layout Geometry), 9.2 (Card Dimensions), 9.3 (Minimum Terminal Size), 9.4 (Render Pipeline), 9.5 (Card Rendering States)
- Work:
  - Single card rendering: face-up (with suit color), face-down (hatched), empty slot (dashed border), all visual states from Section 9.5 (cursor hover, selected, hint target)
  - Pile rendering: fanned tableau column (overlapping cards, only top card full-height), stacked foundation (only top card visible), stock/waste
  - Layout engine: calculate positions for 7 tableau columns + stock/waste/foundations. Minimum size check (78×24).
  - Header bar: score, moves, time, seed, draw mode
  - Footer bar: context-sensitive keybinding hints
  - `Renderer.Render(state, cursor, config) string` — full board composition
  - “Terminal too small” fallback message
- Tests:
  - Add `init()` color profile lock to all renderer test files
  - `renderer/card_test.go`: Golden file tests for each card state (face-up red, face-up black, face-down, empty, selected, hint-highlighted). Use `t.Run()` subtests so each gets its own golden file.
  - `renderer/renderer_test.go`: Golden file test for full board render using seed 42 deal. Separate golden for “terminal too small” message.
- Estimate: Large

**T9: Input Translator**

- Inputs: T1 (Bubbletea importable for `tea.KeyMsg` / `tea.MouseMsg` types)
- Outputs: `tui/input.go`, `tui/messages.go`
- Sections: 8.1 (Input Translator), 8.2 (Input Binding Table), 7.3 (Message Types)
- Work:
  - `GameAction` enum with all actions from Section 8.1
  - `TranslateInput(tea.Msg) (GameAction, interface{})` — pure function mapping every key/mouse binding from Section 8.2
  - All custom `Msg` types from Section 7.3: `NewGameMsg`, `RestartDealMsg`, `GameWonMsg`, `ChangeScreenMsg`, `TickMsg`, `CelebrationTickMsg`, `ConfigChangedMsg`, `ThemeChangedMsg`, `AutoCompleteStepMsg`
- Tests:
  - `tui/input_test.go`: Table-driven tests covering every binding in Section 8.2. Arrow keys, vim keys, number keys, spacebar, enter, escape, ctrl+z, ctrl+y, mouse clicks. Verify that unmapped keys produce `ActionNone`.
- Estimate: Small

-----

#### Stage 6: Interactive Core

**T10: Hint Engine** ‖ **T11: Cursor + Board Model**

**T10: Hint Engine**

- Inputs: T2 (GameState), T3 (rules — to find valid moves)
- Outputs: `engine/hint.go`
- Sections: 12.4 (Hint System)
- Work:
  - `Hint` struct: source pile, card index, destination pile, priority
  - `FindHints(state) []Hint` — enumerate all valid moves, assign priorities per Section 12.4 (foundation > expose face-down > King to empty > build length > stock flip)
  - Sort by priority descending
- Tests:
  - `engine/hint_test.go`: Construct known board positions where specific hints are expected. Verify correct ordering. Verify empty hint list when no moves available. Verify foundation move is always highest priority.
- Estimate: Medium

**T11: Cursor Model + Board Model**

- Inputs: T6 (commands to execute moves), T8 (renderer to produce View output), T9 (input translator for Update logic)
- Outputs: `tui/cursor.go`, `tui/board.go`
- Sections: 8.3 (Cursor Model), 8.4 (Move Execution Modes), 7.2 (Root Model — board portion), 7.4 (Update Flow — board portion)
- Work:
  - `PileID` enum and `CursorState` struct (current pile, card index, drag state)
  - Cursor navigation logic: left/right between piles, up/down within tableau, tab cycling order (stock→waste→foundations→tableau), number key jumps
  - `BoardModel` implementing `tea.Model`: stores engine reference, cursor, config
  - `BoardModel.Update()`: translates input actions to cursor movements or engine commands. Implements drag-style flow (enter to pick up, move, enter to place) and shortcut two-step flow (number key source, number key or `f` destination). Silent rejection on invalid moves.
  - `BoardModel.View()`: delegates to renderer
  - Elapsed time tick subscription
- Tests:
  - `tui/cursor_test.go`: Navigation logic — left from tableau[0] goes to waste, right from tableau[6] wraps or stops, up/down within columns respects face-up count, tab visits all piles in order
  - `tui/board_test.go`: teatest Pattern A (golden) for initial board render with seed 42. Pattern C (model state) for drag pick-up/cancel, valid move execution, invalid move rejection, undo/redo through board model.
- Estimate: Large

-----

#### Stage 7: Integration

**T12: GameEngine Wiring** ‖ **T13: App Shell + Screen Routing**

**T12: GameEngine Wiring**

- Inputs: T6 (commands, history), T7 (scoring), T10 (hints)
- Outputs: `engine/game.go` (updated — implements `GameEngine` interface), `engine/interfaces.go` (updated — full `GameEngine` interface)
- Sections: 4.1 (GameEngine Interface)
- Work:
  - Full `GameEngine` interface from Section 4.1
  - `Game` struct implementing `GameEngine`: wires together `GameState`, `History`, `StandardScorer`
  - `Execute(cmd)`: validate, execute, record in history, update score
  - `Undo()` / `Redo()`: delegate to history, reverse/reapply score
  - `ValidMoves()`: delegate to hint engine
  - `IsWon()`: all 4 foundations have 13 cards
  - `IsAutoCompletable()`: all remaining cards face-up
  - `NewGame(seed, drawCount)`: create deck, shuffle, deal
  - `RestartDeal()`: re-deal using stored seed
  - Auto-flip logic: after any move that exposes a face-down tableau card, append `FlipTableauCardCmd` via `CompoundCmd`
- Tests:
  - `engine/game_test.go`: Full GameEngine integration tests. Create engine, execute moves, verify score updates, undo all the way back, verify state matches initial deal. Test `IsWon()` and `IsAutoCompletable()` with constructed states.
  - `engine/integration_test.go`: Scripted game playthroughs with seed 42 (from Section 14.4). Play a known sequence of moves, assert final score and state. Undo all moves, assert return to initial state.
- Estimate: Medium

**T13: App Shell + Screen Routing**

- Inputs: T4 (config), T11 (BoardModel), T12 (GameEngine)
- Outputs: `tui/app.go`, `main.go`
- Sections: 7.1 (Application States), 7.2 (Root Model), 7.4 (Update Flow — routing portion)
- Work:
  - `AppScreen` enum: Menu, Playing, Paused, Help, QuitConfirm, Win
  - `AppModel` struct: current screen, engine, config, theme, sub-models, window size
  - `AppModel.Update()`: routes messages to active sub-model based on screen, handles `ChangeScreenMsg`, `WindowSizeMsg`, `NewGameMsg`
  - `AppModel.View()`: delegates to active sub-model’s view, or renders “too small” warning
  - `main.go`: initialize config, theme, engine, create `AppModel`, run `tea.NewProgram` with mouse support enabled
  - Window size tracking and minimum size check
- Tests:
  - `tui/app_test.go`: teatest Pattern C (model state) for screen transitions — verify `ChangeScreenMsg` updates `app.screen` correctly for all transitions in the screen flow diagram (Section 7.1)
- Estimate: Medium

-----

#### Stage 8: Polish (Mostly Parallel)

All tasks in this stage depend on T13 (working app shell) but are largely independent of each other. They can be parallelized across up to four streams, or worked sequentially by a single agent.

**T14: Settings Menu**

- Inputs: T13 (app shell routes to menu screen)
- Outputs: `tui/menu.go`
- Sections: 11.1 (Menu Screen)
- Work: Menu sub-model with draw mode toggle, theme selector, auto-move toggle, seed display, “Start New Game” action. Emits `NewGameMsg` and `ConfigChangedMsg`.
- Tests: teatest golden for menu layout. Model state test for config changes.
- Estimate: Small

**T15: Help Overlay, Pause Screen, Quit Dialog**

- Inputs: T13 (app shell routes to these screens)
- Outputs: `tui/help.go`, `tui/pause.go`, `tui/dialog.go`
- Sections: 12.5 (Pause), 12.7 (Quit Confirmation), Section 8.2 (keybinding list for help)
- Work:
  - Help overlay: renders all keybindings from Section 8.2, dismisses on Esc or F1
  - Pause screen: hides all cards, shows “Game Paused” message, stops timer, any key resumes
  - Quit dialog: centered Yes/No dialog, arrow keys or y/n to select
- Tests: teatest golden for each screen. Model state test for pause→resume timer behavior.
- Estimate: Small

**T16: Mouse Input Support**

- Inputs: T11 (board model handles `ActionSelect`), T13 (app shell enables mouse)
- Outputs: Updated `tui/input.go`, updated `tui/cursor.go`
- Sections: 8.4 (Mouse click behavior)
- Work:
  - Extend `TranslateInput` to handle `tea.MouseMsg` — map click coordinates to pile and card index using layout geometry from renderer
  - Click on stock = flip, click on card = select/place, click on empty area = deselect
  - Hit-testing function: given (x, y) and current layout, return `(PileID, cardIndex)` or nil
- Tests: Table-driven tests for hit-testing with known layout coordinates from seed 42. teatest model state tests for mouse-driven selection and placement.
- Estimate: Medium

**T17: Auto-Complete + Auto-Move**

- Inputs: T12 (GameEngine.IsAutoCompletable, Execute), T13 (app shell for tick loop)
- Outputs: Updated `tui/board.go`
- Sections: 12.2 (Auto-Move to Foundation), 12.3 (Auto-Complete)
- Work:
  - Auto-move: after each player action, if `Config.AutoMoveEnabled`, check for safe-to-move cards (both colors of rank-1 already on foundations), execute `MoveToFoundationCmd` via `CompoundCmd`
  - Auto-complete: when `IsAutoCompletable()` returns true, enter animation loop. Each `AutoCompleteStepMsg` tick moves the lowest-rank eligible card to foundation. Any keypress interrupts.
  - `tea.Tick` subscription for animation timing (~100ms per step)
- Tests: Model state tests for auto-complete progression. Verify interrupt-on-keypress stops the loop.
- Estimate: Medium

**T18: Win Celebration**

- Inputs: T12 (GameEngine.IsWon), T13 (app shell routes to win screen)
- Outputs: `tui/celebration.go`
- Sections: 12.6 (Win Celebration)
- Work:
  - `CelebrationModel`: renders congratulations message with final score, move count, time
  - Cascading card animation via `tea.Tick` and frame-by-frame rendering
  - “New Game” and “Quit” options
- Tests: teatest model state test for win detection trigger. Golden for static congratulations message (animation frames are non-deterministic and tested via VHS instead).
- Estimate: Small

**T19: VHS Tapes + CI Pipeline**

- Inputs: T13 (compilable binary), all T14–T18 features
- Outputs: All tape files, VHS baseline outputs, `.github/workflows/test.yml`
- Sections: 14.7 (VHS Visual Regression), 14.8 (CI Pipeline)
- Work:
  - Write all five tape files from Section 14.7.2
  - Generate and commit baseline `.txt` and `.png` outputs
  - Create GitHub Actions workflow from Section 14.8
  - Verify full pipeline: `go test ./...` + VHS tapes + `git diff --exit-code`
- Tests: The CI pipeline itself is the test. All prior unit/golden/VHS tests must pass.
- Estimate: Medium

**T20: Final Integration + Smoke Test**

- Inputs: All tasks complete
- Outputs: Release-ready binary
- Work:
  - Manual playthrough: start game, make moves, undo/redo, use hints, cycle themes, pause/resume, auto-complete a game, verify win celebration
  - Fix any integration issues discovered during playthrough
  - Run full test suite: `go test ./... && vhs tapes/*.tape && git diff --exit-code testdata/vhs/*.txt`
  - Verify minimum terminal size warning
  - Verify all five themes render correctly
- Estimate: Small

-----

### 15.3 Parallelism Summary

```
Stage  │ Tasks                         │ Parallel │ Bottleneck
───────┼───────────────────────────────┼──────────┼──────────────────────────
  1    │ T1                            │    1     │ —
  2    │ T2                            │    1     │ —
  3    │ T3, T4, T5                    │    3     │ T3 (rules, medium)
  4    │ T6                            │    1     │ T6 (commands, large)
  5    │ T7, T8, T9                    │    3     │ T8 (renderer, large)
  6    │ T10, T11                      │    2     │ T11 (board model, large)
  7    │ T12, T13                      │    2     │ T13 (app shell, medium)
  8    │ T14, T15, T16, T17, T18, T19, T20 │  4* │ T19 (needs all others)
```

*Stage 8 tasks T14–T18 can run as up to 4 parallel streams. T19 and T20 must wait for all others.

### 15.4 Critical Path

The longest sequential chain determines the minimum calendar time:

```
T1 → T2 → T3 → T6 → T7 → T12 → T13 → T17 → T19 → T20
 S     M    M    L    S     M      M      M      M     S
```

This chain passes through the engine core (T2→T3→T6), scoring (T7), wiring (T12), app shell (T13), auto-complete (T17), and CI (T19). Every other task can be parallelized off this path.

The three large tasks — T6 (commands), T8 (renderer), and T11 (board model) — are the real effort centers. T6 is on the critical path. T8 and T11 are off it (T8 runs parallel with T7 and T9; T11 runs parallel with T10) but they must complete before their dependents can start.

### 15.5 Recommended Agent Assignment

If using multiple AI coding agents, the optimal assignment is:

**Agent A (Engine Specialist):** T1 → T2 → T3 → T6 → T7 → T10 → T12
This agent owns the entire `engine/` package from start to finish. It never touches Bubbletea or Lipgloss. It produces a fully tested, pure Go game logic library.

**Agent B (Renderer Specialist):** T5 → T8 → (wait for T11) → T18
This agent owns `theme/` and `renderer/`, plus the celebration animation. It needs the engine types from T2 (read-only) and its own theme types. It can start T5 as soon as T1 completes, and T8 as soon as T2 and T5 complete.

**Agent C (TUI Specialist):** T4 → T9 → T11 → T13 → T14 → T15 → T16 → T17
This agent owns `config/`, `tui/`, and `main.go`. It builds the interactive shell. T4 and T9 can start early (Stage 3/5), but the board model (T11) must wait for T6 (commands), T8 (renderer), and T9 (input).

**Agent D (Test & CI Specialist):** T19 → T20
This agent takes over after all features are complete. It writes VHS tapes, generates baselines, creates the CI pipeline, and performs the final integration smoke test. Alternatively, T19 and T20 can be handled by any of the other agents after their primary work is done.

### 15.6 Handoff Contracts

Each agent boundary requires a clear contract so agents can work without blocking each other. These are the interfaces and types that must be agreed on before parallel work begins.

**Engine → TUI contract** (defined in T2 + T6, consumed by T11 + T13):

- `engine.GameState` struct (read-only access for rendering)
- `engine.Card`, `engine.Suit`, `engine.Rank` types
- `engine.Command` interface
- All concrete command types (for the TUI to construct)
- `engine.GameEngine` interface (for the TUI to call)
- `engine.PileID` type and constants (shared between engine and TUI cursor)

**Theme → Renderer contract** (defined in T5, consumed by T8):

- `theme.Theme` struct with all color fields
- `theme.ThemeRegistry` with `Get(name)` and `Next(current)`

**Renderer → TUI contract** (defined in T8, consumed by T11):

- `renderer.Renderer` struct with `Render(state, cursor, config) string`
- Layout constants (card width, minimum terminal size) exported for hit-testing in T16

**Recommendation:** Have Agent A produce stub files for all engine interfaces and types (T2 step) before Agents B and C begin their work. This allows all three agents to compile against real types from day one, even if the implementations are not yet complete. The stubs should include the interface definitions, type definitions, and constructor signatures with `panic("not implemented")` bodies.

-----

## 16. Dependencies

### 16.1 Application Dependencies

|Package                             |Purpose                                                        |
|------------------------------------|---------------------------------------------------------------|
|`github.com/charmbracelet/bubbletea`|TUI framework (Elm arch)                                       |
|`github.com/charmbracelet/lipgloss` |Terminal styling                                               |
|`github.com/charmbracelet/bubbles`  |Reusable TUI components (optional, for timer/spinner if needed)|
|Go stdlib `math/rand`               |Deterministic shuffle                                          |
|Go stdlib `time`                    |Elapsed timer                                                  |
|Go stdlib `fmt`, `strings`          |String building                                                |

The engine package uses **only** the Go standard library.

### 16.2 Test Dependencies

|Package                                 |Purpose                                         |
|----------------------------------------|------------------------------------------------|
|`github.com/charmbracelet/x/exp/teatest`|Bubbletea test harness (TestModel, WaitFor)     |
|`github.com/charmbracelet/x/exp/golden` |Golden file assertion and update workflow       |
|`github.com/muesli/termenv`             |Color profile locking for CI-stable golden files|

### 16.3 External Tools (not Go packages)

|Tool  |Purpose                          |Install                                            |
|------|---------------------------------|---------------------------------------------------|
|VHS   |Visual regression testing        |`brew install charmbracelet/tap/vhs`               |
|ffmpeg|Required by VHS for rendering    |`brew install ffmpeg`                              |
|ttyd  |Required by VHS for terminal sim |`brew install ttyd`                                |
|sequin|Human-readable golden file viewer|`go install github.com/charmbracelet/sequin@latest`|