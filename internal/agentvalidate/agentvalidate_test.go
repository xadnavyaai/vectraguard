package agentvalidate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
)

func TestValidatePathSingleFile(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "danger.sh")
	if err := os.WriteFile(script, []byte("rm -rf /\n"), 0o744); err != nil {
		t.Fatalf("write script: %v", err)
	}

	results, err := ValidatePath(script, config.DefaultConfig().Policies)
	if err != nil {
		t.Fatalf("ValidatePath error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != script {
		t.Errorf("expected path %s, got %s", script, results[0].Path)
	}
	if len(results[0].Findings) == 0 {
		t.Fatalf("expected findings for dangerous script")
	}
}

func TestValidatePathDirectoryFiltersScripts(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "danger.sh")
	other := filepath.Join(dir, "README.md")

	if err := os.WriteFile(script, []byte("rm -rf /\n"), 0o744); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.WriteFile(other, []byte("not a script\n"), 0o644); err != nil {
		t.Fatalf("write other: %v", err)
	}

	results, err := ValidatePath(dir, config.DefaultConfig().Policies)
	if err != nil {
		t.Fatalf("ValidatePath error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (only scripts), got %d", len(results))
	}
	if results[0].Path != script {
		t.Errorf("expected result for %s, got %s", script, results[0].Path)
	}
}

