package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func runExec(ctx context.Context, cmdArgs []string, interactive bool, sessionID string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Check for user bypass
	bypassEnvVar := cfg.GuardLevel.BypassEnvVar
	if bypassEnvVar == "" {
		bypassEnvVar = "VECTRAGUARD_BYPASS"
	}
	
	if cfg.GuardLevel.AllowUserBypass && os.Getenv(bypassEnvVar) != "" {
		bypassValue := os.Getenv(bypassEnvVar)
		// Require a specific non-trivial value that agents are unlikely to guess
		// Users can set: export VECTRAGUARD_BYPASS="$(date +%s | sha256sum | head -c 16)"
		// Or a simpler approach: export VECTRAGUARD_BYPASS="i-am-human-$(whoami)"
		if len(bypassValue) >= 10 && !isLikelyAgentBypass(bypassValue) {
			logger.Info("command executed with user bypass", map[string]any{
				"command": strings.Join(cmdArgs, " "),
				"bypass":  "user authenticated",
			})
			// Execute without protection
			return executeCommandDirectly(cmdArgs)
		}
	}
	
	// Check guard level
	if cfg.GuardLevel.Level == config.GuardLevelOff {
		logger.Info("guard level is OFF - executing without protection", map[string]any{
			"command": strings.Join(cmdArgs, " "),
		})
		return executeCommandDirectly(cmdArgs)
	}

	cmdName := cmdArgs[0]
	args := cmdArgs[1:]

	// Build command string for analysis
	cmdString := strings.Join(cmdArgs, " ")

	// Analyze command for risks
	findings := analyzer.AnalyzeScript("inline-command", []byte(cmdString), cfg.Policies)
	
	riskLevel := "low"
	var findingCodes []string
	
	// Filter findings based on guard level
	filteredFindings := filterFindingsByGuardLevel(findings, cfg.GuardLevel.Level)
	
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

		// Determine if approval is required based on guard level
		requiresApproval := shouldRequireApproval(riskLevel, cfg.GuardLevel.Level)
		
		// Handle interactive approval or blocking
		if requiresApproval {
			if interactive {
				if !promptForApproval(riskLevel, cmdString, filteredFindings) {
					logger.Info("command execution denied by user", map[string]any{
						"command": cmdString,
					})
					return &exitError{message: "execution denied", code: 3}
				}
			} else {
				logger.Error("risky command blocked", map[string]any{
					"command":    cmdString,
					"risk_level": riskLevel,
					"guard_level": cfg.GuardLevel.Level,
				})
				return &exitError{
					message: fmt.Sprintf("%s risk command blocked by guard level %s (use --interactive to approve, or set bypass)", 
						riskLevel, cfg.GuardLevel.Level),
					code: 3,
				}
			}
		}
	}

	// Execute command
	start := time.Now()
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			logger.Error("command execution failed", map[string]any{
				"command": cmdString,
				"error":   err.Error(),
			})
			return fmt.Errorf("execute command: %w", err)
		}
	}

	// Track in session if available
	if sessionID != "" || session.GetCurrentSession() != "" {
		if sessionID == "" {
			sessionID = session.GetCurrentSession()
		}
		
		workspace, _ := os.Getwd()
		mgr, err := session.NewManager(workspace, logger)
		if err == nil {
			sess, err := mgr.Load(sessionID)
			if err == nil {
				cmdRecord := session.Command{
					Timestamp: start,
					Command:   cmdName,
					Args:      args,
					ExitCode:  exitCode,
					Duration:  duration,
					RiskLevel: riskLevel,
					Approved:  interactive || riskLevel == "low",
					Findings:  findingCodes,
				}
				_ = mgr.AddCommand(sess, cmdRecord)
			}
		}
	}

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

func promptForApproval(riskLevel, cmdString string, findings []analyzer.Finding) bool {
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

	fmt.Fprintf(os.Stderr, "Do you want to proceed? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
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

// executeCommandDirectly executes a command without protection
func executeCommandDirectly(cmdArgs []string) error {
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
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
			return &exitError{
				message: fmt.Sprintf("command exited with code %d", exitErr.ExitCode()),
				code:    exitErr.ExitCode(),
			}
		}
		return fmt.Errorf("execute command: %w", err)
	}
	
	return nil
}

