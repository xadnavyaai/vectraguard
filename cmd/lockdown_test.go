package cmd

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/lockdown"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// helper to override HOME for lockdown tests so we don't touch the real state
func withTempHomeCmd(t *testing.T) func() {
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

func TestRunLockdownEnableDisableStatus(t *testing.T) {
	restore := withTempHomeCmd(t)
	defer restore()

	var buf bytes.Buffer
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", &buf))

	// Initially disabled
	if err := runLockdownStatus(ctx); err != nil {
		t.Fatalf("status (initial): %v", err)
	}

	// Enable
	if err := runLockdownEnable(ctx, "unit test"); err != nil {
		t.Fatalf("enable: %v", err)
	}
	st, err := lockdown.GetState()
	if err != nil {
		t.Fatalf("GetState after enable: %v", err)
	}
	if !st.Enabled {
		t.Fatalf("expected lockdown enabled after enable")
	}

	// Disable
	if err := runLockdownDisable(ctx); err != nil {
		t.Fatalf("disable: %v", err)
	}
	st, err = lockdown.GetState()
	if err != nil {
		t.Fatalf("GetState after disable: %v", err)
	}
	if st.Enabled {
		t.Fatalf("expected lockdown disabled after disable")
	}
}

