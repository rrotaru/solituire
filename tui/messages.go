package tui

import (
	"time"

	"solituire/config"
	"solituire/theme"
)

// Game lifecycle

type NewGameMsg struct {
	Seed      int64
	DrawCount int
}

type RestartDealMsg struct{}

type GameWonMsg struct{}

// Navigation

type ChangeScreenMsg struct{ Screen AppScreen }

// Ticks

type TickMsg time.Time           // elapsed timer updates
type CelebrationTickMsg struct{} // win animation frames

// Config

type ConfigChangedMsg struct{ Config *config.Config }
type ThemeChangedMsg struct{ Theme *theme.Theme }

// Auto-complete

type AutoCompleteStepMsg struct{} // triggers one foundation move per tick
