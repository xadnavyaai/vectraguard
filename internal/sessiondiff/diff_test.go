package sessiondiff

import (
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/session"
)

func TestComputeBasicDiff(t *testing.T) {
	now := time.Now()
	ops := []session.FileOperation{
		{Timestamp: now, Operation: "create", Path: "/workspace/a.txt"},
		{Timestamp: now, Operation: "modify", Path: "/workspace/a.txt"},
		{Timestamp: now, Operation: "create", Path: "/workspace/b.txt"},
		{Timestamp: now, Operation: "delete", Path: "/workspace/b.txt"},
		{Timestamp: now, Operation: "modify", Path: "/workspace/c.txt"},
	}

	sum := Compute(ops, "")

	if len(sum.Added) != 1 || sum.Added[0] != "/workspace/a.txt" {
		t.Fatalf("expected a.txt as added, got %#v", sum.Added)
	}
	if len(sum.Modified) != 1 || sum.Modified[0] != "/workspace/c.txt" {
		t.Fatalf("expected c.txt as modified, got %#v", sum.Modified)
	}
	if len(sum.Deleted) != 1 || sum.Deleted[0] != "/workspace/b.txt" {
		t.Fatalf("expected b.txt as deleted, got %#v", sum.Deleted)
	}
}

func TestComputeWithRootFilter(t *testing.T) {
	now := time.Now()
	ops := []session.FileOperation{
		{Timestamp: now, Operation: "create", Path: "/workspace/a.txt"},
		{Timestamp: now, Operation: "create", Path: "/other/b.txt"},
	}

	sum := Compute(ops, "/workspace")

	if len(sum.Added) != 1 || sum.Added[0] != "/workspace/a.txt" {
		t.Fatalf("expected only /workspace/a.txt, got %#v", sum.Added)
	}
}

