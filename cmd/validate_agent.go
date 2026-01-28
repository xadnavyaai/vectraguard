package cmd

import (
	"context"
	"fmt"

	"github.com/vectra-guard/vectra-guard/internal/agentvalidate"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runValidateAgent(ctx context.Context, targetPath string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	results, err := agentvalidate.ValidatePath(targetPath, cfg.Policies)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		logger.Info("agent validation passed", map[string]any{
			"path": targetPath,
		})
		return nil
	}

	for _, res := range results {
		for _, f := range res.Findings {
			logger.Warn("agent validation finding", map[string]any{
				"path":           res.Path,
				"line":           f.Line,
				"code":           f.Code,
				"severity":       f.Severity,
				"description":    f.Description,
				"recommendation": f.Recommendation,
			})
		}
	}

	return &exitError{
		message: fmt.Sprintf("validation issues detected in %d script(s)", len(results)),
		code:    2,
	}
}

