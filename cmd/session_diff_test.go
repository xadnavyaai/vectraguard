package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func TestRunSessionDiffTextOutput(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	var buf bytes.Buffer
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", &buf))

	logger := logging.FromContext(ctx)

	// Use temp HOME so session storage is isolated.
	t.Setenv("HOME", filepath.Join(dir, "home"))

	mgr, err := session.NewManager(dir, logger)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	sess, err := mgr.Start("test-agent", dir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	now := time.Now()
	ops := []session.FileOperation{
		{Timestamp: now, Operation: "create", Path: filepath.Join(dir, "a.txt"), RiskLevel: "low", Allowed: true},
		{Timestamp: now, Operation: "modify", Path: filepath.Join(dir, "a.txt"), RiskLevel: "low", Allowed: true},
	}
	for _, op := range ops {
		if err := mgr.AddFileOperation(sess, op); err != nil {
			t.Fatalf("AddFileOperation: %v", err)
		}
	}

	if err := runSessionDiff(ctx, sess.ID, "", false); err != nil {
		t.Fatalf("runSessionDiff (text): %v", err)
	}
}

func TestRunSessionDiffJSONOutput(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	var buf bytes.Buffer
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", &buf))

	logger := logging.FromContext(ctx)
	t.Setenv("HOME", filepath.Join(dir, "home"))

	mgr, err := session.NewManager(dir, logger)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	sess, err := mgr.Start("test-agent", dir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	now := time.Now()
	op := session.FileOperation{
		Timestamp: now,
		Operation: "create",
		Path:      filepath.Join(dir, "a.txt"),
		RiskLevel: "low",
		Allowed:   true,
	}
	if err := mgr.AddFileOperation(sess, op); err != nil {
		t.Fatalf("AddFileOperation: %v", err)
	}

	// Capture stdout for JSON output.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	err = runSessionDiff(ctx, sess.ID, "", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runSessionDiff (json): %v", err)
	}

	out, _ := io.ReadAll(r)
	if len(out) == 0 {
		t.Fatalf("expected JSON output, got empty")
	}
}

