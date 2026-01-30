package secscan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanPathGoPythonC(t *testing.T) {
	dir := t.TempDir()

	goFile := filepath.Join(dir, "danger.go")
	pyFile := filepath.Join(dir, "danger.py")
	cFile := filepath.Join(dir, "danger.c")

	if err := os.WriteFile(goFile, []byte(`
package main
import "os/exec"
func main() {
  exec.Command("sh", "-c", "rm -rf /")
}
`), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	if err := os.WriteFile(pyFile, []byte(`
import os, subprocess, requests
eval("print('danger')")
subprocess.run("rm -rf /", shell=True)
requests.get("http://example.com")
`), 0o644); err != nil {
		t.Fatalf("write py file: %v", err)
	}

	if err := os.WriteFile(cFile, []byte(`
#include <stdio.h>
#include <stdlib.h>
int main() {
  gets(NULL);
  system("rm -rf /");
}
`), 0o644); err != nil {
		t.Fatalf("write c file: %v", err)
	}

	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected findings, got none")
	}

	hasGo := false
	hasPy := false
	hasC := false
	for _, f := range findings {
		switch f.Language {
		case "go":
			hasGo = true
		case "python":
			hasPy = true
		case "c":
			hasC = true
		}
	}
	if !hasGo || !hasPy || !hasC {
		t.Fatalf("expected findings for go, python, and c, got: go=%v python=%v c=%v", hasGo, hasPy, hasC)
	}
}

func TestScanPathLanguageFilter(t *testing.T) {
	dir := t.TempDir()

	goFile := filepath.Join(dir, "onlygo.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "os/exec"
func main() { exec.Command("sh", "-c", "rm -rf /") }
`), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	pyFile := filepath.Join(dir, "ignored.py")
	if err := os.WriteFile(pyFile, []byte(`eval("print('hi')")`), 0o644); err != nil {
		t.Fatalf("write py file: %v", err)
	}

	opts := Options{
		Languages: map[string]bool{"go": true}, // only Go enabled
	}
	findings, err := ScanPath(dir, opts)
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected findings for go-only scan")
	}
	for _, f := range findings {
		if f.Language != "go" {
			t.Fatalf("expected only go findings, got language=%s in %#v", f.Language, f)
		}
	}
}

func TestScanPathDetectsExternalHTTP(t *testing.T) {
	dir := t.TempDir()

	pyFile := filepath.Join(dir, "api.py")
	if err := os.WriteFile(pyFile, []byte(`
url = "https://api.example.com/call"
requests.get(url)
`), 0o644); err != nil {
		t.Fatalf("write py file: %v", err)
	}

	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	var found bool
	for _, f := range findings {
		if f.Code == "PY_EXTERNAL_HTTP" {
			found = true
			if f.Severity != "medium" {
				t.Errorf("expected severity medium, got %s", f.Severity)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected PY_EXTERNAL_HTTP finding for https://api.example.com, got: %#v", findings)
	}
}

func TestScanPathSkipsCommentOnlyLines(t *testing.T) {
	dir := t.TempDir()
	pyFile := filepath.Join(dir, "example.py")
	// Comment-only lines should not produce findings (reduce FPs from doc URLs, etc.).
	content := []byte(`# See https://api.example.com/docs
# host = "0.0.0.0"
url = "https://real-call.com"  # inline comment
`)
	if err := os.WriteFile(pyFile, content, 0o644); err != nil {
		t.Fatalf("write py file: %v", err)
	}
	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	// Should have exactly one PY_EXTERNAL_HTTP (the real-call.com line), not from comment lines.
	var externalCount int
	for _, f := range findings {
		if f.Code == "PY_EXTERNAL_HTTP" {
			externalCount++
		}
	}
	if externalCount != 1 {
		t.Errorf("expected 1 PY_EXTERNAL_HTTP (non-comment line only), got %d: %#v", externalCount, findings)
	}
}

func TestScanPathDetectsBindAllInterfaces(t *testing.T) {
	dir := t.TempDir()

	pyFile := filepath.Join(dir, "server.py")
	if err := os.WriteFile(pyFile, []byte(`
app.run(host="0.0.0.0", port=8080)
`), 0o644); err != nil {
		t.Fatalf("write py file: %v", err)
	}

	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath error: %v", err)
	}
	var found bool
	for _, f := range findings {
		if f.Code == "BIND_ALL_INTERFACES" {
			found = true
			if f.Severity != "medium" {
				t.Errorf("expected severity medium, got %s", f.Severity)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected BIND_ALL_INTERFACES finding for 0.0.0.0:8080, got: %#v", findings)
	}
}

func TestScanPathSingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "single.go")
	if err := os.WriteFile(f, []byte(`package main
import "os/exec"
func main() { exec.Command("ls") }
`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	findings, err := ScanPath(f, Options{})
	if err != nil {
		t.Fatalf("ScanPath single file: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding for single file")
	}
	for _, fnd := range findings {
		if fnd.File != f {
			t.Errorf("finding file should be %q, got %q", f, fnd.File)
		}
	}
}

func TestScanPathNonExistent(t *testing.T) {
	_, err := ScanPath(filepath.Join(t.TempDir(), "nonexistent"), Options{})
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
	if !strings.Contains(err.Error(), "stat") {
		t.Errorf("error should mention stat, got: %v", err)
	}
}

func TestScanPathSkipsVendorAndGit(t *testing.T) {
	dir := t.TempDir()
	mainGo := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainGo, []byte(`package main
func main() { println("ok") }
`), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	vendorDir := filepath.Join(dir, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatalf("mkdir vendor: %v", err)
	}
	vendorGo := filepath.Join(vendorDir, "x.go")
	if err := os.WriteFile(vendorGo, []byte(`package vendor
import "os/exec"
func X() { exec.Command("sh", "-c", "rm -rf /") }
`), 0o644); err != nil {
		t.Fatalf("write vendor file: %v", err)
	}
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	gitGo := filepath.Join(gitDir, "hack.go")
	if err := os.WriteFile(gitGo, []byte(`package git
import "os/exec"
func H() { exec.Command("rm", "-rf", "/") }
`), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	// main.go is safe; vendor and .git should be skipped, so no findings.
	if len(findings) != 0 {
		t.Fatalf("expected no findings (vendor and .git skipped); got %d: %#v", len(findings), findings)
	}
}

func TestLocalhostDoesNotTriggerExternalHTTP(t *testing.T) {
	dir := t.TempDir()
	pyFile := filepath.Join(dir, "local.py")
	content := `
url = "http://127.0.0.1:8080/health"
url2 = "https://localhost/api"
url3 = "http://[::1]:3000"
requests.get(url)
requests.get(url2)
requests.get(url3)
`
	if err := os.WriteFile(pyFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write py file: %v", err)
	}
	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	for _, f := range findings {
		if f.Code == "PY_EXTERNAL_HTTP" || f.Code == "GO_EXTERNAL_HTTP" {
			t.Errorf("localhost/127.0.0.1/::1 should not produce external HTTP finding, got %s", f.Code)
		}
	}
}

func TestScanPathNilLanguagesDefaultsToAll(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "a.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "os/exec"
func main() { exec.Command("ls") }
`), 0o644); err != nil {
		t.Fatalf("write go: %v", err)
	}
	findings, err := ScanPath(dir, Options{}) // nil Languages
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected findings when Languages is nil (defaults to go,python,c)")
	}
}

func TestFixtureE2E(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "fixture")
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("fixture dir not found: %v", err)
	}

	findings, err := ScanPath(fixtureDir, Options{})
	if err != nil {
		t.Fatalf("ScanPath fixture: %v", err)
	}

	// Expected rule codes from risky.go, risky.py, risky.c (vendor/skipped.go must be skipped).
	wantCodes := map[string]bool{
		"GO_EXEC_COMMAND": true, "GO_DANGEROUS_SHELL": true, "GO_NET_HTTP": true,
		"GO_ENV_READ": true, "GO_SYSTEM_WRITE": true, "GO_EXTERNAL_HTTP": true, "BIND_ALL_INTERFACES": true,
		"PY_EVAL": true, "PY_EXEC": true, "PY_SUBPROCESS": true, "PY_REMOTE_HTTP": true,
		"PY_ENV_ACCESS": true, "PY_EXTERNAL_HTTP": true,
		"C_GETS": true, "C_SHELL_EXEC": true, "C_UNSAFE_STRING": true, "C_MEMCPY": true,
		"C_RAW_SOCKET": true,
	}
	gotCodes := make(map[string]bool)
	for _, f := range findings {
		gotCodes[f.Code] = true
		// Fixture must not include vendor.
		if strings.Contains(f.File, "vendor") {
			t.Errorf("findings must not include vendor/skipped.go, got file %s", f.File)
		}
	}
	for code := range wantCodes {
		if !gotCodes[code] {
			t.Errorf("fixture e2e: missing expected finding code %q; got %v", code, gotCodes)
		}
	}
}

func TestScanPathDetectsGoRuleCodes(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantCode string
	}{
		{"exec", `exec.Command("ls")`, "GO_EXEC_COMMAND"},
		{"danger_shell", `run("rm -rf /")`, "GO_DANGEROUS_SHELL"},
		{"net_http", `"net/http"`, "GO_NET_HTTP"},
		{"env_read", `os.Getenv("X")`, "GO_ENV_READ"},
		{"system_write", `WriteFile("/etc/x", nil, 0)`, "GO_SYSTEM_WRITE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			f := filepath.Join(dir, "x.go")
			body := "package main\nfunc main() { " + tt.content + " }\n"
			if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			findings, err := ScanPath(dir, Options{Languages: map[string]bool{"go": true}})
			if err != nil {
				t.Fatalf("ScanPath: %v", err)
			}
			var found bool
			for _, fnd := range findings {
				if fnd.Code == tt.wantCode {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected code %q, got findings: %#v", tt.wantCode, findings)
			}
		})
	}
}

func TestScanPathDetectsPythonRuleCodes(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantCode string
	}{
		{"eval", `eval("1")`, "PY_EVAL"},
		{"exec", `exec("x=1")`, "PY_EXEC"},
		{"subprocess", `subprocess.run("id")`, "PY_SUBPROCESS"},
		{"requests", `requests.get("http://x.com")`, "PY_REMOTE_HTTP"},
		{"env", `os.environ["X"]`, "PY_ENV_ACCESS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			f := filepath.Join(dir, "x.py")
			if err := os.WriteFile(f, []byte(tt.content+"\n"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			findings, err := ScanPath(dir, Options{Languages: map[string]bool{"python": true}})
			if err != nil {
				t.Fatalf("ScanPath: %v", err)
			}
			var found bool
			for _, fnd := range findings {
				if fnd.Code == tt.wantCode {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected code %q, got findings: %#v", tt.wantCode, findings)
			}
		})
	}
}

func TestScanPathDetectsCRuleCodes(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantCode string
	}{
		{"gets", `gets(buf);`, "C_GETS"},
		{"system", `system("id");`, "C_SHELL_EXEC"},
		{"strcpy", `strcpy(a,b);`, "C_UNSAFE_STRING"},
		{"memcpy", `memcpy(d,s,n);`, "C_MEMCPY"},
		{"socket", `socket(AF_INET, SOCK_STREAM, 0);`, "C_RAW_SOCKET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			f := filepath.Join(dir, "x.c")
			body := "#include <stdio.h>\nint main() { " + tt.content + " return 0; }\n"
			if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			findings, err := ScanPath(dir, Options{Languages: map[string]bool{"c": true}})
			if err != nil {
				t.Fatalf("ScanPath: %v", err)
			}
			var found bool
			for _, fnd := range findings {
				if fnd.Code == tt.wantCode {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected code %q, got findings: %#v", tt.wantCode, findings)
			}
		})
	}
}

func TestScanPathDetectsConfigRuleCodes(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		ext      string
		wantCode string
	}{
		{"bind", `host: "0.0.0.0"`, ".yaml", "BIND_ALL_INTERFACES"},
		{"trust_proxy", `trustProxy: true`, ".yml", "LOCALHOST_TRUST_PROXY"},
		{"auth_false", `auth: false`, ".json", "UNAUTHENTICATED_ACCESS"},
		{"secure_disabled", `secure: false`, ".yaml", "UNAUTHENTICATED_ACCESS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			f := filepath.Join(dir, "app"+tt.ext)
			if err := os.WriteFile(f, []byte(tt.content+"\n"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			findings, err := ScanPath(dir, Options{Languages: map[string]bool{"config": true}})
			if err != nil {
				t.Fatalf("ScanPath: %v", err)
			}
			var found bool
			for _, fnd := range findings {
				if fnd.Code == tt.wantCode && fnd.Language == "config" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected config finding code %q, got findings: %#v", tt.wantCode, findings)
			}
		})
	}
}
