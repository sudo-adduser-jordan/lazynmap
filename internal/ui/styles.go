package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	label         lipgloss.Style
	value         lipgloss.Style
	muted         lipgloss.Style
	primary       lipgloss.Style
	accent        lipgloss.Style
	success       lipgloss.Style
	warning       lipgloss.Style
	error         lipgloss.Style
	border        lipgloss.Style
	focusedBorder lipgloss.Style
	panelTitle    lipgloss.Style
	focusedTitle  lipgloss.Style
	panelHint     lipgloss.Style
	footer        lipgloss.Style
	helpKey       lipgloss.Style
	detailKey     lipgloss.Style
	selected      lipgloss.Style
	selectedDim   lipgloss.Style
}

func newStyles() styles {
	primaryColor := lipgloss.AdaptiveColor{Light: "#006b8f", Dark: "#61dafb"}
	accentColor := lipgloss.AdaptiveColor{Light: "#237a45", Dark: "#87d787"}
	mutedColor := lipgloss.AdaptiveColor{Light: "#6c6c6c", Dark: "#9a9a9a"}
	borderColor := lipgloss.AdaptiveColor{Light: "#b8b8b8", Dark: "#555555"}
	selectedBackground := lipgloss.AdaptiveColor{Light: "#d7e8f2", Dark: "#303030"}
	selectedForeground := lipgloss.AdaptiveColor{Light: "#111111", Dark: "#eeeeee"}
	valueColor := lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}

	return styles{
		label: lipgloss.NewStyle().
			Bold(true).
			Foreground(mutedColor),
		value: lipgloss.NewStyle().
			Foreground(valueColor),
		muted: lipgloss.NewStyle().
			Foreground(mutedColor),
		primary: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor),
		accent: lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor),
		success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#237a45", Dark: "#87d787"}),
		warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#8a5d00", Dark: "#ffd866"}),
		error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#a30000", Dark: "#ff6b6b"}),
		border: lipgloss.NewStyle().
			Foreground(borderColor),
		focusedBorder: lipgloss.NewStyle().
			Foreground(accentColor),
		panelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor),
		focusedTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor),
		panelHint: lipgloss.NewStyle().
			Foreground(mutedColor),
		footer: lipgloss.NewStyle().
			Foreground(mutedColor),
		helpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor),
		detailKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(mutedColor),
		selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(selectedForeground).
			Background(selectedBackground),
		selectedDim: lipgloss.NewStyle().
			Foreground(mutedColor).
			Background(selectedBackground),
	}
}
