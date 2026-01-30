package agentvalidate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
)

// Result holds validation findings for a single script.
type Result struct {
	Path     string
	Findings []analyzer.Finding
}

// ValidatePath validates a script file or all supported scripts under a
// directory. It reuses the core analyzer engine but focuses on agent scripts
// (shell, Python, etc.).
func ValidatePath(path string, policy config.PolicyConfig) ([]Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if !info.IsDir() {
		f, err := validateFile(path, policy)
		if err != nil {
			return nil, err
		}
		if len(f.Findings) == 0 {
			return nil, nil
		}
		return []Result{f}, nil
	}

	var results []Result
	err = filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !isSupportedScript(p) {
			return nil
		}
		r, err := validateFile(p, policy)
		if err != nil {
			return nil
		}
		if len(r.Findings) > 0 {
			results = append(results, r)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func validateFile(path string, policy config.PolicyConfig) (Result, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read script %s: %w", path, err)
	}
	findings := analyzer.AnalyzeScript(path, content, policy)
	return Result{
		Path:     path,
		Findings: findings,
	}, nil
}

func isSupportedScript(path string) bool {
	switch filepath.Ext(path) {
	case ".sh", ".bash", ".zsh", ".py":
		return true
	default:
		return false
	}
}
