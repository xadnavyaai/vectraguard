package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/vectra-guard/vectra-guard/internal/seed"
)

func runSeedAgents(ctx context.Context, target string, force bool) error {
	if target == "" {
		target = "."
	}

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("target not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target is not a directory: %s", target)
	}

	fmt.Printf("🧭 Seeding agent instructions into: %s\n", target)
	results, err := seed.WriteAgentInstructions(target, force)
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
