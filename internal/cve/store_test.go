package cve

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	store, err := LoadStore(path)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}

	ref := PackageRef{Ecosystem: "npm", Name: "left-pad", Version: "1.3.0"}
	entry := PackageVuln{
		Package:     ref,
		RetrievedAt: time.Now().UTC(),
		Vulnerabilities: []Vulnerability{
			{ID: "CVE-2025-0001", Summary: "Test", Severity: "high", CVSS: 7.5},
		},
	}
	store.Set(entry)

	if err := store.Save(); err != nil {
		t.Fatalf("save store: %v", err)
	}

	reloaded, err := LoadStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}

	got, ok := reloaded.Get(ref)
	if !ok {
		t.Fatalf("expected entry to exist")
	}
	if len(got.Vulnerabilities) != 1 || got.Vulnerabilities[0].ID != "CVE-2025-0001" {
		t.Fatalf("unexpected vulnerabilities: %+v", got.Vulnerabilities)
	}
}

func TestStoreFreshness(t *testing.T) {
	store := &Store{
		Cache: Cache{
			Entries: make(map[string]PackageVuln),
		},
	}

	ref := PackageRef{Ecosystem: "npm", Name: "lodash", Version: "4.17.21"}
	entry := PackageVuln{
		Package:     ref,
		RetrievedAt: time.Now().Add(-2 * time.Hour),
	}

	store.Set(entry)
	cached, ok := store.Get(ref)
	if !ok {
		t.Fatalf("expected cache entry")
	}
	if store.IsFresh(cached, time.Hour) {
		t.Fatalf("expected entry to be stale")
	}
	if !store.IsFresh(cached, 0) {
		t.Fatalf("expected entry to be fresh when maxAge disabled")
	}
}
