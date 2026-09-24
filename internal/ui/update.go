package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sudo-adduser-jordan/lazynmap/internal/history"
	"github.com/sudo-adduser-jordan/lazynmap/internal/nmap"
)

type historyItem struct {
	entry history.Entry
}

func (i historyItem) Title() string {
	target := i.entry.Target
	if target == "" {
		target = "unknown target"
	}
	return fmt.Sprintf("%s  %s", target, i.entry.ProfileName)
}

func (i historyItem) Description() string {
	finished := i.entry.FinishedAt
	if finished.IsZero() {
		finished = i.entry.Result.FinishedAt
	}
	openPorts := openPortCount(i.entry.Result.Scan.Hosts)
	return fmt.Sprintf("%s  ·  %d host(s)  ·  %d open", finished.Local().Format("Jan 02 15:04"), len(i.entry.Result.Scan.Hosts), openPorts)
}

func (i historyItem) FilterValue() string {
	return strings.Join([]string{i.entry.Target, i.entry.ProfileName, i.entry.Result.Scan.NmapArgs}, " ")
}

func historyItems(entries []history.Entry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, historyItem{entry: entry})
	}
	return items
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}

	if m.showHelp {
		switch msg.String() {
		case "q", "esc", "?":
			m.showHelp = false
		}
		return m, nil
	}

	// Keep the conventional interrupt key available while editing input.
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	// Help is deliberately available even while an input is focused. The
	// other letter shortcuts remain ordinary text while editing a target.
	if msg.String() == "?" {
		m.showHelp = true
		return m, nil
	}

	if m.focus == focusFilter {
		switch msg.String() {
		case "enter":
			m.setFocus(focusHosts)
			m.rebuildTables()
			return m, nil
		case "esc":
			m.filter.SetValue("")
			m.setFocus(focusHosts)
			m.rebuildTables()
			return m, nil
		case "tab":
			m.setFocus(focusHosts)
			return m, nil
		case "shift+tab":
			m.setFocus(focusHosts)
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		return m, cmd
	}

	if m.focus == focusTarget || m.focus == focusPorts {
		switch msg.String() {
		case "enter":
			return m.startScan()
		case "esc":
			m.setFocus(focusHosts)
			return m, nil
		case "tab":
			m.cycleFocus(1)
			return m, nil
		case "shift+tab":
			m.cycleFocus(-1)
			return m, nil
		default:
			input := &m.target
			if m.focus == focusPorts {
				input = &m.ports
			}
			var cmd tea.Cmd
			*input, cmd = input.Update(msg)
			return m, cmd
		}
	}

	keyString := msg.String()
	switch keyString {
	case "enter":
		return m.startScan()
	case "q", "ctrl+c":
		if m.state == stateScanning || m.state == stateCancelling {
			m.cancelScan()
		}
		m.quitting = true
		return m, tea.Quit
	case "esc":
		if m.state == stateScanning || m.state == stateCancelling {
			m.cancelScan()
		} else if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.rebuildTables()
		} else if m.focus == focusDetail {
			m.setFocus(focusHosts)
		}
		return m, nil
	case "r":
		m.setFocus(focusTarget)
		return m, nil
	case "p":
		if len(m.profiles) > 0 {
			m.profileIndex = (m.profileIndex + 1) % len(m.profiles)
			message := "Profile: " + m.currentProfile().Name
			if m.currentProfile().ID == "syn" && os.Geteuid() != 0 {
				message += " (requires root)"
			}
			m.setStatus(stateIdle, message)
		}
		return m, nil
	case "/":
		m.setFocus(focusFilter)
		m.filter.CursorEnd()
		return m, nil
	case "e":
		if m.current == nil {
			m.setStatus(stateWarning, "There is no scan result to export")
			return m, nil
		}
		result := *m.current
		m.setStatus(stateIdle, "Exporting JSON…")
		return m, exportResult(result)
	case "x":
		m.cancelScan()
		return m, nil
	case "tab":
		m.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil
	}

	if m.focus == focusHistory {
		if handled, cmd := m.handleHistoryKey(msg); handled {
			return m, cmd
		}
		return m, nil
	}

	if m.focus == focusDetail {
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	if m.focus == focusHosts {
		if handled, cmd := m.handleTableKey(&m.hosts, msg, true); handled {
			return m, cmd
		}
	}
	if m.focus == focusPortsTable {
		if handled, cmd := m.handleTableKey(&m.portsTable, msg, false); handled {
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) handleHistoryKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	keyString := msg.String()
	switch keyString {
	case "j":
		m.history.CursorDown()
		m.syncHistorySelection()
		return true, nil
	case "k":
		m.history.CursorUp()
		m.syncHistorySelection()
		return true, nil
	case "up", "down", "pgup", "pgdown":
		var cmd tea.Cmd
		m.history, cmd = m.history.Update(msg)
		m.syncHistorySelection()
		return true, cmd
	case "home", "g":
		m.history.Select(0)
		m.syncHistorySelection()
		return true, nil
	case "end", "G":
		m.history.Select(max(0, len(m.entries)-1))
		m.syncHistorySelection()
		return true, nil
	default:
		return false, nil
	}
}

func (m *Model) handleTableKey(target *table.Model, msg tea.KeyMsg, hosts bool) (bool, tea.Cmd) {
	keyString := msg.String()
	switch keyString {
	case "j":
		target.MoveDown(1)
	case "k":
		target.MoveUp(1)
	case "home", "g":
		target.GotoTop()
	case "end", "G":
		target.GotoBottom()
	case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d":
		var cmd tea.Cmd
		*target, cmd = target.Update(msg)
		if hosts {
			m.refreshPorts()
		} else {
			m.refreshDetails()
		}
		return true, cmd
	default:
		return false, nil
	}
	if hosts {
		m.refreshPorts()
	} else {
		m.refreshDetails()
	}
	return true, nil
}

func (m *Model) cycleFocus(direction int) {
	focuses := []focusPane{focusTarget, focusPorts, focusHistory, focusHosts, focusPortsTable, focusDetail}
	currentIndex := 0
	for index, focus := range focuses {
		if focus == m.focus {
			currentIndex = index
			break
		}
	}
	next := (currentIndex + direction + len(focuses)) % len(focuses)
	m.setFocus(focuses[next])
}

func (m *Model) setFocus(focus focusPane) {
	m.target.Blur()
	m.ports.Blur()
	m.filter.Blur()
	m.hosts.Blur()
	m.portsTable.Blur()

	m.focus = focus
	switch focus {
	case focusTarget:
		m.target.Focus()
	case focusPorts:
		m.ports.Focus()
	case focusFilter:
		m.filter.Focus()
	case focusHistory:
		// The list owns its own focus state; selection is driven by the model.
	case focusHosts:
		m.hosts.Focus()
	case focusPortsTable:
		m.portsTable.Focus()
	case focusDetail:
		// The viewport has no focus state; key messages are routed here by
		// the parent model.
	}
}

func (m *Model) syncHistorySelection() {
	if len(m.entries) == 0 {
		m.current = nil
		m.rebuildTables()
		return
	}
	index := m.history.Index()
	if index < 0 {
		index = 0
	}
	if index >= len(m.entries) {
		index = len(m.entries) - 1
	}
	m.current = &m.entries[index].Result
	m.rebuildTables()
}

func (m *Model) rebuildTables() {
	m.visible = nil
	m.visiblePorts = nil
	if m.current == nil {
		m.hosts.SetRows(nil)
		m.portsTable.SetRows(nil)
		m.detail.SetContent("Run a scan to inspect hosts and ports.")
		return
	}

	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	for _, host := range m.current.Scan.Hosts {
		if query == "" || hostMatches(host, query) {
			m.visible = append(m.visible, host)
		}
	}
	if m.hosts.Cursor() >= len(m.visible) {
		m.hosts.SetCursor(max(0, len(m.visible)-1))
	}

	rows := make([]table.Row, 0, len(m.visible))
	for _, host := range m.visible {
		address := host.PrimaryAddress()
		if host.DisplayName() != address {
			address = fmt.Sprintf("%s (%s)", host.DisplayName(), address)
		}
		latency := "-"
		if host.Latency > 0 {
			latency = host.Latency.Round(100 * time.Microsecond).String()
		}
		rows = append(rows, table.Row{
			address,
			m.stateText(host.State),
			fmt.Sprintf("%d open", host.OpenPortCount()),
			host.OS,
			latency,
		})
	}
	m.hosts.SetRows(rows)
	m.refreshPorts()
}

func (m *Model) refreshPorts() {
	m.visiblePorts = nil
	if m.current == nil || len(m.visible) == 0 {
		m.portsTable.SetRows(nil)
		m.refreshDetails()
		return
	}
	hostIndex := clamp(m.hosts.Cursor(), 0, len(m.visible)-1)
	host := m.visible[hostIndex]
	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	for _, port := range host.Ports {
		if query == "" || portMatches(port, query) {
			m.visiblePorts = append(m.visiblePorts, port)
		}
	}
	if m.portsTable.Cursor() >= len(m.visiblePorts) {
		m.portsTable.SetCursor(max(0, len(m.visiblePorts)-1))
	}
	rows := make([]table.Row, 0, len(m.visiblePorts))
	for _, port := range m.visiblePorts {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", port.Number),
			port.Protocol,
			m.stateText(port.State),
			port.Service,
			portVersion(port),
		})
	}
	m.portsTable.SetRows(rows)
	m.refreshDetails()
}

func (m *Model) refreshDetails() {
	if m.current == nil || len(m.visible) == 0 {
		m.detail.SetContent("No hosts matched the current filter.")
		return
	}
	hostIndex := clamp(m.hosts.Cursor(), 0, len(m.visible)-1)
	host := m.visible[hostIndex]
	lines := []string{
		fmt.Sprintf("%s  [%s]", host.DisplayName(), host.State),
		fmt.Sprintf("Addresses: %s", host.AddressList()),
	}
	if len(host.Hostnames) > 0 {
		lines = append(lines, "Hostnames: "+strings.Join(host.Hostnames, ", "))
	}
	if host.StateReason != "" {
		lines = append(lines, "Reason: "+host.StateReason)
	}
	if host.OS != "" {
		lines = append(lines, fmt.Sprintf("OS: %s (%d%%)", host.OS, host.OSAccuracy))
	}
	if len(m.visiblePorts) > 0 {
		portIndex := clamp(m.portsTable.Cursor(), 0, len(m.visiblePorts)-1)
		port := m.visiblePorts[portIndex]
		lines = append(lines, "", fmt.Sprintf("Port %d/%s  [%s]", port.Number, port.Protocol, port.State))
		if port.Service != "" {
			lines = append(lines, "Service: "+port.Service)
		}
		if port.Product != "" || port.Version != "" {
			lines = append(lines, "Product: "+strings.TrimSpace(port.Product+" "+port.Version))
		}
		if port.Reason != "" {
			lines = append(lines, "Port reason: "+port.Reason)
		}
		if port.Extra != "" {
			lines = append(lines, "Extra: "+port.Extra)
		}
	}
	m.detail.SetContent(strings.Join(lines, "\n"))
}

func hostMatches(host nmap.Host, query string) bool {
	values := []string{host.PrimaryAddress(), host.AddressList(), host.State, host.StateReason, host.OS}
	values = append(values, host.Hostnames...)
	for _, port := range host.Ports {
		values = append(values, port.State, port.Service, port.Product, port.Version, fmt.Sprintf("%d", port.Number))
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func portMatches(port nmap.Port, query string) bool {
	values := []string{port.State, port.Service, port.Product, port.Version, port.Protocol, fmt.Sprintf("%d", port.Number)}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func (m *Model) stateText(state string) string {
	switch strings.ToLower(state) {
	case "open", "up":
		return m.styles.success.Render(state)
	case "closed", "down":
		return m.styles.error.Render(state)
	case "filtered", "open|filtered":
		return m.styles.warning.Render(state)
	default:
		return state
	}
}

func portVersion(port nmap.Port) string {
	version := strings.TrimSpace(port.Product + " " + port.Version)
	if version == "" {
		return "-"
	}
	return version
}
