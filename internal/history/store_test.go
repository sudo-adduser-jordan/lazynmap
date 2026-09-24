package history

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sudo-adduser-jordan/lazynmap/internal/nmap"
)

func TestSaveLoadAndAdd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "history.json")
	store := &Store{Path: path}
	result := nmap.Result{
		FinishedAt: time.Now(),
		Scan:       nmap.Scan{Hosts: []nmap.Host{{State: "up"}}},
	}
	request := nmap.Request{Target: "localhost", Profile: nmap.Profiles[0]}
	entries := store.Add(request, result)
	if err := Save(path, entries); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Target != "localhost" || loaded[0].ProfileName != nmap.Profiles[0].Name {
		t.Fatalf("loaded entries = %#v", loaded)
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	entries, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %#v, want empty", entries)
	}
}
