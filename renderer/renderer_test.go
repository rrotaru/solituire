package renderer

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"
	"solituire/config"
	"solituire/engine"
	"solituire/theme"
)

// init is defined in card_test.go and locks lipgloss to ASCII profile.

// newSeed42State builds the canonical seed-42 game state used across tests.
func newSeed42State(drawCount int) *engine.GameState {
	deck := engine.NewDeck()
	engine.Shuffle(deck, 42)
	state := engine.Deal(deck, drawCount)
	state.Seed = 42
	return state
}

func TestRendererFullBoard(t *testing.T) {
	state := newSeed42State(1)
	r := New(theme.Classic)
	r.SetSize(80, 30)

	cfg := config.DefaultConfig()
	cursor := CursorState{Pile: engine.PileTableau0, CardIndex: 0}

	got := r.Render(state, cursor, cfg)
	golden.RequireEqual(t, []byte(got))
}

func TestRendererFullBoardDraw3(t *testing.T) {
	state := newSeed42State(3)
	r := New(theme.Classic)
	r.SetSize(80, 30)

	cfg := &config.Config{DrawCount: 3, ThemeName: "classic"}
	cursor := CursorState{Pile: engine.PileStock, CardIndex: 0}

	got := r.Render(state, cursor, cfg)
	golden.RequireEqual(t, []byte(got))
}

func TestRendererTooSmall(t *testing.T) {
	state := newSeed42State(1)
	r := New(theme.Classic)
	r.SetSize(40, 12)

	cfg := config.DefaultConfig()
	cursor := CursorState{}

	got := r.Render(state, cursor, cfg)
	golden.RequireEqual(t, []byte(got))
}
