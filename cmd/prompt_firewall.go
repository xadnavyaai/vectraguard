package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/promptfw"
)

func runPromptFirewall(ctx context.Context, fromFile string) error {
	logger := logging.FromContext(ctx)

	text, err := readPromptInput(fromFile)
	if err != nil {
		return err
	}

	result := promptfw.Analyze(text)

	fields := map[string]any{
		"risk_level": result.RiskLevel,
		"score":      fmt.Sprintf("%.2f", result.Score),
	}
	if len(result.Reasons) > 0 {
		fields["reasons"] = result.Reasons
	}

	switch result.RiskLevel {
	case "high":
		logger.Warn("prompt blocked: malicious or risky instructions detected", fields)
		return &exitError{message: "prompt blocked by firewall", code: 2}
	case "medium":
		logger.Warn("prompt flagged: potentially risky instructions", fields)
	default:
		logger.Info("prompt allowed", fields)
	}

	return nil
}

func readPromptInput(fromFile string) (string, error) {
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return "", fmt.Errorf("read prompt file: %w", err)
		}
		return string(data), nil
	}

	// Read from stdin until EOF.
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("stat stdin: %w", err)
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no prompt provided on stdin and no --file specified")
	}

	var b strings.Builder
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		b.WriteString(line)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("read stdin: %w", err)
		}
	}
	return b.String(), nil
}

