package secscan

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding represents a potential security issue in source code.
type Finding struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Language    string `json:"language"`
	Severity    string `json:"severity"` // low, medium, high, critical
	Code        string `json:"code"`     // short identifier (e.g., GO_EXEC_COMMAND)
	Description string `json:"description"`
}

// Options controls how the security scan is performed.
type Options struct {
	Languages map[string]bool
}

// defaultLanguages if none are specified.
var defaultLanguages = []string{"go", "python", "c"}

// ScanPath walks a directory tree and performs language-specific security checks.
func ScanPath(root string, opts Options) ([]Finding, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}

	if opts.Languages == nil || len(opts.Languages) == 0 {
		opts.Languages = make(map[string]bool)
		for _, lang := range defaultLanguages {
			opts.Languages[lang] = true
		}
	}

	// Single file.
	if !info.IsDir() {
		lang := languageForExt(filepath.Ext(root))
		if !opts.Languages[lang] {
			return nil, nil
		}
		fs, err := scanFile(root, lang)
		if err != nil {
			return nil, err
		}
		return fs, nil
	}

	var findings []Finding
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			// Skip typical vendor/build dirs.
			name := strings.ToLower(d.Name())
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".venv" || name == "venv" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		lang := languageForExt(filepath.Ext(path))
		if !opts.Languages[lang] {
			return nil
		}
		fs, err := scanFile(path, lang)
		if err != nil {
			// best-effort: log upstream, skip file
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

func languageForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".c", ".h":
		return "c"
	case ".yaml", ".yml", ".json":
		return "config"
	default:
		return ""
	}
}

func scanFile(path, lang string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var findings []Finding
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		switch lang {
		case "go":
			findings = append(findings, scanGoLine(path, line, lineNum)...)
		case "python":
			findings = append(findings, scanPythonLine(path, line, lineNum)...)
		case "c":
			findings = append(findings, scanCLine(path, line, lineNum)...)
		case "config":
			findings = append(findings, scanConfigLine(path, line, lineNum)...)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return findings, nil
}

var (
	goExecCommandRe = regexp.MustCompile(`exec\.Command\(`)
	goShellDangerRe = regexp.MustCompile(`(?i)(rm\s+-rf\s+/|curl\s+.+\|\s*sh)`)
	goNetHTTPRe     = regexp.MustCompile(`"net/http"`)
	goEnvReadRe     = regexp.MustCompile(`os\.Getenv\(`)
	goWriteSystemRe = regexp.MustCompile(`WriteFile\("(/etc|/var|/usr)/`)
	pyEvalRe        = regexp.MustCompile(`\beval\(`)
	pyExecRe        = regexp.MustCompile(`\bexec\(`)
	pySubprocessRe  = regexp.MustCompile(`\bsubprocess\.`)
	pyOSSystemRe    = regexp.MustCompile(`\bos\.system\(`)
	pyRequestsRe    = regexp.MustCompile(`\brequests\.`)
	pyEnvRe         = regexp.MustCompile(`os\.environ|\bENV\b|os\.getenv\(`)
	pyDotEnvRe      = regexp.MustCompile(`\.env("|'| )`)
	cSystemRe       = regexp.MustCompile(`\bsystem\(`)
	cPopenRe        = regexp.MustCompile(`\bpopen\(`)
	cExecFamilyRe   = regexp.MustCompile(`\b(execv|execve|execl|execvp)\b`)
	cGetsRe         = regexp.MustCompile(`\bgets\(`)
	cStrcpyRe       = regexp.MustCompile(`\bstrcpy\(`)
	cStrcatRe       = regexp.MustCompile(`\bstrcat\(`)
	cMemUnsafeRe    = regexp.MustCompile(`\bmemcpy\(`)
	cSocketRe       = regexp.MustCompile(`\bsocket\(`)
	// External HTTP(S) URL: match URL then check host is not localhost/127.0.0.1/::1
	externalHTTPRe      = regexp.MustCompile(`https?://([^\s/]+)`)
	bindAllInterfacesRe = regexp.MustCompile(`0\.0\.0\.0|"0\.0\.0\.0"`)

	// Config/deployment: control-panel and reverse-proxy misconfig
	configBindRe         = regexp.MustCompile(`(?i)(host|listen|bind|address).*0\.0\.0\.0|0\.0\.0\.0.*(:|,)`)
	configTrustProxyRe   = regexp.MustCompile(`(?i)(trust[_\s]?proxy|X-Forwarded-For|forwarded.*trust)`)
	configUnauthAccessRe = regexp.MustCompile(`(?i)(auth|authentication|secure).*:\s*(false|0|off|no|disabled)`)
)

func scanGoLine(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return nil
	}
	lower := strings.ToLower(line)
	var out []Finding

	if goExecCommandRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "go",
			Severity:    "high",
			Code:        "GO_EXEC_COMMAND",
			Description: "Use of exec.Command; ensure inputs are validated and sandboxed.",
		})
	}
	if goShellDangerRe.MatchString(lower) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "go",
			Severity:    "critical",
			Code:        "GO_DANGEROUS_SHELL",
			Description: "Dangerous shell pattern (rm -rf / or curl|sh) detected in Go code.",
		})
	}
	if goNetHTTPRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "go",
			Severity:    "medium",
			Code:        "GO_NET_HTTP",
			Description: "Use of net/http; ensure remote calls are authenticated and sanitized.",
		})
	}
	if goEnvReadRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "go",
			Severity:    "medium",
			Code:        "GO_ENV_READ",
			Description: "Environment variable access; avoid leaking credentials.",
		})
	}
	if goWriteSystemRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "go",
			Severity:    "high",
			Code:        "GO_SYSTEM_WRITE",
			Description: "Writing to system directory (/etc, /var, /usr); review for safety.",
		})
	}
	for _, submatch := range externalHTTPRe.FindAllStringSubmatch(line, -1) {
		if len(submatch) > 1 && !isLocalhostHost(submatch[1]) {
			out = append(out, Finding{
				File:        path,
				Line:        lineNum,
				Language:    "go",
				Severity:    "medium",
				Code:        "GO_EXTERNAL_HTTP",
				Description: "Non-localhost HTTP(S) URL; ensure not used with untrusted input (SSRF risk).",
			})
			break
		}
	}
	if bindAllInterfacesRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "go",
			Severity:    "medium",
			Code:        "BIND_ALL_INTERFACES",
			Description: "Binding to 0.0.0.0 exposes the service on all interfaces; ensure auth and TLS are enabled.",
		})
	}
	return out
}

func isLocalhostHost(host string) bool {
	host = strings.TrimSpace(host)

	// Handle IPv6 addresses in URL host, which are typically wrapped in brackets,
	// e.g. http://[::1]:3000. In that case, extract the inner address first.
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end > 0 {
			host = host[1:end]
		}
	} else {
		// Strip optional port for non-bracketed hosts like 127.0.0.1:8080.
		if i := strings.Index(host, ":"); i >= 0 {
			host = host[:i]
		}
	}

	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	if strings.HasPrefix(host, "127.") {
		return true
	}
	return false
}

func scanPythonLine(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return nil
	}
	lower := strings.ToLower(line)
	var out []Finding

	if pyEvalRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "python",
			Severity:    "high",
			Code:        "PY_EVAL",
			Description: "Use of eval(); avoid executing dynamic code from untrusted input.",
		})
	}
	if pyExecRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "python",
			Severity:    "high",
			Code:        "PY_EXEC",
			Description: "Use of exec(); avoid executing dynamic code from untrusted input.",
		})
	}
	if pySubprocessRe.MatchString(line) || pyOSSystemRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "python",
			Severity:    "medium",
			Code:        "PY_SUBPROCESS",
			Description: "Use of subprocess or os.system; ensure commands are validated and sandboxed.",
		})
	}
	if pyRequestsRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "python",
			Severity:    "medium",
			Code:        "PY_REMOTE_HTTP",
			Description: "Remote HTTP request; validate URLs and responses to avoid SSRF/data leaks.",
		})
	}
	if pyEnvRe.MatchString(line) || pyDotEnvRe.MatchString(lower) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "python",
			Severity:    "medium",
			Code:        "PY_ENV_ACCESS",
			Description: "Environment or .env access; avoid exposing secrets in logs or responses.",
		})
	}
	for _, submatch := range externalHTTPRe.FindAllStringSubmatch(line, -1) {
		if len(submatch) > 1 && !isLocalhostHost(submatch[1]) {
			out = append(out, Finding{
				File:        path,
				Line:        lineNum,
				Language:    "python",
				Severity:    "medium",
				Code:        "PY_EXTERNAL_HTTP",
				Description: "Non-localhost HTTP(S) URL; ensure not used with untrusted input (SSRF risk).",
			})
			break
		}
	}
	if bindAllInterfacesRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "python",
			Severity:    "medium",
			Code:        "BIND_ALL_INTERFACES",
			Description: "Binding to 0.0.0.0 exposes the service on all interfaces; ensure auth and TLS are enabled.",
		})
	}
	return out
}

func scanCLine(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return nil
	}
	var out []Finding

	if cSystemRe.MatchString(line) || cPopenRe.MatchString(line) || cExecFamilyRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "c",
			Severity:    "high",
			Code:        "C_SHELL_EXEC",
			Description: "Use of system/popen/exec*; avoid spawning shells with untrusted input.",
		})
	}
	if cGetsRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "c",
			Severity:    "critical",
			Code:        "C_GETS",
			Description: "Use of gets(); this is inherently unsafe and should be removed.",
		})
	}
	if cStrcpyRe.MatchString(line) || cStrcatRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "c",
			Severity:    "high",
			Code:        "C_UNSAFE_STRING",
			Description: "Use of unsafe string functions (strcpy/strcat); risk of buffer overflow.",
		})
	}
	if cMemUnsafeRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "c",
			Severity:    "medium",
			Code:        "C_MEMCPY",
			Description: "Use of memcpy(); ensure bounds are validated to avoid buffer overflows.",
		})
	}
	if cSocketRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "c",
			Severity:    "medium",
			Code:        "C_RAW_SOCKET",
			Description: "Raw socket use; review for network abuse or exfiltration.",
		})
	}
	if bindAllInterfacesRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "c",
			Severity:    "medium",
			Code:        "BIND_ALL_INTERFACES",
			Description: "Binding to 0.0.0.0 exposes the service on all interfaces; ensure auth and TLS are enabled.",
		})
	}
	return out
}

func scanConfigLine(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return nil
	}
	var out []Finding
	lower := strings.ToLower(line)

	if configBindRe.MatchString(line) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "config",
			Severity:    "medium",
			Code:        "BIND_ALL_INTERFACES",
			Description: "Config binds to 0.0.0.0; control panels must use auth and TLS when exposed.",
		})
	}
	if configTrustProxyRe.MatchString(lower) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "config",
			Severity:    "medium",
			Code:        "LOCALHOST_TRUST_PROXY",
			Description: "Trust proxy / X-Forwarded-For can treat remote clients as local; ensure auth is not bypassed.",
		})
	}
	if configUnauthAccessRe.MatchString(lower) {
		out = append(out, Finding{
			File:        path,
			Line:        lineNum,
			Language:    "config",
			Severity:    "high",
			Code:        "UNAUTHENTICATED_ACCESS",
			Description: "Config disables auth or secure access; control panels must require authentication.",
		})
	}
	return out
}
