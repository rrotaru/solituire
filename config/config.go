package config

import "fmt"

// Config holds the player-configurable settings for a session.
// Settings are not persisted to disk — they live for the duration of the process.
type Config struct {
	DrawCount       int    // 1 or 3
	ThemeName       string // registered theme name (e.g. "classic")
	AutoMoveEnabled bool   // auto-move safe cards to foundation after each turn
	Seed            int64  // 0 = generate random seed at game start
}

// DefaultConfig returns a Config with sane defaults:
//   - DrawCount: 1
//   - ThemeName: "classic"
//   - AutoMoveEnabled: false
//   - Seed: 0 (random)
func DefaultConfig() *Config {
	return &Config{
		DrawCount:       1,
		ThemeName:       "classic",
		AutoMoveEnabled: false,
		Seed:            0,
	}
}

// Validate checks that the Config fields are within acceptable bounds.
// Returns an error describing the first invalid field, or nil if all are valid.
func (c *Config) Validate() error {
	if c.DrawCount != 1 && c.DrawCount != 3 {
		return fmt.Errorf("config: DrawCount must be 1 or 3, got %d", c.DrawCount)
	}
	if c.ThemeName == "" {
		return fmt.Errorf("config: ThemeName must not be empty")
	}
	return nil
}
