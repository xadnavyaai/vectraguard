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
`)
	case "context":
		return printHelp(`Context summaries:

  vg context summarize code <file> --max 5
  vg context summarize docs <file> --max 3
  vg context summarize advanced <file> --max 3

Advanced mode parses Go files and surfaces high-impact functions with call-graph signals.
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
	default:
		return printHelp(fmt.Sprintf("Unknown help topic: %s\n", topic))
	}
}

func printHelp(message string) error {
	_, err := fmt.Fprint(os.Stdout, message)
	return err
}
