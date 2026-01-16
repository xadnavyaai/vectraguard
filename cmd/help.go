package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func runHelp(ctx context.Context, topic string) error {
	topic = strings.ToLower(strings.TrimSpace(topic))
	switch topic {
	case "", "all":
		return printHelp(`Vectra Guard help:

  vg help [topic]

Topics:
  context   Summarize code/docs for navigation
  roadmap   Track repo-specific planning items
  init      Initialize config (including repo-local)
  sandbox   Install sandbox dependencies
  seed      Seed agent instructions into a repo
`)
	case "context":
		return printHelp(`Context summaries:

  vg context summarize code <file|directory> [--max 5] [--output text|json] [--since <commit|date>]
  vg context summarize docs <file|directory> [--max 3] [--output text|json] [--since <commit|date>]
  vg context summarize advanced <file|directory> [--max 3] [--output text|json] [--since <commit|date>]

Summarize a single file or entire repository. When given a directory, processes all
relevant files and groups results by file. Results are cached in .vectra-guard/cache/
for faster subsequent runs.

Advanced mode parses Go files and surfaces high-impact functions with call-graph signals.

Options:
  --max N          Maximum number of summary lines per file (default: 5)
  --output FORMAT  Output format: text (default) or json
  --since REF      Only process files changed since commit/date (e.g., HEAD~1, 2024-01-01)

Examples:
  vg context summarize code cmd/root.go
  vg context summarize code . --output json              # JSON output for agents
  vg context summarize code . --since HEAD~1             # Only changed files
  vg context summarize code . --since 2024-01-01         # Since date
  vg context summarize advanced internal/ --max 10        # More summary lines
`)
	case "roadmap":
		return printHelp(`Roadmap planning:

  vg roadmap add --title "Title" --summary "Details" --tags "agent,plan"
  vg roadmap list [--status planned]
  vg roadmap show <id>
  vg roadmap status <id> <status>
  vg roadmap log <id> --note "Update" --session <session-id>

Roadmaps are stored per workspace under ~/.vectra-guard/roadmaps.
`)
	case "init":
		return printHelp(`Initialization:

  vg init            # writes vectra-guard.yaml in repo root
  vg init --toml     # writes vectra-guard.toml
  vg init --local    # writes .vectra-guard/config.yaml and sets cache_dir

Local init creates a repo-scoped cache directory at .vectra-guard/cache.
`)
	case "sandbox":
		return printHelp(`Sandbox dependencies:

  vg sandbox deps install [--force] [--dry-run]

Install Docker/Podman + bubblewrap for sandboxing. Use --force to remove
conflicting binaries (e.g., /usr/local/bin/hub-tool) on macOS. Use --dry-run
or DRY_RUN=1 to preview commands.
`)
	case "seed":
		return printHelp(`Seed agent instructions:

  vg seed agents [--target .] [--force]

Creates agent instruction files for Cursor, Claude, Codex, VS Code,
Copilot, and Windsurf inside a target repository.
`)
	default:
		return printHelp(fmt.Sprintf("Unknown help topic: %s\n", topic))
	}
}

func printHelp(message string) error {
	_, err := fmt.Fprint(os.Stdout, message)
	return err
}
