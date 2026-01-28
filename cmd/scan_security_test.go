package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunScanSecurityFindsIssues(t *testing.T) {
	dir := t.TempDir()

	goFile := filepath.Join(dir, "danger.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "os/exec"
func main() { exec.Command("sh", "-c", "rm -rf /") }
`), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runScanSecurity(ctx, dir, "go")
	if err == nil {
		t.Fatalf("expected exit error for security findings")
	}
	if exitErr, ok := err.(*exitError); !ok || exitErr.code != 2 {
		t.Fatalf("expected exit error code 2, got %#v", err)
	}
}

func TestRunScanSecurityNoIssues(t *testing.T) {
	dir := t.TempDir()

	// A harmless Go file that should not trigger any rules.
	goFile := filepath.Join(dir, "safe.go")
	if err := os.WriteFile(goFile, []byte(`package main
func main() { println("ok") }
`), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	if err := runScanSecurity(ctx, dir, "go"); err != nil {
		t.Fatalf("expected no error for safe code, got %v", err)
	}
}


