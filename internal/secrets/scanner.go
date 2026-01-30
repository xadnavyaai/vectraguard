package secrets

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Finding represents a potential secret detected in a file.
type Finding struct {
	File      string  `json:"file"`
	Line      int     `json:"line"`
	Match     string  `json:"match"`
	PatternID string  `json:"pattern_id"`
	Entropy   float64 `json:"entropy"`
	Severity  string  `json:"severity"`
}

// Options controls how scanning is performed.
type Options struct {
	// Allowlist contains exact values that should be ignored even if they match
	// a detector (e.g., known test keys).
	Allowlist map[string]struct{}
	// IgnorePaths is a list of path globs or directory prefixes (relative to the
	// scan root). If any pattern matches a file's path, that file is skipped.
	// Patterns ending with "/" match any file under that directory prefix.
	// Otherwise filepath.Match is used (e.g. "*.min.js", "vendor/*").
	IgnorePaths []string
}

var (
	// Known secret detectors. These are intentionally conservative and can be
	// extended over time.
	secretDetectors = []struct {
		id   string
		re   *regexp.Regexp
		crit bool
	}{
		{
			id:   "AWS_ACCESS_KEY_ID",
			re:   regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			crit: true,
		},
		{
			id:   "AWS_SECRET_ACCESS_KEY",
			re:   regexp.MustCompile(`(?i)aws_secret_access_key['"\s:=]+([A-Za-z0-9/+=]{40})`),
			crit: true,
		},
		{
			id:   "GENERIC_API_KEY",
			re:   regexp.MustCompile(`(?i)(api[_-]?key|token|secret)['"\s:=]+([A-Za-z0-9_\-]{20,})`),
			crit: true,
		},
		{
			id:   "PRIVATE_KEY_BLOCK",
			re:   regexp.MustCompile(`-----BEGIN (RSA |DSA |EC |OPENSSH )?PRIVATE KEY-----`),
			crit: true,
		},
	}

	// Entropy candidate: long, mostly random-looking token.
	entropyCandidateRe = regexp.MustCompile(`([A-Za-z0-9+/=_-]{20,})`)
)

// ScanPath walks the given path (file or directory) and returns all detected
// secret findings. It is safe for use on large trees; binary files and common
// vendor directories are skipped.
func ScanPath(root string, opts Options) ([]Finding, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}

	var findings []Finding

	if !info.IsDir() {
		if shouldIgnorePath(filepath.Base(root), opts.IgnorePaths) {
			return nil, nil
		}
		fs, err := scanFile(root, opts)
		if err != nil {
			return nil, err
		}
		return fs, nil
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Best-effort: skip unreadable paths but continue.
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipFile(path) {
			return nil
		}
		rel, errRel := filepath.Rel(root, path)
		if errRel == nil && shouldIgnorePath(filepath.ToSlash(rel), opts.IgnorePaths) {
			return nil
		}
		fs, err := scanFile(path, opts)
		if err != nil {
			// Best-effort: loggable upstream; do not abort entire scan.
			return nil
		}
		findings = append(findings, fs...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return findings, nil
}

func scanFile(path string, opts Options) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	if isProbablyBinary(f) {
		return nil, nil
	}

	var findings []Finding
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Run explicit detectors first.
		for _, det := range secretDetectors {
			matches := det.re.FindAllStringSubmatch(line, -1)
			if len(matches) == 0 {
				continue
			}
			for _, m := range matches {
				matchVal := m[0]
				// If regex has capture groups, prefer the first non-empty group as the value.
				if len(m) > 1 && m[1] != "" {
					matchVal = m[1]
				}
				if isAllowlisted(matchVal, opts.Allowlist) {
					continue
				}
				entropy := shannonEntropy(matchVal)
				severity := "high"
				if det.crit {
					severity = "critical"
				}
				findings = append(findings, Finding{
					File:      path,
					Line:      lineNum,
					Match:     matchVal,
					PatternID: det.id,
					Entropy:   entropy,
					Severity:  severity,
				})
			}
		}

		// Entropy-based fallback: look for long random-looking tokens.
		candidates := entropyCandidateRe.FindAllString(line, -1)
		for _, cand := range candidates {
			if isAllowlisted(cand, opts.Allowlist) {
				continue
			}
			entropy := shannonEntropy(cand)
			// Threshold chosen based on typical secret scanners; tuneable.
			if entropy < 3.5 {
				continue
			}
			findings = append(findings, Finding{
				File:      path,
				Line:      lineNum,
				Match:     cand,
				PatternID: "ENTROPY_CANDIDATE",
				Entropy:   entropy,
				Severity:  "medium",
			})
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return findings, nil
}

// shouldIgnorePath returns true if relPath (slash-separated, relative to scan root)
// matches any of the given patterns. Patterns ending with "/" are directory prefixes;
// others are matched with filepath.Match against relPath and the base name.
func shouldIgnorePath(relPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	relPath = filepath.ToSlash(relPath)
	base := filepath.Base(relPath)
	for _, p := range patterns {
		p = strings.TrimSpace(filepath.ToSlash(p))
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/") {
			prefix := strings.TrimSuffix(p, "/")
			if prefix == "" || relPath == prefix || strings.HasPrefix(relPath, prefix+"/") {
				return true
			}
			continue
		}
		matched, _ := filepath.Match(p, relPath)
		if matched {
			return true
		}
		matched, _ = filepath.Match(p, base)
		if matched {
			return true
		}
	}
	return false
}

func shouldSkipDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".hg", ".svn", ".vectra-guard", "node_modules", "vendor", "dist", "build", ".venv", "venv":
		return true
	default:
		return false
	}
}

func shouldSkipFile(path string) bool {
	// Skip obviously irrelevant or huge files.
	lower := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".gif") ||
		strings.HasSuffix(lower, ".pdf") ||
		strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".gz") ||
		strings.HasSuffix(lower, ".tar") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".jar") ||
		strings.HasSuffix(lower, ".exe") ||
		strings.HasSuffix(lower, ".dll") {
		return true
	}
	// Skip lockfiles: full of integrity hashes and resolved URLs that match
	// ENTROPY_CANDIDATE; they inflate secret counts and are not app secrets.
	switch lower {
	case "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb",
		"poetry.lock", "pipfile.lock", "pdm.lock", "uv.lock",
		"cargo.lock", "go.sum", "composer.lock":
		return true
	}
	if strings.HasSuffix(lower, ".lock") {
		return true
	}
	return false
}

// isProbablyBinary does a quick sniff of the first chunk of a file to see if it
// contains many non-UTF-8 bytes or NULs.
func isProbablyBinary(f *os.File) bool {
	const sniffSize = 8000

	buf := make([]byte, sniffSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	// Reset for subsequent scanning.
	if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
		_ = seekErr
	}
	sniff := buf[:n]

	// Heuristic: if not valid UTF-8 and contains NULs, treat as binary.
	if !utf8.Valid(sniff) {
		if bytesContainsNull(sniff) {
			return true
		}
	}
	return false
}

func bytesContainsNull(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	// Treat as bytes; for secrets this is typically fine.
	freq := make(map[byte]int)
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
	}
	l := float64(len(s))
	var entropy float64
	for _, count := range freq {
		p := float64(count) / l
		entropy -= p * math.Log2(p)
	}
	// Avoid NaN.
	if math.IsNaN(entropy) {
		return 0
	}
	return entropy
}

func isAllowlisted(value string, allow map[string]struct{}) bool {
	if len(allow) == 0 {
		return false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if _, ok := allow[value]; ok {
		return true
	}
	return false
}
