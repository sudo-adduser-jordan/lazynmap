package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sudo-adduser-jordan/lazynmap/internal/nmap"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return NewModel()
}

func TestModelRendersAndHandlesProfileCycle(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 36}, {Width: 80, Height: 24}, {Width: 60, Height: 20}} {
		model := newTestModel(t)
		updated, _ := model.Update(size)
		modelPtr := updated.(*Model)
		model = *modelPtr
		view := model.View()
		if strings.TrimSpace(view) == "" {
			t.Fatalf("View returned an empty screen at %dx%d", size.Width, size.Height)
		}
		if got := lipgloss.Height(view); got > size.Height {
			t.Fatalf("view height = %d, want <= %d at %dx%d", got, size.Height, size.Width, size.Height)
		}
		if got := lipgloss.Width(view); got > size.Width {
			t.Fatalf("view width = %d, want <= %d at %dx%d", got, size.Width, size.Width, size.Height)
		}
	}

	model := newTestModel(t)
	model.setFocus(focusHosts)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	modelPtr := updated.(*Model)
	model = *modelPtr
	if model.profileIndex != 1 {
		t.Fatalf("profile index = %d, want 1", model.profileIndex)
	}
}

func TestInputTabMovesBetweenPanes(t *testing.T) {
	model := newTestModel(t)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = *(updated.(*Model))
	if model.focus != focusPorts {
		t.Fatalf("focus after tab = %v, want ports", model.focus)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = *(updated.(*Model))
	if model.focus != focusTarget {
		t.Fatalf("focus after shift+tab = %v, want target", model.focus)
	}
}

func TestScanResultPopulatesTablesAndFilter(t *testing.T) {
	model := newTestModel(t)
	result := nmap.Result{
		StartedAt:  time.Now().Add(-time.Second),
		FinishedAt: time.Now(),
		Duration:   time.Second,
		Scan: nmap.Scan{
			Hosts: []nmap.Host{{
				State:     "up",
				Addresses: []nmap.Address{{Type: "ipv4", Address: "127.0.0.1"}},
				Ports:     []nmap.Port{{Number: 22, Protocol: "tcp", State: "open", Service: "ssh"}},
			}},
		},
	}
	updated, _ := model.Update(scanFinishedMsg{result: result})
	modelPtr := updated.(*Model)
	model = *modelPtr
	if len(model.visible) != 1 || len(model.visiblePorts) != 1 {
		t.Fatalf("visible results = %d hosts / %d ports", len(model.visible), len(model.visiblePorts))
	}
	if strings.TrimSpace(model.View()) == "" {
		t.Fatal("result view is empty")
	}

	model.filter.SetValue("http")
	model.rebuildTables()
	if len(model.visible) != 0 {
		t.Fatalf("filter matched %d hosts, want 0", len(model.visible))
	}
	model.filter.SetValue("ssh")
	model.rebuildTables()
	if len(model.visible) != 1 || len(model.visiblePorts) != 1 {
		t.Fatalf("service filter matched %d hosts / %d ports", len(model.visible), len(model.visiblePorts))
	}
}
