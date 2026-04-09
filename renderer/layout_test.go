package renderer

import (
	"testing"

	"solituire/engine"
)

// newSeed42DrawState builds the canonical seed-42 draw-1 game state used
// across layout hit-test tests.
func newSeed42DrawState() *engine.GameState {
	deck := engine.NewDeck()
	engine.Shuffle(deck, 42)
	state := engine.Deal(deck, 1)
	state.Seed = 42
	return state
}

// TestPileHitTestWithWidth exercises pileHitTestWithWidth with known terminal
// coordinates derived from the seed-42 draw-1 deal at termWidth=MinTermWidth.
//
// Layout geometry at termWidth=78:
//
//	topRow = 2   (header row 0 + blank spacer row 1)
//	tabRow = 10  (topRow + CardHeight(7) + blank spacer(1))
//
//	foundationStartX(78) = 78 - (4×9 + 3×1) = 78 - 39 = 39
//
//	Pile         X    Y-start
//	Stock        0    2
//	Waste        10   2
//	Foundation0  39   2
//	Foundation1  49   2
//	Foundation2  59   2
//	Foundation3  69   2
//	Tableau0     0    10
//	Tableau1     10   10   (1 fd stub at row 10, 1 fu at rows 11..17)
//	Tableau6     60   10   (6 fd stubs rows 10..15, 1 fu at rows 16..22)
func TestPileHitTestWithWidth(t *testing.T) {
	state := newSeed42DrawState()

	type wantHit struct {
		pile      engine.PileID
		cardIndex int
	}

	tests := []struct {
		name string
		x, y int
		want *wantHit // nil = expect miss (ok=false)
	}{
		// ── Stock ────────────────────────────────────────────────────────────
		{"stock top-left corner", 0, 2, &wantHit{engine.PileStock, 0}},
		{"stock center", 4, 5, &wantHit{engine.PileStock, 0}},
		{"stock bottom-right corner", 8, 8, &wantHit{engine.PileStock, 0}},

		// ── Waste (empty at deal time, wasteVisCount clamps to 1) ────────────
		{"waste left edge", 10, 2, &wantHit{engine.PileWaste, 0}},
		{"waste center", 14, 5, &wantHit{engine.PileWaste, 0}},
		{"waste right edge", 18, 8, &wantHit{engine.PileWaste, 0}},

		// ── Foundations ──────────────────────────────────────────────────────
		{"foundation 0", 39, 2, &wantHit{engine.PileFoundation0, 0}},
		{"foundation 1", 49, 4, &wantHit{engine.PileFoundation1, 0}},
		{"foundation 2", 59, 6, &wantHit{engine.PileFoundation2, 0}},
		{"foundation 3", 69, 2, &wantHit{engine.PileFoundation3, 0}},

		// ── Tableau T0 (0 fd, 1 fu occupying rows 10..16) ────────────────────
		{"T0 fu top row", 4, 10, &wantHit{engine.PileTableau0, 0}},
		{"T0 fu bottom row", 4, 16, &wantHit{engine.PileTableau0, 0}},
		{"T0 center", 4, 13, &wantHit{engine.PileTableau0, 0}},

		// ── Tableau T1 (1 fd at row 10, 1 fu at rows 11..17) ─────────────────
		{"T1 fd stub", 14, 10, &wantHit{engine.PileTableau1, 0}},
		{"T1 fu top row", 14, 11, &wantHit{engine.PileTableau1, 1}},
		{"T1 fu bottom row", 14, 17, &wantHit{engine.PileTableau1, 1}},

		// ── Tableau T2 (2 fd at rows 10,11; 1 fu at rows 12..18) ─────────────
		{"T2 fd stub 0", 24, 10, &wantHit{engine.PileTableau2, 0}},
		{"T2 fd stub 1", 24, 11, &wantHit{engine.PileTableau2, 1}},
		{"T2 fu card", 24, 12, &wantHit{engine.PileTableau2, 2}},
		{"T2 fu bottom row", 24, 18, &wantHit{engine.PileTableau2, 2}},

		// ── Tableau T6 (6 fd stubs rows 10..15; 1 fu at rows 16..22) ─────────
		{"T6 fd stub 0", 64, 10, &wantHit{engine.PileTableau6, 0}},
		{"T6 fd stub 3", 64, 13, &wantHit{engine.PileTableau6, 3}},
		{"T6 fd stub 5", 64, 15, &wantHit{engine.PileTableau6, 5}},
		{"T6 fu card top row", 64, 16, &wantHit{engine.PileTableau6, 6}},
		{"T6 fu card bottom row", 64, 22, &wantHit{engine.PileTableau6, 6}},
		{"T6 last valid x", 68, 16, &wantHit{engine.PileTableau6, 6}},

		// ── Misses ───────────────────────────────────────────────────────────
		{"above top row", 4, 1, nil},
		{"stock-waste gap", 9, 2, nil},
		{"gap before foundation 0", 38, 2, nil},
		{"gap row below top piles", 4, 9, nil}, // tabRow=10; row 9 is the blank spacer
		{"right of T6", 69, 10, nil},           // T6 occupies x=[60,68]; x=69 is outside
		{"below T0 fu card", 4, 17, nil},        // T0 fu ends at row 16 (10+7-1)
		{"far below tableau", 4, 30, nil},
		{"between waste and foundation", 28, 2, nil},
	}

	termWidth := MinTermWidth // 78

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pile, cardIdx, ok := PileHitTestWithWidth(tt.x, tt.y, state, termWidth)
			if tt.want == nil {
				if ok {
					t.Errorf("expected miss at (%d,%d), got pile=%v cardIndex=%d",
						tt.x, tt.y, pile, cardIdx)
				}
				return
			}
			if !ok {
				t.Errorf("expected hit at (%d,%d), got miss", tt.x, tt.y)
				return
			}
			if pile != tt.want.pile {
				t.Errorf("(%d,%d): pile = %v, want %v", tt.x, tt.y, pile, tt.want.pile)
			}
			if cardIdx != tt.want.cardIndex {
				t.Errorf("(%d,%d): cardIndex = %d, want %d", tt.x, tt.y, cardIdx, tt.want.cardIndex)
			}
		})
	}
}

// TestPileHitTest_DefaultWidth verifies that PileHitTest (no explicit width)
// returns the same result as PileHitTestWithWidth(MinTermWidth) for a
// coordinate that hits the stock pile regardless of terminal width.
func TestPileHitTest_DefaultWidth(t *testing.T) {
	state := newSeed42DrawState()
	// Stock is at (0,2) regardless of terminal width.
	p1, c1, ok1 := PileHitTest(4, 5, state)
	p2, c2, ok2 := PileHitTestWithWidth(4, 5, state, MinTermWidth)
	if ok1 != ok2 || p1 != p2 || c1 != c2 {
		t.Errorf("PileHitTest and PileHitTestWithWidth(MinTermWidth) disagree: "+
			"got (%v,%d,%v) vs (%v,%d,%v)", p1, c1, ok1, p2, c2, ok2)
	}
}

// TestPileHitTestWaste_Draw3Expansion verifies that in draw-3 mode after a
// stock flip, clicking any of the visible side-by-side waste cards registers
// as PileWaste — the hit region expands to wasteVisCount*CardWidth.
func TestPileHitTestWaste_Draw3Expansion(t *testing.T) {
	deck := engine.NewDeck()
	engine.Shuffle(deck, 42)
	state := engine.Deal(deck, 3)
	state.DrawCount = 3

	// Flip stock once; in draw-3 mode up to 3 cards go to waste.
	flipCmd := &engine.FlipStockCmd{}
	if err := flipCmd.Execute(state); err != nil {
		t.Fatalf("FlipStockCmd.Execute: %v", err)
	}

	visCount := len(state.Waste.VisibleCards())
	if visCount < 2 {
		t.Skip("fewer than 2 waste cards visible — cannot test draw-3 expansion")
	}

	// wasteOriginX = CardWidth + ColGap = 9 + 1 = 10
	wasteOriginX := CardWidth + ColGap
	// Click in the rightmost visible waste card.
	rightCardX := wasteOriginX + (visCount-1)*CardWidth + 1
	clickY := 4 // within top-row Y range [2, 2+CardHeight-1] = [2, 8]

	pile, _, ok := PileHitTestWithWidth(rightCardX, clickY, state, MinTermWidth)
	if !ok || pile != engine.PileWaste {
		t.Errorf("rightmost draw-3 waste card at x=%d: got pile=%v ok=%v, want PileWaste ok=true",
			rightCardX, pile, ok)
	}
}

// TestPileHitTestWithWidth_WiderTerminal verifies that foundation positions
// shift right when a wider terminal is used, keeping the gap between waste
// and foundations free.
func TestPileHitTestWithWidth_WiderTerminal(t *testing.T) {
	state := newSeed42DrawState()
	const wide = 120

	// With termWidth=120: foundationStartX = 120 - 39 = 81.
	// F0 occupies x=[81,89].  A click at (39, 2) — which is F0 at termWidth=78 —
	// must be a miss at termWidth=120.
	_, _, ok := PileHitTestWithWidth(39, 2, state, wide)
	if ok {
		t.Errorf("x=39 should miss all piles at termWidth=120 (F0 starts at x=81)")
	}

	// F0 at x=81 must be a hit.
	pile, _, ok := PileHitTestWithWidth(81, 2, state, wide)
	if !ok || pile != engine.PileFoundation0 {
		t.Errorf("x=81 at termWidth=120: got pile=%v ok=%v, want Foundation0 ok=true",
			pile, ok)
	}
}
