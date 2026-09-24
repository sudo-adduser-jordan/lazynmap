package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	title        lipgloss.Style
	label        lipgloss.Style
	value        lipgloss.Style
	muted        lipgloss.Style
	accent       lipgloss.Style
	success      lipgloss.Style
	warning      lipgloss.Style
	error        lipgloss.Style
	panel        lipgloss.Style
	focusedPanel lipgloss.Style
	panelTitle   lipgloss.Style
	footer       lipgloss.Style
	helpKey      lipgloss.Style
}

func newStyles() styles {
	return styles{
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f5f7fa")).
			Background(lipgloss.Color("#5b4bdb")).
			Padding(0, 1),
		label: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#a7a9b7")),
		value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5f7fa")),
		muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777b8c")),
		accent: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#8b7cff")),
		success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#55d187")),
		warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0c674")),
		error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ff6b7a")),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475a")).
			Padding(0, 1),
		focusedPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8b7cff")).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#c9c5ff")),
		footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9295a5")),
		helpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0c674")),
	}
}
