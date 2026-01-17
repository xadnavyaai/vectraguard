package daemon

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestCommandApprovalDeniesDenylist(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	t.Setenv("VECTRAGUARD_BYPASS", "")

	logger := logging.NewLogger("text", io.Discard)
	cfg := config.DefaultConfig()
	cfg.GuardLevel.Level = config.GuardLevelMedium

	daemon, err := New(workspace, "agent", cfg, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	sess, err := daemon.sessionMgr.Start("agent", workspace)
	if err != nil {
		t.Fatalf("Start session error = %v", err)
	}
	daemon.session = sess

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go daemon.processCommands(ctx)

	approved := daemon.InterceptCommand("rm", []string{"-rf", "/"})
	if approved {
		t.Fatal("expected denylist command to be rejected")
	}

	if len(daemon.session.Commands) != 1 {
		t.Fatalf("expected 1 command recorded, got %d", len(daemon.session.Commands))
	}

	cmd := daemon.session.Commands[0]
	if cmd.Approved {
		t.Errorf("expected command to be unapproved")
	}
	if cmd.RiskLevel != "critical" {
		t.Errorf("expected critical risk level, got %s", cmd.RiskLevel)
	}
	if len(cmd.Findings) == 0 {
		t.Error("expected findings to be recorded")
	}
}

func TestCommandApprovalRespectsGuardLevelOff(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	t.Setenv("VECTRAGUARD_BYPASS", "")

	logger := logging.NewLogger("text", io.Discard)
	cfg := config.DefaultConfig()
	cfg.GuardLevel.Level = config.GuardLevelOff

	daemon, err := New(workspace, "agent", cfg, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	sess, err := daemon.sessionMgr.Start("agent", workspace)
	if err != nil {
		t.Fatalf("Start session error = %v", err)
	}
	daemon.session = sess

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go daemon.processCommands(ctx)

	approved := daemon.InterceptCommand("rm", []string{"-rf", "/"})
	if !approved {
		t.Fatal("expected guard level off to allow commands")
	}
}

func TestCommandApprovalAllowsAllowlist(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	t.Setenv("VECTRAGUARD_BYPASS", "")

	logger := logging.NewLogger("text", io.Discard)
	cfg := config.DefaultConfig()
	cfg.GuardLevel.Level = config.GuardLevelHigh
	cfg.Policies.Allowlist = []string{"echo"}

	daemon, err := New(workspace, "agent", cfg, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	sess, err := daemon.sessionMgr.Start("agent", workspace)
	if err != nil {
		t.Fatalf("Start session error = %v", err)
	}
	daemon.session = sess

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go daemon.processCommands(ctx)

	approved := daemon.InterceptCommand("echo", []string{"ok"})
	if !approved {
		t.Fatal("expected allowlisted command to be approved")
	}
}
