package renderer

import (
	"image"

	"solituire/engine"
)

// Card and terminal size constants exported for Agent C's hit-testing (T16).
const (
	CardWidth     = 7 // rendered width (no borders)
	CardHeight    = 5 // rendered height (no borders)
	MinTermWidth  = 61
	MinTermHeight = 23
	ColGap        = 1                      // gap between tableau columns
	BoardWidth    = 7*CardWidth + 6*ColGap // = 55, matches 7 tableau columns
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
	// (rows 2..2+CardHeight, always padded to CardHeight+1 for the arrow row),
	// "" spacer, tableau.
	topRow := 2
	// Tableau row starts after the top-row piles (CardHeight+1) + the blank spacer row.
	tabRow := topRow + CardHeight + 2

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

// PileHitTestWithCursor maps (x, y) to a pile and card index using the same
// per-column layout the renderer drew for the given cursor. This matters when a
// tableau column is keyboard-"lifted": the focal card is rendered in full at a
// shifted position, so a cursor-unaware hit test would map clicks to the wrong
// card. Callers that drive mouse input from the live cursor should use this.
//
// Foundation positions are derived from the game state (waste visible count) so
// draw-3 mode is handled correctly. Returns (pileID, cardIndex, true) on hit,
// or (0, 0, false) on miss. cardIndex is 0-based from the top of the pile's
// visible cards. Pass a zero CursorState when no cursor is active.
func PileHitTestWithCursor(x, y int, state *engine.GameState, cursor CursorState) (engine.PileID, int, bool) {
	return pileHitTest(x, y, state, cursor)
}

func pileHitTest(x, y int, state *engine.GameState, cursor CursorState) (engine.PileID, int, bool) {
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

		// Mirror the renderer's layout: the focal card occupies a full CardHeight
		// while every other face-up card is a 1-row peek, and the arrow (when
		// shown) inserts a row directly below the focal card.
		focalFi := tableauFocalFi(pid, fdCount, len(fuCards), cursor)
		arrow := pileShowsArrow(pid, cursor)

		row := o.Y
		// Face-down stubs
		for ci := 0; ci < fdCount; ci++ {
			if y == row {
				return pid, ci, true
			}
			row++
		}
		// Face-up fanned cards: the focal card occupies full CardHeight, the rest
		// occupy 1 row (peek). The arrow row below the focal card is a miss.
		for fi := range fuCards {
			cardIdx := fdCount + fi
			height := 1
			if fi == focalFi {
				height = CardHeight
			}
			if y >= row && y < row+height {
				return pid, cardIdx, true
			}
			row += height
			if fi == focalFi && arrow {
				row++
			}
		}
	}

	return 0, 0, false
}

// hitCard returns true if (x, y) falls within a CardWidth × CardHeight region
// starting at origin o.
func hitCard(x, y int, o image.Point) bool {
	return x >= o.X && x < o.X+CardWidth && y >= o.Y && y < o.Y+CardHeight
}
