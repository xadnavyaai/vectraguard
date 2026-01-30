package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunValidateAgentDirectory(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "danger.sh")
	if err := os.WriteFile(script, []byte("rm -rf /\n"), 0o744); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runValidateAgent(ctx, dir)
	if err == nil {
		t.Fatalf("expected exit error for validation issues")
	}
	if exitErr, ok := err.(*exitError); !ok || exitErr.code != 2 {
		t.Fatalf("expected exit error code 2, got %#v", err)
	}
}

func TestRunValidateAgentCleanDirectory(t *testing.T) {
	dir := t.TempDir()
	// Non-script file should be ignored.
	other := filepath.Join(dir, "README.md")
	if err := os.WriteFile(other, []byte("no scripts here\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	if err := runValidateAgent(ctx, dir); err != nil {
		t.Fatalf("expected no validation issues, got %v", err)
	}
}
