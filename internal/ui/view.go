package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/sudo-adduser-jordan/lazynmap/internal/history"
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
	bodyHeight := max(3, height-lipgloss.Height(header)-lipgloss.Height(footer))
	body := m.bodyView(width, bodyHeight)
	screen := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	switch {
	case m.showHistory:
		popup := m.historyPopup(width, height)
		return placeOverlay(screen, popup, width, height)
	case m.showHelp:
		popup := m.helpPopup(width, height)
		return placeOverlay(screen, popup, width, height)
	default:
		return screen
	}
}

func (m Model) headerView(width int) string {
	profile := m.currentProfile()
	title := fmt.Sprintf(
		"Scan · %s · %s · %s",
		profile.Name,
		m.stateLabel(),
		m.elapsed().Round(time.Second),
	)

	hint := "enter run · ? help"
	if m.focus == focusFilter {
		hint = "enter apply · esc clear"
	}
	content := m.scanFieldsView(width-2) + "\n" + m.styles.muted.Render("$ "+m.commandPreview())
	focused := m.focus == focusTarget || m.focus == focusPorts || m.focus == focusFilter
	return panelWithHint(m.styles, title, hint, content, width, scanPanelHeight, focused)
}

func (m Model) scanFieldsView(width int) string {
	width = max(8, width)
	if m.focus == focusFilter {
		filter := m.filter
		filter.Width = max(8, width-24)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			filter.View(),
			"  ",
			m.styles.muted.Render("enter apply · esc clear"),
		)
	}

	portsWidth := clamp(width*20/100, 10, 22)
	targetWidth := clamp(width-portsWidth-19, 14, 46)
	target := m.target
	ports := m.ports
	target.Width = targetWidth
	ports.Width = portsWidth
	line := lipgloss.JoinHorizontal(lipgloss.Top, target.View(), "  ", ports.View())
	if ansi.StringWidth(line)+12 <= width {
		line += "  " + m.styles.primary.Render("enter run")
	}
	return line
}

func (m Model) bodyView(width, height int) string {
	layout := calculateBodyLayout(width, height)
	hostContent := m.styles.muted.Render("No scan results yet.\n\nStart a scan above to inspect hosts and ports.")
	if m.current != nil && len(m.visible) == 0 {
		hostContent = m.styles.muted.Render("No hosts match the current filter.")
	} else if len(m.visible) > 0 {
		hostContent = m.hosts.View()
	}
	hostTitle := "Hosts"
	if m.current != nil {
		hostTitle = fmt.Sprintf("Hosts (%d/%d)", len(m.visible), len(m.current.Scan.Hosts))
		if strings.TrimSpace(m.filter.Value()) == "" {
			hostTitle = fmt.Sprintf("Hosts (%d)", len(m.visible))
		}
	}
	left := panel(m.styles, hostTitle, hostContent, layout.leftWidth, height, m.focus == focusHosts)
	right := m.resultPanels(layout)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) resultPanels(layout screenLayout) string {
	hostLabel := "-"
	portsTitle := "Ports"
	if len(m.visible) > 0 {
		host := m.visible[clamp(m.hosts.Cursor(), 0, len(m.visible)-1)]
		hostLabel = host.DisplayName()
		if host.PrimaryAddress() != hostLabel {
			hostLabel = fmt.Sprintf("%s (%s)", hostLabel, host.PrimaryAddress())
		}
		portsTitle = fmt.Sprintf("Ports · %s (%d open)", hostLabel, host.OpenPortCount())
	}

	portsContent := m.styles.muted.Render("No ports to show.")
	if m.current == nil {
		portsContent = m.styles.muted.Render("Run a scan to discover open ports.")
	} else if len(m.visible) == 0 {
		portsContent = m.styles.muted.Render("No hosts match the current filter.")
	} else if len(m.visiblePorts) == 0 {
		portsContent = m.styles.muted.Render("No ports reported for this host.")
	} else {
		portsContent = m.portsTable.View()
	}
	portsPanel := panel(
		m.styles,
		portsTitle,
		portsContent,
		layout.rightWidth,
		layout.portsHeight,
		m.focus == focusPortsTable,
	)

	if layout.detailsHeight == 0 {
		return portsPanel
	}
	detailsTitle := "Details"
	if hostLabel != "-" {
		detailsTitle = "Details · " + hostLabel
	}
	detailsPanel := panel(
		m.styles,
		detailsTitle,
		m.detail.View(),
		layout.rightWidth,
		layout.detailsHeight,
		m.focus == focusDetail,
	)
	return lipgloss.JoinVertical(lipgloss.Left, detailsPanel, portsPanel)
}

func (m Model) historyPopup(width, height int) string {
	popupWidth := min(68, max(12, width-4))
	popupHeight := min(18, max(5, height-2))
	contentWidth := max(1, popupWidth-2)
	contentHeight := max(1, popupHeight-2)
	rightHint := ""
	content := m.styles.muted.Render("No saved scans yet.")
	if len(m.entries) > 0 {
		rightHint = fmt.Sprintf("%d/%d", m.historyIndex+1, len(m.entries))
		visibleItems := max(1, contentHeight/2)
		start := max(0, m.historyIndex-visibleItems+1)
		if start > len(m.entries)-visibleItems {
			start = max(0, len(m.entries)-visibleItems)
		}

		lines := make([]string, 0, visibleItems*2)
		for index := start; index < len(m.entries) && index < start+visibleItems; index++ {
			title, description := historyEntryText(m.entries[index])
			if index == m.historyIndex {
				lines = append(lines,
					m.styles.selected.Render(padRight(truncate(title, contentWidth), contentWidth)),
					m.styles.selectedDim.Render(padRight(truncate(description, contentWidth), contentWidth)),
				)
				continue
			}
			lines = append(lines,
				m.styles.value.Render(truncate(title, contentWidth)),
				m.styles.muted.Render(truncate(description, contentWidth)),
			)
		}
		content = strings.Join(lines, "\n")
	}
	return panelWithHint(m.styles, "Scan history", rightHint, content, popupWidth, popupHeight, true)
}

func historyEntryText(entry history.Entry) (string, string) {
	target := entry.Target
	if target == "" {
		target = "unknown target"
	}
	profile := entry.ProfileName
	if profile == "" {
		profile = "Custom"
	}
	finished := entry.FinishedAt
	if finished.IsZero() {
		finished = entry.Result.FinishedAt
	}
	return fmt.Sprintf("%s  ·  %s", target, profile), fmt.Sprintf(
		"%s  ·  %d host(s)  ·  %d open",
		finished.Local().Format("Jan 02 15:04"),
		len(entry.Result.Scan.Hosts),
		openPortCount(entry.Result.Scan.Hosts),
	)
}

func (m Model) helpPopup(width, height int) string {
	popupWidth := min(64, max(18, width-4))
	popupHeight := min(23, max(7, height-2))
	heading := func(value string) string {
		return m.styles.primary.Render(strings.ToUpper(value))
	}
	item := func(key, description string) string {
		return "  " + m.styles.helpKey.Render(padRight(key, 12)) + description
	}
	lines := []string{
		heading("Scanning"),
		item("enter", "Run the configured scan"),
		item("x / esc", "Cancel an active scan"),
		item("e", "Export the selected result as JSON"),
		"",
		heading("Navigation"),
		item("tab", "Move between panes"),
		item("shift+tab", "Move to the previous pane"),
		item("j / k", "Move down / up"),
		item("g / G", "Jump to first / last row"),
		item("/", "Filter hosts and ports"),
		item("h", "Open history from a result pane"),
		"",
		heading("Scan setup"),
		item("r", "Edit the target"),
		item("p", "Cycle scan profile"),
		"",
		heading("General"),
		item("esc / ?", "Close this help"),
		item("q", "Quit from the main view"),
		"",
		m.styles.muted.Render("Only scan systems and networks you are authorized to test."),
	}
	content := strings.Join(lines, "\n")
	return panel(m.styles, "Help · lazynmap", content, popupWidth, popupHeight, true)
}

func (m Model) footerView(width int) string {
	overlay := ""
	switch {
	case m.showHistory:
		overlay = "history"
	case m.showHelp:
		overlay = "help"
	}

	left := m.statusView()
	if overlay != "" || m.statusMessage == "" {
		left = m.styles.primary.Render(m.keys.footerHelp(m.focus, m.state, overlay))
	}
	right := m.styles.muted.Render("lazynmap")
	return joinEdges(left, right, max(1, width))
}

func (m Model) statusView() string {
	if m.statusMessage == "" {
		return m.styles.footer.Render("Ready")
	}
	switch m.statusKind {
	case stateSuccess:
		return m.styles.success.Render("✓ " + m.statusMessage)
	case stateScanning:
		return m.styles.primary.Render("… " + m.statusMessage)
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
	return panelWithHint(styles, title, "", content, width, height, focused)
}

func panelWithHint(styles styles, title, hint, content string, width, height int, focused bool) string {
	width = max(4, width)
	height = max(3, height)
	innerWidth := width - 2
	bodyHeight := height - 2

	borderStyle := styles.border
	titleStyle := styles.panelTitle
	if focused {
		borderStyle = styles.focusedBorder
		titleStyle = styles.focusedTitle
	}

	titleText := ""
	leftWidth := 0
	if title != "" && innerWidth >= 3 {
		titleText = "─" + truncate(title, innerWidth-2) + " "
		leftWidth = ansi.StringWidth(titleText)
	}
	hintText := ""
	rightWidth := 0
	if hint != "" && leftWidth+2 <= innerWidth {
		hintText = truncate(hint, innerWidth-leftWidth-2)
		rightWidth = ansi.StringWidth(hintText) + 2
	}
	fillWidth := max(0, innerWidth-leftWidth-rightWidth)

	var top strings.Builder
	top.WriteString(borderStyle.Render("╭"))
	top.WriteString(titleStyle.Render(titleText))
	top.WriteString(borderStyle.Render(strings.Repeat("─", fillWidth)))
	if hintText != "" {
		top.WriteString(borderStyle.Render(" "))
		top.WriteString(styles.panelHint.Render(hintText))
		top.WriteString(borderStyle.Render("─"))
	}
	top.WriteString(borderStyle.Render("╮"))

	lines := strings.Split(content, "\n")
	body := make([]string, 0, bodyHeight)
	for index := 0; index < bodyHeight; index++ {
		line := ""
		if index < len(lines) {
			line = lines[index]
		}
		line = truncate(line, innerWidth)
		body = append(body, padRight(line, innerWidth))
	}

	rendered := make([]string, 0, height)
	rendered = append(rendered, top.String())
	for _, line := range body {
		rendered = append(rendered, borderStyle.Render("│")+line+borderStyle.Render("│"))
	}
	rendered = append(rendered, borderStyle.Render("╰"+strings.Repeat("─", innerWidth)+"╯"))
	return strings.Join(rendered, "\n")
}

func placeOverlay(background, popup string, width, height int) string {
	popupLines := strings.Split(popup, "\n")
	popupWidth := lipgloss.Width(popup)
	popupHeight := lipgloss.Height(popup)
	x := max(0, (width-popupWidth)/2)
	y := max(0, (height-popupHeight)/2)
	backgroundLines := strings.Split(background, "\n")

	lines := make([]string, 0, height)
	for row := 0; row < height; row++ {
		baseLine := ""
		if row < len(backgroundLines) {
			baseLine = backgroundLines[row]
		}
		if row < y || row >= y+popupHeight {
			lines = append(lines, padRight(truncate(baseLine, width), width))
			continue
		}

		popupLine := ""
		if row-y < len(popupLines) {
			popupLine = popupLines[row-y]
		}
		before := padRight(ansi.Cut(baseLine, 0, x), x)
		center := padRight(ansi.Cut(popupLine, 0, popupWidth), popupWidth)
		afterWidth := max(0, width-x-popupWidth)
		after := padRight(ansi.Cut(baseLine, x+popupWidth, width), afterWidth)
		lines = append(lines, before+center+after)
	}
	return strings.Join(lines, "\n")
}

func joinEdges(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	right = truncate(right, width)
	rightWidth := ansi.StringWidth(right)
	if rightWidth >= width {
		return right
	}
	left = truncate(left, width-rightWidth-1)
	leftWidth := ansi.StringWidth(left)
	return left + strings.Repeat(" ", width-leftWidth-rightWidth) + right
}

func padRight(value string, width int) string {
	padding := max(0, width-ansi.StringWidth(value))
	return value + strings.Repeat(" ", padding)
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
