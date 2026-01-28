package lockdown

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents the persisted lockdown state.
type State struct {
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedBy string    `json:"updated_by,omitempty"`
}

// stateFilePath returns the path to the global lockdown state file.
func stateFilePath() string {
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return ""
		}
	}
	return filepath.Join(home, ".vectra-guard", "lockdown.json")
}

// GetState loads the current lockdown state. If the file does not exist, a
// disabled state is returned.
func GetState() (State, error) {
	path := stateFilePath()
	if path == "" {
		return State{}, fmt.Errorf("could not resolve lockdown state path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{Enabled: false}, nil
		}
		return State{}, fmt.Errorf("read lockdown state: %w", err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, fmt.Errorf("parse lockdown state: %w", err)
	}
	return st, nil
}

// SetState persists the given lockdown state.
func SetState(st State) error {
	path := stateFilePath()
	if path == "" {
		return fmt.Errorf("could not resolve lockdown state path")
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create lockdown directory: %w", err)
		}
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lockdown state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write lockdown state: %w", err)
	}
	return nil
}

