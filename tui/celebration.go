package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"solituire/theme"
)

// celebCard holds the animation state of a single falling card symbol.
type celebCard struct {
	symbol string // e.g. "A♠", "K♥", "10♦"
	col    int    // terminal column of the symbol's left edge
	row    int    // current terminal row (may be negative while off-screen above)
	speed  int    // rows advanced per CelebrationTickMsg (1 or 2)
}

// CelebrationModel is the Bubbletea sub-model for ScreenWin.
// It renders a congratulations box with final stats and drives a cascading
// card animation via CelebrationTickMsg ticks.
type CelebrationModel struct {
	score     int
	moves     int
	elapsed   time.Duration
	drawCount int
	th        theme.Theme

	frame   int         // incremented each tick; 0 = static (golden-testable)
	cards   []celebCard // current animation positions — always a fresh slice on mutation
	windowW int
	windowH int
}

// cascadeSymbols is the fixed, deterministic set of card symbols used in the
// cascade animation. The order and contents never change, which keeps the
// animation reproducible for frame-based inspection.
var cascadeSymbols = []string{
	"A♠", "2♥", "3♦", "4♣",
	"5♠", "6♥", "7♦", "8♣",
	"9♠", "10♥", "J♦", "Q♣", "K♠",
}

// NewCelebrationModel constructs a CelebrationModel with stats captured at
// the moment the game was won. drawCount is forwarded into any NewGameMsg
// emitted so the next game respects the same draw mode.
func NewCelebrationModel(score, moves int, elapsed time.Duration, th theme.Theme, drawCount int) CelebrationModel {
	m := CelebrationModel{
		score:     score,
		moves:     moves,
		elapsed:   elapsed,
		drawCount: drawCount,
		th:        th,
		windowW:   78,
		windowH:   24,
	}
	m.cards = buildCascadeCards(m.windowW, m.windowH)
	return m
}

// Init starts the animation tick chain immediately.
func (m CelebrationModel) Init() tea.Cmd {
	return celebTickCmd()
}

// Update handles animation ticks, window resizes, and key input.
func (m CelebrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CelebrationTickMsg:
		m.frame++
		// Copy the card slice so the previous model value is unmodified
		// (Bubbletea treats models as immutable value types).
		next := make([]celebCard, len(m.cards))
		copy(next, m.cards)
		advanceCascadeCards(next, m.windowH)
		m.cards = next
		return m, celebTickCmd()

	case tea.WindowSizeMsg:
		m.windowW = msg.Width
		m.windowH = msg.Height
		m.cards = buildCascadeCards(m.windowW, m.windowH)
		m.frame = 0
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlN:
			dc := m.drawCount
			return m, func() tea.Msg { return NewGameMsg{Seed: 0, DrawCount: dc} }
		case msg.Type == tea.KeyRunes && len(msg.Runes) > 0 &&
			(msg.Runes[0] == 'q' || msg.Runes[0] == 'Q'):
			return m, func() tea.Msg { return ChangeScreenMsg{Screen: ScreenQuitConfirm} }
		case msg.Type == tea.KeyCtrlC:
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the win screen. Frame 0 returns only the congratulation box
// (deterministic — safe for golden tests). Subsequent frames render the full
// terminal with cascading card symbols overlaid around the centered box.
func (m CelebrationModel) View() string {
	box := m.renderBox()
	if m.frame == 0 {
		return box
	}
	return m.renderAnimated(box)
}

// renderBox builds the styled congratulations box (no cascading cards).
func (m CelebrationModel) renderBox() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(m.th.HeaderForeground).
		Background(m.th.HeaderBackground).
		Bold(true).
		Padding(0, 1)

	statsStyle := lipgloss.NewStyle().
		Foreground(m.th.FooterForeground)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.th.FooterForeground).
		Faint(true)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(m.th.CursorBorder).
		Padding(1, 4)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("  You Win!  "))
	sb.WriteString("\n\n")
	sb.WriteString(statsStyle.Render(fmt.Sprintf("Score:  %d", m.score)))
	sb.WriteByte('\n')
	sb.WriteString(statsStyle.Render(fmt.Sprintf("Moves:  %d", m.moves)))
	sb.WriteByte('\n')
	sb.WriteString(statsStyle.Render(fmt.Sprintf("Time:   %s", formatElapsed(m.elapsed))))
	sb.WriteString("\n\n")
	sb.WriteString(hintStyle.Render("[Ctrl+N] New Game    [Q] Quit"))

	return boxStyle.Render(sb.String())
}

// renderAnimated composes the full terminal screen: the congratulation box
// centered on a background field that also contains cascading card symbols.
func (m CelebrationModel) renderAnimated(box string) string {
	boxLines := strings.Split(box, "\n")
	boxH := len(boxLines)
	boxW := 0
	for _, l := range boxLines {
		if w := lipgloss.Width(l); w > boxW {
			boxW = w
		}
	}

	topPad := (m.windowH - boxH) / 2
	if topPad < 0 {
		topPad = 0
	}
	leftPad := (m.windowW - boxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	// Index cards by their current row for O(n) lookup while building lines.
	type cardAtRow struct {
		col    int
		symbol string
	}
	byRow := make(map[int][]cardAtRow, len(m.cards))
	for _, c := range m.cards {
		if c.row >= 0 && c.row < m.windowH {
			byRow[c.row] = append(byRow[c.row], cardAtRow{col: c.col, symbol: c.symbol})
		}
	}

	var sb strings.Builder
	for row := 0; row < m.windowH; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}
		boxRow := row - topPad
		if boxRow >= 0 && boxRow < boxH {
			// Part of the congratulation box.
			sb.WriteString(strings.Repeat(" ", leftPad))
			sb.WriteString(boxLines[boxRow])
		} else {
			// Background line: spaces with any card symbols overlaid.
			line := []rune(strings.Repeat(" ", m.windowW))
			for _, c := range byRow[row] {
				sym := []rune(c.symbol)
				end := c.col + len(sym)
				if c.col >= 0 && end <= m.windowW {
					copy(line[c.col:], sym)
				}
			}
			sb.WriteString(string(line))
		}
	}
	return sb.String()
}

// buildCascadeCards creates the initial card positions spread evenly across
// the terminal width. Cards are staggered vertically so they enter the screen
// at different times, producing a cascade effect rather than all arriving at once.
func buildCascadeCards(windowW, windowH int) []celebCard {
	n := len(cascadeSymbols)
	cards := make([]celebCard, n)

	colStep := windowW / n
	if colStep < 4 {
		colStep = 4
	}

	for i, sym := range cascadeSymbols {
		col := i * colStep
		if col+len([]rune(sym)) > windowW {
			col = windowW - len([]rune(sym)) - 1
		}
		if col < 0 {
			col = 0
		}
		// Stagger start rows: earlier cards start on-screen, later ones above it.
		startRow := (i * windowH / n) - windowH
		if startRow > 0 {
			startRow = -startRow
		}
		// Speed alternates 1/2 for visual variety.
		speed := 1 + (i % 2)

		cards[i] = celebCard{
			symbol: sym,
			col:    col,
			row:    startRow,
			speed:  speed,
		}
	}
	return cards
}

// advanceCascadeCards moves each card down by its speed. Cards that fall below
// the screen wrap back to just above the top.
func advanceCascadeCards(cards []celebCard, windowH int) {
	for i := range cards {
		cards[i].row += cards[i].speed
		if cards[i].row >= windowH {
			cards[i].row = -2
		}
	}
}

// formatElapsed formats a duration as "M:SS" (e.g. "3:07").
func formatElapsed(d time.Duration) string {
	total := int(d.Seconds())
	if total < 0 {
		total = 0
	}
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

// celebTickCmd fires a CelebrationTickMsg after 80 ms.
func celebTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return CelebrationTickMsg{}
	})
}
