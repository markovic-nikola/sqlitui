package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/markovic-nikola/sqlitui/ui"
)

func main() {
	var path string
	if len(os.Args) >= 2 {
		path = os.Args[1]
	}

	p := tea.NewProgram(ui.NewModel(path), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
