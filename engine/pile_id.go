package engine

// PileID identifies a pile in GameState without pointer equality.
type PileID uint8

const (
	PileStock PileID = iota
	PileWaste
	PileFoundation0 // foundations are PileFoundation0 + index (0-3)
	PileFoundation1
	PileFoundation2
	PileFoundation3
	PileTableau0 // tableau columns are PileTableau0 + index (0-6)
	PileTableau1
	PileTableau2
	PileTableau3
	PileTableau4
	PileTableau5
	PileTableau6
)

// IsTableau reports whether p identifies one of the seven tableau columns.
func (p PileID) IsTableau() bool {
	return p >= PileTableau0 && p <= PileTableau6
}

// IsFoundation reports whether p identifies one of the four foundation piles.
func (p PileID) IsFoundation() bool {
	return p >= PileFoundation0 && p <= PileFoundation3
}

// TableauIndex returns the 0-based column index for a tableau PileID.
// The result is only meaningful when IsTableau reports true.
func (p PileID) TableauIndex() int {
	return int(p - PileTableau0)
}

// FoundationIndex returns the 0-based index for a foundation PileID.
// The result is only meaningful when IsFoundation reports true.
func (p PileID) FoundationIndex() int {
	return int(p - PileFoundation0)
}
