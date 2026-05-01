package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"var-cli/tui"
)

func main() {
	m := tui.NewModel()

	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Hubo un error fatal al iniciar la app: %v", err)
		os.Exit(1)
	}
}
