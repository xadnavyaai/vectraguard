package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func runExec(ctx context.Context, cmdArgs []string, interactive bool, sessionID string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmdName := cmdArgs[0]
	args := cmdArgs[1:]

	// Build command string for analysis
	cmdString := strings.Join(cmdArgs, " ")
	tracker := initSessionTracker(sessionID, logger)
	if tracker != nil {
		sessionID = tracker.id
	}

	// CRITICAL: Always analyze commands FIRST, even if guard level is OFF
	// This ensures critical commands like "rm -rf /" are ALWAYS blocked
	// regardless of guard level configuration
	findings := analyzer.AnalyzeScript("inline-command", []byte(cmdString), cfg.Policies)
	trackingRiskLevel, trackingFindingCodes := summarizeFindings(findings)
	
	// Check for CRITICAL commands that MUST be blocked regardless of guard level
	// These commands can destroy the system and should NEVER execute
	criticalCodes := []string{
		"DANGEROUS_DELETE_ROOT",
		"DANGEROUS_DELETE_HOME",
		"PROTECTED_DIRECTORY_ACCESS", // Also block protected directory access (e.g., rm -rf /)
		"FORK_BOMB",
		"SENSITIVE_ENV_ACCESS",
		"DOTENV_FILE_READ",
	}
	
	hasCriticalCode := false
	for _, f := range findings {
		for _, code := range criticalCodes {
			if f.Code == code {
				hasCriticalCode = true
				logger.Error("CRITICAL command blocked - cannot be bypassed", map[string]any{
					"command":    cmdString,
					"code":       code,
					"severity":   f.Severity,
					"description": f.Description,
				})
				recordCommandAttempt(tracker, session.Command{
					Timestamp: time.Now(),
					Command:   cmdName,
					Args:      args,
					ExitCode:  3,
					Duration:  0,
					RiskLevel: trackingRiskLevel,
					Approved:  false,
					Findings:  trackingFindingCodes,
					Metadata: map[string]interface{}{
						"blocked":       true,
						"block_reason":  "critical_command",
						"block_code":    code,
						"guard_level":   cfg.GuardLevel.Level,
						"repeat_detect": false,
					},
				})
				return &exitError{
					message: fmt.Sprintf("CRITICAL: Command '%s' is blocked for safety. %s", cmdString, f.Recommendation),
					code:    3,
				}
			}
		}
		if hasCriticalCode {
			break
		}
	}

	riskLevel := "low"
	var findingCodes []string
	
	// Filter findings based on guard level
	filteredFindings := filterFindingsByGuardLevel(findings, cfg.GuardLevel.Level)
	
	// Debug: Log if findings were filtered out
	if len(findings) > 0 && len(filteredFindings) == 0 {
		logger.Warn("findings filtered out by guard level", map[string]any{
			"total_findings": len(findings),
			"filtered_findings": len(filteredFindings),
			"guard_level": cfg.GuardLevel.Level,
			"findings": findings,
		})
	}
	
	if len(filteredFindings) > 0 {
		// Determine highest risk level
		for _, f := range filteredFindings {
			findingCodes = append(findingCodes, f.Code)
			switch f.Severity {
			case "critical":
				riskLevel = "critical"
			case "high":
				if riskLevel != "critical" {
					riskLevel = "high"
				}
			case "medium":
				if riskLevel != "critical" && riskLevel != "high" {
					riskLevel = "medium"
				}
			}
		}

		// Log findings
		for _, f := range filteredFindings {
			logger.Warn("command risk detected", map[string]any{
				"command":        cmdString,
				"code":           f.Code,
				"severity":       f.Severity,
				"description":    f.Description,
				"recommendation": f.Recommendation,
			})
		}

		// Resolve auto guard level to concrete level before checking approval
		effectiveGuardLevel := cfg.GuardLevel.Level
		if effectiveGuardLevel == config.GuardLevelAuto {
			// Auto mode: treat as medium for blocking (conservative)
			effectiveGuardLevel = config.GuardLevelMedium
		}
		
		// Determine if approval is required based on guard level
		requiresApproval := shouldRequireApproval(riskLevel, effectiveGuardLevel)
		
		// Handle interactive approval or blocking
		if requiresApproval {
			if interactive {
				approval := promptForApproval(riskLevel, cmdString, filteredFindings)
				if !approval.approved {
					logger.Info("command execution denied by user", map[string]any{
						"command": cmdString,
					})
					recordCommandAttempt(tracker, session.Command{
						Timestamp: time.Now(),
						Command:   cmdName,
						Args:      args,
						ExitCode:  3,
						Duration:  0,
						RiskLevel: trackingRiskLevel,
						Approved:  false,
						Findings:  trackingFindingCodes,
						Metadata: map[string]interface{}{
							"blocked":      true,
							"block_reason": "user_denied",
							"guard_level":  cfg.GuardLevel.Level,
						},
					})
					return &exitError{message: "execution denied", code: 3}
				}
				
				// Handle "remember" functionality
				if approval.remember && cfg.Sandbox.Enabled {
					trustStore, err := sandbox.NewTrustStore(cfg.Sandbox.TrustStorePath)
					if err == nil {
						duration := time.Duration(0) // Never expire by default
						if approval.duration > 0 {
							duration = approval.duration
						}
						if err := trustStore.Add(cmdString, duration, "User approved"); err == nil {
							fmt.Fprintln(os.Stderr, "✅ Approved and remembered")
							logger.Info("command added to trust store", map[string]any{
								"command": cmdString,
							})
						}
					}
				}
			} else {
				logger.Error("risky command blocked", map[string]any{
					"command":    cmdString,
					"risk_level": riskLevel,
					"guard_level": cfg.GuardLevel.Level,
				})
				recordCommandAttempt(tracker, session.Command{
					Timestamp: time.Now(),
					Command:   cmdName,
					Args:      args,
					ExitCode:  3,
					Duration:  0,
					RiskLevel: trackingRiskLevel,
					Approved:  false,
					Findings:  trackingFindingCodes,
					Metadata: map[string]interface{}{
						"blocked":      true,
						"block_reason": "guard_level_block",
						"guard_level":  cfg.GuardLevel.Level,
					},
				})
				return &exitError{
					message: fmt.Sprintf("%s risk command blocked by guard level %s (use --interactive to approve, or set bypass)", 
						riskLevel, cfg.GuardLevel.Level),
					code: 3,
				}
			}
		}
	}

	// Repeated action detection (prevents rapid destructive loops)
	if tracker != nil {
		repeatDecision := evaluateRepeatProtection(tracker.sess, cmdName, args, trackingRiskLevel, trackingFindingCodes)
		if repeatDecision.block {
			logger.Error("repeated high-risk action blocked", map[string]any{
				"command":     cmdString,
				"risk_level":  trackingRiskLevel,
				"repeat_key":  repeatDecision.key,
				"repeat_count": repeatDecision.count,
				"repeat_window_seconds": int(repeatDecision.window.Seconds()),
			})
			recordCommandAttempt(tracker, session.Command{
				Timestamp: time.Now(),
				Command:   cmdName,
				Args:      args,
				ExitCode:  3,
				Duration:  0,
				RiskLevel: trackingRiskLevel,
				Approved:  false,
				Findings:  trackingFindingCodes,
				Metadata: map[string]interface{}{
					"blocked":       true,
					"block_reason":  "repeat_protection",
					"repeat_key":    repeatDecision.key,
					"repeat_count":  repeatDecision.count,
					"repeat_window": repeatDecision.window.String(),
				},
			})
			return &exitError{
				message: fmt.Sprintf("repeated action blocked (%d times in %s). Slow down or review command intent.", repeatDecision.count, repeatDecision.window.String()),
				code:    3,
			}
		}
		if repeatDecision.warn {
			logger.Warn("repeated action warning", map[string]any{
				"command":     cmdString,
				"risk_level":  trackingRiskLevel,
				"repeat_key":  repeatDecision.key,
				"repeat_count": repeatDecision.count,
				"repeat_window_seconds": int(repeatDecision.window.Seconds()),
			})
			fmt.Fprintf(os.Stderr, "Warning: repeated action detected (%d/%d in %s). Slow down to avoid a block.\n",
				repeatDecision.count,
				repeatDecision.limit,
				repeatDecision.window.String(),
			)
		}
	}

	// Block external HTTP(S) endpoints when using vg/vectra-guard
	if hasExternalHTTP(cmdString) && os.Getenv("VECTRAGUARD_ALLOW_NET") == "" {
		logger.Error("external http(s) blocked in guarded execution", map[string]any{
			"command": cmdString,
		})
		recordCommandAttempt(tracker, session.Command{
			Timestamp: time.Now(),
			Command:   cmdName,
			Args:      args,
			ExitCode:  3,
			Duration:  0,
			RiskLevel: trackingRiskLevel,
			Approved:  false,
			Findings:  trackingFindingCodes,
			Metadata: map[string]interface{}{
				"blocked":      true,
				"block_reason": "external_http_blocked",
			},
		})
		return &exitError{
			message: "external http(s) endpoints are blocked when using vg/vectra-guard. Set VECTRAGUARD_ALLOW_NET=1 or run directly to override.",
			code:    3,
		}
	}

	// Check guard level AFTER critical command check
	// Only non-critical commands can bypass when guard level is OFF
	if cfg.GuardLevel.Level == config.GuardLevelOff {
		logger.Info("guard level is OFF - executing without protection", map[string]any{
			"command": cmdString,
		})
		start := time.Now()
		exitCode, err := executeCommandDirectly(cmdArgs)
		duration := time.Since(start)
		recordCommandAttempt(tracker, session.Command{
			Timestamp: start,
			Command:   cmdName,
			Args:      args,
			ExitCode:  exitCode,
			Duration:  duration,
			RiskLevel: trackingRiskLevel,
			Approved:  true,
			Findings:  trackingFindingCodes,
			Metadata: map[string]interface{}{
				"guard_level": "off",
				"execution":   "direct",
			},
		})
		return err
	}

	// Block sudo when using vg/vectra-guard unless explicitly allowed
	if cmdName == "sudo" {
		if os.Getenv("VECTRAGUARD_ALLOW_SUDO") == "" {
			logger.Error("sudo blocked in guarded execution", map[string]any{
				"command": cmdString,
			})
			recordCommandAttempt(tracker, session.Command{
				Timestamp: time.Now(),
				Command:   cmdName,
				Args:      args,
				ExitCode:  3,
				Duration:  0,
				RiskLevel: trackingRiskLevel,
				Approved:  false,
				Findings:  trackingFindingCodes,
				Metadata: map[string]interface{}{
					"blocked":      true,
					"block_reason": "sudo_blocked",
				},
			})
			return &exitError{
				message: "sudo is blocked when using vg/vectra-guard. Run sudo directly or set VECTRAGUARD_ALLOW_SUDO=1 to override.",
				code:    3,
			}
		}
	}

	// PRE-EXECUTION PERMISSION ASSESSMENT
	// For critical commands, enforce mandatory sandboxing BEFORE any execution
	if riskLevel == "critical" {
		criticalCodes := []string{
			"DANGEROUS_DELETE_ROOT",
			"DANGEROUS_DELETE_HOME",
			"PROTECTED_DIRECTORY_ACCESS", // Also block protected directory access (e.g., rm -rf /)
			"FORK_BOMB",
			"SENSITIVE_ENV_ACCESS",
			"DOTENV_FILE_READ",
		}
		
		hasCriticalCode := false
		for _, f := range filteredFindings {
			for _, code := range criticalCodes {
				if f.Code == code {
					hasCriticalCode = true
					break
				}
			}
			if hasCriticalCode {
				break
			}
		}
		
		if hasCriticalCode {
			// CRITICAL: These commands MUST be sandboxed - no bypass allowed
			// This check happens BEFORE bypass checks to prevent critical commands from bypassing
			logger.Error("CRITICAL command detected - mandatory sandbox required", map[string]any{
				"command":    cmdString,
				"risk_level": riskLevel,
				"findings":   findingCodes,
			})
			
			// CRITICAL commands CANNOT be bypassed - even with user bypass
			// This is a hard security requirement
			
			// Even if sandbox is disabled, we MUST enforce it for critical commands
			// This is a safety override that cannot be bypassed
			if !cfg.Sandbox.Enabled {
				recordCommandAttempt(tracker, session.Command{
					Timestamp: time.Now(),
					Command:   cmdName,
					Args:      args,
					ExitCode:  3,
					Duration:  0,
					RiskLevel: trackingRiskLevel,
					Approved:  false,
					Findings:  trackingFindingCodes,
					Metadata: map[string]interface{}{
						"blocked":      true,
						"block_reason": "sandbox_required",
						"guard_level":  cfg.GuardLevel.Level,
					},
				})
				return &exitError{
					message: fmt.Sprintf("CRITICAL: Command '%s' requires sandboxing but sandbox is disabled. Enable sandbox in config to proceed.", cmdString),
					code:    3,
				}
			}
		}
	}

	// Check for user bypass (AFTER critical command check)
	// Critical commands cannot be bypassed, but other commands can with proper authentication
	bypassEnvVar := cfg.GuardLevel.BypassEnvVar
	if bypassEnvVar == "" {
		bypassEnvVar = "VECTRAGUARD_BYPASS"
	}
	
	if cfg.GuardLevel.AllowUserBypass && os.Getenv(bypassEnvVar) != "" {
		// CRITICAL: Do not allow bypass for critical commands
		if riskLevel == "critical" {
			criticalCodes := []string{
				"DANGEROUS_DELETE_ROOT",
				"DANGEROUS_DELETE_HOME",
				"PROTECTED_DIRECTORY_ACCESS", // Also block protected directory access (e.g., rm -rf /)
				"FORK_BOMB",
				"SENSITIVE_ENV_ACCESS",
				"DOTENV_FILE_READ",
			}
			
			hasCriticalCode := false
			for _, f := range filteredFindings {
				for _, code := range criticalCodes {
					if f.Code == code {
						hasCriticalCode = true
						break
					}
				}
				if hasCriticalCode {
					break
				}
			}
			
			if hasCriticalCode {
				logger.Error("CRITICAL command cannot be bypassed", map[string]any{
					"command":    cmdString,
					"risk_level": riskLevel,
					"bypass":     "blocked for critical commands",
				})
				return &exitError{
					message: fmt.Sprintf("CRITICAL: Command '%s' cannot be bypassed. Critical commands require mandatory sandboxing.", cmdString),
					code:    3,
				}
			}
		}
		
		bypassValue := os.Getenv(bypassEnvVar)
		// Require a specific non-trivial value that agents are unlikely to guess
		// Users can set: export VECTRAGUARD_BYPASS="$(date +%s | sha256sum | head -c 16)"
		// Or a simpler approach: export VECTRAGUARD_BYPASS="i-am-human-$(whoami)"
		if len(bypassValue) >= 10 && !isLikelyAgentBypass(bypassValue) {
			logger.Info("command executed with user bypass", map[string]any{
				"command": cmdString,
				"bypass":  "user authenticated",
			})
			// Execute without protection (only for non-critical commands)
			start := time.Now()
			exitCode, err := executeCommandDirectly(cmdArgs)
			duration := time.Since(start)
			recordCommandAttempt(tracker, session.Command{
				Timestamp: start,
				Command:   cmdName,
				Args:      args,
				ExitCode:  exitCode,
				Duration:  duration,
				RiskLevel: trackingRiskLevel,
				Approved:  true,
				Findings:  trackingFindingCodes,
				Metadata: map[string]interface{}{
					"bypass":     true,
					"execution":  "direct",
					"guard_level": cfg.GuardLevel.Level,
				},
			})
			return err
		}
	}
	
	// Create sandbox executor
	executor, err := sandbox.NewExecutor(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize sandbox executor", map[string]any{
			"error": err.Error(),
		})
		
		// For critical commands, we cannot fallback to direct execution
		if riskLevel == "critical" {
			recordCommandAttempt(tracker, session.Command{
				Timestamp: time.Now(),
				Command:   cmdName,
				Args:      args,
				ExitCode:  3,
				Duration:  0,
				RiskLevel: trackingRiskLevel,
				Approved:  false,
				Findings:  trackingFindingCodes,
				Metadata: map[string]interface{}{
					"blocked":      true,
					"block_reason": "sandbox_init_failed",
					"guard_level":  cfg.GuardLevel.Level,
				},
			})
			return &exitError{
				message: fmt.Sprintf("CRITICAL: Cannot execute critical command without sandbox. Sandbox initialization failed: %v", err),
				code:    3,
			}
		}
		
		// Fallback to direct execution only for non-critical commands
		start := time.Now()
		exitCode, execErr := executeCommandDirectly(cmdArgs)
		duration := time.Since(start)
		recordCommandAttempt(tracker, session.Command{
			Timestamp: start,
			Command:   cmdName,
			Args:      args,
			ExitCode:  exitCode,
			Duration:  duration,
			RiskLevel: trackingRiskLevel,
			Approved:  true,
			Findings:  trackingFindingCodes,
			Metadata: map[string]interface{}{
				"execution":          "direct",
				"sandbox_init_failed": true,
			},
		})
		return execErr
	}
	
	// Decide execution mode (host vs sandbox)
	// Pass findings so sandbox can make informed decisions
	decision := executor.DecideExecutionMode(ctx, cmdArgs, riskLevel, filteredFindings)
	
	// Show user-friendly notice
	displayExecutionNotice(decision, riskLevel)
	
	// Execute command in chosen mode
	start := time.Now()
	err = executor.Execute(ctx, cmdArgs, decision)
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			logger.Error("command execution failed", map[string]any{
				"command": cmdString,
				"error":   err.Error(),
				"mode":    decision.Mode,
			})
			return fmt.Errorf("execute command: %w", err)
		}
	}

	recordCommandAttempt(tracker, session.Command{
		Timestamp: start,
		Command:   cmdName,
		Args:      args,
		ExitCode:  exitCode,
		Duration:  duration,
		RiskLevel: trackingRiskLevel,
		Approved:  interactive || riskLevel == "low",
		Findings:  trackingFindingCodes,
		Metadata: map[string]interface{}{
			"execution": decision.Mode,
		},
	})

	logger.Info("command executed", map[string]any{
		"command":   cmdString,
		"exit_code": exitCode,
		"duration":  duration.String(),
		"risk":      riskLevel,
	})

	if exitCode != 0 {
		return &exitError{message: fmt.Sprintf("command exited with code %d", exitCode), code: exitCode}
	}

	return nil
}

type approvalResult struct {
	approved bool
	remember bool
	duration time.Duration
}

func promptForApproval(riskLevel, cmdString string, findings []analyzer.Finding) approvalResult {
	result := approvalResult{approved: false, remember: false, duration: 0}
	
	fmt.Fprintf(os.Stderr, "\n⚠️  Command requires approval\n")
	fmt.Fprintf(os.Stderr, "Command: %s\n", cmdString)
	fmt.Fprintf(os.Stderr, "Risk Level: %s\n\n", strings.ToUpper(riskLevel))
	
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "Security concerns:\n")
		for i, f := range findings {
			fmt.Fprintf(os.Stderr, "%d. [%s] %s\n", i+1, f.Code, f.Description)
			fmt.Fprintf(os.Stderr, "   Recommendation: %s\n", f.Recommendation)
		}
		fmt.Fprintln(os.Stderr)
	}

	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  y  - Yes, run once\n")
	fmt.Fprintf(os.Stderr, "  r  - Yes, and remember (trust permanently)\n")
	fmt.Fprintf(os.Stderr, "  n  - No, cancel\n")
	fmt.Fprintf(os.Stderr, "\nChoose [y/r/N]: ")
	
	var response string
	fmt.Scanln(&response)
	
	response = strings.ToLower(strings.TrimSpace(response))
	
	switch response {
	case "y", "yes":
		result.approved = true
	case "r", "remember":
		result.approved = true
		result.remember = true
	default:
		result.approved = false
	}
	
	return result
}

// filterFindingsByGuardLevel filters findings based on the configured guard level
func filterFindingsByGuardLevel(findings []analyzer.Finding, level config.GuardLevel) []analyzer.Finding {
	if level == config.GuardLevelOff {
		return nil
	}
	
	if level == config.GuardLevelParanoid {
		return findings // Return all findings
	}
	
	var filtered []analyzer.Finding
	for _, f := range findings {
		switch level {
		case config.GuardLevelLow:
			// Only critical
			if f.Severity == "critical" {
				filtered = append(filtered, f)
			}
		case config.GuardLevelMedium:
			// Critical and high
			if f.Severity == "critical" || f.Severity == "high" {
				filtered = append(filtered, f)
			}
		case config.GuardLevelHigh:
			// Critical, high, and medium
			if f.Severity == "critical" || f.Severity == "high" || f.Severity == "medium" {
				filtered = append(filtered, f)
			}
		case config.GuardLevelAuto:
			// Auto mode: treat as medium for filtering (conservative)
			if f.Severity == "critical" || f.Severity == "high" {
				filtered = append(filtered, f)
			}
		default:
			// Unknown guard level - be conservative and include all findings
			// This handles edge cases where level might not match expected constants
			filtered = append(filtered, f)
		}
	}
	
	return filtered
}

// shouldRequireApproval determines if a command should require approval
func shouldRequireApproval(riskLevel string, guardLevel config.GuardLevel) bool {
	if guardLevel == config.GuardLevelParanoid {
		return true // Everything requires approval
	}
	
	switch guardLevel {
	case config.GuardLevelLow:
		return riskLevel == "critical"
	case config.GuardLevelMedium:
		return riskLevel == "critical" || riskLevel == "high"
	case config.GuardLevelHigh:
		return riskLevel == "critical" || riskLevel == "high" || riskLevel == "medium"
	case config.GuardLevelAuto:
		// Auto mode: treat as medium (conservative)
		return riskLevel == "critical" || riskLevel == "high"
	default:
		return false
	}
}

// isLikelyAgentBypass checks if the bypass value looks like it was set by an AI agent
func isLikelyAgentBypass(value string) bool {
	// Simple heuristics to detect agent-generated bypass values
	agentPatterns := []string{
		"bypass", "agent", "ai", "automated", "script",
		"cursor", "copilot", "gpt", "claude",
		"true", "yes", "1", "enabled",
	}
	
	lowerValue := strings.ToLower(value)
	for _, pattern := range agentPatterns {
		if strings.Contains(lowerValue, pattern) {
			return true
		}
	}
	
	// If it's too simple (less than 10 chars), likely not a proper bypass
	if len(value) < 10 {
		return true
	}
	
	return false
}

// displayExecutionNotice shows a user-friendly notice about execution mode
func displayExecutionNotice(decision sandbox.ExecutionDecision, riskLevel string) {
	if decision.Mode == sandbox.ExecutionModeHost {
		// Only show notice for interesting cases
		if riskLevel != "low" && decision.Reason != "" {
			fmt.Fprintf(os.Stderr, "🏠 Running on host: %s\n", decision.Reason)
		}
		return
	}
	
	// Sandbox execution - always inform user
	notice := "📦 Running in sandbox"
	
	if decision.ShouldCache {
		notice += " (cached)"
	}
	
	notice += "."
	
	// Explain the "why"
	if decision.Reason != "" {
		reasonParts := strings.Split(decision.Reason, "+")
		if len(reasonParts) > 1 {
			// Multiple reasons: format nicely
			notice += fmt.Sprintf("\n   Why: %s", strings.TrimSpace(reasonParts[0]))
			for _, part := range reasonParts[1:] {
				notice += fmt.Sprintf(" + %s", strings.TrimSpace(part))
			}
		} else {
			notice += fmt.Sprintf("\n   Why: %s", decision.Reason)
		}
	}
	
	fmt.Fprintln(os.Stderr, notice)
}

type sessionTracker struct {
	id   string
	mgr  *session.Manager
	sess *session.Session
}

func initSessionTracker(sessionID string, logger *logging.Logger) *sessionTracker {
	if sessionID == "" {
		workspace, err := os.Getwd()
		if err != nil {
			return nil
		}
		sessionID = session.GetCurrentSessionForWorkspace(workspace)
	}
	if sessionID == "" {
		return nil
	}

	workspace, err := os.Getwd()
	if err != nil {
		return nil
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return nil
	}

	sess, err := mgr.Load(sessionID)
	if err != nil {
		return nil
	}

	return &sessionTracker{
		id:   sessionID,
		mgr:  mgr,
		sess: sess,
	}
}

func recordCommandAttempt(tracker *sessionTracker, cmd session.Command) {
	if tracker == nil {
		return
	}
	_ = tracker.mgr.AddCommand(tracker.sess, cmd)
}

func summarizeFindings(findings []analyzer.Finding) (string, []string) {
	riskLevel := "low"
	var codes []string
	for _, f := range findings {
		codes = append(codes, f.Code)
		switch f.Severity {
		case "critical":
			riskLevel = "critical"
		case "high":
			if riskLevel != "critical" {
				riskLevel = "high"
			}
		case "medium":
			if riskLevel != "critical" && riskLevel != "high" {
				riskLevel = "medium"
			}
		}
	}
	return riskLevel, codes
}

type repeatDecision struct {
	block  bool
	warn   bool
	count  int
	limit  int
	window time.Duration
	key    string
}

func evaluateRepeatProtection(sess *session.Session, cmdName string, args []string, riskLevel string, findingCodes []string) repeatDecision {
	const (
		repeatWindow    = 30 * time.Second
		repeatMaxHigh   = 3
		repeatMaxMedium = 5
	)

	if sess == nil {
		return repeatDecision{}
	}

	sensitive := isRepeatSensitiveCommand(cmdName) || hasSensitiveFinding(findingCodes)
	threshold := 0
	switch riskLevel {
	case "critical", "high":
		threshold = repeatMaxHigh
	case "medium":
		threshold = repeatMaxMedium
	default:
		if sensitive {
			threshold = repeatMaxMedium
		}
	}

	if threshold == 0 {
		return repeatDecision{}
	}

	now := time.Now()
	key := normalizeCommand(cmdName, args)
	if sensitive {
		key = cmdName
	}

	count := 0
	for _, cmd := range sess.Commands {
		if now.Sub(cmd.Timestamp) > repeatWindow {
			continue
		}
		if sensitive {
			if cmd.Command == cmdName {
				count++
			}
			continue
		}
		if normalizeCommand(cmd.Command, cmd.Args) == key {
			count++
		}
	}

	if count+1 > threshold {
		return repeatDecision{
			block:  true,
			count:  count + 1,
			limit:  threshold,
			window: repeatWindow,
			key:    key,
		}
	}

	warn := count+1 == threshold
	return repeatDecision{
		block:  false,
		warn:   warn,
		count:  count + 1,
		limit:  threshold,
		window: repeatWindow,
		key:    key,
	}
}

func isRepeatSensitiveCommand(cmdName string) bool {
	switch cmdName {
	case "rm", "mv", "cp", "chmod", "chown", "dd", "mkfs":
		return true
	default:
		return false
	}
}

func hasSensitiveFinding(codes []string) bool {
	sensitiveCodes := map[string]struct{}{
		"DANGEROUS_DELETE_ROOT":     {},
		"DANGEROUS_DELETE_HOME":     {},
		"PROTECTED_DIRECTORY_ACCESS": {},
		"FORK_BOMB":                 {},
		"SENSITIVE_ENV_ACCESS":      {},
		"DOTENV_FILE_READ":          {},
		"PRIVATE_KEY_ACCESS":        {},
	}
	for _, code := range codes {
		if _, ok := sensitiveCodes[code]; ok {
			return true
		}
	}
	return false
}

func normalizeCommand(cmdName string, args []string) string {
	parts := append([]string{cmdName}, args...)
	return strings.Join(parts, " ")
}

var urlPattern = regexp.MustCompile(`https?://[^\s'"]+`)

func hasExternalHTTP(cmdString string) bool {
	matches := urlPattern.FindAllString(cmdString, -1)
	for _, raw := range matches {
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := parsed.Hostname()
		if host == "" {
			continue
		}
		if isLocalHost(host) {
			continue
		}
		return true
	}
	return false
}

func isLocalHost(host string) bool {
	if host == "localhost" || host == "::1" || host == "127.0.0.1" {
		return true
	}
	return strings.HasPrefix(host, "127.")
}

// executeCommandDirectly executes a command without protection
func executeCommandDirectly(cmdArgs []string) (int, error) {
	if len(cmdArgs) == 0 {
		return 1, fmt.Errorf("no command specified")
	}
	
	cmdName := cmdArgs[0]
	args := cmdArgs[1:]
	
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), &exitError{
				message: fmt.Sprintf("command exited with code %d", exitErr.ExitCode()),
				code:    exitErr.ExitCode(),
			}
		}
		return 1, fmt.Errorf("execute command: %w", err)
	}
	
	return 0, nil
}

