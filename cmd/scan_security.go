package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/secscan"
)

func runScanSecurity(ctx context.Context, targetPath string, languagesCSV string) error {
	logger := logging.FromContext(ctx)

	opts := secscan.Options{
		Languages: make(map[string]bool),
	}
	if languagesCSV != "" {
		for _, part := range strings.Split(languagesCSV, ",") {
			lang := strings.TrimSpace(strings.ToLower(part))
			if lang == "" {
				continue
			}
			opts.Languages[lang] = true
		}
	}

	findings, err := secscan.ScanPath(targetPath, opts)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		logger.Info("security scan completed (no findings)", map[string]any{
			"path": targetPath,
		})
		return nil
	}

	for _, f := range findings {
		logger.Warn("security finding", map[string]any{
			"file":        f.File,
			"line":        f.Line,
			"language":    f.Language,
			"severity":    f.Severity,
			"code":        f.Code,
			"description": f.Description,
		})
	}

	// Non-zero exit to signal issues in CI / automation.
	return &exitError{
		message: fmt.Sprintf("security issues detected in %d location(s)", len(findings)),
		code:    2,
	}
}

