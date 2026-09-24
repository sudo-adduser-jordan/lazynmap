package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Run      key.Binding
	Focus    key.Binding
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

func (k keyMap) footerHelp() string {
	bindings := []key.Binding{k.Run, k.Focus, k.Profile, k.Next, k.Filter, k.Export, k.Help, k.Quit}
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		parts = append(parts, binding.Help().Key+" "+binding.Help().Desc)
	}
	return joinHelp(parts)
}

func joinHelp(parts []string) string {
	result := ""
	for index, part := range parts {
		if index > 0 {
			result += "   "
		}
		result += part
	}
	return result
}

func defaultKeyMap() keyMap {
	return keyMap{
		Run:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run scan")),
		Focus:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "edit target")),
		Cancel:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "cancel scan")),
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
