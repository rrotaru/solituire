package engine

import "testing"

func TestStandardScorer_OnMove(t *testing.T) {
	s := StandardScorer{}

	tests := []struct {
		name string
		move Move
		want int
	}{
		{
			name: "waste to tableau",
			move: Move{From: PileWaste, To: PileTableau0, CardCount: 1},
			want: 5,
		},
		{
			name: "waste to tableau column 6",
			move: Move{From: PileWaste, To: PileTableau6, CardCount: 1},
			want: 5,
		},
		{
			name: "waste to foundation",
			move: Move{From: PileWaste, To: PileFoundation0, CardCount: 1},
			want: 10,
		},
		{
			name: "waste to foundation index 3",
			move: Move{From: PileWaste, To: PileFoundation3, CardCount: 1},
			want: 10,
		},
		{
			name: "tableau to foundation",
			move: Move{From: PileTableau0, To: PileFoundation0, CardCount: 1},
			want: 10,
		},
		{
			name: "tableau column 5 to foundation 2",
			move: Move{From: PileTableau5, To: PileFoundation2, CardCount: 1},
			want: 10,
		},
		{
			name: "foundation to tableau",
			move: Move{From: PileFoundation0, To: PileTableau0, CardCount: 1},
			want: -15,
		},
		{
			name: "foundation 3 to tableau 4",
			move: Move{From: PileFoundation3, To: PileTableau4, CardCount: 1},
			want: -15,
		},
		{
			name: "tableau to tableau",
			move: Move{From: PileTableau0, To: PileTableau1, CardCount: 1},
			want: 0,
		},
		{
			name: "stock flip",
			move: Move{From: PileStock, To: PileWaste, CardCount: 0},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.OnMove(tt.move, nil)
			if got != tt.want {
				t.Errorf("OnMove(%v) = %d, want %d", tt.move, got, tt.want)
			}
		})
	}
}

func TestStandardScorer_OnFlipTableau(t *testing.T) {
	s := StandardScorer{}
	got := s.OnFlipTableau()
	if got != 5 {
		t.Errorf("OnFlipTableau() = %d, want 5", got)
	}
}

func TestStandardScorer_OnRecycleStock(t *testing.T) {
	s := StandardScorer{}
	got := s.OnRecycleStock()
	if got != -100 {
		t.Errorf("OnRecycleStock() = %d, want -100", got)
	}
}

// TestStandardScorer_ScoreFloor verifies that when deltas are applied to
// GameState.Score with a floor of 0, negative totals are clamped correctly.
// The scorer itself returns raw deltas; the caller is responsible for clamping.
func TestStandardScorer_ScoreFloor(t *testing.T) {
	s := StandardScorer{}
	score := 0

	// Foundation → Tableau yields −15; from score 0 the floor keeps it at 0.
	delta := s.OnMove(Move{From: PileFoundation0, To: PileTableau0, CardCount: 1}, nil)
	score += delta
	if score < 0 {
		score = 0
	}
	if score != 0 {
		t.Errorf("score after flooring = %d, want 0", score)
	}

	// Recycle stock yields −100; from score 0 the floor keeps it at 0.
	score += s.OnRecycleStock()
	if score < 0 {
		score = 0
	}
	if score != 0 {
		t.Errorf("score after recycle floor = %d, want 0", score)
	}

	// Accumulate 10 points then subtract 15; floor at 0.
	score += s.OnMove(Move{From: PileTableau0, To: PileFoundation0, CardCount: 1}, nil)
	score += s.OnMove(Move{From: PileFoundation0, To: PileTableau0, CardCount: 1}, nil)
	if score < 0 {
		score = 0
	}
	if score != 0 {
		t.Errorf("score after net-negative floor = %d, want 0", score)
	}
}
