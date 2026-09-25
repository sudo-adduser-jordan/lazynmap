package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Run      key.Binding
	Focus    key.Binding
	History  key.Binding
	Cancel   key.Binding
	Next     key.Binding
	Previous key.Binding
	Filter   key.Binding
	Profile  key.Binding
	Export   key.Binding
	Help     key.Binding
	Quit     key.Binding
	Enter    key.Binding
	Escape   key.Binding
}

func (k keyMap) footerHelp(focus focusPane, state scanState, overlay string) string {
	switch overlay {
	case "help":
		return "Close help: esc / ?"
	case "history":
		return "Open: enter · Navigate: j/k · Close: esc"
	}

	if state == stateScanning || state == stateCancelling {
		return "Cancel scan: x / esc · Quit: q"
	}

	switch focus {
	case focusTarget, focusPorts:
		return "Run scan: enter · Next field: tab · Results: esc · Help: ?"
	case focusFilter:
		return "Apply filter: enter · Clear and close: esc · Help: ?"
	case focusHistory:
		return "Open: enter · Navigate: j/k · Close: esc"
	case focusPortsTable:
		return "Navigate: j/k · Details: tab · Filter: / · Export: e · Help: ?"
	case focusDetail:
		return "Scroll: j/k / arrows · Hosts: esc · Help: ?"
	default:
		return "Run: enter · History: h · Filter: / · Profile: p · Export: e · Help: ?"
	}
}

func defaultKeyMap() keyMap {
	return keyMap{
		Run:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run scan")),
		Focus:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "edit target")),
		History:  key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "scan history")),
		Cancel:   key.NewBinding(key.WithKeys("x", "esc"), key.WithHelp("x", "cancel scan")),
		Next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
		Previous: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous pane")),
		Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Profile:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "cycle profile")),
		Export:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export JSON")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Enter:    key.NewBinding(key.WithKeys("enter")),
		Escape:   key.NewBinding(key.WithKeys("esc")),
	}
}
