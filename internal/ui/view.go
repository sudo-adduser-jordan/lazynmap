package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/sudo-adduser-jordan/lazynmap/internal/nmap"
)

func (m Model) View() string {
	width := m.width
	if width <= 0 {
		width = 100
	}
	height := m.height
	if height <= 0 {
		height = 30
	}

	header := m.headerView(width)
	footer := m.footerView(width)
	bodyHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyHeight < 14 {
		bodyHeight = 14
	}

	var body string
	if m.showHelp {
		body = m.helpView(width, bodyHeight)
	} else {
		body = m.bodyView(width, bodyHeight)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) headerView(width int) string {
	profile := m.currentProfile()
	state := m.stateLabel()
	elapsed := m.elapsed().Round(time.Second).String()
	title := m.styles.title.Render(" lazynmap ")
	meta := m.styles.muted.Render(fmt.Sprintf("%s  ·  %s  ·  %s", profile.Name, state, elapsed))
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, " ", meta)

	targetWidth := max(18, min(42, width/3-4))
	portsWidth := max(14, min(28, width/5-2))
	m.target.Width = targetWidth
	m.ports.Width = portsWidth
	config := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.target.View(),
		"  ",
		m.ports.View(),
		"  ",
		m.styles.label.Render("Profile "),
		m.styles.accent.Render(profile.Name),
	)
	config = truncate(config, max(20, width-2))
	command := m.styles.muted.Render(truncate(m.commandPreview(), max(20, width-2)))
	return lipgloss.JoinVertical(lipgloss.Left, header, config, command)
}

func (m Model) bodyView(width, height int) string {
	leftWidth := clamp(width*28/100, 24, 38)
	rightWidth := max(1, width-leftWidth)
	left := m.historyPanel(leftWidth, height)
	right := m.resultPanels(rightWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) historyPanel(width, height int) string {
	var content string
	if len(m.entries) == 0 {
		content = m.styles.muted.Render("No saved scans yet.\n\nPress r, enter a target, then enter.")
	} else {
		content = m.history.View()
	}
	title := "History"
	if m.focus == focusHistory {
		title += "  • focused"
	}
	return panel(m.styles, title, content, width, height, m.focus == focusHistory)
}

func (m Model) resultPanels(width, height int) string {
	hostContent := "No hosts found."
	if len(m.visible) > 0 {
		hostContent = m.hosts.View()
	}
	portsContent := "No ports found."
	if len(m.visiblePorts) > 0 {
		portsContent = m.portsTable.View()
	}
	hostTitle := "Hosts"
	if m.focus == focusHosts {
		hostTitle += "  • focused"
	}
	portsTitle := "Ports"
	if m.focus == focusPortsTable {
		portsTitle += "  • focused"
	}

	// On very short terminals, keep the two result tables visible and hide
	// the detail pane rather than allowing the bordered panels to overflow.
	if height < 17 {
		hostHeight := max(5, height/2)
		portsHeight := max(5, height-hostHeight)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			panel(m.styles, hostTitle, hostContent, width, hostHeight, m.focus == focusHosts),
			panel(m.styles, portsTitle, portsContent, width, portsHeight, m.focus == focusPortsTable),
		)
	}

	hostHeight := max(5, height*45/100)
	portsHeight := max(5, height*30/100)
	detailHeight := max(4, height-hostHeight-portsHeight)
	detailTitle := "Details"
	if m.focus == focusDetail {
		detailTitle += "  • focused"
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		panel(m.styles, hostTitle, hostContent, width, hostHeight, m.focus == focusHosts),
		panel(m.styles, portsTitle, portsContent, width, portsHeight, m.focus == focusPortsTable),
		panel(m.styles, detailTitle, m.detail.View(), width, detailHeight, m.focus == focusDetail),
	)
}

func (m Model) helpView(width, height int) string {
	lines := []string{
		m.styles.panelTitle.Render("Keyboard shortcuts"),
		"",
		"  enter       run the configured scan",
		"  r           edit the target",
		"  p           cycle scan profile",
		"  tab         move between panes",
		"  shift+tab   move to the previous pane",
		"  /           filter hosts and ports",
		"  x / esc     cancel an active scan",
		"  e           export the selected result as JSON",
		"  ?           close this help",
		"  q           quit",
		"",
		m.styles.muted.Render("Profiles use nmap arguments directly and never invoke a shell."),
		m.styles.muted.Render("Only scan hosts and networks you are authorized to test."),
	}
	content := strings.Join(lines, "\n")
	return panel(m.styles, "Help", content, width, height, true)
}

func (m Model) footerView(width int) string {
	status := m.statusView()
	keys := m.styles.footer.Render(m.keys.footerHelp())
	return lipgloss.JoinVertical(
		lipgloss.Left,
		truncate(status, max(20, width-2)),
		truncate(keys, max(20, width-2)),
	)
}

func (m Model) statusView() string {
	if m.statusMessage == "" {
		return m.styles.footer.Render("Ready")
	}
	switch m.statusKind {
	case stateSuccess:
		return m.styles.success.Render("✓ " + m.statusMessage)
	case stateScanning:
		return m.styles.accent.Render("⟳ " + m.statusMessage)
	case stateCancelling:
		return m.styles.warning.Render("… " + m.statusMessage)
	case stateWarning:
		return m.styles.warning.Render("! " + m.statusMessage)
	case stateError:
		return m.styles.error.Render("✗ " + m.statusMessage)
	default:
		return m.styles.footer.Render(m.statusMessage)
	}
}

func (m Model) stateLabel() string {
	switch m.state {
	case stateScanning:
		return "scanning"
	case stateCancelling:
		return "cancelling"
	case stateSuccess:
		return "complete"
	case stateError:
		return "error"
	case stateCanceled:
		return "cancelled"
	default:
		return "ready"
	}
}

func (m Model) elapsed() time.Duration {
	if (m.state == stateScanning || m.state == stateCancelling) && !m.scanStarted.IsZero() {
		return time.Since(m.scanStarted)
	}
	if m.current != nil {
		return m.current.Duration
	}
	return 0
}

func (m Model) commandPreview() string {
	request, err := m.request()
	if err != nil {
		return "nmap " + m.currentProfile().Name
	}
	if strings.TrimSpace(request.Target) == "" {
		request.Target = "<target>"
	}
	args, err := nmap.BuildArgs(request, "scan.xml")
	if err != nil {
		return "nmap " + m.currentProfile().Name
	}
	return nmap.CommandLine("nmap", args)
}

func panel(styles styles, title, content string, width, height int, focused bool) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	box := styles.panel
	if focused {
		box = styles.focusedPanel
	}
	box = box.Width(max(1, width-2)).Height(max(1, height-2))
	titleStyle := styles.panelTitle
	if focused {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#ffffff"))
	}
	body := titleStyle.Render(truncate(title, max(1, width-4)))
	if content != "" {
		body += "\n" + content
	}
	return box.Render(body)
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}
