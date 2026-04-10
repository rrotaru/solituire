package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/golden"
	"solituire/theme"
)

// board_test.go's init() sets lipgloss.SetColorProfile(termenv.Ascii) for the
// whole tui test package — no second init() is needed here.

// newTestCelebration returns a CelebrationModel with fixed inputs so tests
// produce deterministic output regardless of when they run.
func newTestCelebration() CelebrationModel {
	reg := theme.NewRegistry()
	th := reg.Get("classic")
	return NewCelebrationModel(1250, 87, 3*time.Minute+42*time.Second, th, 1)
}

// updateCeleb calls Update and type-asserts the result to CelebrationModel.
func updateCeleb(m CelebrationModel, msg tea.Msg) (CelebrationModel, tea.Cmd) {
	result, cmd := m.Update(msg)
	return result.(CelebrationModel), cmd
}

// ---------------------------------------------------------------------------
// Golden test — static frame 0 (no animation cards rendered)
// ---------------------------------------------------------------------------

func TestCelebrationView(t *testing.T) {
	m := newTestCelebration()
	// frame == 0: deterministic, no cascading cards
	got := m.View()
	golden.RequireEqual(t, []byte(got))
}

// ---------------------------------------------------------------------------
// Animation tick tests
// ---------------------------------------------------------------------------

func TestCelebrationModel_TickAdvancesFrame(t *testing.T) {
	m := newTestCelebration()
	if m.frame != 0 {
		t.Fatalf("initial frame = %d, want 0", m.frame)
	}
	updated, cmd := updateCeleb(m, CelebrationTickMsg{})
	if updated.frame != 1 {
		t.Errorf("frame after one tick = %d, want 1", updated.frame)
	}
	if cmd == nil {
		t.Error("CelebrationTickMsg: returned nil Cmd — animation tick chain would break")
	}
}

func TestCelebrationModel_TickChainContinues(t *testing.T) {
	m := newTestCelebration()
	// Fire several ticks and confirm the chain stays alive.
	for i := 0; i < 5; i++ {
		var cmd tea.Cmd
		m, cmd = updateCeleb(m, CelebrationTickMsg{})
		if cmd == nil {
			t.Fatalf("tick %d: returned nil Cmd — animation tick chain broken", i+1)
		}
	}
	if m.frame != 5 {
		t.Errorf("frame after 5 ticks = %d, want 5", m.frame)
	}
}

func TestCelebrationModel_TickDoesNotMutateOriginal(t *testing.T) {
	m := newTestCelebration()
	// Capture first card position before any tick.
	origRow := m.cards[0].row
	// Tick advances the *updated* model's cards but must not mutate m.cards.
	_, _ = updateCeleb(m, CelebrationTickMsg{})
	if m.cards[0].row != origRow {
		t.Error("tick mutated original model's card slice — Bubbletea immutability violated")
	}
}

func TestCelebrationModel_TickAdvancesCardRows(t *testing.T) {
	m := newTestCelebration()
	origRow := m.cards[0].row
	updated, _ := updateCeleb(m, CelebrationTickMsg{})
	// Each card advances by at least 1 row per tick.
	if updated.cards[0].row <= origRow {
		t.Errorf("card[0].row after tick = %d, want > %d", updated.cards[0].row, origRow)
	}
}

func TestCelebrationModel_AnimatedViewNonEmpty(t *testing.T) {
	m := newTestCelebration()
	// Advance to frame 1 so renderAnimated is called.
	m, _ = updateCeleb(m, CelebrationTickMsg{})
	if m.frame != 1 {
		t.Fatalf("expected frame 1, got %d", m.frame)
	}
	view := m.View()
	if view == "" {
		t.Error("View() returned empty string on animated frame")
	}
}

func TestCelebrationModel_AnimatedViewContainsStats(t *testing.T) {
	m := newTestCelebration()
	m, _ = updateCeleb(m, CelebrationTickMsg{}) // frame 1 → animated
	view := m.View()
	for _, want := range []string{"1250", "87", "3:42"} {
		if !strings.Contains(view, want) {
			t.Errorf("animated View() missing %q", want)
		}
	}
}

// ---------------------------------------------------------------------------
// Key input tests
// ---------------------------------------------------------------------------

func TestCelebrationModel_CtrlN_EmitsNewGameMsg(t *testing.T) {
	m := newTestCelebration()
	_, cmd := updateCeleb(m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if cmd == nil {
		t.Fatal("Ctrl+N: returned nil Cmd, expected NewGameMsg")
	}
	msg := cmd()
	if _, ok := msg.(NewGameMsg); !ok {
		t.Errorf("Ctrl+N: got %T, want NewGameMsg", msg)
	}
}

func TestCelebrationModel_CtrlN_DrawCountPreserved(t *testing.T) {
	reg := theme.NewRegistry()
	th := reg.Get("classic")
	// Construct with drawCount=3.
	m := NewCelebrationModel(100, 10, time.Minute, th, 3)
	_, cmd := updateCeleb(m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if cmd == nil {
		t.Fatal("Ctrl+N: returned nil Cmd")
	}
	msg := cmd()
	ngm, ok := msg.(NewGameMsg)
	if !ok {
		t.Fatalf("Ctrl+N: got %T, want NewGameMsg", msg)
	}
	if ngm.DrawCount != 3 {
		t.Errorf("NewGameMsg.DrawCount = %d, want 3", ngm.DrawCount)
	}
}

func TestCelebrationModel_Q_EmitsQuitConfirm(t *testing.T) {
	m := newTestCelebration()
	_, cmd := updateCeleb(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("'q': returned nil Cmd, expected ChangeScreenMsg")
	}
	msg := cmd()
	csm, ok := msg.(ChangeScreenMsg)
	if !ok || csm.Screen != ScreenQuitConfirm {
		t.Errorf("'q': got %v, want ChangeScreenMsg{ScreenQuitConfirm}", msg)
	}
}

func TestCelebrationModel_UpperQ_EmitsQuitConfirm(t *testing.T) {
	m := newTestCelebration()
	_, cmd := updateCeleb(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Q")})
	if cmd == nil {
		t.Fatal("'Q': returned nil Cmd, expected ChangeScreenMsg")
	}
	msg := cmd()
	csm, ok := msg.(ChangeScreenMsg)
	if !ok || csm.Screen != ScreenQuitConfirm {
		t.Errorf("'Q': got %v, want ChangeScreenMsg{ScreenQuitConfirm}", msg)
	}
}

func TestCelebrationModel_UnknownKey_NoOp(t *testing.T) {
	m := newTestCelebration()
	updated, cmd := updateCeleb(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil {
		t.Errorf("unknown key 'x': expected nil Cmd, got non-nil")
	}
	if updated.frame != m.frame {
		t.Error("unknown key changed frame counter")
	}
}

// ---------------------------------------------------------------------------
// Window resize tests
// ---------------------------------------------------------------------------

func TestCelebrationModel_WindowSizeMsg_UpdatesDimensions(t *testing.T) {
	m := newTestCelebration()
	updated, _ := updateCeleb(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if updated.windowW != 120 {
		t.Errorf("windowW = %d, want 120", updated.windowW)
	}
	if updated.windowH != 40 {
		t.Errorf("windowH = %d, want 40", updated.windowH)
	}
}

func TestCelebrationModel_WindowSizeMsg_ResetsAnimationToFrame0(t *testing.T) {
	m := newTestCelebration()
	for i := 0; i < 5; i++ {
		m, _ = updateCeleb(m, CelebrationTickMsg{})
	}
	if m.frame == 0 {
		t.Skip("frames did not advance — skipping reset assertion")
	}
	updated, _ := updateCeleb(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if updated.frame != 0 {
		t.Errorf("frame after WindowSizeMsg = %d, want 0", updated.frame)
	}
}

func TestCelebrationModel_WindowSizeMsg_RebuildsCascadeCards(t *testing.T) {
	m := newTestCelebration() // windowW=78
	updated, _ := updateCeleb(m, tea.WindowSizeMsg{Width: 160, Height: 48})
	// Cards must be rebuilt for the new width — check no card falls outside bounds.
	for i, c := range updated.cards {
		sym := []rune(c.symbol)
		end := c.col + len(sym)
		if end > updated.windowW {
			t.Errorf("card[%d] col %d + symLen %d = %d exceeds windowW %d",
				i, c.col, len(sym), end, updated.windowW)
		}
	}
}

// ---------------------------------------------------------------------------
// AppModel integration tests
// ---------------------------------------------------------------------------

// TestAppModel_GameWonMsg_InitializesCelebration verifies that receiving
// GameWonMsg transitions to ScreenWin and starts the celebration animation.
func TestAppModel_GameWonMsg_InitializesCelebration(t *testing.T) {
	app := newTestApp()
	result, cmd := app.Update(GameWonMsg{})
	updated := result.(AppModel)

	if updated.screen != ScreenWin {
		t.Errorf("screen = %v, want ScreenWin", updated.screen)
	}
	if cmd == nil {
		t.Error("GameWonMsg: returned nil Cmd — celebration.Init() not called")
	}
}

// TestAppModel_WinScreen_ViewUsesCelebrationModel verifies that the ScreenWin
// view is no longer the old stub string but comes from CelebrationModel.
func TestAppModel_WinScreen_ViewUsesCelebrationModel(t *testing.T) {
	app := newTestApp()
	result, _ := app.Update(GameWonMsg{})
	app = result.(AppModel)

	view := app.View()
	if view == "You won! Press Ctrl+N for a new game." {
		t.Error("ScreenWin View() still returns stub string — CelebrationModel not wired")
	}
	if view == "" {
		t.Error("ScreenWin View() returned empty string")
	}
	// The celebration box must contain the win message.
	if !strings.Contains(view, "You Win!") {
		t.Errorf("ScreenWin View() missing 'You Win!': got %q", view)
	}
}

// TestAppModel_WinScreen_CelebTickDelegated confirms that CelebrationTickMsg
// is forwarded to CelebrationModel when on ScreenWin.
func TestAppModel_WinScreen_CelebTickDelegated(t *testing.T) {
	result, _ := newTestApp().Update(GameWonMsg{})
	app := result.(AppModel)
	// frame must be 0 right after GameWonMsg.
	if app.celebration.frame != 0 {
		t.Fatalf("celebration.frame = %d after GameWonMsg, want 0", app.celebration.frame)
	}
	result2, _ := app.Update(CelebrationTickMsg{})
	app2 := result2.(AppModel)
	if app2.celebration.frame != 1 {
		t.Errorf("celebration.frame = %d after CelebrationTickMsg, want 1", app2.celebration.frame)
	}
}

// TestAppModel_WinScreen_CtrlNStartsNewGame confirms that key handling is
// now delegated through CelebrationModel (which emits NewGameMsg).
func TestAppModel_WinScreen_CtrlNStartsNewGame(t *testing.T) {
	result, _ := newTestApp().Update(GameWonMsg{})
	app := result.(AppModel)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if cmd == nil {
		t.Fatal("ScreenWin Ctrl+N: returned nil Cmd, expected NewGameMsg")
	}
	msg := cmd()
	if _, ok := msg.(NewGameMsg); !ok {
		t.Errorf("ScreenWin Ctrl+N: got %T, want NewGameMsg", msg)
	}
}

// TestAppModel_WinScreen_WindowSizePropagated confirms that tea.WindowSizeMsg
// is forwarded to CelebrationModel when on ScreenWin.
func TestAppModel_WinScreen_WindowSizePropagated(t *testing.T) {
	result, _ := newTestApp().Update(GameWonMsg{})
	app := result.(AppModel)
	result2, _ := app.Update(tea.WindowSizeMsg{Width: 130, Height: 45})
	app2 := result2.(AppModel)
	if app2.celebration.windowW != 130 {
		t.Errorf("celebration.windowW = %d, want 130", app2.celebration.windowW)
	}
	if app2.celebration.windowH != 45 {
		t.Errorf("celebration.windowH = %d, want 45", app2.celebration.windowH)
	}
}

// ---------------------------------------------------------------------------
// formatElapsed helper
// ---------------------------------------------------------------------------

func TestFormatElapsed(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00"},
		{30 * time.Second, "0:30"},
		{time.Minute, "1:00"},
		{3*time.Minute + 7*time.Second, "3:07"},
		{3*time.Minute + 42*time.Second, "3:42"},
		{59*time.Minute + 59*time.Second, "59:59"},
		{-time.Second, "0:00"}, // negative clamped to zero
	}
	for _, tc := range cases {
		got := formatElapsed(tc.d)
		if got != tc.want {
			t.Errorf("formatElapsed(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
