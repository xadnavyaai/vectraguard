package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunScanSecretsDetectsFindingAndReturnsExitError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	content := []byte(`
aws_access_key_id: AKIAIOSFODNN7EXAMPLE
`)
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runScanSecrets(ctx, dir, "")
	if err == nil {
		t.Fatalf("expected exit error when secrets are detected")
	}
	if exitErr, ok := err.(*exitError); !ok || exitErr.code != 2 {
		t.Fatalf("expected exit error code 2, got %#v", err)
	}
}

func TestRunScanSecretsSucceedsWhenClean(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	content := []byte(`
name: test
description: safe content only
`)
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	if err := runScanSecrets(ctx, dir, ""); err != nil {
		t.Fatalf("expected no error for clean scan, got %v", err)
	}
}

func TestRunScanSecretsRespectsAllowlistFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	secret := "AKIAIOSFODNN7EXAMPLE"
	content := []byte(`
aws_access_key_id: ` + secret + `
`)
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("# known test key\n"+secret+"\n"), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	if err := runScanSecrets(ctx, dir, allowlistPath); err != nil {
		t.Fatalf("expected no error when all matches are allowlisted, got %v", err)
	}
}
