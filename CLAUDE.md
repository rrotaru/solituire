# CLAUDE.md — Agent guide for solituire

## Mandatory setup

Run this once at the start of every session before making any changes:

```bash
make setup
```

This installs golangci-lint, govulncheck, and lefthook, then wires up the
pre-commit, pre-push, and commit-msg git hooks. Without it, commits will
not be validated and the lint toolchain will not be available.

If `make setup` has already been run in this environment, run at minimum:

```bash
make hooks   # re-installs git hooks
```

## Before every commit

```bash
make check
```

This runs the full local CI suite in order: `fmt → vet → lint → tidy → test → build`.
All steps must pass with zero errors before committing. Fix failures before
proceeding — do not bypass hooks with `LEFTHOOK=0`.

## Project overview

Go 1.24 TUI application (Klondike Solitaire) built with bubbletea + lipgloss.

```
engine/      Game rules, state machine, scoring, hint engine
renderer/    ANSI terminal rendering (cards, piles, layout)
tui/         Bubbletea app model, input handling, menus, cursor
theme/       5 built-in colour themes
config/      Persistent user configuration
```

## Key commands

| Command | Purpose |
|---|---|
| `make build` | Build `./klondike` binary |
| `make test` | Run all tests |
| `make lint` | Run golangci-lint |
| `make fmt` | Auto-format all Go files |
| `make check` | Full local CI (run before every commit) |
| `make golden-update` | Regenerate golden snapshot files |
| `make test-race` | Run tests with race detector |

## Commit message format

All commits **must** follow [Conventional Commits](https://www.conventionalcommits.org/).
The `commit-msg` hook enforces this — commits with non-conforming messages are
rejected.

```
<type>[optional scope]: <description>

Types: feat  fix  docs  style  refactor  test  chore  perf  ci  build  revert
```

Examples:
```
feat: add nord theme
fix(renderer): correct card overlap in draw-3 mode
test: add golden file for draw-3 board
chore: update dependencies
```

Semver impact: `feat` → minor bump, `fix` → patch bump, `feat!` → major bump.

## Golden files

The renderer and TUI tests use snapshot (golden) files for visual regression.
If your change intentionally alters rendered output, regenerate them:

```bash
make golden-update
```

Then review the diff (`git diff renderer/testdata/ tui/testdata/`) and commit
the updated `.golden` files alongside the code change. Never update golden
files without reviewing the diff.

## VHS assets

`docs/demo/*.gif` and `testdata/vhs/*.png` are generated from tape scripts in
`tapes/`. **These must be regenerated whenever a UI change affects the rendered
output** (card layout, colours, menus, animations, etc.). Commit updated GIFs
alongside the code change.

### Running VHS locally (no Docker)

Install VHS and its dependencies once:

```bash
go install github.com/charmbracelet/vhs@v0.9.0
# ffmpeg and ttyd must also be present (apt install ffmpeg ttyd on Debian/Ubuntu)
```

Then regenerate all tapes (mirrors what `make vhs` and CI do):

```bash
export PATH=$PATH:$(go env GOPATH)/bin
for tape in tapes/*.tape; do VHS_NO_SANDBOX=true vhs "$tape"; done
```

`VHS_NO_SANDBOX=true` is required when running as root (e.g. in CI containers).
Running all tapes updates both `docs/demo/*.gif` and `testdata/vhs/*.png`.

### Running VHS via Docker

```bash
make vhs   # requires a running Docker daemon
```

After either method, review the diff and commit the updated files:

```bash
git diff docs/demo/ testdata/vhs/
git add docs/demo/*.gif testdata/vhs/*.png
```

`testdata/vhs/*.png` are visual baseline screenshots committed alongside the
GIFs — they must be kept in sync when rendering changes.

## Linting

The project must stay lint-clean (`golangci-lint run` reports 0 issues).
If you introduce new code, verify before committing:

```bash
make lint
```

The pre-commit hook uses `--new-from-rev HEAD` to check only newly introduced
issues. `make lint` (used in `make check`) runs the full analysis — both must
pass.

## Test fixtures note

Only `renderer/` and `tui/` packages define the `-update` flag (via
`charmbracelet/x/exp/golden`). `make golden-update` is scoped to those
packages. Do not pass `-update` to `go test ./...` directly.
