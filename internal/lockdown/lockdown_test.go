package lockdown

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to override HOME for tests
func withTempHome(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	prev := os.Getenv("HOME")
	if err := os.Setenv("HOME", dir); err != nil {
		t.Fatalf("setenv HOME: %v", err)
	}
	return func() {
		_ = os.Setenv("HOME", prev)
	}
}

func TestGetStateDefaultDisabled(t *testing.T) {
	restore := withTempHome(t)
	defer restore()

	st, err := GetState()
	if err != nil {
		t.Fatalf("GetState error: %v", err)
	}
	if st.Enabled {
		t.Fatalf("expected lockdown disabled by default")
	}
}

func TestSetAndGetStateRoundTrip(t *testing.T) {
	restore := withTempHome(t)
	defer restore()

	want := State{
		Enabled:   true,
		Reason:    "test",
		UpdatedBy: "unit-test",
	}
	if err := SetState(want); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	path := stateFilePath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected state file at %s: %v", path, err)
	}

	got, err := GetState()
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if !got.Enabled || got.Reason != want.Reason || got.UpdatedBy != want.UpdatedBy {
		t.Fatalf("round-trip mismatch: got %#v, want %#v", got, want)
	}
}

func TestStateFilePathUsesHome(t *testing.T) {
	restore := withTempHome(t)
	defer restore()

	path := stateFilePath()
	if path == "" {
		t.Fatalf("stateFilePath returned empty path")
	}
	if filepath.Dir(path) == "" {
		t.Fatalf("stateFilePath has no directory: %s", path)
	}
}

