package cve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverPackagesFindsManifests(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "package.json"), `{
  "dependencies": {
    "left-pad": "1.3.0",
    "lodash": "^4.17.0"
  }
}`)
	writeFile(t, filepath.Join(dir, "package-lock.json"), `{
  "name": "demo",
  "version": "1.0.0",
  "lockfileVersion": 2,
  "packages": {
    "": {"name": "demo", "version": "1.0.0"},
    "node_modules/left-pad": {"name": "left-pad", "version": "1.3.0"},
    "node_modules/lodash": {"version": "4.17.21"}
  },
  "dependencies": {
    "left-pad": {"version": "1.3.0"},
    "lodash": {"version": "4.17.21"}
  }
}`)
	writeFile(t, filepath.Join(dir, "go.mod"), `module example.com/demo

go 1.20

require github.com/pkg/errors v0.9.1
`)

	packages, warnings, err := DiscoverPackages(dir)
	if err != nil {
		t.Fatalf("discover packages: %v", err)
	}
	if len(packages) == 0 {
		t.Fatalf("expected packages to be detected")
	}

	want := map[string]bool{
		"npm|left-pad|1.3.0":              true,
		"npm|lodash|4.17.21":              true,
		"go|github.com/pkg/errors|v0.9.1": true,
	}

	for _, pkg := range packages {
		key := strings.ToLower(pkg.Key())
		if _, ok := want[key]; ok {
			want[key] = false
		}
	}

	for key, missing := range want {
		if missing {
			t.Fatalf("expected package %s to be detected", key)
		}
	}

	foundWarning := false
	for _, warn := range warnings {
		if strings.Contains(warn, "non-exact version") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("expected non-exact version warning")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
