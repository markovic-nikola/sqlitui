package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/markovic-nikola/sqlitui/ui"
	"github.com/markovic-nikola/sqlitui/update"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("sqlitui %s (%s, %s)\n", version, commit, date)
			return
		case "--update":
			update.Run(version)
			return
		}
	}

	var path string
	if len(os.Args) >= 2 {
		path = os.Args[1]
	}

	showUpdateNotice := update.CheckInBackground(version)

	p := tea.NewProgram(ui.NewModel(path), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	showUpdateNotice()
}
