package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
	"github.com/vectra-guard/vectra-guard/internal/sessiondiff"
)

func runSessionDiff(ctx context.Context, sessionID, rootPath string, jsonOutput bool) error {
	logger := logging.FromContext(ctx)

	workspace, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return fmt.Errorf("create session manager: %w", err)
	}

	sess, err := mgr.Load(sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	sum := sessiondiff.Compute(sess.FileOps, rootPath)

	if jsonOutput {
		out := struct {
			SessionID string               `json:"session_id"`
			Workspace string               `json:"workspace"`
			Root      string               `json:"root,omitempty"`
			Diff      sessiondiff.Summary  `json:"diff"`
		}{
			SessionID: sess.ID,
			Workspace: sess.Workspace,
			Root:      rootPath,
			Diff:      sum,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Fprintf(os.Stdout, "Session %s filesystem changes", sess.ID)
	if rootPath != "" {
		fmt.Fprintf(os.Stdout, " under %s", rootPath)
	}
	fmt.Fprintln(os.Stdout, ":")

	if len(sum.Added) == 0 && len(sum.Modified) == 0 && len(sum.Deleted) == 0 {
		fmt.Fprintln(os.Stdout, "  (no recorded file changes)")
		return nil
	}

	if len(sum.Added) > 0 {
		fmt.Fprintln(os.Stdout, "  Added:")
		for _, p := range sum.Added {
			fmt.Fprintf(os.Stdout, "    + %s\n", p)
		}
	}
	if len(sum.Modified) > 0 {
		fmt.Fprintln(os.Stdout, "  Modified:")
		for _, p := range sum.Modified {
			fmt.Fprintf(os.Stdout, "    ~ %s\n", p)
		}
	}
	if len(sum.Deleted) > 0 {
		fmt.Fprintln(os.Stdout, "  Deleted:")
		for _, p := range sum.Deleted {
			fmt.Fprintf(os.Stdout, "    - %s\n", p)
		}
	}
	return nil
}

