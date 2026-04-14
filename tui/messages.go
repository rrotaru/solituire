package tui

import (
	"time"

	"solituire/config"
	"solituire/theme"
)

// AppScreen identifies which screen the application is currently showing.
// Defined here (not app.go) so that ChangeScreenMsg compiles in T9.
// T13 must NOT redefine this type.
type AppScreen int

const (
	ScreenMenu AppScreen = iota
	ScreenPlaying
	ScreenPaused
	ScreenHelp
	ScreenQuitConfirm
	ScreenWin
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
