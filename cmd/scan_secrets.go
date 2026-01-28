package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/secrets"
)

func runScanSecrets(ctx context.Context, targetPath string, allowlistPath string) error {
	logger := logging.FromContext(ctx)

	opts := secrets.Options{
		Allowlist: nil,
	}
	if allowlistPath != "" {
		allowlist, err := loadAllowlist(allowlistPath)
		if err != nil {
			return fmt.Errorf("load allowlist: %w", err)
		}
		opts.Allowlist = allowlist
	}

	findings, err := secrets.ScanPath(targetPath, opts)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		logger.Info("no secrets detected", map[string]any{
			"path": targetPath,
		})
		return nil
	}

	for _, f := range findings {
		logger.Warn("secret detected", map[string]any{
			"file":      f.File,
			"line":      f.Line,
			"pattern":   f.PatternID,
			"match":     f.Match,
			"entropy":   fmt.Sprintf("%.2f", f.Entropy),
			"severity":  f.Severity,
			"scan_path": targetPath,
		})
	}

	// Non-zero exit code so callers (including CI) can fail on findings.
	return &exitError{message: "secrets detected", code: 2}
}

func loadAllowlist(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	allow := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		allow[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return allow, nil
}

