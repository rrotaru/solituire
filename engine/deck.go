package engine

import "math/rand"

// NewDeck returns an ordered 52-card deck: Spades A-K, Hearts A-K, Diamonds A-K, Clubs A-K.
// All cards are face-down; Deal flips the appropriate top cards.
func NewDeck() []Card {
	suits := []Suit{Spades, Hearts, Diamonds, Clubs}
	deck := make([]Card, 0, 52)
	for _, suit := range suits {
		for rank := Ace; rank <= King; rank++ {
			deck = append(deck, Card{Suit: suit, Rank: rank, FaceUp: false})
		}
	}
	return deck
}

// Shuffle performs a deterministic Fisher-Yates shuffle of deck in-place using seed.
// The same seed always produces the same order.
func Shuffle(deck []Card, seed int64) []Card {
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

// Deal distributes a shuffled deck into a new GameState following standard Klondike layout:
//   - Column i receives (i+1) cards: bottom i are face-down, top 1 is face-up.
//   - Remaining 24 cards go to Stock (all face-down).
//   - Waste and all Foundations start empty.
//
// The caller is responsible for setting GameState.DrawCount and GameState.Seed after Deal.
func Deal(deck []Card) *GameState {
	state := &GameState{
		Waste: &WastePile{},
		Stock: &StockPile{},
	}
	for i := 0; i < 4; i++ {
		state.Foundations[i] = &FoundationPile{}
	}

	idx := 0
	for col := 0; col < 7; col++ {
		pile := &TableauPile{}
		for row := 0; row <= col; row++ {
			card := deck[idx]
			idx++
			card.FaceUp = (row == col) // only the top card is face-up
			pile.Cards = append(pile.Cards, card)
		}
		state.Tableau[col] = pile
	}

	// Remaining 24 cards go to stock, face-down
	for ; idx < len(deck); idx++ {
		card := deck[idx]
		card.FaceUp = false
		state.Stock.Cards = append(state.Stock.Cards, card)
	}

	return state
}
