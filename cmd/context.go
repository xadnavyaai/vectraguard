package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/summarizer"
)

func runContextSummarize(ctx context.Context, mode, path string, maxItems int) error {
	logger := logging.FromContext(ctx)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	mode = strings.ToLower(mode)
	var summary []string
	switch mode {
	case "code":
		summary = summarizer.SummarizeCode(string(content), maxItems)
	case "advanced":
		summary = summarizer.SummarizeCodeAdvanced(string(content), maxItems)
	case "docs", "doc", "text":
		summary = summarizer.SummarizeText(string(content), maxItems)
	default:
		return fmt.Errorf("unknown summarize mode: %s", mode)
	}

	if len(summary) == 0 {
		logger.Info("summary produced no highlights", map[string]any{"path": path, "mode": mode})
		return nil
	}

	for _, line := range summary {
		logger.Info("summary line", map[string]any{"path": path, "mode": mode, "summary": line})
	}
	return nil
}
