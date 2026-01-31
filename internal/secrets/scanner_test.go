package secrets

import (
	"os"
	"path/filepath"
	"strings"
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

func TestScanPathRespectsIgnorePaths(t *testing.T) {
	dir := t.TempDir()

	// File under a path we will ignore (not in shouldSkipDir so Walk visits it).
	ignoredDir := filepath.Join(dir, "external", "foo")
	if err := os.MkdirAll(ignoredDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ignoredFile := filepath.Join(ignoredDir, "bar.txt")
	secretContent := []byte("aws_access_key_id: AKIAIOSFODNN7EXAMPLE\n")
	if err := os.WriteFile(ignoredFile, secretContent, 0o644); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}

	// File at root that should be scanned.
	appFile := filepath.Join(dir, "app.py")
	if err := os.WriteFile(appFile, []byte("token: abcdEFGHijklMNOPqrstUVWX12345678\n"), 0o644); err != nil {
		t.Fatalf("write app file: %v", err)
	}

	opts := Options{IgnorePaths: []string{"external/", "external/*"}}
	findings, err := ScanPath(dir, opts)
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}

	for _, f := range findings {
		if f.File == ignoredFile {
			t.Fatalf("expected no findings from ignored path %s, got finding: %#v", ignoredFile, f)
		}
	}
	// We expect at least one finding from app.py (entropy candidate).
	foundApp := false
	for _, f := range findings {
		if f.File == appFile {
			foundApp = true
			break
		}
	}
	if !foundApp {
		var files []string
		for _, f := range findings {
			files = append(files, f.File)
		}
		t.Fatalf("expected at least one finding from %s, got %d findings from: %v", appFile, len(findings), files)
	}
}

func TestScanPathEntropyRequiresSecretContext(t *testing.T) {
	// ENTROPY_CANDIDATE only when line has secret context (token=, api_key:, etc.).
	// Paths/slugs without context should not be flagged.
	dir := t.TempDir()
	path := filepath.Join(dir, "readme.md")
	content := []byte(`See https://github.com/SomeOrg/some-repo/issues/123 and 1-eliminating-waterfalls.`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	for _, f := range findings {
		if f.PatternID == "ENTROPY_CANDIDATE" {
			t.Errorf("expected no ENTROPY_CANDIDATE when line has no secret context (path/slug only), got %q in %s", f.Match, f.File)
		}
	}
}

func TestScanPathEntropyFlagsWithSecretContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.env")
	// Use "credential" so line has secret context but no GENERIC_API_KEY match (that regex is api_key|token|secret).
	content := []byte(`credential: abcdEFGHijklMNOPqrstUVWX12345678`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	var found bool
	for _, f := range findings {
		if f.PatternID == "ENTROPY_CANDIDATE" && strings.Contains(f.Match, "abcdEFGH") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ENTROPY_CANDIDATE when line has api_key= and high-entropy value, got %d findings: %#v", len(findings), findings)
	}
}

func TestScanPathSkipsLockfiles(t *testing.T) {
	dir := t.TempDir()

	// Lockfiles are full of high-entropy hashes; we skip them by default so
	// secret counts reflect app/config, not dependency integrity blobs.
	lockFiles := []string{"package-lock.json", "yarn.lock", "poetry.lock", "cargo.lock", "other.lock"}
	for _, name := range lockFiles {
		path := filepath.Join(dir, name)
		// Content that would trigger ENTROPY_CANDIDATE in a normal file.
		content := []byte(`{"integrity":"sha512-abcdEFGHijklMNOPqrstUVWX1234567890ABCDEFGHijklMNOPqrstUVWX12345678=="}`)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings when only lockfiles present (skip lockfiles by default), got %d: %#v", len(findings), findings)
	}
}
