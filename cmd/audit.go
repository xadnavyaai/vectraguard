package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

type auditSummary struct {
	Tool   string
	Path   string
	Total  int
	Counts map[string]int
	Mode   string
}

func runAudit(ctx context.Context, tool string, targetPath string, failOnFindings bool, autoInstall bool) error {
	logger := logging.FromContext(ctx)
	if targetPath == "" {
		targetPath = "."
	}

	switch strings.ToLower(tool) {
	case "npm":
		if autoInstall {
			if err := ensureNpm(ctx); err != nil {
				return err
			}
		}
		summary, err := runNpmAudit(targetPath)
		if err != nil {
			return err
		}
		logAuditSummary(logger, summary)
		if failOnFindings && summary.Total > 0 {
			return &exitError{message: "npm audit found vulnerabilities", code: 2}
		}
	case "python", "pip":
		if autoInstall {
			if err := ensurePipAudit(ctx); err != nil {
				return err
			}
		}
		summary, err := runPythonAudit(targetPath)
		if err != nil {
			return err
		}
		logAuditSummary(logger, summary)
		if failOnFindings && summary.Total > 0 {
			return &exitError{message: "python audit found vulnerabilities", code: 2}
		}
	default:
		return fmt.Errorf("unsupported audit tool: %s", tool)
	}

	return nil
}

func runNpmAudit(targetPath string) (auditSummary, error) {
	if !commandAvailable("npm") {
		return auditSummary{}, fmt.Errorf("npm not found in PATH")
	}
	if !fileExists(filepath.Join(targetPath, "package.json")) {
		return auditSummary{}, fmt.Errorf("package.json not found in %s", targetPath)
	}

	cmd := exec.Command("npm", "audit", "--json")
	cmd.Dir = targetPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()

	if len(output) == 0 && err != nil {
		return auditSummary{}, fmt.Errorf("npm audit failed: %s", strings.TrimSpace(stderr.String()))
	}
	counts, total, parseErr := parseNpmAuditOutput(output)
	if parseErr != nil {
		return auditSummary{}, parseErr
	}

	return auditSummary{
		Tool:   "npm",
		Path:   targetPath,
		Total:  total,
		Counts: counts,
		Mode:   "project",
	}, nil
}

type pipAuditEntry struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Vulns   []pipAuditVuln `json:"vulns"`
}

type pipAuditVuln struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	FixVersions []string `json:"fix_versions"`
}

func runPythonAudit(targetPath string) (auditSummary, error) {
	mode := "environment"
	args := []string{"-f", "json"}
	reqPath := filepath.Join(targetPath, "requirements.txt")
	if fileExists(reqPath) {
		mode = "requirements.txt"
		args = append([]string{"-r", reqPath}, args...)
	}

	cmd, err := pythonAuditCommand(args)
	if err != nil {
		return auditSummary{}, err
	}
	cmd.Dir = targetPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if len(output) == 0 && err != nil {
		return auditSummary{}, fmt.Errorf("pip-audit failed: %s", strings.TrimSpace(stderr.String()))
	}

	var entries []pipAuditEntry
	entries, err = parsePipAuditOutput(output)
	if err != nil {
		return auditSummary{}, err
	}

	counts := map[string]int{}
	total := 0
	for _, entry := range entries {
		for _, vuln := range entry.Vulns {
			sev := strings.ToLower(strings.TrimSpace(vuln.Severity))
			if sev == "" {
				sev = "unknown"
			}
			counts[sev] = counts[sev] + 1
			total++
		}
	}

	return auditSummary{
		Tool:   "python",
		Path:   targetPath,
		Total:  total,
		Counts: counts,
		Mode:   mode,
	}, nil
}

func pythonAuditCommand(args []string) (*exec.Cmd, error) {
	if commandAvailable("pip-audit") {
		return exec.Command("pip-audit", args...), nil
	}
	if commandAvailable("python3") {
		return exec.Command("python3", append([]string{"-m", "pip_audit"}, args...)...), nil
	}
	if commandAvailable("python") {
		return exec.Command("python", append([]string{"-m", "pip_audit"}, args...)...), nil
	}
	return nil, fmt.Errorf("pip-audit not found (install with: python -m pip install pip-audit)")
}

func parsePipAuditOutput(output []byte) ([]pipAuditEntry, error) {
	start := bytes.IndexAny(output, "{[")
	if start < 0 {
		return nil, fmt.Errorf("parse pip-audit output: no JSON found")
	}

	trimmed := output[start:]
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	if trimmed[0] == '{' {
		wrapped := struct {
			Dependencies []pipAuditEntry `json:"dependencies"`
		}{}
		if err := decoder.Decode(&wrapped); err != nil {
			return nil, fmt.Errorf("parse pip-audit output: %w", err)
		}
		return wrapped.Dependencies, nil
	}

	var entries []pipAuditEntry
	if err := decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("parse pip-audit output: %w", err)
	}
	return entries, nil
}

func logAuditSummary(logger *logging.Logger, summary auditSummary) {
	fields := map[string]any{
		"tool":  summary.Tool,
		"path":  summary.Path,
		"mode":  summary.Mode,
		"total": summary.Total,
	}
	if len(summary.Counts) > 0 {
		fields["counts"] = summary.Counts
	}

	if summary.Total > 0 {
		logger.Warn("package audit findings", fields)
	} else {
		logger.Info("package audit clean", fields)
	}
}

func ensureNpm(ctx context.Context) error {
	if commandAvailable("npm") {
		return nil
	}
	logger := logging.FromContext(ctx)
	logger.Info("npm not found; attempting install", nil)

	switch runtime.GOOS {
	case "darwin":
		if !commandAvailable("brew") {
			return fmt.Errorf("npm not found and Homebrew missing (install Node.js first)")
		}
		return runInstallCommand("brew", "install", "node")
	case "linux":
		if commandAvailable("apt-get") {
			if err := runInstallCommand("sudo", "apt-get", "update", "-y"); err != nil {
				return err
			}
			return runInstallCommand("sudo", "apt-get", "install", "-y", "nodejs", "npm")
		}
		if commandAvailable("dnf") {
			return runInstallCommand("sudo", "dnf", "install", "-y", "nodejs", "npm")
		}
		if commandAvailable("yum") {
			return runInstallCommand("sudo", "yum", "install", "-y", "nodejs", "npm")
		}
	}
	return fmt.Errorf("npm not found (install Node.js/npm first)")
}

func ensurePipAudit(ctx context.Context) error {
	if commandAvailable("pip-audit") {
		return nil
	}
	if commandAvailable("python3") || commandAvailable("python") {
		logger := logging.FromContext(ctx)
		logger.Info("pip-audit not found; attempting install", nil)
		if commandAvailable("python3") {
			return runInstallCommand("python3", "-m", "pip", "install", "--user", "pip-audit")
		}
		return runInstallCommand("python", "-m", "pip", "install", "--user", "pip-audit")
	}
	return fmt.Errorf("pip-audit not found (install with: python -m pip install pip-audit)")
}

func runInstallCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s", name, strings.TrimSpace(string(output)))
	}
	return nil
}

func parseNpmAuditOutput(output []byte) (map[string]int, int, error) {
	report := struct {
		Metadata struct {
			Vulnerabilities map[string]int `json:"vulnerabilities"`
		} `json:"metadata"`
		Error interface{} `json:"error"`
	}{}
	if parseErr := json.Unmarshal(output, &report); parseErr != nil {
		return nil, 0, fmt.Errorf("parse npm audit output: %w", parseErr)
	}

	counts := map[string]int{}
	total := 0
	for sev, count := range report.Metadata.Vulnerabilities {
		counts[sev] = count
		total += count
	}
	return counts, total, nil
}
