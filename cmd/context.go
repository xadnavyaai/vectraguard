package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/summarizer"
)

type contextCacheEntry struct {
	Path    string   `json:"path"`
	Mode    string   `json:"mode"`
	Summary []string `json:"summary"`
}

type contextSummaryOutput struct {
	Mode      string              `json:"mode"`
	Path      string              `json:"path"`
	FileCount int                 `json:"fileCount,omitempty"`
	Files     []fileSummaryOutput `json:"files,omitempty"`
	Summary   []string            `json:"summary,omitempty"`
}

type fileSummaryOutput struct {
	Path    string   `json:"path"`
	Summary []string `json:"summary"`
}

func runContextSummarize(ctx context.Context, mode, path string, maxItems int, outputFormat, since string) error {
	logger := logging.FromContext(ctx)
	mode = strings.ToLower(mode)
	outputFormat = strings.ToLower(outputFormat)

	// Check if path is a directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		return runContextSummarizeRepo(ctx, mode, path, maxItems, outputFormat, since, logger)
	}

	// Single file mode
	return runContextSummarizeFile(ctx, mode, path, maxItems, outputFormat, logger)
}

func runContextSummarizeFile(ctx context.Context, mode, filePath string, maxItems int, outputFormat string, logger *logging.Logger) error {
	// Try to load from cache first
	cacheDir := getContextCacheDir()
	if cacheDir != "" {
		if cached, err := loadFromCache(cacheDir, filePath, mode); err == nil && cached != nil {
			logger.Info("using cached summary", map[string]any{"path": filePath, "mode": mode})
			return outputSummary(filePath, mode, []fileSummaryOutput{
				{Path: filePath, Summary: cached.Summary},
			}, outputFormat, 1)
		}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

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
		logger.Info("summary produced no highlights", map[string]any{"path": filePath, "mode": mode})
		return outputSummary(filePath, mode, []fileSummaryOutput{}, outputFormat, 0)
	}

	// Save to cache
	if cacheDir != "" {
		if err := saveToCache(cacheDir, filePath, mode, summary); err != nil {
			logger.Info("failed to save cache", map[string]any{"error": err.Error()})
		}
	}

	// Output results
	return outputSummary(filePath, mode, []fileSummaryOutput{
		{Path: filePath, Summary: summary},
	}, outputFormat, 1)
}

func runContextSummarizeRepo(ctx context.Context, mode, repoPath string, maxItems int, outputFormat, since string, logger *logging.Logger) error {
	// Find repo root (look for .git or .vectra-guard)
	repoRoot := findRepoRoot(repoPath)
	if repoRoot == "" {
		repoRoot = repoPath
	}

	cacheDir := getContextCacheDir()
	if cacheDir == "" {
		// Try to create cache dir if .vectra-guard exists
		if hasVectraGuardDir(repoRoot) {
			cacheDir = filepath.Join(repoRoot, ".vectra-guard", "cache", "context-summaries")
			if err := os.MkdirAll(cacheDir, 0o755); err == nil {
				logger.Info("created context cache directory", map[string]any{"path": cacheDir})
			}
		}
	}

	// Get changed files if --since is specified
	var changedFiles map[string]bool
	if since != "" {
		var err error
		changedFiles, err = getChangedFiles(repoRoot, since)
		if err != nil {
			logger.Info("failed to get changed files, processing all", map[string]any{"error": err.Error()})
			changedFiles = nil
		} else if len(changedFiles) > 0 {
			logger.Info("filtering to changed files", map[string]any{"count": len(changedFiles), "since": since})
		}
	}

	// Collect all relevant files
	var files []string
	err := filepath.Walk(repoPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			// Skip hidden directories and common ignore patterns
			base := filepath.Base(p)
			if strings.HasPrefix(base, ".") && base != "." && base != ".." {
				if base == ".git" || base == ".vectra-guard" || base == "node_modules" || base == "vendor" {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Determine file type and mode
		if shouldProcessFile(p, mode) {
			// If filtering by changes, check if file was changed
			if changedFiles != nil {
				relPath, _ := filepath.Rel(repoRoot, p)
				if !changedFiles[relPath] && !changedFiles[p] {
					return nil // Skip unchanged files
				}
			}
			files = append(files, p)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	if len(files) == 0 {
		logger.Info("no files found to summarize", map[string]any{"path": repoPath, "mode": mode, "since": since})
		return outputSummary(repoPath, mode, []fileSummaryOutput{}, outputFormat, 0)
	}

	logger.Info("summarizing repository", map[string]any{
		"path":      repoPath,
		"mode":      mode,
		"fileCount": len(files),
		"since":     since,
	})

	// Process each file
	allSummaries := make([]fileSummaryOutput, 0, len(files))
	for _, file := range files {
		relPath, _ := filepath.Rel(repoPath, file)
		if relPath == "" {
			relPath = filepath.Base(file)
		}

		// Try cache first
		var summary []string
		if cacheDir != "" {
			if cached, err := loadFromCache(cacheDir, file, mode); err == nil && cached != nil {
				summary = cached.Summary
			}
		}

		// If not cached, compute
		if summary == nil {
			content, err := os.ReadFile(file)
			if err != nil {
				logger.Info("skipping file", map[string]any{"path": file, "error": err.Error()})
				continue
			}

			switch mode {
			case "code":
				summary = summarizer.SummarizeCode(string(content), maxItems)
			case "advanced":
				summary = summarizer.SummarizeCodeAdvanced(string(content), maxItems)
			case "docs", "doc", "text":
				summary = summarizer.SummarizeText(string(content), maxItems)
			}

			// Save to cache
			if cacheDir != "" && len(summary) > 0 {
				if err := saveToCache(cacheDir, file, mode, summary); err != nil {
					logger.Info("failed to save cache", map[string]any{"path": file, "error": err.Error()})
				}
			}
		}

		if len(summary) > 0 {
			allSummaries = append(allSummaries, fileSummaryOutput{
				Path:    relPath,
				Summary: summary,
			})
		}
	}

	// Output results
	return outputSummary(repoPath, mode, allSummaries, outputFormat, len(allSummaries))
}

func outputSummary(path, mode string, files []fileSummaryOutput, outputFormat string, fileCount int) error {
	if outputFormat == "json" {
		output := contextSummaryOutput{
			Mode:      mode,
			Path:      path,
			FileCount: fileCount,
			Files:     files,
		}
		// For single file, also include summary at top level
		if len(files) == 1 {
			output.Summary = files[0].Summary
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Text output
	if len(files) == 0 {
		return nil
	}

	if len(files) == 1 {
		// Single file output (simple)
		for _, line := range files[0].Summary {
			fmt.Println(line)
		}
		return nil
	}

	// Multiple files output (grouped)
	for _, file := range files {
		fmt.Printf("\n📄 %s\n", file.Path)
		fmt.Println(strings.Repeat("─", 60))
		for _, line := range file.Summary {
			fmt.Printf("  %s\n", line)
		}
	}
	return nil
}

func getChangedFiles(repoRoot, since string) (map[string]bool, error) {
	// Check if we're in a git repo
	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	// Try to parse as time first
	var sinceArg string
	if t, err := time.Parse("2006-01-02", since); err == nil {
		// Date format - use --since flag
		sinceArg = fmt.Sprintf("--since=%s", t.Format("2006-01-02"))
	} else if t, err := time.Parse(time.RFC3339, since); err == nil {
		// RFC3339 format
		sinceArg = fmt.Sprintf("--since=%s", t.Format(time.RFC3339))
	} else {
		// Assume it's a commit reference (HEAD~1, commit hash, etc.)
		sinceArg = since
	}

	// Run git diff to get changed files
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=ACMR", sinceArg)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		// Try as commit range
		cmd = exec.Command("git", "diff", "--name-only", "--diff-filter=ACMR", sinceArg+"..HEAD")
		cmd.Dir = repoRoot
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git diff failed: %w", err)
		}
	}

	changedFiles := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			changedFiles[line] = true
			// Also add absolute path version
			absPath := filepath.Join(repoRoot, line)
			changedFiles[absPath] = true
		}
	}

	return changedFiles, nil
}

func shouldProcessFile(filePath string, mode string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	base := filepath.Base(filePath)

	// Skip binary and common ignore patterns
	if strings.HasPrefix(base, ".") {
		return false
	}

	switch mode {
	case "code", "advanced":
		// Code files
		codeExts := []string{".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".h", ".rs", ".rb", ".php", ".swift", ".kt", ".scala", ".sh", ".bash", ".zsh", ".fish"}
		for _, e := range codeExts {
			if ext == e {
				return true
			}
		}
		// Also check for common code file names
		if base == "Makefile" || base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.") {
			return true
		}
		return false

	case "docs", "doc", "text":
		// Documentation files
		docExts := []string{".md", ".txt", ".rst", ".adoc", ".org", ".tex"}
		for _, e := range docExts {
			if ext == e {
				return true
			}
		}
		// Common doc file names
		if strings.EqualFold(base, "readme") || strings.EqualFold(base, "license") || strings.EqualFold(base, "changelog") {
			return true
		}
		return false

	default:
		return false
	}
}

func findRepoRoot(startPath string) string {
	dir := startPath
	for {
		gitDir := filepath.Join(dir, ".git")
		vectraDir := filepath.Join(dir, ".vectra-guard")
		if _, err := os.Stat(gitDir); err == nil {
			return dir
		}
		if _, err := os.Stat(vectraDir); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func hasVectraGuardDir(path string) bool {
	vectraDir := filepath.Join(path, ".vectra-guard")
	_, err := os.Stat(vectraDir)
	return err == nil
}

func getContextCacheDir() string {
	// First check if we're in a repo with .vectra-guard
	if cwd, err := os.Getwd(); err == nil {
		repoRoot := findRepoRoot(cwd)
		if repoRoot != "" {
			cacheDir := filepath.Join(repoRoot, ".vectra-guard", "cache", "context-summaries")
			if _, err := os.Stat(cacheDir); err == nil {
				return cacheDir
			}
			// Try to create it
			if hasVectraGuardDir(repoRoot) {
				if err := os.MkdirAll(cacheDir, 0o755); err == nil {
					return cacheDir
				}
			}
		}
	}

	// Fallback to global cache
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	cacheDir := filepath.Join(home, ".vectra-guard", "cache", "context-summaries")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return ""
	}
	return cacheDir
}

func cacheKey(filePath, mode string) string {
	hash := sha256.Sum256([]byte(filePath + ":" + mode))
	return hex.EncodeToString(hash[:16]) + ".json"
}

func loadFromCache(cacheDir, filePath, mode string) (*contextCacheEntry, error) {
	key := cacheKey(filePath, mode)
	cacheFile := filepath.Join(cacheDir, key)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var entry contextCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Verify the path matches (in case of hash collision)
	if entry.Path != filePath || entry.Mode != mode {
		return nil, fmt.Errorf("cache entry mismatch")
	}

	return &entry, nil
}

func saveToCache(cacheDir, filePath, mode string, summary []string) error {
	key := cacheKey(filePath, mode)
	cacheFile := filepath.Join(cacheDir, key)

	entry := contextCacheEntry{
		Path:    filePath,
		Mode:    mode,
		Summary: summary,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0o644)
}
