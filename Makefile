# Makefile — developer task runner for solituire
# Run 'make' or 'make check' to execute the full local CI suite.
# Run 'make setup' on a fresh clone to install tools and git hooks.

BINARY     := klondike
GO         := go
GOLANGCI   := golangci-lint
LEFTHOOK   := lefthook
VHS_IMAGE  := ghcr.io/charmbracelet/vhs
VHS_FLAGS  := --rm --privileged --shm-size=512m -v "$(CURDIR):/vhs" -w /vhs

.PHONY: all build test test-race lint fmt vet tidy check \
        golden-update vhs hooks setup vuln clean

# Default: run the full local CI suite.
all: check

## ── Build ────────────────────────────────────────────────────────────────────

build:
	$(GO) build -o $(BINARY) .

## ── Quality checks ───────────────────────────────────────────────────────────

# Run all quality checks (mirrors what CI and pre-push hooks enforce).
check: fmt vet lint tidy test build

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	$(GOLANGCI) run --timeout=3m

# Verify go.mod and go.sum are tidy (no drift from module graph).
tidy:
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum

test:
	$(GO) test ./...

# Run tests with the race detector — useful before significant refactors.
# The bubbletea event loop uses goroutines, so races are plausible.
test-race:
	$(GO) test -race ./...

# Scan dependencies for known vulnerabilities.
# Recommended before releases or after bumping dependencies.
vuln:
	govulncheck ./...

## ── Test fixtures ────────────────────────────────────────────────────────────

# Regenerate all golden files after renderer or TUI layout changes.
# Review the diff carefully before committing — these are your visual baselines.
golden-update:
	$(GO) test ./renderer/... ./tui/... -update

## ── VHS documentation assets ─────────────────────────────────────────────────

# Run all VHS tapes and regenerate docs/demo/*.gif and testdata/vhs/*.png.
# Requires Docker. Run after visual or gameplay changes that should be
# reflected in the README screenshots and animated demos.
vhs:
	mkdir -p testdata/vhs docs/demo
	@for tape in tapes/*.tape; do \
		echo "=== $$tape ==="; \
		docker run $(VHS_FLAGS) $(VHS_IMAGE) "$$tape"; \
	done

## ── Git hooks ────────────────────────────────────────────────────────────────

hooks:
	$(LEFTHOOK) install

## ── Developer setup ──────────────────────────────────────────────────────────

# Install all required developer tools and git hooks.
# Run once after cloning, or when tool versions change.
setup:
	@echo "Installing golangci-lint..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing govulncheck..."
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Installing lefthook..."
	$(GO) install github.com/evilmartians/lefthook@latest
	@echo "Installing git hooks..."
	$(MAKE) hooks
	@echo ""
	@echo "Setup complete. Run 'make check' to verify everything works."

## ── Housekeeping ─────────────────────────────────────────────────────────────

clean:
	rm -f $(BINARY)
