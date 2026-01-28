package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanPathDetectsAwsAccessKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := []byte(`
aws_access_key_id: AKIAIOSFODNN7EXAMPLE
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding, got 0")
	}

	foundAWS := false
	for _, f := range findings {
		if f.PatternID == "AWS_ACCESS_KEY_ID" {
			foundAWS = true
			if f.File != path {
				t.Errorf("expected file %s, got %s", path, f.File)
			}
			if f.Line <= 0 {
				t.Errorf("expected positive line number, got %d", f.Line)
			}
			if f.Entropy <= 0 {
				t.Errorf("expected positive entropy, got %f", f.Entropy)
			}
		}
	}
	if !foundAWS {
		t.Fatalf("expected AWS_ACCESS_KEY_ID finding, got %#v", findings)
	}
}

func TestScanPathDetectsEntropyCandidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "random.txt")

	// 32+ random-like chars, not matching a specific regex
	content := []byte(`
token: abcdEFGHijklMNOPqrstUVWX12345678
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding, got 0")
	}

	foundEntropy := false
	for _, f := range findings {
		if f.PatternID == "ENTROPY_CANDIDATE" {
			foundEntropy = true
			if f.Entropy < 3.5 {
				t.Errorf("expected high entropy >= 3.5, got %f", f.Entropy)
			}
		}
	}
	if !foundEntropy {
		t.Fatalf("expected ENTROPY_CANDIDATE finding, got %#v", findings)
	}
}

func TestScanPathRespectsAllowlist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	secret := "AKIAIOSFODNN7EXAMPLE"
	content := []byte(`
aws_access_key_id: ` + secret + `
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	allow := map[string]struct{}{secret: {}}
	findings, err := ScanPath(dir, Options{Allowlist: allow})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings due to allowlist, got %#v", findings)
	}
}

