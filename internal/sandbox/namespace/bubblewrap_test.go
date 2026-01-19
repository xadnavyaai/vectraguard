package namespace

import (
	"testing"
)

func TestBubblewrapConfig(t *testing.T) {
	config := BubblewrapConfig{
		Workspace:    "/tmp/test-workspace",
		CacheDir:     "/tmp/test-cache",
		AllowNetwork: false,
		BindMounts: []BindMount{
			{
				Source:   "/tmp/source",
				Target:   "/tmp/target",
				ReadOnly: true,
			},
		},
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	executor := NewBubblewrapExecutor(config)
	if executor == nil {
		t.Fatal("NewBubblewrapExecutor() returned nil")
	}

	// Test buildBubblewrapArgs
	args := executor.buildBubblewrapArgs([]string{"echo", "test"})
	if len(args) == 0 {
		t.Error("buildBubblewrapArgs() returned empty args")
	}

	// Verify some key arguments are present
	hasRoBind := false
	hasDev := false
	hasTmpfs := false
	hasUnshareAll := false
	hasCapDrop := false

	for i := 0; i < len(args); i++ {
		if args[i] == "--ro-bind" {
			hasRoBind = true
		}
		if args[i] == "--dev" {
			hasDev = true
		}
		if args[i] == "--tmpfs" {
			hasTmpfs = true
		}
		if args[i] == "--unshare-all" {
			hasUnshareAll = true
		}
		if args[i] == "--cap-drop" {
			hasCapDrop = true
		}
	}

	if !hasRoBind {
		t.Error("Missing --ro-bind argument")
	}
	if !hasDev {
		t.Error("Missing --dev argument")
	}
	if !hasTmpfs {
		t.Error("Missing --tmpfs argument")
	}
	if !hasUnshareAll {
		t.Error("Missing --unshare-all argument")
	}
	if !hasCapDrop {
		t.Error("Missing --cap-drop argument")
	}

	t.Logf("Generated %d bubblewrap arguments", len(args))
}

func TestGetDefaultCacheBinds(t *testing.T) {
	executor := NewBubblewrapExecutor(BubblewrapConfig{})
	binds := executor.getDefaultCacheBinds()

	t.Logf("Found %d default cache binds", len(binds))
	for i, bind := range binds {
		t.Logf("  Bind %d: %s -> %s (ro=%v)", i, bind.Source, bind.Target, bind.ReadOnly)
	}
}

func TestIsBubblewrapAvailable(t *testing.T) {
	available := IsBubblewrapAvailable()
	t.Logf("Bubblewrap available: %v", available)

	if available {
		version, err := GetVersion()
		if err != nil {
			t.Errorf("GetVersion() error = %v", err)
		} else {
			t.Logf("Bubblewrap version: %s", version)
		}
	}
}
