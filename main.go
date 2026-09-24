package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sudo-adduser-jordan/lazynmap/internal/ui"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printHelp()
		return
	}

	model := ui.NewModel()
	program := tea.NewProgram(&model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazynmap: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`lazynmap - a small terminal UI for nmap

Usage:
  lazynmap

The interactive UI is launched without a target so that the target can be
entered safely in the terminal. Press ? inside the UI for keybindings.

Nmap must be installed and available on PATH.`)
}
