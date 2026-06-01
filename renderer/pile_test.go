package renderer

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
	"solituire/engine"
	"solituire/theme"
)

// init locking lipgloss to the ASCII profile lives in card_test.go and applies
// to the whole package test binary.

// liftTestPile returns a tableau column with 2 face-down cards beneath a 4-card
// face-up run (K♠ Q♥ J♠ 10♥). Face-up cards occupy slice indices 2,3,4,5. It is
// used to exercise the lifted-focal rendering and the matching hit-test
// geometry — the seed-42 deal never has a multi-card face-up run.
func liftTestPile() *engine.TableauPile {
	return &engine.TableauPile{Cards: []engine.Card{
		{Suit: engine.Spades, Rank: engine.Five, FaceUp: false},
		{Suit: engine.Hearts, Rank: engine.Four, FaceUp: false},
		{Suit: engine.Spades, Rank: engine.King, FaceUp: true},
		{Suit: engine.Hearts, Rank: engine.Queen, FaceUp: true},
		{Suit: engine.Spades, Rank: engine.Jack, FaceUp: true},
		{Suit: engine.Hearts, Rank: engine.Ten, FaceUp: true},
	}}
}

// TestRenderTableauPileLifted verifies that the cursor's focal card is rendered
// in full with the arrow beneath it, while the cards below it become a small
// stack of peeks. The bottom-card case must match the historical layout (arrow
// pinned to the bottom of the column).
func TestRenderTableauPileLifted(t *testing.T) {
	pile := liftTestPile()
	tests := []struct {
		name      string
		cardIndex int
	}{
		{"bottom_card", 5}, // focal = 10♥ (bottom): arrow at bottom, no visible lift
		{"middle_card", 3}, // focal = Q♥: J♠ & 10♥ form a stack below the arrow
		{"top_card", 2},    // focal = K♠: the whole run trails below the arrow
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := CursorState{Pile: engine.PileTableau0, CardIndex: tt.cardIndex}
			got := RenderTableauPile(pile, 0, cursor, theme.Classic)
			golden.RequireEqual(t, []byte(got))
		})
	}
}

// TestRenderTableauPileEmptyArrow guards against regressing the empty-column
// arrow: RenderTableauPile now embeds its own arrow, so the empty-pile path must
// still draw one when the cursor is on it (and none otherwise).
func TestRenderTableauPileEmptyArrow(t *testing.T) {
	empty := &engine.TableauPile{}

	withCursor := RenderTableauPile(empty, 0, CursorState{Pile: engine.PileTableau0}, theme.Classic)
	if !strings.Contains(withCursor, "↑") {
		t.Errorf("empty pile under cursor should show an arrow, got:\n%s", withCursor)
	}

	noCursor := RenderTableauPile(empty, 0, CursorState{Pile: engine.PileStock}, theme.Classic)
	if strings.Contains(noCursor, "↑") {
		t.Errorf("empty pile without cursor should not show an arrow, got:\n%s", noCursor)
	}
}

// TestPileHitTestLiftedColumn verifies that mouse hit-testing follows the lifted
// layout the renderer drew: the focal card occupies a full CardHeight, the arrow
// row beneath it is a miss, and the trailing peeks map to the correct indices.
func TestPileHitTestLiftedColumn(t *testing.T) {
	state := newSeed42DrawState()
	state.Tableau[0] = liftTestPile()

	// Cursor lifts the focal card to Q♥ (CardIndex 3 = face-up index 1).
	// Column 0 origin X=0, Y=9. Rows: 9,10 face-down stubs; 11 K♠ peek;
	// 12..16 Q♥ focal (full); 17 arrow (miss); 18 J♠ peek; 19 10♥ peek.
	lifted := CursorState{Pile: engine.PileTableau0, CardIndex: 3}

	cases := []struct {
		name    string
		y       int
		wantIdx int
		wantOK  bool
	}{
		{"face_down_top", 9, 0, true},
		{"king_peek", 11, 2, true},
		{"queen_focal_full", 14, 3, true},
		{"arrow_row_miss", 17, 0, false},
		{"jack_peek", 18, 4, true},
		{"ten_peek", 19, 5, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pile, idx, ok := PileHitTestWithCursor(3, c.y, state, lifted)
			if ok != c.wantOK {
				t.Fatalf("ok=%v, want %v (pile=%v idx=%d)", ok, c.wantOK, pile, idx)
			}
			if c.wantOK && (pile != engine.PileTableau0 || idx != c.wantIdx) {
				t.Fatalf("got (%v, %d), want (Tableau0, %d)", pile, idx, c.wantIdx)
			}
		})
	}

	// Without the lift, the same y=14 click maps to the bottom card (10♥, index
	// 5) because the bottom card is the one rendered full — proving the hit test
	// follows the cursor-driven layout.
	if pile, idx, ok := PileHitTest(3, 14, state); !ok || pile != engine.PileTableau0 || idx != 5 {
		t.Fatalf("default hit test got (%v, %d, %v), want (Tableau0, 5, true)", pile, idx, ok)
	}
}
