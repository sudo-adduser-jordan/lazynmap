package nmap

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const sampleNmapXML = `<?xml version="1.0" encoding="UTF-8"?>
<nmaprun scanner="nmap" args="nmap -sT -p 22,80 example.test" start="1700000000" startstr="Tue Nov 14 22:13:20 2023" version="7.94" xmloutputversion="1.05">
  <host>
    <status state="up" reason="echo-reply" reason_ttl="54"/>
    <address addr="192.0.2.10" addrtype="ipv4"/>
    <hostnames>
      <hostname name="example.test" type="user"/>
    </hostnames>
    <ports>
      <port protocol="tcp" portid="22">
        <state state="open" reason="syn-ack" reason_ttl="54"/>
        <service name="ssh" product="OpenSSH" version="9.6p1" confidence="10"/>
        <extra name="banner">SSH-2.0-OpenSSH_9.6p1</extra>
      </port>
      <port protocol="tcp" portid="80">
        <state state="closed" reason="conn-refused" reason_ttl="0"/>
        <service name="http" confidence="10"/>
      </port>
    </ports>
    <times srtt="123.4" rttvar="100.0" to="100000"/>
    <os>
      <osmatch name="Linux 5.X" accuracy="95" line="123"/>
    </os>
  </host>
  <runstats>
    <finished time="1700000010" elapsed="10.00" exit="success" summary="Nmap done"/>
    <hosts up="1" down="0" total="1"/>
  </runstats>
</nmaprun>`

func TestBuildArgs(t *testing.T) {
	request := Request{
		Target:  "example.test",
		Profile: Profiles[0],
		Ports:   "22,80,443",
	}
	args, err := BuildArgs(request, "/tmp/scan.xml")
	if err != nil {
		t.Fatalf("BuildArgs returned error: %v", err)
	}
	want := []string{"-oX", "/tmp/scan.xml", "-sT", "-T4", "--top-ports", "1000", "-p", "22,80,443", "example.test"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestBuildArgsCustomPortsReplaceFullProfile(t *testing.T) {
	args, err := BuildArgs(Request{Target: "localhost", Profile: Profiles[3], Ports: "22,443"}, "")
	if err != nil {
		t.Fatalf("BuildArgs returned error: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-p-") || !strings.Contains(joined, "-p 22,443") {
		t.Fatalf("full profile ports were not replaced: %v", args)
	}
}

func TestBuildArgsRejectsOptionLikeTarget(t *testing.T) {
	_, err := BuildArgs(Request{Target: "--script=malicious", Profile: Profiles[0]}, "")
	if err == nil || !strings.Contains(err.Error(), "must not start") {
		t.Fatalf("expected target validation error, got %v", err)
	}
}

func TestParseXML(t *testing.T) {
	scan, err := ParseXML(strings.NewReader(sampleNmapXML))
	if err != nil {
		t.Fatalf("ParseXML returned error: %v", err)
	}
	if scan.Scanner != "nmap" {
		t.Fatalf("scanner = %q, want nmap", scan.Scanner)
	}
	if scan.Duration != 10*time.Second {
		t.Fatalf("duration = %s, want 10s", scan.Duration)
	}
	if scan.Stats != (Stats{HostsUp: 1, HostsDown: 0, HostsTotal: 1}) {
		t.Fatalf("stats = %#v", scan.Stats)
	}
	if len(scan.Hosts) != 1 {
		t.Fatalf("hosts = %d, want 1", len(scan.Hosts))
	}
	host := scan.Hosts[0]
	if host.PrimaryAddress() != "192.0.2.10" || host.DisplayName() != "example.test" {
		t.Fatalf("host identity = %q / %q", host.PrimaryAddress(), host.DisplayName())
	}
	if host.OpenPortCount() != 1 {
		t.Fatalf("open ports = %d, want 1", host.OpenPortCount())
	}
	if host.Latency != 123400*time.Microsecond {
		t.Fatalf("latency = %s, want 123.4ms", host.Latency)
	}
	if host.OS != "Linux 5.X" || host.OSAccuracy != 95 {
		t.Fatalf("OS = %q (%d%%)", host.OS, host.OSAccuracy)
	}
	if len(host.Ports) != 2 || host.Ports[0].Number != 22 || host.Ports[1].Number != 80 {
		t.Fatalf("ports were not normalized/sorted: %#v", host.Ports)
	}
	if host.Ports[0].Service != "ssh" || host.Ports[0].Version != "9.6p1" {
		t.Fatalf("service data missing: %#v", host.Ports[0])
	}
}

func TestRunnerUsesXMLOutput(t *testing.T) {
	directory := t.TempDir()
	binary := filepath.Join(directory, "fake-nmap")
	script := `#!/bin/sh
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-oX" ]; then
    output="$2"
    shift 2
  else
    shift
  fi
done
cat > "$output" <<'XML'
` + sampleNmapXML + `
XML
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	runner := Runner{Binary: binary}
	result, err := runner.Run(context.Background(), Request{Target: "example.test", Profile: Profiles[0]})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Scan.Hosts) != 1 {
		t.Fatalf("parsed hosts = %d, want 1", len(result.Scan.Hosts))
	}
	if len(result.Command) < 2 || result.Command[0] != binary {
		t.Fatalf("command = %#v", result.Command)
	}
	if result.Duration != 10*time.Second {
		t.Fatalf("result duration = %s, want 10s", result.Duration)
	}

	// The runner must not leave its temporary XML file behind.
	if len(result.Command) > 2 {
		outputPath := result.Command[2]
		if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
			t.Fatalf("temporary output still exists or stat failed unexpectedly: %v", err)
		}
	}
}
