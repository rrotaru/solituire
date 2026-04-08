package main

import (
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

	reg := theme.NewRegistry()
	th := reg.Get(cfg.ThemeName)

	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	eng := engine.NewGame(seed, cfg.DrawCount)
	rend := renderer.New(th)
	app := tui.NewAppModel(eng, rend, cfg, reg)

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "solituire: %v\n", err)
		os.Exit(1)
	}
}
