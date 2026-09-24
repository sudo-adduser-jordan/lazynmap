package nmap

import (
	"fmt"
	"strings"
	"time"
)

// Profile is a named, safe set of arguments that can be passed to nmap.
//
// The arguments are deliberately kept as a slice rather than a command string.
// This lets the UI and runner pass each argument directly to exec.Command and
// avoids shell interpolation.
type Profile struct {
	ID          string
	Name        string
	Description string
	Args        []string
}

// Profiles contains the scan profiles exposed by lazynmap.
var Profiles = []Profile{
	{
		ID:          "quick",
		Name:        "Quick TCP",
		Description: "Top 1000 TCP ports using an unprivileged connect scan",
		Args:        []string{"-sT", "-T4", "--top-ports", "1000"},
	},
	{
		ID:          "service",
		Name:        "Service scan",
		Description: "Top 1000 TCP ports with service and version detection",
		Args:        []string{"-sT", "-T4", "--top-ports", "1000", "-sV"},
	},
	{
		ID:          "discovery",
		Name:        "Host discovery",
		Description: "Discover hosts without probing TCP ports",
		Args:        []string{"-sn", "-T4"},
	},
	{
		ID:          "full",
		Name:        "Full TCP",
		Description: "Scan all TCP ports using a connect scan",
		Args:        []string{"-sT", "-T4", "-p-"},
	},
	{
		ID:          "syn",
		Name:        "SYN scan",
		Description: "Fast top-1000 TCP scan; requires root or equivalent capabilities",
		Args:        []string{"-sS", "-T4", "--top-ports", "1000"},
	},
}

// ProfileByID returns a profile by its stable identifier.
func ProfileByID(id string) (Profile, bool) {
	for _, profile := range Profiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return Profile{}, false
}

// Request describes one nmap invocation.
type Request struct {
	Target    string
	Profile   Profile
	Ports     string
	ExtraArgs []string
}

// Result contains the machine-readable result and execution metadata for a
// completed scan.
type Result struct {
	Scan       Scan          `json:"scan"`
	Command    []string      `json:"command"`
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt time.Time     `json:"finished_at"`
	Duration   time.Duration `json:"duration"`
	Stderr     string        `json:"stderr,omitempty"`
}

// Scan is the normalized subset of nmap's XML output used by the UI.
type Scan struct {
	Scanner    string        `json:"scanner,omitempty"`
	NmapArgs   string        `json:"nmap_args,omitempty"`
	StartedAt  time.Time     `json:"started_at,omitempty"`
	FinishedAt time.Time     `json:"finished_at,omitempty"`
	Duration   time.Duration `json:"duration"`
	Hosts      []Host        `json:"hosts"`
	Stats      Stats         `json:"stats"`
}

// Stats contains the runstats section from an nmap XML document.
type Stats struct {
	HostsUp    int `json:"hosts_up"`
	HostsDown  int `json:"hosts_down"`
	HostsTotal int `json:"hosts_total"`
}

// Host is a normalized nmap host result.
type Host struct {
	Addresses   []Address     `json:"addresses"`
	Hostnames   []string      `json:"hostnames,omitempty"`
	State       string        `json:"state"`
	StateReason string        `json:"state_reason,omitempty"`
	Ports       []Port        `json:"ports"`
	OS          string        `json:"os,omitempty"`
	OSAccuracy  int           `json:"os_accuracy,omitempty"`
	Latency     time.Duration `json:"latency,omitempty"`
}

// Address is one address reported for a host.
type Address struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	Vendor  string `json:"vendor,omitempty"`
}

// Port is a normalized nmap port result.
type Port struct {
	Number     int     `json:"number"`
	Protocol   string  `json:"protocol"`
	State      string  `json:"state"`
	Reason     string  `json:"reason,omitempty"`
	ReasonTTL  float64 `json:"reason_ttl,omitempty"`
	Service    string  `json:"service,omitempty"`
	Product    string  `json:"product,omitempty"`
	Version    string  `json:"version,omitempty"`
	Extra      string  `json:"extra,omitempty"`
	Confidence int     `json:"confidence,omitempty"`
}

// PrimaryAddress returns the most useful address for display.
func (h Host) PrimaryAddress() string {
	for _, wanted := range []string{"ipv4", "ipv6"} {
		for _, address := range h.Addresses {
			if address.Type == wanted && address.Address != "" {
				return address.Address
			}
		}
	}
	if len(h.Addresses) > 0 {
		return h.Addresses[0].Address
	}
	return "-"
}

// DisplayName returns a hostname when one is available, otherwise the
// primary address.
func (h Host) DisplayName() string {
	if len(h.Hostnames) > 0 && h.Hostnames[0] != "" {
		return h.Hostnames[0]
	}
	return h.PrimaryAddress()
}

// OpenPortCount returns the number of open ports in the host result.
func (h Host) OpenPortCount() int {
	count := 0
	for _, port := range h.Ports {
		if strings.EqualFold(port.State, "open") {
			count++
		}
	}
	return count
}

// AddressList returns all addresses as a compact display string.
func (h Host) AddressList() string {
	values := make([]string, 0, len(h.Addresses))
	for _, address := range h.Addresses {
		if address.Address != "" {
			values = append(values, address.Address)
		}
	}
	return strings.Join(values, ", ")
}

// BuildArgs constructs the argv passed to nmap. outputPath is omitted when it
// is empty, which is useful for displaying a command preview in the UI.
func BuildArgs(request Request, outputPath string) ([]string, error) {
	target := strings.TrimSpace(request.Target)
	if target == "" {
		return nil, fmt.Errorf("target is required")
	}
	if strings.HasPrefix(target, "-") {
		return nil, fmt.Errorf("target must not start with '-'")
	}
	if strings.ContainsAny(target, "\r\n\t ") {
		return nil, fmt.Errorf("target must be a single host, network, or range")
	}
	if strings.IndexByte(target, 0) >= 0 {
		return nil, fmt.Errorf("target contains an invalid NUL byte")
	}

	profile := request.Profile
	if profile.ID == "" {
		profile = Profiles[0]
	}

	args := make([]string, 0, len(profile.Args)+len(request.ExtraArgs)+5)
	if outputPath != "" {
		args = append(args, "-oX", outputPath)
	}

	ports := strings.TrimSpace(request.Ports)
	profileArgs := profile.Args
	if ports != "" && profile.ID == "full" {
		// A user-supplied port list should replace the full profile's -p-,
		// rather than relying on nmap's last-option-wins behavior.
		profileArgs = make([]string, 0, len(profile.Args))
		for _, arg := range profile.Args {
			if arg != "-p-" {
				profileArgs = append(profileArgs, arg)
			}
		}
	}
	args = append(args, profileArgs...)

	if ports != "" {
		if strings.HasPrefix(ports, "-") || strings.ContainsAny(ports, "\r\n\t ") {
			return nil, fmt.Errorf("ports must be a comma-separated port specification")
		}
		args = append(args, "-p", ports)
	}

	for _, arg := range request.ExtraArgs {
		if strings.IndexByte(arg, 0) >= 0 {
			return nil, fmt.Errorf("extra arguments contain an invalid NUL byte")
		}
		args = append(args, arg)
	}

	// The target is intentionally the final argv element. It is never passed
	// through a shell, and BuildArgs rejects leading option-like targets.
	args = append(args, target)
	return args, nil
}

// CommandLine formats an argv for display in the UI. It is not used to execute
// the command; execution always uses the argument slice directly.
func CommandLine(binary string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteArg(binary))
	for _, arg := range args {
		parts = append(parts, quoteArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteArg(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return !(r == '-' || r == '_' || r == '.' || r == '/' || r == ':' || r == '=' || r == ',' || r == '@' || r == '%' || r == '+' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) == -1 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}
