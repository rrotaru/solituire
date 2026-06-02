# solituire

> Klondike Solitaire in your terminal, built with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

![gameplay demo](docs/demo/gameplay.gif)

## Features

- **Full Klondike rules** — draw-1 and draw-3 modes, unlimited stock recycling
- **Keyboard & mouse** — arrows, number shortcuts, click to select
- **Undo / Redo** — every move is reversible with `u` / `r`
- **Hint engine** — press `h` to highlight a valid move
- **Foundation shortcuts** — press `f` to send the selected card to its foundation, or toggle continuous auto-move with `Ctrl+A`
- **7 built-in themes** — Classic, Dracula, Solarized Dark, Solarized Light, Nord, Catppuccin, Tokyo Night
- **Seeded games** — replay any game exactly with `--seed`
- **Score & timer** — standard Klondike point system tracked in the header

![themes demo](docs/demo/themes.gif)

## Installation

Requires Go 1.24+.

```bash
git clone https://github.com/rrotaru/solituire
cd solituire
go build -o klondike .
./klondike
```

## Usage

```
./klondike [flags]

Flags:
  --seed int64   RNG seed for the deal (0 = random)
  --draw int     Cards to draw per stock flip: 1 or 3 (default 1)
  --version      Print version and exit
```

When `--draw` is supplied the menu is skipped and the game starts immediately.

```bash
# Random game, pick draw mode in the menu
./klondike

# Reproducible draw-3 game
./klondike --seed 42 --draw 3
```

## Controls

| Key(s) | Action |
|---|---|
| `←` `→` | Move cursor between piles |
| `↑` `↓` | Move cursor within a tableau column |
| `1`–`7` | Jump to tableau column |
| `Tab` / `Shift+Tab` | Cycle to next / previous pile |
| `Enter` | Pick up / place selected card |
| `Space` | Draw from stock |
| `f` | Move selected card to foundation |
| `h` | Show a hint |
| `u` | Undo |
| `r` | Redo |
| `t` | Cycle theme |
| `p` | Pause / resume |
| `Ctrl+A` | Toggle continuous auto-move to foundation |
| `Ctrl+N` | Start a new game |
| `Ctrl+R` | Restart the same deal |
| `?` | Keybind help |
| `Escape` | Cancel selection / dismiss overlay |
| `q` | Quit |
| Mouse click | Select and place cards |

## Themes

Cycle through themes in-game with `t`, or select one from the start menu.

| Name | Description |
|---|---|
| Classic | Green felt — the timeless Solitaire look |
| Dracula | Purple and pink on dark backgrounds |
| Solarized Dark | Warm tones on the Solarized dark base |
| Solarized Light | Warm tones on the Solarized light base |
| Nord | Cool blues and greys from the Nord palette |
| Catppuccin | Catppuccin Mocha — soft pastels on a dark base |
| Tokyo Night | Deep blues with bright accents (Night variant) |

## Scoring

| Action | Points |
|---|---|
| Waste → Tableau | +5 |
| Waste → Foundation | +10 |
| Tableau → Foundation | +10 |
| Foundation → Tableau | −15 |
| Flip face-down card | +5 |
| Recycle stock | −100 |

Score never goes below 0.

## Development

```bash
# Run all tests
go test ./...

# Build
go build -o klondike .

# Visual regression (requires Docker + VHS image)
docker run --rm --privileged --shm-size=512m \
  -v "$PWD:/vhs" -w /vhs \
  ghcr.io/charmbracelet/vhs tapes/board-initial.tape
```

Terminal must be at least **61 × 23** characters; the game shows a warning if the window is too small.

## License

See [LICENSE](LICENSE).
