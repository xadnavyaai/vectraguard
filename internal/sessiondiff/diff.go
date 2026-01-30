package sessiondiff

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/session"
)

// Summary represents a high-level view of filesystem changes during a session.
type Summary struct {
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Deleted  []string `json:"deleted"`
}

// Compute builds a diff summary from recorded file operations. If root is
// non-empty, only paths under that prefix are considered.
func Compute(ops []session.FileOperation, root string) Summary {
	type opKind string
	const (
		opAdded    opKind = "added"
		opModified opKind = "modified"
		opDeleted  opKind = "deleted"
	)

	normalize := func(p string) string {
		if p == "" {
			return ""
		}
		return filepath.Clean(p)
	}

	root = normalize(root)
	last := make(map[string]opKind)

	for _, op := range ops {
		path := normalize(op.Path)
		if path == "" {
			continue
		}
		if root != "" {
			// Keep only paths under root; treat equal or prefixed.
			if path != root && !strings.HasPrefix(path, root+string(filepath.Separator)) {
				continue
			}
		}
		switch strings.ToLower(op.Operation) {
		case "create":
			last[path] = opAdded
		case "modify":
			// If we previously saw "added", keep it as added; otherwise mark modified.
			if last[path] != opAdded {
				last[path] = opModified
			}
		case "delete":
			last[path] = opDeleted
		default:
			// read or unknown operations do not affect the diff.
			continue
		}
	}

	added := make([]string, 0)
	modified := make([]string, 0)
	deleted := make([]string, 0)

	for path, kind := range last {
		switch kind {
		case opAdded:
			added = append(added, path)
		case opModified:
			modified = append(modified, path)
		case opDeleted:
			deleted = append(deleted, path)
		}
	}

	sort.Strings(added)
	sort.Strings(modified)
	sort.Strings(deleted)

	return Summary{
		Added:    added,
		Modified: modified,
		Deleted:  deleted,
	}
}
