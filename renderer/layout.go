package renderer

import (
	"image"

	"solituire/engine"
)

// Card and terminal size constants exported for Agent C's hit-testing (T16).
const (
	CardWidth     = 9 // total lipgloss-rendered width including borders
	CardHeight    = 7 // max full-card height including borders
	MinTermWidth  = 78
	MinTermHeight = 25
	ColGap        = 1 // gap between tableau columns
)

// stockWasteX returns the x position of the stock pile.
func stockWasteX() int { return 0 }

// computeFoundationStartX returns the x offset of Foundation0 in the rendered
// top row given the number of visible waste cards. It mirrors the gap
// calculation in renderTopRow so that hit-testing and rendering are always
// consistent — including in draw-3 mode where the waste pile can be 2–3 cards
// wide and the gap shrinks (or clamps to 1) accordingly.
func computeFoundationStartX(wasteVisCount int) int {
	if wasteVisCount < 1 {
		wasteVisCount = 1
	}
	tableauWidth := 7*CardWidth + 6*ColGap     // = 69
	foundationsWidth := 4*CardWidth + 3*ColGap // = 39
	leftWidth := CardWidth + ColGap + wasteVisCount*CardWidth
	gap := tableauWidth - leftWidth - foundationsWidth
	if gap < 1 {
		gap = 1
	}
	return leftWidth + gap
}

// tableauX returns the x position of tableau column idx.
func tableauX(idx int) int {
	return idx * (CardWidth + ColGap)
}

// pileOrigins returns the top-left terminal coordinate of each pile's
// render region. Row 0 = header, row 1 = stock/waste/foundation row.
// wasteVisCount is the number of visible waste cards (affects foundation x in
// draw-3 mode); pass 1 when the waste state is not known.
func pileOrigins(wasteVisCount int) map[engine.PileID]image.Point {
	// Render() join order: header (row 0), "" spacer (row 1), top-row piles
	// (rows 2..2+CardHeight-1), "" spacer, tableau.
	topRow := 2
	// Tableau row starts after the top-row piles + the blank spacer row.
	tabRow := topRow + CardHeight + 1

	origins := map[engine.PileID]image.Point{
		engine.PileStock: {X: stockWasteX(), Y: topRow},
		engine.PileWaste: {X: stockWasteX() + CardWidth + ColGap, Y: topRow},
	}

	fStartX := computeFoundationStartX(wasteVisCount)
	for i := 0; i < 4; i++ {
		pid := engine.PileID(engine.PileFoundation0 + engine.PileID(i))
		origins[pid] = image.Point{X: fStartX + i*(CardWidth+ColGap), Y: topRow}
	}

	for i := 0; i < 7; i++ {
		pid := engine.PileID(engine.PileTableau0 + engine.PileID(i))
		origins[pid] = image.Point{X: tableauX(i), Y: tabRow}
	}

	return origins
}

// PileHitTest maps terminal coordinates (x, y) to a pile and card index.
// Foundation positions are derived from the game state (waste visible count)
// so draw-3 mode is handled correctly. Returns (pileID, cardIndex, true) on
// hit, or (0, 0, false) on miss. cardIndex is 0-based from the top of the
// pile's visible cards.
func PileHitTest(x, y int, state *engine.GameState) (engine.PileID, int, bool) {
	return pileHitTestWithWidth(x, y, state, 0)
}

// PileHitTestWithWidth is equivalent to PileHitTest. The termWidth parameter
// is accepted for API compatibility but is no longer used; foundation
// positions are computed from the game state instead.
func PileHitTestWithWidth(x, y int, state *engine.GameState, termWidth int) (engine.PileID, int, bool) {
	return pileHitTestWithWidth(x, y, state, termWidth)
}

func pileHitTestWithWidth(x, y int, state *engine.GameState, _ int) (engine.PileID, int, bool) {
	// Derive waste visible count from state so foundation x-positions match
	// the rendered layout in all draw modes (draw-1 and draw-3).
	wasteVisCount := len(state.Waste.VisibleCards())
	if wasteVisCount < 1 {
		wasteVisCount = 1
	}
	origins := pileOrigins(wasteVisCount)

	// Check stock
	if hitCard(x, y, origins[engine.PileStock]) {
		return engine.PileStock, 0, true
	}

	// Check waste.
	// In draw-3 mode RenderWastePile places multiple full cards side-by-side;
	// the playable top card is the rightmost one. Expand the hit region to
	// cover all visible cards so clicks on the rightmost card are not missed.
	wasteOrigin := origins[engine.PileWaste]
	wasteHitWidth := wasteVisCount * CardWidth
	if x >= wasteOrigin.X && x < wasteOrigin.X+wasteHitWidth &&
		y >= wasteOrigin.Y && y < wasteOrigin.Y+CardHeight {
		return engine.PileWaste, 0, true
	}

	// Check foundations
	for i := 0; i < 4; i++ {
		pid := engine.PileID(engine.PileFoundation0 + engine.PileID(i))
		if hitCard(x, y, origins[pid]) {
			return pid, 0, true
		}
	}

	// Check tableau columns
	for i := 0; i < 7; i++ {
		pid := engine.PileID(engine.PileTableau0 + engine.PileID(i))
		o := origins[pid]
		pile := state.Tableau[i]

		if x < o.X || x >= o.X+CardWidth {
			continue
		}

		if pile.IsEmpty() {
			if y >= o.Y && y < o.Y+CardHeight {
				return pid, 0, true
			}
			continue
		}

		// Each face-down card occupies 1 row (stub top line).
		fdCount := pile.FaceDownCount()
		fuCards := pile.FaceUpCards()

		row := o.Y
		// Face-down stubs
		for ci := 0; ci < fdCount; ci++ {
			if y == row {
				return pid, ci, true
			}
			row++
		}
		// Face-up fanned cards: all but last occupy 2 rows, last occupies full CardHeight
		for fi := range fuCards {
			cardIdx := fdCount + fi
			height := 2
			if fi == len(fuCards)-1 {
				height = CardHeight
			}
			if y >= row && y < row+height {
				return pid, cardIdx, true
			}
			row += height
		}
	}

	return 0, 0, false
}

// hitCard returns true if (x, y) falls within a CardWidth × CardHeight region
// starting at origin o.
func hitCard(x, y int, o image.Point) bool {
	return x >= o.X && x < o.X+CardWidth && y >= o.Y && y < o.Y+CardHeight
}
