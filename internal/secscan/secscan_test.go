package secscan

import (
	"os"
	"path/filepath"
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


