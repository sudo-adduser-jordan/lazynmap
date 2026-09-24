# lazynmap

A small, keyboard-driven terminal UI for [nmap](https://nmap.org/), built with
Go and [Charm](https://charm.sh/).

`lazynmap` keeps the workflow focused on the terminal: enter a target, choose a
scan profile, watch the scan status, and inspect hosts, ports, services, and
details without leaving the TUI.

## Features

- Bubble Tea / Lip Gloss terminal interface
- Safe argument construction without a shell
- Nmap XML parsing instead of scraping human-readable output
- Quick TCP, service/version, host discovery, full TCP, and SYN profiles
- Custom port specifications such as `22,80,443` or `1-1024`
- Host and port tables with status colors and detail inspection
- Text filtering across addresses, states, services, and versions
- Cancelable scans with elapsed-time feedback
- Local scan history stored as JSON
- JSON export of the selected result
- Responsive resizing and a built-in help screen

## Requirements

- Go 1.23 or newer
- `nmap` installed and available on `PATH`

Most Linux distributions provide nmap through their package manager, for
example:

```sh
sudo apt install nmap
sudo pacman -S nmap
```

Some scan types require elevated privileges. The default **Quick TCP** and
**Service scan** profiles use `-sT`, which can run without root. The **SYN
scan** profile uses `-sS` and requires root or equivalent capabilities;
lazynmap does not invoke `sudo` automatically.

## Run

```sh
go run .
```

Or build a binary:

```sh
go build -o lazynmap .
./lazynmap
```

The target is entered inside the application so that it is always visible
before a scan starts. Press `?` for the complete keybinding list.

## Keys

| Key | Action |
|---|---|
| `enter` | Run the configured scan |
| `r` | Focus the target input |
| `p` | Cycle scan profiles |
| `tab` / `shift+tab` | Move between panes |
| `j` / `k`, arrows | Navigate hosts, ports, or history |
| `/` | Filter results |
| `x` or `esc` | Cancel an active scan |
| `e` | Export the selected result as JSON |
| `?` | Toggle help |
| `q` | Quit |

Letter shortcuts are active while navigating panes. While editing the target,
ports, or filter, use `tab`/`shift+tab` or `esc` to leave the input; `ctrl+c`
always quits.

## Safety

Only scan systems and networks you own or have explicit permission to test.
The application passes target and port values as individual process arguments,
but it cannot determine whether a target is authorized. Review the command
preview before running a scan.

## Development

```sh
gofmt -w main.go internal
go test ./...
go vet ./...
```

The project is intentionally small and has no daemon or background service. A
scan is started only after an explicit `enter`, runs in a cancellable child
process, and stores its normalized result in the local history file when it
completes.
