package nmap

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type xmlScan struct {
	XMLName  xml.Name    `xml:"nmaprun"`
	Scanner  string      `xml:"scanner,attr"`
	Args     string      `xml:"args,attr"`
	Start    string      `xml:"start,attr"`
	StartStr string      `xml:"startstr,attr"`
	Hosts    []xmlHost   `xml:"host"`
	RunStats xmlRunStats `xml:"runstats"`
}

type xmlHost struct {
	Status    xmlStatus    `xml:"status"`
	Addresses []xmlAddress `xml:"address"`
	Hostnames xmlHostnames `xml:"hostnames"`
	Ports     xmlPorts     `xml:"ports"`
	Times     xmlTimes     `xml:"times"`
	OS        xmlOS        `xml:"os"`
}

type xmlStatus struct {
	State    string `xml:"state,attr"`
	Reason   string `xml:"reason,attr"`
	ReasonTT string `xml:"reason_ttl,attr"`
}

type xmlAddress struct {
	Type   string `xml:"addrtype,attr"`
	Addr   string `xml:"addr,attr"`
	Vendor string `xml:"vendor,attr"`
}

type xmlHostnames struct {
	Hostnames []xmlHostname `xml:"hostname"`
}

type xmlHostname struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type xmlPorts struct {
	Ports []xmlPort `xml:"port"`
}

type xmlPort struct {
	PortID   int          `xml:"portid,attr"`
	Protocol string       `xml:"protocol,attr"`
	State    xmlPortState `xml:"state"`
	Service  xmlService   `xml:"service"`
	Extra    []xmlExtra   `xml:"extra"`
}

type xmlPortState struct {
	State     string  `xml:"state,attr"`
	Reason    string  `xml:"reason,attr"`
	ReasonTTL float64 `xml:"reason_ttl,attr"`
}

type xmlService struct {
	Name       string `xml:"name,attr"`
	Product    string `xml:"product,attr"`
	Version    string `xml:"version,attr"`
	Confidence int    `xml:"confidence,attr"`
	Method     string `xml:"method,attr"`
}

type xmlExtra struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type xmlTimes struct {
	SRTT   string `xml:"srtt,attr"`
	RTTVar string `xml:"rttvar,attr"`
	To     string `xml:"to,attr"`
}

type xmlOS struct {
	Matches []xmlOSMatch `xml:"osmatch"`
}

type xmlOSMatch struct {
	Name     string `xml:"name,attr"`
	Accuracy int    `xml:"accuracy,attr"`
	Line     int    `xml:"line,attr"`
}

type xmlRunStats struct {
	Finished xmlFinished `xml:"finished"`
	Hosts    xmlCounts   `xml:"hosts"`
}

type xmlFinished struct {
	Time    string  `xml:"time,attr"`
	Elapsed float64 `xml:"elapsed,attr"`
	Exit    string  `xml:"exit,attr"`
	Summary string  `xml:"summary,attr"`
}

type xmlCounts struct {
	Up    int `xml:"up,attr"`
	Down  int `xml:"down,attr"`
	Total int `xml:"total,attr"`
}

// ParseXML parses an nmap XML document from r.
func ParseXML(r io.Reader) (Scan, error) {
	var document xmlScan
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&document); err != nil {
		return Scan{}, fmt.Errorf("decode nmap XML: %w", err)
	}

	started := parseEpoch(document.Start)
	finished := parseEpoch(document.RunStats.Finished.Time)
	duration := time.Duration(document.RunStats.Finished.Elapsed * float64(time.Second))
	if duration <= 0 && !started.IsZero() && !finished.IsZero() {
		duration = finished.Sub(started)
	}

	scan := Scan{
		Scanner:    document.Scanner,
		NmapArgs:   document.Args,
		StartedAt:  started,
		FinishedAt: finished,
		Duration:   duration,
		Hosts:      make([]Host, 0, len(document.Hosts)),
		Stats: Stats{
			HostsUp:    document.RunStats.Hosts.Up,
			HostsDown:  document.RunStats.Hosts.Down,
			HostsTotal: document.RunStats.Hosts.Total,
		},
	}

	for _, xmlHostValue := range document.Hosts {
		host := Host{
			State:       strings.ToLower(xmlHostValue.Status.State),
			StateReason: xmlHostValue.Status.Reason,
			Addresses:   make([]Address, 0, len(xmlHostValue.Addresses)),
			Ports:       make([]Port, 0, len(xmlHostValue.Ports.Ports)),
		}
		for _, address := range xmlHostValue.Addresses {
			host.Addresses = append(host.Addresses, Address{
				Type:    address.Type,
				Address: address.Addr,
				Vendor:  address.Vendor,
			})
		}
		for _, hostname := range xmlHostValue.Hostnames.Hostnames {
			if hostname.Name != "" {
				host.Hostnames = append(host.Hostnames, hostname.Name)
			}
		}
		for _, port := range xmlHostValue.Ports.Ports {
			host.Ports = append(host.Ports, Port{
				Number:     port.PortID,
				Protocol:   strings.ToLower(port.Protocol),
				State:      strings.ToLower(port.State.State),
				Reason:     port.State.Reason,
				ReasonTTL:  port.State.ReasonTTL,
				Service:    port.Service.Name,
				Product:    port.Service.Product,
				Version:    port.Service.Version,
				Confidence: port.Service.Confidence,
				Extra:      formatExtra(port.Extra),
			})
		}
		sort.SliceStable(host.Ports, func(i, j int) bool {
			if host.Ports[i].Number == host.Ports[j].Number {
				return host.Ports[i].Protocol < host.Ports[j].Protocol
			}
			return host.Ports[i].Number < host.Ports[j].Number
		})
		host.OS, host.OSAccuracy = bestOSMatch(xmlHostValue.OS)
		host.Latency = parseMilliseconds(xmlHostValue.Times.SRTT)
		scan.Hosts = append(scan.Hosts, host)
	}

	return scan, nil
}

// ParseXMLFile parses an nmap XML file.
func ParseXMLFile(path string) (Scan, error) {
	file, err := os.Open(path)
	if err != nil {
		return Scan{}, fmt.Errorf("open nmap XML: %w", err)
	}
	defer file.Close()
	return ParseXML(file)
}

func bestOSMatch(os xmlOS) (string, int) {
	best := xmlOSMatch{}
	for _, match := range os.Matches {
		if match.Accuracy > best.Accuracy || (match.Accuracy == best.Accuracy && match.Name != "" && best.Name == "") {
			best = match
		}
	}
	return best.Name, best.Accuracy
}

func formatExtra(extra []xmlExtra) string {
	values := make([]string, 0, len(extra))
	for _, value := range extra {
		trimmed := strings.TrimSpace(value.Value)
		if trimmed == "" {
			continue
		}
		if value.Name != "" {
			values = append(values, value.Name+"="+trimmed)
		} else {
			values = append(values, trimmed)
		}
	}
	return strings.Join(values, "; ")
}

func parseEpoch(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0)
}

func parseMilliseconds(value string) time.Duration {
	if value == "" {
		return 0
	}
	milliseconds, err := strconv.ParseFloat(value, 64)
	if err != nil || milliseconds <= 0 {
		return 0
	}
	return time.Duration(milliseconds * float64(time.Millisecond))
}
