package engine

import "fmt"

// Color represents card color for alternating-color tableau rules.
type Color uint8

const (
	Black Color = iota
	Red
)

// Suit represents one of the four card suits.
type Suit uint8

const (
	Spades   Suit = iota // Black
	Hearts               // Red
	Diamonds             // Red
	Clubs                // Black
)

// Color returns the color of the suit.
func (s Suit) Color() Color {
	if s == Hearts || s == Diamonds {
		return Red
	}
	return Black
}

// Symbol returns the Unicode symbol for the suit.
func (s Suit) Symbol() string {
	switch s {
	case Spades:
		return "♠"
	case Hearts:
		return "♥"
	case Diamonds:
		return "♦"
	case Clubs:
		return "♣"
	}
	return "?"
}

// String returns the full name of the suit.
func (s Suit) String() string {
	switch s {
	case Spades:
		return "Spades"
	case Hearts:
		return "Hearts"
	case Diamonds:
		return "Diamonds"
	case Clubs:
		return "Clubs"
	}
	return "Unknown"
}

// Rank represents card rank from Ace (1) to King (13).
type Rank uint8

const (
	Ace   Rank = iota + 1
	Two
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King // = 13
)

// String returns the display string for the rank.
func (r Rank) String() string {
	switch r {
	case Ace:
		return "A"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	default:
		return fmt.Sprintf("%d", int(r))
	}
}

// Card is a single playing card. It is a value type; piles hold []Card.
type Card struct {
	Suit   Suit
	Rank   Rank
	FaceUp bool
}

// Color returns the color of the card's suit.
func (c Card) Color() Color {
	return c.Suit.Color()
}

// String returns a short display string. Face-down cards show "??".
func (c Card) String() string {
	if !c.FaceUp {
		return "??"
	}
	return c.Rank.String() + c.Suit.Symbol()
}
