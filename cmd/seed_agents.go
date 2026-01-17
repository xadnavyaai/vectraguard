package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/seed"
)

func runSeedAgents(ctx context.Context, target string, force bool, targets []string, listOnly bool) error {
	if target == "" {
		target = "."
	}

	if listOnly {
		available := seed.AvailableTargets()
		keys := make([]string, 0, len(available))
		for key := range available {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Println("Available targets:")
		for _, key := range keys {
			fmt.Printf("- %s\n", key)
		}
		return nil
	}

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("target not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target is not a directory: %s", target)
	}

	fmt.Printf("🧭 Seeding agent instructions into: %s\n", target)
	results, err := seed.WriteAgentInstructions(target, force, targets)
	if err != nil {
		return err
	}

	for _, r := range results {
		switch r.Status {
		case "written":
			fmt.Printf("✅ Wrote: %s\n", r.Path)
		case "skipped":
			fmt.Printf("↷ Exists, skipping: %s\n", r.Path)
		default:
			fmt.Printf("• %s: %s\n", r.Path, r.Status)
		}
	}

	fmt.Println("Done.")
	return nil
}

func parseSeedTargets(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' '
	})
	var out []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, strings.ToLower(trimmed))
		}
	}
	return out
}
