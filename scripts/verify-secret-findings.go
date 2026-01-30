// verify-secret-findings runs secret scan on each repo, samples random findings,
// verifies each by reading the file at the reported line, and assesses whether
// the finding is a true issue (real secret/credential) vs false positive.
//
// Usage: go run scripts/verify-secret-findings.go
// Or:   go run scripts/verify-secret-findings.go /path/to/repo1
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/secrets"
)

func main() {
	root := "test-workspaces"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	repos, err := listRepos(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list repos: %v\n", err)
		os.Exit(1)
	}

	// Fixed seed for reproducible "random" sampling
	rng := rand.New(rand.NewSource(42))

	var failed int
	for _, repoPath := range repos {
		fmt.Printf("\n=== %s ===\n", filepath.Base(repoPath))
		findings, err := secrets.ScanPath(repoPath, secrets.Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  scan error: %v\n", err)
			failed++
			continue
		}
		if len(findings) == 0 {
			fmt.Println("  (no secret findings)")
			continue
		}

		// Prefer known-pattern findings (true issues) when present, then fill with random.
		knownPattern := make([]int, 0)
		rest := make([]int, 0)
		for i := range findings {
			switch findings[i].PatternID {
			case "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "GENERIC_API_KEY", "PRIVATE_KEY_BLOCK":
				knownPattern = append(knownPattern, i)
			default:
				rest = append(rest, i)
			}
		}
		// Sample: up to 2 known-pattern (true issues), then fill to 3 total with random rest
		n := 3
		if len(findings) < n {
			n = len(findings)
		}
		var indices []int
		for _, i := range knownPattern {
			if len(indices) >= n {
				break
			}
			indices = append(indices, i)
		}
		if len(indices) < n && len(rest) > 0 {
			perm := rng.Perm(len(rest))
			for _, p := range perm {
				if len(indices) >= n {
					break
				}
				indices = append(indices, rest[p])
			}
		}
		perm := indices
		for _, i := range perm {
			f := findings[i]
			line, present := getLine(f.File, f.Line)
			ok := present && strings.Contains(line, f.Match)
			trueIssue := isLikelyTrueIssue(f, line)
			status := "OK"
			if !ok {
				status = "FAIL"
				failed++
			}
			issueLabel := "FP"
			if trueIssue {
				issueLabel = "TRUE_ISSUE"
			}
			rel, _ := filepath.Rel(repoPath, f.File)
			if rel == "" || strings.HasPrefix(rel, "..") {
				rel = f.File
			}
			matchPreview := f.Match
			if len(matchPreview) > 44 {
				matchPreview = matchPreview[:41] + "..."
			}
			linePreview := strings.TrimSpace(line)
			if len(linePreview) > 60 {
				linePreview = linePreview[:57] + "..."
			}
			fmt.Printf("  [%s] %s %s:%d pattern=%s match=%q\n", status, issueLabel, rel, f.Line, f.PatternID, matchPreview)
			fmt.Printf("      line: %q\n", linePreview)
		}
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d verification(s) failed\n", failed)
		os.Exit(1)
	}
	fmt.Println("\nAll sampled findings verified.")
}

func listRepos(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		paths = append(paths, filepath.Join(root, name))
	}
	return paths, nil
}

// getLine returns the line at the given 1-based line number, and whether it was found.
func getLine(filePath string, lineNum int) (string, bool) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var n int
	for scanner.Scan() {
		n++
		if n == lineNum {
			return scanner.Text(), true
		}
	}
	return "", false
}

// isLikelyTrueIssue returns true if the finding is specified as a real secret/credential
// that should be fixed, vs a false positive (path, model ID, config key, etc.).
func isLikelyTrueIssue(f secrets.Finding, line string) bool {
	// Known-pattern findings: scanner explicitly flags these as secrets.
	switch f.PatternID {
	case "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "GENERIC_API_KEY", "PRIVATE_KEY_BLOCK":
		return true
	}
	// ENTROPY_CANDIDATE: heuristics for likely false positives.
	if f.PatternID != "ENTROPY_CANDIDATE" {
		return false
	}
	m := strings.ToLower(f.Match)
	lineLower := strings.ToLower(line)
	// Path-like: URLs, import paths, file paths
	if strings.Contains(m, ".com/") || strings.Contains(m, "github.com") ||
		strings.Contains(m, "/") && (strings.HasPrefix(m, "com/") || strings.Contains(m, "site-packages/")) {
		return false
	}
	// Model/provider IDs (common in config, not credentials by themselves)
	if strings.Contains(m, "openai") || strings.Contains(m, "anthropic") ||
		strings.Contains(m, "together_ai") || strings.Contains(m, "openrouter") ||
		strings.Contains(lineLower, "model") && (strings.Contains(m, "claude") || strings.Contains(m, "gpt-")) {
		return false
	}
	// Test/code identifiers: TestXxx, xxx_test, CamelCase type names
	if regexp.MustCompile(`^test[A-Z]|_test$|_test/`).MatchString(m) ||
		regexp.MustCompile(`[A-Z][a-z]+[A-Z]`).MatchString(m) && len(m) < 45 {
		return false
	}
	// Config/key names (snake_case), constants (UPPER_SNAKE)
	if regexp.MustCompile(`^[a-z][a-z0-9_]{10,}$`).MatchString(m) || regexp.MustCompile(`^[A-Z][A-Z0-9_]{10,}$`).MatchString(m) {
		return false
	}
	// Long opaque token (base64-ish, hex-ish) with no path/model flavor → possible true issue
	if len(f.Match) >= 32 && !strings.Contains(m, "/") && !strings.Contains(m, "_") {
		return true
	}
	return false
}
