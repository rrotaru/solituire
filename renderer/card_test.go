package renderer

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/muesli/termenv"
	"solituire/engine"
	"solituire/theme"
)

func init() {
	// Lock color profile to ASCII for deterministic golden output.
	lipgloss.SetColorProfile(termenv.Ascii)
}

func TestRenderCard(t *testing.T) {
	th := theme.Classic

	tests := []struct {
		name string
		cc   cardContent
	}{
		{
			name: "face_up_red",
			cc: cardContent{
				card:  engine.Card{Suit: engine.Hearts, Rank: engine.Ace, FaceUp: true},
				state: cardNormal,
			},
		},
		{
			name: "face_up_black",
			cc: cardContent{
				card:  engine.Card{Suit: engine.Spades, Rank: engine.King, FaceUp: true},
				state: cardNormal,
			},
		},
		{
			name: "face_down",
			cc:   cardContent{state: cardFaceDown},
		},
		{
			name: "empty_slot",
			cc:   cardContent{state: cardEmpty},
		},
		{
			name: "cursor_hover",
			cc: cardContent{
				card:  engine.Card{Suit: engine.Diamonds, Rank: engine.Queen, FaceUp: true},
				state: cardCursor,
			},
		},
		{
			name: "selected",
			cc: cardContent{
				card:  engine.Card{Suit: engine.Clubs, Rank: engine.Jack, FaceUp: true},
				state: cardSelected,
			},
		},
		{
			name: "hint_target",
			cc: cardContent{
				card:  engine.Card{Suit: engine.Hearts, Rank: engine.Ten, FaceUp: true},
				state: cardHintTo,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderCard(tc.cc, th)
			golden.RequireEqual(t, []byte(got))
		})
	}
}
