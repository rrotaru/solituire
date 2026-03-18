package engine

import "testing"

// buildState returns a zeroed GameState with all piles initialized and empty.
func buildState() *GameState {
	s := &GameState{
		Stock:     &StockPile{},
		Waste:     &WastePile{DrawCount: 1},
		DrawCount: 1,
	}
	for i := range s.Tableau {
		s.Tableau[i] = &TableauPile{}
	}
	for i := range s.Foundations {
		s.Foundations[i] = &FoundationPile{}
	}
	return s
}

// faceUpCard constructs a face-up Card.
func faceUpCard(r Rank, s Suit) Card { return Card{Rank: r, Suit: s, FaceUp: true} }

// faceDownCard constructs a face-down Card.
func faceDownCard(r Rank, s Suit) Card { return Card{Rank: r, Suit: s, FaceUp: false} }

// TestValidateMove_TableauToTableau covers Section 13.1 rules.
func TestValidateMove_TableauToTableau(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *GameState
		move    Move
		wantErr bool
	}{
		{
			name: "red 6 onto black 7",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				s.Tableau[1].Cards = []Card{faceUpCard(Six, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "black 5 onto red 6",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Six, Hearts)}
				s.Tableau[1].Cards = []Card{faceUpCard(Five, Spades)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "red 6 onto red 7 rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Hearts)}
				s.Tableau[1].Cards = []Card{faceUpCard(Six, Diamonds)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "black 6 onto black 7 rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				s.Tableau[1].Cards = []Card{faceUpCard(Six, Clubs)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "wrong rank (5 onto 7) rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				s.Tableau[1].Cards = []Card{faceUpCard(Five, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "King to empty column",
			setup: func() *GameState {
				s := buildState()
				// Tableau[0] is empty.
				s.Tableau[1].Cards = []Card{faceUpCard(King, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "non-King to empty column rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[1].Cards = []Card{faceUpCard(Queen, Spades)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "move face-down card rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				s.Tableau[1].Cards = []Card{faceDownCard(Six, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "valid 2-card sequence (6H,5S) onto 7C",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Clubs)}
				s.Tableau[1].Cards = []Card{faceUpCard(Six, Hearts), faceUpCard(Five, Spades)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 2},
			wantErr: false,
		},
		{
			name: "invalid sub-sequence (6H,5H same color) rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Clubs)}
				s.Tableau[1].Cards = []Card{faceUpCard(Six, Hearts), faceUpCard(Five, Diamonds)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 2},
			wantErr: true,
		},
		{
			name: "CardCount exceeds face-up cards rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				s.Tableau[1].Cards = []Card{faceUpCard(Six, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 5},
			wantErr: true,
		},
		{
			name: "CardCount zero rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				s.Tableau[1].Cards = []Card{faceUpCard(Six, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 0},
			wantErr: true,
		},
		{
			name: "source pile empty rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				// Tableau[1] is empty.
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "face-down cards not counted in source",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				// Tableau[1] has one face-down and one face-up card.
				s.Tableau[1].Cards = []Card{faceDownCard(Nine, Clubs), faceUpCard(Six, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "CardCount=2 reaching into face-down cards rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				s.Tableau[1].Cards = []Card{faceDownCard(Nine, Clubs), faceUpCard(Six, Hearts)}
				return s
			},
			move:    Move{From: PileTableau1, To: PileTableau0, CardCount: 2},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.setup()
			err := ValidateMove(state, tt.move)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateMove_WasteToTableau covers Section 13.2 rules.
func TestValidateMove_WasteToTableau(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *GameState
		move    Move
		wantErr bool
	}{
		{
			name: "valid placement (6H onto 7S)",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Six, Hearts)}
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileTableau0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "wrong color (6H onto 7H) rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Six, Hearts)}
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Hearts)}
				return s
			},
			move:    Move{From: PileWaste, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "waste empty rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "King to empty tableau",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(King, Spades)}
				// Tableau[0] is empty.
				return s
			},
			move:    Move{From: PileWaste, To: PileTableau0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "non-King to empty tableau rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Queen, Hearts)}
				return s
			},
			move:    Move{From: PileWaste, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "CardCount != 1 rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Six, Hearts)}
				s.Tableau[0].Cards = []Card{faceUpCard(Seven, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileTableau0, CardCount: 2},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.setup()
			err := ValidateMove(state, tt.move)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateMove_ToFoundation covers Section 13.3 (Waste→Foundation and Tableau→Foundation).
func TestValidateMove_ToFoundation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *GameState
		move    Move
		wantErr bool
	}{
		{
			name: "Ace from waste to empty foundation",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Ace, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "2S from waste onto AS in foundation",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Two, Spades)}
				s.Foundations[0].Cards = []Card{faceUpCard(Ace, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "wrong suit (2H onto AS) rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Two, Hearts)}
				s.Foundations[0].Cards = []Card{faceUpCard(Ace, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "wrong rank (3S onto AS) rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Three, Spades)}
				s.Foundations[0].Cards = []Card{faceUpCard(Ace, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "non-Ace to empty foundation rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Two, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "waste empty rejected",
			setup: func() *GameState {
				s := buildState()
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "waste CardCount zero rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Ace, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 0},
			wantErr: true,
		},
		{
			name: "waste CardCount > 1 rejected",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Ace, Spades)}
				return s
			},
			move:    Move{From: PileWaste, To: PileFoundation0, CardCount: 2},
			wantErr: true,
		},
		{
			name: "Ace from tableau to empty foundation",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Ace, Hearts)}
				return s
			},
			move:    Move{From: PileTableau0, To: PileFoundation1, CardCount: 1},
			wantErr: false,
		},
		{
			name: "tableau face-down top card rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceDownCard(Ace, Hearts)}
				return s
			},
			move:    Move{From: PileTableau0, To: PileFoundation1, CardCount: 1},
			wantErr: true,
		},
		{
			name: "tableau CardCount > 1 rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(Ace, Hearts)}
				return s
			},
			move:    Move{From: PileTableau0, To: PileFoundation1, CardCount: 2},
			wantErr: true,
		},
		{
			name: "tableau empty rejected",
			setup: func() *GameState {
				s := buildState()
				return s
			},
			move:    Move{From: PileTableau0, To: PileFoundation0, CardCount: 1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.setup()
			err := ValidateMove(state, tt.move)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateMove_FoundationToTableau covers Section 13.4 rules.
func TestValidateMove_FoundationToTableau(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *GameState
		move    Move
		wantErr bool
	}{
		{
			name: "King from foundation to empty tableau",
			setup: func() *GameState {
				s := buildState()
				s.Foundations[0].Cards = []Card{
					faceUpCard(Ace, Spades), faceUpCard(Two, Spades),
					faceUpCard(Three, Spades), faceUpCard(Four, Spades),
					faceUpCard(Five, Spades), faceUpCard(Six, Spades),
					faceUpCard(Seven, Spades), faceUpCard(Eight, Spades),
					faceUpCard(Nine, Spades), faceUpCard(Ten, Spades),
					faceUpCard(Jack, Spades), faceUpCard(Queen, Spades),
					faceUpCard(King, Spades),
				}
				return s
			},
			move:    Move{From: PileFoundation0, To: PileTableau3, CardCount: 1},
			wantErr: false,
		},
		{
			name: "QH onto KS from foundation",
			setup: func() *GameState {
				s := buildState()
				s.Foundations[1].Cards = []Card{
					faceUpCard(Ace, Hearts), faceUpCard(Two, Hearts),
					faceUpCard(Three, Hearts), faceUpCard(Four, Hearts),
					faceUpCard(Five, Hearts), faceUpCard(Six, Hearts),
					faceUpCard(Seven, Hearts), faceUpCard(Eight, Hearts),
					faceUpCard(Nine, Hearts), faceUpCard(Ten, Hearts),
					faceUpCard(Jack, Hearts), faceUpCard(Queen, Hearts),
				}
				s.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
				return s
			},
			move:    Move{From: PileFoundation1, To: PileTableau0, CardCount: 1},
			wantErr: false,
		},
		{
			name: "wrong color (QS onto KS) rejected",
			setup: func() *GameState {
				s := buildState()
				s.Foundations[0].Cards = []Card{
					faceUpCard(Ace, Spades), faceUpCard(Two, Spades),
					faceUpCard(Three, Spades), faceUpCard(Four, Spades),
					faceUpCard(Five, Spades), faceUpCard(Six, Spades),
					faceUpCard(Seven, Spades), faceUpCard(Eight, Spades),
					faceUpCard(Nine, Spades), faceUpCard(Ten, Spades),
					faceUpCard(Jack, Spades), faceUpCard(Queen, Spades),
				}
				s.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
				return s
			},
			move:    Move{From: PileFoundation0, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "empty foundation rejected",
			setup: func() *GameState {
				s := buildState()
				s.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
				return s
			},
			move:    Move{From: PileFoundation0, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "non-King to empty tableau from foundation rejected",
			setup: func() *GameState {
				s := buildState()
				s.Foundations[0].Cards = []Card{faceUpCard(Ace, Spades), faceUpCard(Two, Spades)}
				// Tableau[0] is empty.
				return s
			},
			move:    Move{From: PileFoundation0, To: PileTableau0, CardCount: 1},
			wantErr: true,
		},
		{
			name: "CardCount > 1 rejected",
			setup: func() *GameState {
				s := buildState()
				s.Foundations[1].Cards = []Card{
					faceUpCard(Ace, Hearts), faceUpCard(Two, Hearts),
					faceUpCard(Three, Hearts), faceUpCard(Four, Hearts),
					faceUpCard(Five, Hearts), faceUpCard(Six, Hearts),
					faceUpCard(Seven, Hearts), faceUpCard(Eight, Hearts),
					faceUpCard(Nine, Hearts), faceUpCard(Ten, Hearts),
					faceUpCard(Jack, Hearts), faceUpCard(Queen, Hearts),
				}
				s.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
				return s
			},
			move:    Move{From: PileFoundation1, To: PileTableau0, CardCount: 2},
			wantErr: true,
		},
		{
			name: "CardCount zero rejected",
			setup: func() *GameState {
				s := buildState()
				s.Foundations[1].Cards = []Card{
					faceUpCard(Ace, Hearts), faceUpCard(Two, Hearts),
					faceUpCard(Three, Hearts), faceUpCard(Four, Hearts),
					faceUpCard(Five, Hearts), faceUpCard(Six, Hearts),
					faceUpCard(Seven, Hearts), faceUpCard(Eight, Hearts),
					faceUpCard(Nine, Hearts), faceUpCard(Ten, Hearts),
					faceUpCard(Jack, Hearts), faceUpCard(Queen, Hearts),
				}
				s.Tableau[0].Cards = []Card{faceUpCard(King, Spades)}
				return s
			},
			move:    Move{From: PileFoundation1, To: PileTableau0, CardCount: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.setup()
			err := ValidateMove(state, tt.move)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateMove_StockFlip covers Section 13.5 rules.
func TestValidateMove_StockFlip(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *GameState
		move    Move
		wantErr bool
	}{
		{
			name: "stock non-empty flip",
			setup: func() *GameState {
				s := buildState()
				s.Stock.Cards = []Card{faceDownCard(Ace, Spades), faceDownCard(Two, Spades)}
				return s
			},
			move:    Move{From: PileStock, To: PileWaste},
			wantErr: false,
		},
		{
			name: "stock empty waste non-empty recycle",
			setup: func() *GameState {
				s := buildState()
				s.Waste.Cards = []Card{faceUpCard(Five, Hearts)}
				return s
			},
			move:    Move{From: PileStock, To: PileWaste},
			wantErr: false,
		},
		{
			name: "both empty rejected",
			setup: func() *GameState {
				return buildState()
			},
			move:    Move{From: PileStock, To: PileWaste},
			wantErr: true,
		},
		{
			name: "wrong destination rejected",
			setup: func() *GameState {
				s := buildState()
				s.Stock.Cards = []Card{faceDownCard(Ace, Spades)}
				return s
			},
			move:    Move{From: PileStock, To: PileTableau0},
			wantErr: true,
		},
		{
			name: "non-zero CardCount rejected",
			setup: func() *GameState {
				s := buildState()
				s.Stock.Cards = []Card{faceDownCard(Ace, Spades), faceDownCard(Two, Hearts)}
				return s
			},
			move:    Move{From: PileStock, To: PileWaste, CardCount: 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.setup()
			err := ValidateMove(state, tt.move)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateMove_UnsupportedPileCombination verifies unknown combinations return an error.
func TestValidateMove_UnsupportedPileCombination(t *testing.T) {
	s := buildState()
	// Foundation → Foundation is not a legal Klondike move.
	err := ValidateMove(s, Move{From: PileFoundation0, To: PileFoundation1, CardCount: 1})
	if err == nil {
		t.Error("expected error for Foundation→Foundation, got nil")
	}
}

// TestValidMoves_NoDuplicates verifies ValidMoves never returns duplicate moves.
func TestValidMoves_NoDuplicates(t *testing.T) {
	deck := Shuffle(NewDeck(), 42)
	state := Deal(deck, 1)

	moves := ValidMoves(state)
	seen := make(map[Move]bool)
	for _, m := range moves {
		if seen[m] {
			t.Errorf("duplicate move returned: %+v", m)
		}
		seen[m] = true
	}
}

// TestValidMoves_EmptyBoard verifies that a fully empty board has no valid moves.
func TestValidMoves_EmptyBoard(t *testing.T) {
	s := buildState()
	moves := ValidMoves(s)
	if len(moves) != 0 {
		t.Errorf("expected 0 moves on empty board, got %d: %+v", len(moves), moves)
	}
}

// TestValidMoves_StockOnlyBoard verifies exactly one move (stock flip) when only stock has cards.
func TestValidMoves_StockOnlyBoard(t *testing.T) {
	s := buildState()
	s.Stock.Cards = []Card{faceDownCard(Ace, Spades), faceDownCard(Two, Hearts)}

	moves := ValidMoves(s)
	if len(moves) != 1 {
		t.Errorf("expected exactly 1 move (stock flip), got %d: %+v", len(moves), moves)
		return
	}
	if moves[0].From != PileStock || moves[0].To != PileWaste {
		t.Errorf("expected stock flip move, got %+v", moves[0])
	}
}

// TestValidMoves_AllMovesAreValid verifies every move returned by ValidMoves passes ValidateMove.
func TestValidMoves_AllMovesAreValid(t *testing.T) {
	deck := Shuffle(NewDeck(), 99)
	state := Deal(deck, 1)

	moves := ValidMoves(state)
	for _, m := range moves {
		if err := ValidateMove(state, m); err != nil {
			t.Errorf("ValidMoves returned invalid move %+v: %v", m, err)
		}
	}
}

// TestIsValidFaceUpSequence tests the internal sequence validator.
func TestIsValidFaceUpSequence(t *testing.T) {
	tests := []struct {
		name  string
		cards []Card
		want  bool
	}{
		{"empty slice", []Card{}, true},
		{"single card", []Card{faceUpCard(Seven, Spades)}, true},
		{"valid two-card (7S,6H)", []Card{faceUpCard(Seven, Spades), faceUpCard(Six, Hearts)}, true},
		{"valid three-card (8H,7S,6H)", []Card{faceUpCard(Eight, Hearts), faceUpCard(Seven, Spades), faceUpCard(Six, Hearts)}, true},
		{"same color rejected", []Card{faceUpCard(Seven, Spades), faceUpCard(Six, Clubs)}, false},
		{"non-sequential rank rejected", []Card{faceUpCard(Seven, Spades), faceUpCard(Five, Hearts)}, false},
		{"ascending rank rejected", []Card{faceUpCard(Six, Hearts), faceUpCard(Seven, Spades)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidFaceUpSequence(tt.cards)
			if got != tt.want {
				t.Errorf("isValidFaceUpSequence() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsValidTableauPlacement tests the internal placement validator.
func TestIsValidTableauPlacement(t *testing.T) {
	tests := []struct {
		name  string
		card  Card
		dest  *TableauPile
		want  bool
	}{
		{"King to empty", faceUpCard(King, Hearts), &TableauPile{}, true},
		{"non-King to empty rejected", faceUpCard(Queen, Spades), &TableauPile{}, false},
		{"red 6 onto black 7", faceUpCard(Six, Hearts), &TableauPile{Cards: []Card{faceUpCard(Seven, Spades)}}, true},
		{"black 5 onto red 6", faceUpCard(Five, Clubs), &TableauPile{Cards: []Card{faceUpCard(Six, Diamonds)}}, true},
		{"same color rejected", faceUpCard(Six, Hearts), &TableauPile{Cards: []Card{faceUpCard(Seven, Hearts)}}, false},
		{"wrong rank rejected", faceUpCard(Five, Spades), &TableauPile{Cards: []Card{faceUpCard(Seven, Hearts)}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidTableauPlacement(tt.card, tt.dest)
			if got != tt.want {
				t.Errorf("isValidTableauPlacement() = %v, want %v", got, tt.want)
			}
		})
	}
}
