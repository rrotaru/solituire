# Contributing to solituire

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.24+ | Build and test |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | latest | Linting |
| [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) | latest | Vulnerability scanning |
| [lefthook](https://github.com/evilmartians/lefthook) | latest | Git hooks |
| [Docker](https://docs.docker.com/get-docker/) | any recent | VHS tape rendering (optional) |

Docker is only required if you want to regenerate the animated GIF demos or
run the full visual smoke-test suite locally. All other checks work without it.

## First-time setup

```bash
git clone https://github.com/rrotaru/solituire
cd solituire
make setup
```

`make setup` installs golangci-lint, govulncheck, and lefthook via `go install`,
then runs `lefthook install` to wire up the git hooks.

If you already have the tools installed, just install the hooks:

```bash
make hooks
```

## Git hooks

Three hooks are configured in `lefthook.yml`:

### `pre-commit` (runs on every `git commit`)

| Check | What it does |
|---|---|
| `gofmt` | Auto-formats any staged `.go` files and re-stages the result |
| `go vet` | Runs static analysis across the whole module |
| `golangci-lint` | Runs the full linter suite (see `.golangci.yml`) |

Target runtime: under 30 seconds.

### `pre-push` (runs on every `git push`)

| Check | What it does |
|---|---|
| `go mod tidy` | Verifies `go.mod`/`go.sum` are in sync with the module graph |
| `go test ./...` | Runs the full test suite including golden-file tests |
| `go build ./...` | Verifies the module compiles cleanly |

Target runtime: under 3 minutes.

### `commit-msg` (runs on every `git commit`)

Enforces [Conventional Commits](https://www.conventionalcommits.org/) format.
This is required because CI uses `go-semantic-release` to derive version bumps
from commit messages automatically.

```
<type>[optional scope][optional !]: <description>

Types: feat  fix  docs  style  refactor  test  chore  perf  ci  build  revert
```

**Examples:**

```
feat: add nord theme
fix(renderer): correct card overlap in draw-3 mode
docs: update controls table in README
feat!: change config file format  ← breaking change, bumps major version
```

**Semver mapping:**

| Commit prefix | Version bump |
|---|---|
| `feat:` | Minor (0.x.0) |
| `fix:` | Patch (0.0.x) |
| `feat!:` or `BREAKING CHANGE:` | Major (x.0.0) |
| `chore:`, `docs:`, `ci:` | No release |

### Bypassing hooks in an emergency

```bash
LEFTHOOK=0 git commit -m "..."   # skip all hooks for one commit
LEFTHOOK=0 git push              # skip all hooks for one push
```

Use this sparingly — the hooks exist to keep CI green.

To reinstall hooks after an accidental uninstall:

```bash
make hooks
```

## Running checks manually

Every hook maps directly to a `make` target:

```bash
make fmt          # auto-format (gofmt -w .)
make vet          # go vet ./...
make lint         # golangci-lint run
make tidy         # go mod tidy + diff check
make test         # go test ./...
make build        # go build -o klondike .
make check        # all of the above in sequence (full local CI)
make test-race    # go test -race ./... (race detector)
make vuln         # govulncheck ./... (vulnerability scan)
```

`make check` is the canonical "does this branch pass CI?" command.

## Regenerating test fixtures (golden files)

The renderer and TUI tests use [charmbracelet/x/exp/golden](https://github.com/charmbracelet/x/tree/main/exp/golden)
for snapshot testing. Each golden file contains the expected ANSI terminal
output for a given render call.

**When to update:** after any change to rendering, layout, themes, or card
symbols that intentionally changes the visual output.

**How to update:**

```bash
make golden-update   # go test ./... -update
```

Then review the diff:

```bash
git diff renderer/testdata/ tui/testdata/
```

Confirm the changes look correct and commit the updated `.golden` files
alongside your code change. Never update golden files without reviewing them —
they are the visual regression baseline.

**Where they live:**

```
renderer/testdata/          ← card rendering and full-board layout snapshots
tui/testdata/               ← board view, menu, celebration screen snapshots
```

## Regenerating documentation assets (VHS)

The README demos and visual smoke tests are produced from VHS tape scripts.

**Requires:** Docker (pulls `ghcr.io/charmbracelet/vhs` automatically).

**When to update:** after changes that affect the gameplay, UI, or themes in
ways that should be visible in the README or in the board-initial smoke test.

**How to update all tapes at once:**

```bash
make vhs
```

This mounts the repo into the VHS Docker container and runs every `.tape` file
in `tapes/`. Output lands in:

```
docs/demo/gameplay.gif       ← README gameplay animation
docs/demo/themes.gif         ← README themes animation
testdata/vhs/board-initial.png  ← board layout smoke-test screenshot
```

Review the outputs visually before committing them.

**Running a single tape:**

```bash
docker run --rm --privileged --shm-size=512m \
  -v "$PWD:/vhs" -w /vhs \
  ghcr.io/charmbracelet/vhs tapes/demo-gameplay.tape
```

**Available tapes:**

| Tape | Output | Purpose |
|---|---|---|
| `tapes/demo-gameplay.tape` | `docs/demo/gameplay.gif` | README gameplay demo |
| `tapes/demo-themes.tape` | `docs/demo/themes.gif` | README themes demo |
| `tapes/board-initial.tape` | `testdata/vhs/board-initial.png` | Board layout smoke test |
| `tapes/card-select.tape` | — | Card selection smoke test |
| `tapes/screens.tape` | — | Screen navigation smoke test |
| `tapes/theme-cycle.tape` | — | Theme cycling smoke test |
| `tapes/too-small.tape` | — | Terminal-too-small warning smoke test |

## Build

Local development build:

```bash
make build
./klondike
```

The binary is named `klondike`. Release builds are multi-platform (Linux,
macOS, Windows × amd64/arm64) and are produced automatically by GoReleaser
in CI when a new version tag is created.

## Project structure

```
engine/          Game rules, state machine, scoring, hint engine
renderer/        ANSI terminal rendering (cards, piles, layout, header/footer)
tui/             Bubbletea app model, keyboard/mouse input, cursor, menus
theme/           5 built-in colour themes
config/          Persistent user config (draw count, theme)
tapes/           VHS tape scripts for smoke tests and demo GIF generation
docs/demo/       Animated GIF assets used in the README
testdata/vhs/    PNG screenshot outputs from VHS smoke tests
```
