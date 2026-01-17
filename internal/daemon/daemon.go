package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

// Daemon runs in the background monitoring agent activity.
type Daemon struct {
	workspace   string
	agentName   string
	sessionMgr  *session.Manager
	session     *session.Session
	logger      *logging.Logger
	config      config.Config
	pidFile     string
	lockFile    string
	mu          sync.Mutex
	interceptCh chan Command
	stopCh      chan struct{}
}

// Command represents an intercepted command.
type Command struct {
	Cmd       string
	Args      []string
	Timestamp time.Time
	PID       int
	PPID      int
	UID       int
	Approved  chan bool // Response channel
}

// New creates a new daemon instance.
func New(workspace, agentName string, cfg config.Config, logger *logging.Logger) (*Daemon, error) {
	sessionMgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return nil, fmt.Errorf("create session manager: %w", err)
	}

	daemonDir := filepath.Join(workspace, ".vectra-guard", "daemon")
	if err := os.MkdirAll(daemonDir, 0o755); err != nil {
		return nil, fmt.Errorf("create daemon directory: %w", err)
	}

	return &Daemon{
		workspace:   workspace,
		agentName:   agentName,
		sessionMgr:  sessionMgr,
		logger:      logger,
		config:      cfg,
		pidFile:     filepath.Join(daemonDir, "daemon.pid"),
		lockFile:    filepath.Join(daemonDir, "daemon.lock"),
		interceptCh: make(chan Command, 100),
		stopCh:      make(chan struct{}),
	}, nil
}

// Start runs the daemon and blocks until stopped.
func (d *Daemon) Start(ctx context.Context) error {
	// Check if daemon is already running
	if d.isRunning() {
		return fmt.Errorf("daemon already running (pid file: %s)", d.pidFile)
	}

	// Acquire exclusive lock
	if err := d.acquireLock(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer d.releaseLock()

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	defer d.removePIDFile()

	// Start session
	sess, err := d.sessionMgr.Start(d.agentName, d.workspace)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	d.session = sess
	session.SetCurrentSessionForWorkspace(d.workspace, sess.ID)

	d.logger.Info("daemon started", map[string]any{
		"session_id": sess.ID,
		"agent":      d.agentName,
		"workspace":  d.workspace,
		"pid":        os.Getpid(),
	})

	// Setup signal handlers
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Start monitoring goroutines
	go d.processCommands(ctx)
	go d.monitorFileSystem(ctx)

	// Wait for stop signal
	select {
	case <-ctx.Done():
		d.logger.Info("daemon stopping (context cancelled)", nil)
	case sig := <-sigCh:
		d.logger.Info("daemon stopping (signal received)", map[string]any{
			"signal": sig.String(),
		})
	case <-d.stopCh:
		d.logger.Info("daemon stopping (stop requested)", nil)
	}

	// Cleanup
	if err := d.sessionMgr.End(d.session); err != nil {
		d.logger.Error("failed to end session", map[string]any{
			"error": err.Error(),
		})
	}

	d.logger.Info("daemon stopped", map[string]any{
		"session_id": sess.ID,
		"commands":   len(sess.Commands),
		"violations": sess.Violations,
	})

	return nil
}

// Stop gracefully stops the daemon.
func (d *Daemon) Stop() {
	close(d.stopCh)
}

// InterceptCommand submits a command for validation.
// Returns true if command should be allowed.
func (d *Daemon) InterceptCommand(cmd string, args []string) bool {
	d.mu.Lock()
	if d.session == nil {
		d.mu.Unlock()
		return true // No active session, allow
	}
	d.mu.Unlock()

	approved := make(chan bool, 1)

	d.interceptCh <- Command{
		Cmd:       cmd,
		Args:      args,
		Timestamp: time.Now(),
		PID:       os.Getpid(),
		PPID:      os.Getppid(),
		UID:       os.Getuid(),
		Approved:  approved,
	}

	// Wait for approval (with timeout)
	select {
	case result := <-approved:
		return result
	case <-time.After(5 * time.Second):
		d.logger.Warn("command approval timeout", map[string]any{
			"command": cmd,
		})
		return false // Deny on timeout
	}
}

func (d *Daemon) processCommands(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case cmd := <-d.interceptCh:
			approved, riskLevel, findings, approvedBy := d.evaluateCommand(cmd)
			d.logger.Debug("command intercepted", map[string]any{
				"command":  cmd.Cmd,
				"args":     cmd.Args,
				"pid":      cmd.PID,
				"approved": approved,
				"risk":     riskLevel,
			})

			if d.session != nil {
				err := d.sessionMgr.AddCommand(d.session, session.Command{
					Timestamp:  cmd.Timestamp,
					Command:    cmd.Cmd,
					Args:       cmd.Args,
					RiskLevel:  riskLevel,
					Approved:   approved,
					ApprovedBy: approvedBy,
					Findings:   findings,
				})
				if err != nil {
					d.logger.Warn("failed to record command", map[string]any{"error": err.Error()})
				}
			}

			// Send approval
			select {
			case cmd.Approved <- approved:
			default:
			}
		}
	}
}

func (d *Daemon) monitorFileSystem(ctx context.Context) {
	paths := d.watchPaths()
	lastStates := make(map[string]time.Time, len(paths))
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			lastStates[path] = info.ModTime()
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			if d.detectFileChanges(paths, lastStates) {
				if err := d.checkIntegrity(); err != nil {
					d.logger.Warn("integrity check failed", map[string]any{
						"error": err.Error(),
					})
				}
			}
			// Periodic check for tampering
			if err := d.checkIntegrity(); err != nil {
				d.logger.Warn("integrity check failed", map[string]any{
					"error": err.Error(),
				})
			}
		}
	}
}

func (d *Daemon) watchPaths() []string {
	paths := map[string]struct{}{}
	if d.workspace == "" {
		return nil
	}

	paths[d.workspace] = struct{}{}
	paths[filepath.Join(d.workspace, ".vectra-guard")] = struct{}{}
	paths[filepath.Join(d.workspace, "vectra-guard.yaml")] = struct{}{}
	paths[filepath.Join(d.workspace, "vectra-guard.yml")] = struct{}{}
	if d.pidFile != "" {
		paths[d.pidFile] = struct{}{}
	}

	var ordered []string
	for path := range paths {
		if _, err := os.Stat(path); err == nil {
			ordered = append(ordered, path)
		}
	}

	return ordered
}

func (d *Daemon) detectFileChanges(paths []string, lastStates map[string]time.Time) bool {
	changed := false
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			if _, ok := lastStates[path]; ok {
				delete(lastStates, path)
				changed = true
			}
			continue
		}

		lastTime, ok := lastStates[path]
		if !ok || info.ModTime().After(lastTime) {
			lastStates[path] = info.ModTime()
			changed = true
		}
	}
	return changed
}

func (d *Daemon) evaluateCommand(cmd Command) (bool, string, []string, string) {
	cmdString := strings.TrimSpace(strings.Join(append([]string{cmd.Cmd}, cmd.Args...), " "))
	if cmdString == "" {
		return true, "low", nil, ""
	}

	if d.config.GuardLevel.AllowUserBypass && d.config.GuardLevel.BypassEnvVar != "" {
		if os.Getenv(d.config.GuardLevel.BypassEnvVar) != "" {
			return true, "low", nil, "bypass_env_var"
		}
	}

	detectionCtx := config.DetectionContext{
		Command:     cmdString,
		GitBranch:   config.GetCurrentGitBranch(d.workspace),
		WorkingDir:  d.workspace,
		Environment: envToMap(os.Environ()),
	}
	level := config.DetectGuardLevel(d.config, detectionCtx)

	findings := analyzer.AnalyzeScript("command", []byte(cmdString), d.config.Policies)
	riskLevel := highestSeverity(findings)
	approved := d.approvalForLevel(level, riskLevel, len(findings) > 0)

	findingsSummary := make([]string, 0, len(findings))
	for _, finding := range findings {
		findingsSummary = append(findingsSummary, fmt.Sprintf("%s:%s", finding.Code, finding.Severity))
	}

	return approved, riskLevel, findingsSummary, "policy"
}

func (d *Daemon) approvalForLevel(level config.GuardLevel, riskLevel string, hasFindings bool) bool {
	switch level {
	case config.GuardLevelOff:
		return true
	case config.GuardLevelLow:
		return riskLevel != "critical"
	case config.GuardLevelMedium:
		return riskLevel != "critical" && riskLevel != "high"
	case config.GuardLevelHigh:
		return riskLevel == "low"
	case config.GuardLevelParanoid:
		return !hasFindings && riskLevel == "low"
	default:
		return riskLevel != "critical" && riskLevel != "high"
	}
}

func highestSeverity(findings []analyzer.Finding) string {
	priority := map[string]int{
		"low":      0,
		"medium":   1,
		"high":     2,
		"critical": 3,
	}
	maxSeverity := "low"
	maxRank := 0
	for _, finding := range findings {
		rank, ok := priority[strings.ToLower(finding.Severity)]
		if ok && rank > maxRank {
			maxRank = rank
			maxSeverity = strings.ToLower(finding.Severity)
		}
	}
	return maxSeverity
}

func envToMap(values []string) map[string]string {
	env := make(map[string]string, len(values))
	for _, entry := range values {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func (d *Daemon) checkIntegrity() error {
	// Check if session file still exists and is valid
	d.mu.Lock()
	sessID := d.session.ID
	d.mu.Unlock()

	sessionPath := filepath.Join(d.workspace, ".vectra-guard", "sessions", sessID+".json")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return fmt.Errorf("session file deleted: possible tampering")
	}

	// Check if PID file was modified
	pidData, err := os.ReadFile(d.pidFile)
	if err != nil {
		return fmt.Errorf("pid file read failed: %w", err)
	}

	expectedPID := fmt.Sprintf("%d", os.Getpid())
	if string(pidData) != expectedPID {
		return fmt.Errorf("pid file tampered: expected %s", expectedPID)
	}

	return nil
}

func (d *Daemon) isRunning() bool {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		return false
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process is alive
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}

func (d *Daemon) acquireLock() error {
	f, err := os.OpenFile(d.lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	f.Close()
	return nil
}

func (d *Daemon) releaseLock() {
	os.Remove(d.lockFile)
}

func (d *Daemon) writePIDFile() error {
	return os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644)
}

func (d *Daemon) removePIDFile() {
	os.Remove(d.pidFile)
}

// GetRunningDaemon returns the PID of a running daemon, or 0 if not running.
func GetRunningDaemon(workspace string) int {
	pidFile := filepath.Join(workspace, ".vectra-guard", "daemon", "daemon.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0
	}

	// Verify process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		return 0
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		return 0
	}

	return pid
}
