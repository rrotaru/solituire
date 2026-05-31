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
// coordinates derived from the seed-42 draw-1 deal (empty waste, visCount=1).
//
// Layout geometry (draw-1, wasteVisCount=1):
//
//	topRow = 2   (header row 0 + blank spacer row 1)
//	tabRow = 8   (topRow + CardHeight(5) + blank spacer(1))
//
//	computeFoundationStartX(1): leftWidth=15, gap=9 → fStartX=24
//
//	Pile         X    Y-start
//	Stock        0    2
//	Waste        8    2
//	Foundation0  24   2
//	Foundation1  32   2
//	Foundation2  40   2
//	Foundation3  48   2
//	Tableau0     0    8
//	Tableau1     8    8   (1 fd stub at row 8, 1 fu at rows 9..13)
//	Tableau6     48   8   (6 fd stubs rows 8..13, 1 fu at rows 14..18)
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
		// ── Stock (x=[0,6], y=[2,6]) ─────────────────────────────────────────
		{"stock top-left corner", 0, 2, &wantHit{engine.PileStock, 0}},
		{"stock center", 3, 4, &wantHit{engine.PileStock, 0}},
		{"stock bottom-right corner", 6, 6, &wantHit{engine.PileStock, 0}},

		// ── Waste (x=[8,14], y=[2,6]) ────────────────────────────────────────
		{"waste left edge", 8, 2, &wantHit{engine.PileWaste, 0}},
		{"waste center", 11, 4, &wantHit{engine.PileWaste, 0}},
		{"waste right edge", 14, 6, &wantHit{engine.PileWaste, 0}},

		// ── Foundations (y=[2,6]) ─────────────────────────────────────────────
		{"foundation 0", 26, 2, &wantHit{engine.PileFoundation0, 0}},
		{"foundation 1", 34, 4, &wantHit{engine.PileFoundation1, 0}},
		{"foundation 2", 42, 2, &wantHit{engine.PileFoundation2, 0}},
		{"foundation 3", 50, 2, &wantHit{engine.PileFoundation3, 0}},

		// ── Tableau T0 (0 fd, 1 fu occupying rows 8..12) ─────────────────────
		{"T0 fu top row", 4, 8, &wantHit{engine.PileTableau0, 0}},
		{"T0 fu bottom row", 4, 12, &wantHit{engine.PileTableau0, 0}},
		{"T0 center", 4, 10, &wantHit{engine.PileTableau0, 0}},

		// ── Tableau T1 (1 fd at row 8, 1 fu at rows 9..13) ───────────────────
		{"T1 fd stub", 10, 8, &wantHit{engine.PileTableau1, 0}},
		{"T1 fu top row", 10, 9, &wantHit{engine.PileTableau1, 1}},
		{"T1 fu bottom row", 10, 13, &wantHit{engine.PileTableau1, 1}},

		// ── Tableau T2 (2 fd at rows 8,9; 1 fu at rows 10..14) ───────────────
		{"T2 fd stub 0", 20, 8, &wantHit{engine.PileTableau2, 0}},
		{"T2 fd stub 1", 20, 9, &wantHit{engine.PileTableau2, 1}},
		{"T2 fu card", 20, 10, &wantHit{engine.PileTableau2, 2}},
		{"T2 fu bottom row", 20, 14, &wantHit{engine.PileTableau2, 2}},

		// ── Tableau T6 (6 fd stubs rows 8..13; 1 fu at rows 14..18) ──────────
		{"T6 fd stub 0", 50, 8, &wantHit{engine.PileTableau6, 0}},
		{"T6 fd stub 3", 50, 11, &wantHit{engine.PileTableau6, 3}},
		{"T6 fd stub 5", 50, 13, &wantHit{engine.PileTableau6, 5}},
		{"T6 fu card top row", 50, 14, &wantHit{engine.PileTableau6, 6}},
		{"T6 fu card bottom row", 50, 18, &wantHit{engine.PileTableau6, 6}},
		{"T6 last valid x", 54, 14, &wantHit{engine.PileTableau6, 6}},

		// ── Misses ───────────────────────────────────────────────────────────
		{"above top row", 4, 1, nil},
		{"stock-waste gap", 7, 2, nil},
		{"gap before foundation 0", 20, 2, nil},
		{"gap row below top piles", 4, 7, nil}, // tabRow=8; row 7 is the blank spacer
		{"right of T6", 55, 10, nil},           // T6 occupies x=[48,54]; x=55 is outside
		{"below T0 fu card", 4, 13, nil},       // T0 fu ends at row 12 (8+5-1)
		{"far below tableau", 4, 30, nil},
		{"between waste and foundation", 18, 2, nil},
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

	// wasteOriginX = CardWidth + ColGap = 7 + 1 = 8
	wasteOriginX := CardWidth + ColGap
	// Click in the rightmost visible waste card.
	rightCardX := wasteOriginX + (visCount-1)*CardWidth + 1
	clickY := 4 // within top-row Y range [2, 2+CardHeight-1] = [2, 6]

	pile, _, ok := PileHitTestWithWidth(rightCardX, clickY, state, MinTermWidth)
	if !ok || pile != engine.PileWaste {
		t.Errorf("rightmost draw-3 waste card at x=%d: got pile=%v ok=%v, want PileWaste ok=true",
			rightCardX, pile, ok)
	}
}

// TestPileHitTestWithWidth_WiderTerminal verifies that foundation positions
// are fixed relative to the tableau width and do not shift with terminal width.
func TestPileHitTestWithWidth_WiderTerminal(t *testing.T) {
	state := newSeed42DrawState()

	// Foundation x-positions come from computeFoundationStartX(wasteVisCount),
	// not from termWidth. F0 starts at x=24 (draw-1); x=28 is within F0=[24,30].
	for _, termWidth := range []int{60, 120, 200} {
		pile, _, ok := PileHitTestWithWidth(28, 2, state, termWidth)
		if !ok || pile != engine.PileFoundation0 {
			t.Errorf("termWidth=%d: x=28 got pile=%v ok=%v, want Foundation0 ok=true",
				termWidth, pile, ok)
		}
		// A click well past the layout boundary must miss all piles.
		_, _, ok = PileHitTestWithWidth(70, 2, state, termWidth)
		if ok {
			t.Errorf("termWidth=%d: x=70 should miss all piles", termWidth)
		}
	}
}

// TestFoundationHitTestDraw3 verifies that in draw-3 mode with 3 visible
// waste cards the foundation hit regions match the rendered positions.
//
// With wasteVisCount=3: leftWidth=29, gap=max(1,-5)=1 → fStartX=30.
// Foundation x-ranges: F0=[30,36], F1=[38,44], F2=[46,52], F3=[54,60].
// Clicking x=20 is inside the expanded waste hit region (x=[8,28]).
func TestFoundationHitTestDraw3(t *testing.T) {
	deck := engine.NewDeck()
	engine.Shuffle(deck, 42)
	state := engine.Deal(deck, 3)
	state.DrawCount = 3

	flipCmd := &engine.FlipStockCmd{}
	if err := flipCmd.Execute(state); err != nil {
		t.Fatalf("FlipStockCmd.Execute: %v", err)
	}

	visCount := len(state.Waste.VisibleCards())
	if visCount < 3 {
		t.Skipf("expected 3 visible waste cards after first draw-3 flip, got %d", visCount)
	}

	// computeFoundationStartX(3) = 30
	fStartX := computeFoundationStartX(visCount)
	if fStartX != 30 {
		t.Fatalf("computeFoundationStartX(%d) = %d, want 30", visCount, fStartX)
	}

	// F0 must be a hit at the actual rendered x.
	pile, _, ok := PileHitTestWithWidth(fStartX, 2, state, MinTermWidth)
	if !ok || pile != engine.PileFoundation0 {
		t.Errorf("draw-3 F0 at x=%d: got pile=%v ok=%v, want Foundation0 ok=true", fStartX, pile, ok)
	}

	// x=20 is inside the expanded waste hit region (x=[8,28]), so it resolves
	// to PileWaste — not Foundation0, which now starts at x=30.
	wastePile, _, ok := PileHitTestWithWidth(20, 2, state, MinTermWidth)
	if !ok || wastePile != engine.PileWaste {
		t.Errorf("draw-3: x=20 should hit PileWaste (waste expands to x=[8,28]), got pile=%v ok=%v",
			wastePile, ok)
	}
}
