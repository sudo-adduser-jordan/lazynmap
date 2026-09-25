package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sudo-adduser-jordan/lazynmap/internal/history"
	"github.com/sudo-adduser-jordan/lazynmap/internal/nmap"
)

type focusPane int

const (
	focusTarget focusPane = iota
	focusPorts
	focusHistory
	focusHosts
	focusPortsTable
	focusDetail
	focusFilter
)

type scanState int

const (
	stateIdle scanState = iota
	stateScanning
	stateCancelling
	stateSuccess
	stateError
	stateCanceled
	stateWarning
)

const scanPanelHeight = 4

type Model struct {
	width  int
	height int

	target textinput.Model
	ports  textinput.Model
	filter textinput.Model

	historyIndex int
	hosts        table.Model
	portsTable   table.Model
	detail       viewport.Model
	runner       nmap.Runner
	store        *history.Store
	entries      []history.Entry
	profiles     []nmap.Profile
	profileIndex int

	current      *nmap.Result
	lastQuery    nmap.Request
	visible      []nmap.Host
	visiblePorts []nmap.Port

	focus         focusPane
	state         scanState
	cancel        context.CancelFunc
	scanStarted   time.Time
	statusKind    scanState
	statusMessage string
	statusAt      time.Time

	showHelp           bool
	showHistory        bool
	historyReturnFocus focusPane
	quitting           bool
	styles             styles
	keys               keyMap
}

// NewModel creates the initial TUI model.
func NewModel() Model {
	styles := newStyles()
	keys := defaultKeyMap()

	target := textinput.New()
	target.Prompt = "Target "
	target.Placeholder = "host, CIDR, or range"
	target.CharLimit = 256
	target.Width = 34
	_ = target.Focus()

	ports := textinput.New()
	ports.Prompt = "Ports "
	ports.Placeholder = "default"
	ports.CharLimit = 128
	ports.Width = 20

	filter := textinput.New()
	filter.Prompt = "Filter "
	filter.Placeholder = "address, service, or state"
	filter.CharLimit = 128
	filter.Width = 28

	for _, input := range []*textinput.Model{&target, &ports, &filter} {
		input.PromptStyle = styles.label
		input.PlaceholderStyle = styles.muted
		input.TextStyle = styles.value
	}

	hostTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "HOST", Width: 22},
			{Title: "STATE", Width: 7},
			{Title: "PORTS", Width: 5},
		}),
		table.WithHeight(8),
		table.WithFocused(false),
	)
	hostStyles := table.DefaultStyles()
	hostStyles.Header = styles.muted.Bold(true)
	hostStyles.Cell = lipgloss.NewStyle()
	hostStyles.Selected = styles.selected
	hostTable.SetStyles(hostStyles)

	portsTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "PORT", Width: 8},
			{Title: "PROTO", Width: 7},
			{Title: "STATE", Width: 10},
			{Title: "SERVICE", Width: 18},
			{Title: "VERSION", Width: 28},
		}),
		table.WithHeight(6),
		table.WithFocused(false),
	)
	portsTable.SetStyles(hostStyles)

	detail := viewport.New(40, 8)
	detail.SetContent(styles.muted.Render("Run a scan to inspect a host."))

	store, storeErr := history.NewStore()
	entries := []history.Entry{}
	initialStatus := ""
	initialStatusKind := stateIdle
	if storeErr != nil {
		initialStatus = "History unavailable: " + storeErr.Error()
		initialStatusKind = stateWarning
	} else {
		entries = store.Entries
	}

	model := Model{
		target:        target,
		ports:         ports,
		filter:        filter,
		hosts:         hostTable,
		portsTable:    portsTable,
		detail:        detail,
		runner:        nmap.NewRunner(),
		store:         store,
		entries:       entries,
		profiles:      nmap.Profiles,
		profileIndex:  0,
		focus:         focusTarget,
		state:         stateIdle,
		statusKind:    initialStatusKind,
		statusMessage: initialStatus,
		styles:        styles,
		keys:          keys,
	}
	if len(entries) > 0 {
		model.current = &model.entries[0].Result
		model.historyIndex = 0
		model.rebuildTables()
	}
	return model
}

// Init starts cursor blinking for the focused input.
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles terminal and user messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tickMsg:
		return m.handleTick(msg)
	case scanFinishedMsg:
		return m.handleScanFinished(msg)
	case scanFailedMsg:
		return m.handleScanFailed(msg)
	case scanCanceledMsg:
		return m.handleScanCanceled(msg)
	case exportFinishedMsg:
		if msg.err != nil {
			m.setStatus(stateError, "Export failed: "+msg.err.Error())
		} else {
			m.setStatus(stateSuccess, "Exported "+msg.path)
		}
		return m, nil
	case historySavedMsg:
		if msg.err != nil {
			m.setStatus(stateWarning, "Scan complete; history could not be saved: "+msg.err.Error())
		}
		return m, nil
	default:
		// Let the focused input consume component messages such as cursor
		// blinks and paste events.
		var cmd tea.Cmd
		switch m.focus {
		case focusTarget:
			m.target, cmd = m.target.Update(msg)
		case focusPorts:
			m.ports, cmd = m.ports.Update(msg)
		case focusFilter:
			m.filter, cmd = m.filter.Update(msg)
		}
		return m, cmd
	}
}

func (m *Model) handleTick(msg tickMsg) (tea.Model, tea.Cmd) {
	if m.state == stateScanning || m.state == stateCancelling {
		return m, tick()
	}
	if m.statusMessage != "" && !m.statusAt.IsZero() && time.Since(m.statusAt) > 5*time.Second && m.statusKind != stateError {
		m.statusMessage = ""
		m.statusKind = stateIdle
	}
	return m, nil
}

type screenLayout struct {
	leftWidth     int
	rightWidth    int
	leftHeight    int
	detailsHeight int
	portsHeight   int
}

func calculateLayout(width, height int) screenLayout {
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	return calculateBodyLayout(width, max(3, height-scanPanelHeight-1))
}

func calculateBodyLayout(width, bodyHeight int) screenLayout {
	leftWidth := clamp(width*30/100, 24, 42)
	rightWidth := max(3, width-leftWidth)
	layout := screenLayout{
		leftWidth:   leftWidth,
		rightWidth:  rightWidth,
		leftHeight:  bodyHeight,
		portsHeight: bodyHeight,
	}
	if bodyHeight < 7 {
		return layout
	}

	layout.detailsHeight = clamp(bodyHeight*30/100, 5, 9)
	layout.detailsHeight = min(layout.detailsHeight, bodyHeight-3)
	layout.portsHeight = bodyHeight - layout.detailsHeight
	return layout
}

func (m *Model) resize() {
	if m.width <= 0 {
		m.width = 100
	}
	if m.height <= 0 {
		m.height = 30
	}

	layout := calculateLayout(m.width, m.height)
	leftInnerWidth := max(3, layout.leftWidth-2)
	m.hosts.SetColumns(hostColumns(leftInnerWidth))
	m.hosts.SetWidth(leftInnerWidth)
	m.hosts.SetHeight(max(1, layout.leftHeight-2))

	rightInnerWidth := max(3, layout.rightWidth-2)
	m.portsTable.SetColumns(portColumns(rightInnerWidth))
	m.portsTable.SetWidth(rightInnerWidth)
	m.portsTable.SetHeight(max(1, layout.portsHeight-2))
	m.detail.Width = rightInnerWidth
	m.detail.Height = max(1, layout.detailsHeight-2)
}

func hostColumns(width int) []table.Column {
	stateWidth := clamp(width*30/100, 5, 7)
	portsWidth := clamp(width*18/100, 4, 5)
	hostWidth := max(1, width-stateWidth-portsWidth)
	return []table.Column{
		{Title: "HOST", Width: hostWidth},
		{Title: "STATE", Width: stateWidth},
		{Title: "PORTS", Width: portsWidth},
	}
}

func portColumns(width int) []table.Column {
	if width < 34 {
		portWidth := clamp(width*25/100, 4, 6)
		stateWidth := clamp(width*30/100, 5, 7)
		serviceWidth := max(1, width-portWidth-stateWidth)
		return []table.Column{
			{Title: "PORT", Width: portWidth},
			{Title: "PROTO", Width: 0},
			{Title: "STATE", Width: stateWidth},
			{Title: "SERVICE", Width: serviceWidth},
			{Title: "VERSION", Width: 0},
		}
	}

	portWidth := 5
	protocolWidth := 5
	stateWidth := 7
	remaining := max(1, width-portWidth-protocolWidth-stateWidth)
	serviceWidth := clamp(remaining*42/100, 8, remaining-1)
	versionWidth := max(1, remaining-serviceWidth)
	return []table.Column{
		{Title: "PORT", Width: portWidth},
		{Title: "PROTO", Width: protocolWidth},
		{Title: "STATE", Width: stateWidth},
		{Title: "SERVICE", Width: serviceWidth},
		{Title: "VERSION", Width: versionWidth},
	}
}

func (m *Model) setStatus(kind scanState, message string) {
	m.statusKind = kind
	m.statusMessage = message
	m.statusAt = time.Now()
}

func (m *Model) currentProfile() nmap.Profile {
	if len(m.profiles) == 0 {
		return nmap.Profiles[0]
	}
	return m.profiles[m.profileIndex]
}

func (m *Model) request() (nmap.Request, error) {
	return nmap.Request{
		Target:  strings.TrimSpace(m.target.Value()),
		Profile: m.currentProfile(),
		Ports:   strings.TrimSpace(m.ports.Value()),
	}, nil
}

func (m *Model) startScan() (tea.Model, tea.Cmd) {
	if m.state == stateScanning || m.state == stateCancelling {
		return m, nil
	}
	request, err := m.request()
	if err == nil {
		_, err = nmap.BuildArgs(request, "")
	}
	if err != nil {
		m.setStatus(stateError, err.Error())
		m.setFocus(focusTarget)
		return m, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.lastQuery = request
	m.scanStarted = time.Now()
	m.state = stateScanning
	m.statusKind = stateScanning
	m.statusMessage = "Scanning " + request.Target + "…"
	if request.Profile.ID == "syn" && os.Geteuid() != 0 {
		m.statusMessage += " (SYN scan requires root)"
	}
	m.statusAt = time.Now()
	m.setFocus(focusHosts)
	return m, tea.Batch(runScan(ctx, m.runner, request), tick())
}

func (m *Model) cancelScan() {
	if (m.state == stateScanning || m.state == stateCancelling) && m.cancel != nil {
		m.cancel()
		m.state = stateCancelling
		m.statusKind = stateCancelling
		m.statusMessage = "Cancelling scan…"
		m.statusAt = time.Now()
	}
}

func (m *Model) handleScanFinished(msg scanFinishedMsg) (tea.Model, tea.Cmd) {
	m.state = stateSuccess
	m.statusKind = stateSuccess
	m.statusAt = time.Now()
	m.current = &msg.result
	m.setStatus(stateSuccess, fmt.Sprintf("Scan complete: %d host(s), %d open port(s)", len(msg.result.Scan.Hosts), openPortCount(msg.result.Scan.Hosts)))
	m.rebuildTables()
	m.setFocus(focusHosts)

	cmds := make([]tea.Cmd, 0, 1)
	if m.store != nil {
		m.entries = m.store.Add(m.lastQuery, msg.result)
		m.store.Entries = m.entries
		m.historyIndex = 0
		entries := append([]history.Entry(nil), m.entries...)
		cmds = append(cmds, saveHistory(m.store.Path, entries))
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleScanFailed(msg scanFailedMsg) (tea.Model, tea.Cmd) {
	m.state = stateError
	if msg.result.Scan.Hosts != nil {
		m.current = &msg.result
		m.rebuildTables()
	}
	m.setStatus(stateError, msg.err.Error())
	return m, nil
}

func (m *Model) handleScanCanceled(msg scanCanceledMsg) (tea.Model, tea.Cmd) {
	m.state = stateCanceled
	m.setStatus(stateWarning, "Scan cancelled")
	return m, nil
}

func runScan(ctx context.Context, runner nmap.Runner, request nmap.Request) tea.Cmd {
	return func() tea.Msg {
		result, err := runner.Run(ctx, request)
		if err != nil {
			if ctx.Err() != nil {
				return scanCanceledMsg{err: err}
			}
			return scanFailedMsg{result: result, err: err}
		}
		return scanFinishedMsg{result: result}
	}
}

func saveHistory(path string, entries []history.Entry) tea.Cmd {
	return func() tea.Msg {
		return historySavedMsg{err: history.Save(path, entries)}
	}
}

type scanFinishedMsg struct {
	result nmap.Result
}

type scanFailedMsg struct {
	result nmap.Result
	err    error
}

type scanCanceledMsg struct {
	err error
}

type historySavedMsg struct {
	err error
}

type exportFinishedMsg struct {
	path string
	err  error
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(now time.Time) tea.Msg {
		return tickMsg(now)
	})
}

func exportResult(result nmap.Result) tea.Cmd {
	return func() tea.Msg {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return exportFinishedMsg{err: err}
		}
		data = append(data, '\n')
		name := fmt.Sprintf("lazynmap-%s.json", time.Now().Format("20060102-150405.000000000"))
		path, err := filepath.Abs(name)
		if err == nil {
			err = os.WriteFile(path, data, 0o600)
		}
		return exportFinishedMsg{path: path, err: err}
	}
}

func openPortCount(hosts []nmap.Host) int {
	count := 0
	for _, host := range hosts {
		count += host.OpenPortCount()
	}
	return count
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
