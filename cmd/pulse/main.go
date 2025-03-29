package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"pulse/internal/config"
	"pulse/internal/docker"
	"pulse/internal/ui"
)

func main() {
	cfg := config.ParseFlags()

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Unable to create Docker client: %v", err)
	}
	defer func() {
		_ = cli.Close()
	}()

	// Use WithAltScreen to enable full-screen mode with proper window size events
	p := tea.NewProgram(
		ui.NewModel(cli, cfg.Debug),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Optional: add mouse support for future enhancements
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting the program: %v\n", err)
		os.Exit(1)
	}
}
