package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

type auditSummary struct {
	Tool   string
	Path   string
	Total  int
	Counts map[string]int
	Mode   string
}

type sessionAuditSummary struct {
	SessionID       string
	Workspace       string
	Agent           string
	SessionCount    int
	Total           int
	RiskCounts      map[string]int
	SourceCounts    map[string]int
	ExecutionCounts map[string]int
	Bypassed        int
	Blocked         int
}

func runAudit(ctx context.Context, tool string, targetPath string, failOnFindings bool, autoInstall bool, sessionID string, allSessions bool) error {
	logger := logging.FromContext(ctx)
	if targetPath == "" {
		targetPath = "."
	}

	switch strings.ToLower(tool) {
	case "session":
		summary, err := runSessionAudit(ctx, sessionID, allSessions)
		if err != nil {
			return err
		}
		logSessionAuditSummary(logger, summary)
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

func runSessionAudit(ctx context.Context, sessionID string, allSessions bool) (sessionAuditSummary, error) {
	logger := logging.FromContext(ctx)
	workspace, err := os.Getwd()
	if err != nil {
		return sessionAuditSummary{}, fmt.Errorf("get workspace: %w", err)
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return sessionAuditSummary{}, fmt.Errorf("create session manager: %w", err)
	}

	if allSessions {
		sessions, err := mgr.List()
		if err != nil {
			return sessionAuditSummary{}, fmt.Errorf("list sessions: %w", err)
		}
		if len(sessions) == 0 {
			return sessionAuditSummary{}, fmt.Errorf("no sessions found")
		}
		return buildSessionAuditSummaryFromSessions(sessions), nil
	}

	if sessionID == "" {
		sessionID = session.GetCurrentSessionForWorkspace(workspace)
	}
	if sessionID == "" {
		return sessionAuditSummary{}, fmt.Errorf("no active session found")
	}

	sess, err := mgr.Load(sessionID)
	if err != nil {
		return sessionAuditSummary{}, fmt.Errorf("load session: %w", err)
	}

	return buildSessionAuditSummary(sess), nil
}

func buildSessionAuditSummary(sess *session.Session) sessionAuditSummary {
	summary := sessionAuditSummary{
		SessionID:       sess.ID,
		Workspace:       sess.Workspace,
		Agent:           sess.AgentName,
		SessionCount:    1,
		Total:           len(sess.Commands),
		RiskCounts:      map[string]int{},
		SourceCounts:    map[string]int{},
		ExecutionCounts: map[string]int{},
	}

	for _, cmd := range sess.Commands {
		risk := strings.TrimSpace(cmd.RiskLevel)
		if risk == "" {
			risk = "unknown"
		}
		summary.RiskCounts[risk] = summary.RiskCounts[risk] + 1

		source := "exec"
		execution := ""
		if cmd.Metadata != nil {
			if val, ok := cmd.Metadata["source"].(string); ok && val != "" {
				source = val
			}
			if val, ok := cmd.Metadata["execution"].(string); ok && val != "" {
				execution = val
			}
			if val, ok := cmd.Metadata["bypass"].(bool); ok && val {
				summary.Bypassed++
			}
			if val, ok := cmd.Metadata["blocked"].(bool); ok && val {
				summary.Blocked++
			}
		}

		summary.SourceCounts[source] = summary.SourceCounts[source] + 1
		if execution != "" {
			summary.ExecutionCounts[execution] = summary.ExecutionCounts[execution] + 1
		}
	}

	return summary
}

func buildSessionAuditSummaryFromSessions(sessions []*session.Session) sessionAuditSummary {
	summary := sessionAuditSummary{
		SessionID:       "all",
		SessionCount:    len(sessions),
		Total:           0,
		RiskCounts:      map[string]int{},
		SourceCounts:    map[string]int{},
		ExecutionCounts: map[string]int{},
	}

	for _, sess := range sessions {
		if summary.Workspace == "" {
			summary.Workspace = sess.Workspace
		}
		if summary.Agent == "" {
			summary.Agent = sess.AgentName
		}
		summary.Total += len(sess.Commands)
		single := buildSessionAuditSummary(sess)
		mergeCounts(summary.RiskCounts, single.RiskCounts)
		mergeCounts(summary.SourceCounts, single.SourceCounts)
		mergeCounts(summary.ExecutionCounts, single.ExecutionCounts)
		summary.Bypassed += single.Bypassed
		summary.Blocked += single.Blocked
	}

	return summary
}

func mergeCounts(dst, src map[string]int) {
	for key, val := range src {
		dst[key] = dst[key] + val
	}
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

func logSessionAuditSummary(logger *logging.Logger, summary sessionAuditSummary) {
	fields := map[string]any{
		"session_id": summary.SessionID,
		"workspace":  summary.Workspace,
		"agent":      summary.Agent,
		"sessions":   summary.SessionCount,
		"total":      summary.Total,
	}
	if len(summary.RiskCounts) > 0 {
		fields["risk_counts"] = summary.RiskCounts
	}
	if len(summary.SourceCounts) > 0 {
		fields["source_counts"] = summary.SourceCounts
	}
	if len(summary.ExecutionCounts) > 0 {
		fields["execution_counts"] = summary.ExecutionCounts
	}
	if summary.Bypassed > 0 {
		fields["bypassed"] = summary.Bypassed
	}
	if summary.Blocked > 0 {
		fields["blocked"] = summary.Blocked
	}

	logger.Info("session audit summary", fields)
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
