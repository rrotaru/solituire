package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"solituire/config"
	"solituire/engine"
	"solituire/renderer"
	"solituire/theme"
	"solituire/tui"
)

func main() {
	cfg := config.DefaultConfig()

	seedFlag := flag.Int64("seed", 0, "RNG seed (0 = random)")
	drawFlag := flag.Int("draw", cfg.DrawCount, "cards to draw per stock flip (1 or 3)")
	flag.Parse()

	// Detect whether --draw was explicitly set on the command line.
	// When it is, we skip the menu and go straight to the game board,
	// because all session parameters are fully specified.
	drawExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "draw" {
			drawExplicit = true
		}
	})

	if *seedFlag != 0 {
		cfg.Seed = *seedFlag
	}
	if *drawFlag != 1 && *drawFlag != 3 {
		fmt.Fprintf(os.Stderr, "klondike: --draw must be 1 or 3 (got %d)\n", *drawFlag)
		os.Exit(1)
	}
	cfg.DrawCount = *drawFlag

	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	reg := theme.NewRegistry()
	th := reg.Get(cfg.ThemeName)
	eng := engine.NewGame(seed, cfg.DrawCount)
	rend := renderer.New(th)
	app := tui.NewAppModel(eng, rend, cfg, reg)

	if drawExplicit {
		app = app.WithScreen(tui.ScreenPlaying)
	}

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "klondike: %v\n", err)
		os.Exit(1)
	}
}
