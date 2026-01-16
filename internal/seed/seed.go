package seed

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/**
var templates embed.FS

type Result struct {
	Path   string
	Status string
}

func WriteAgentInstructions(targetDir string, force bool) ([]Result, error) {
	if targetDir == "" {
		targetDir = "."
	}

	targets := map[string]string{
		"AGENTS.md":                            "templates/AGENTS.md",
		"CLAUDE.md":                            "templates/CLAUDE.md",
		"CODEX.md":                             "templates/CODEX.md",
		".github/copilot-instructions.md":      "templates/.github/copilot-instructions.md",
		".cursor/rules/vectra-guard.md":        "templates/.cursor/rules/vectra-guard.md",
		".windsurf/rules.md":                   "templates/.windsurf/rules.md",
		".vscode/vectra-guard.instructions.md": "templates/.vscode/vectra-guard.instructions.md",
	}

	var results []Result
	for dstRel, srcPath := range targets {
		dst := filepath.Join(targetDir, dstRel)
		if !force {
			if _, err := os.Stat(dst); err == nil {
				results = append(results, Result{Path: dst, Status: "skipped"})
				continue
			}
		}

		payload, err := fs.ReadFile(templates, srcPath)
		if err != nil {
			return results, fmt.Errorf("read template %s: %w", srcPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return results, fmt.Errorf("create directory for %s: %w", dst, err)
		}
		if err := os.WriteFile(dst, payload, 0o644); err != nil {
			return results, fmt.Errorf("write %s: %w", dst, err)
		}
		results = append(results, Result{Path: dst, Status: "written"})
	}

	return results, nil
}
