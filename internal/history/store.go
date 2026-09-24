package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sudo-adduser-jordan/lazynmap/internal/nmap"
)

const maxEntries = 30

// Entry is a persisted scan result shown in the TUI history pane.
type Entry struct {
	ID          string      `json:"id"`
	Target      string      `json:"target"`
	ProfileID   string      `json:"profile_id"`
	ProfileName string      `json:"profile_name"`
	FinishedAt  time.Time   `json:"finished_at"`
	Result      nmap.Result `json:"result"`
}

// Store persists a bounded scan history as JSON.
type Store struct {
	Path    string
	Entries []Entry
}

// NewStore loads the user's history from the XDG config directory.
func NewStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	entries, err := Load(path)
	if err != nil {
		// Keep the path available so a later completed scan can repair a
		// missing or malformed history file.
		return &Store{Path: path}, err
	}
	return &Store{Path: path, Entries: entries}, nil
}

// DefaultPath returns the lazynmap history file path.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(configDir, "lazynmap", "history.json"), nil
}

// Load reads a history file. A missing file is treated as an empty history.
func Load(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read scan history: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode scan history: %w", err)
	}
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	return entries, nil
}

// Save writes entries atomically, creating the parent directory when needed.
func Save(path string, entries []Entry) error {
	if path == "" {
		return nil
	}
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode scan history: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(filepath.Dir(path), ".history-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary history file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect temporary history file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write scan history: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close scan history: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace scan history: %w", err)
	}
	return nil
}

// Add prepends a result to the history and returns a bounded copy.
func (s *Store) Add(request nmap.Request, result nmap.Result) []Entry {
	profileName := request.Profile.Name
	if profileName == "" {
		profileName = "Custom"
	}
	entry := Entry{
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
		Target:      request.Target,
		ProfileID:   request.Profile.ID,
		ProfileName: profileName,
		FinishedAt:  result.FinishedAt,
		Result:      result,
	}
	entries := make([]Entry, 0, min(maxEntries, len(s.Entries)+1))
	entries = append(entries, entry)
	entries = append(entries, s.Entries...)
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	return entries
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
