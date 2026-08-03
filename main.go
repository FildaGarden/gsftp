package main

import (
	"fmt"
	"os"

	"gsftp/pkg/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	model := ui.NewModel()

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	model.SetProgramRef(p)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running SFTP TUI application: %v\n", err)
		os.Exit(1)
	}
}
