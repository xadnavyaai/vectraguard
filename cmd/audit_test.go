package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/secscan"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func TestParsePipAuditOutputWrapped(t *testing.T) {
	payload := []byte(`warning: extra
{"dependencies":[{"name":"requests","version":"2.31.0","vulns":[{"id":"GHSA-1","severity":"high"}]},{"name":"idna","version":"3.11","vulns":[]}]}`)
	entries, err := parsePipAuditOutput(payload)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if len(entries[0].Vulns) != 1 {
		t.Fatalf("expected 1 vuln, got %d", len(entries[0].Vulns))
	}
}

func TestParsePipAuditOutputArray(t *testing.T) {
	payload := []byte(`[{"name":"flask","version":"3.0.0","vulns":[{"id":"GHSA-2","severity":"medium"}]}]`)
	entries, err := parsePipAuditOutput(payload)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestParseNpmAuditOutput(t *testing.T) {
	payload := []byte(`{"metadata":{"vulnerabilities":{"low":1,"high":2}}}`)
	counts, total, err := parseNpmAuditOutput(payload)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if counts["high"] != 2 {
		t.Fatalf("expected high=2, got %d", counts["high"])
	}
}

func TestBuildSessionAuditSummary(t *testing.T) {
	sess := &session.Session{
		ID:        "session-1",
		Workspace: "/tmp/repo",
		AgentName: "tester",
		Commands: []session.Command{
			{
				Command:   "echo",
				Args:      []string{"hi"},
				RiskLevel: "low",
				Metadata: map[string]interface{}{
					"source": "shell-tracker",
				},
			},
			{
				Command:   "npm",
				Args:      []string{"install"},
				RiskLevel: "medium",
				Metadata: map[string]interface{}{
					"execution": "sandbox",
				},
			},
			{
				Command:   "curl",
				Args:      []string{"http://example.com"},
				RiskLevel: "high",
				Metadata: map[string]interface{}{
					"execution": "host",
					"bypass":    true,
				},
			},
			{
				Command:   "sudo",
				Args:      []string{"rm", "-rf", "/"},
				RiskLevel: "critical",
				Metadata: map[string]interface{}{
					"blocked": true,
				},
			},
		},
	}

	summary := buildSessionAuditSummary(sess)
	if summary.Total != 4 {
		t.Fatalf("expected total 4, got %d", summary.Total)
	}
	if summary.RiskCounts["low"] != 1 || summary.RiskCounts["medium"] != 1 || summary.RiskCounts["high"] != 1 || summary.RiskCounts["critical"] != 1 {
		t.Fatalf("unexpected risk counts: %#v", summary.RiskCounts)
	}
	if summary.SourceCounts["shell-tracker"] != 1 || summary.SourceCounts["exec"] != 3 {
		t.Fatalf("unexpected source counts: %#v", summary.SourceCounts)
	}
	if summary.ExecutionCounts["sandbox"] != 1 || summary.ExecutionCounts["host"] != 1 {
		t.Fatalf("unexpected execution counts: %#v", summary.ExecutionCounts)
	}
	if summary.Bypassed != 1 {
		t.Fatalf("expected bypassed 1, got %d", summary.Bypassed)
	}
	if summary.Blocked != 1 {
		t.Fatalf("expected blocked 1, got %d", summary.Blocked)
	}
}

func TestBuildSessionAuditSummaryFromSessions(t *testing.T) {
	sessions := []*session.Session{
		{
			ID:        "s1",
			Workspace: "/tmp/repo-a",
			AgentName: "a",
			Commands: []session.Command{
				{RiskLevel: "low", Metadata: map[string]interface{}{"source": "shell-tracker"}},
				{RiskLevel: "high", Metadata: map[string]interface{}{"execution": "sandbox"}},
			},
		},
		{
			ID:        "s2",
			Workspace: "/tmp/repo-b",
			AgentName: "b",
			Commands: []session.Command{
				{RiskLevel: "medium", Metadata: map[string]interface{}{"execution": "host", "bypass": true}},
				{RiskLevel: "critical", Metadata: map[string]interface{}{"blocked": true}},
			},
		},
	}

	summary := buildSessionAuditSummaryFromSessions(sessions)
	if summary.SessionCount != 2 {
		t.Fatalf("expected 2 sessions, got %d", summary.SessionCount)
	}
	if summary.Total != 4 {
		t.Fatalf("expected total 4, got %d", summary.Total)
	}
	if summary.RiskCounts["low"] != 1 || summary.RiskCounts["high"] != 1 || summary.RiskCounts["medium"] != 1 || summary.RiskCounts["critical"] != 1 {
		t.Fatalf("unexpected risk counts: %#v", summary.RiskCounts)
	}
	if summary.SourceCounts["shell-tracker"] != 1 || summary.SourceCounts["exec"] != 3 {
		t.Fatalf("unexpected source counts: %#v", summary.SourceCounts)
	}
	if summary.ExecutionCounts["sandbox"] != 1 || summary.ExecutionCounts["host"] != 1 {
		t.Fatalf("unexpected execution counts: %#v", summary.ExecutionCounts)
	}
	if summary.Bypassed != 1 {
		t.Fatalf("expected bypassed 1, got %d", summary.Bypassed)
	}
	if summary.Blocked != 1 {
		t.Fatalf("expected blocked 1, got %d", summary.Blocked)
	}
}

func TestHasPackageFindings(t *testing.T) {
	tests := []struct {
		name   string
		input  []auditSummary
		expect bool
	}{
		{
			name:   "no audits",
			input:  nil,
			expect: false,
		},
		{
			name: "audits with zero findings",
			input: []auditSummary{
				{Tool: "npm", Total: 0},
				{Tool: "python", Total: 0},
			},
			expect: false,
		},
		{
			name: "audits with findings",
			input: []auditSummary{
				{Tool: "npm", Total: 1},
				{Tool: "python", Total: 0},
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasPackageFindings(tt.input)
			if got != tt.expect {
				t.Fatalf("expected %v, got %v", tt.expect, got)
			}
		})
	}
}

func TestGetRemediationForKnownCodes(t *testing.T) {
	for _, code := range []string{"PY_ENV_ACCESS", "PY_SUBPROCESS", "GO_EXEC_COMMAND", "BIND_ALL_INTERFACES"} {
		rem := getRemediation(code)
		if rem == "" {
			t.Errorf("getRemediation(%q) expected non-empty, got %q", code, rem)
		}
	}
	// Unknown code returns empty
	if getRemediation("UNKNOWN_CODE") != "" {
		t.Errorf("getRemediation(UNKNOWN_CODE) expected empty, got non-empty")
	}
}

func TestRepoAuditSummaryJSONRoundTrip(t *testing.T) {
	summary := repoAuditSummary{
		Path: "/tmp/repo",
		CodeFindings: []secscan.Finding{
			{File: "a.py", Line: 1, Language: "python", Severity: "medium", Code: "PY_ENV_ACCESS", Description: "env access"},
			{File: "b.go", Line: 2, Language: "go", Severity: "high", Code: "GO_EXEC_COMMAND", Description: "exec"},
		},
		CodeFindingsTotal: 2,
		CodeBySeverity:    map[string]int{"medium": 1, "high": 1},
		CodeByLanguage:    map[string]int{"python": 1, "go": 1},
		SecretsTotal:      0,
		PackageAudits:     nil,
	}
	var buf bytes.Buffer
	// Encode the same struct we emit so we can decode and assert remediation is present.
	out := repoAuditJSONOut{
		Path: summary.Path,
		CodeFindings: []codeFindingJSON{
			{File: "a.py", Line: 1, Severity: "medium", Code: "PY_ENV_ACCESS", Description: "env access", Remediation: getRemediation("PY_ENV_ACCESS")},
			{File: "b.go", Line: 2, Severity: "high", Code: "GO_EXEC_COMMAND", Description: "exec", Remediation: getRemediation("GO_EXEC_COMMAND")},
		},
		CodeBySeverity: summary.CodeBySeverity,
		SecretsTotal:   summary.SecretsTotal,
		PackageAudits:  nil,
	}
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		t.Fatalf("encode: %v", err)
	}
	var decoded repoAuditJSONOut
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Path != summary.Path {
		t.Errorf("path: got %q", decoded.Path)
	}
	if len(decoded.CodeFindings) != 2 {
		t.Fatalf("code_findings: expected 2, got %d", len(decoded.CodeFindings))
	}
	if decoded.CodeFindings[0].Remediation == "" {
		t.Errorf("code_findings[0].remediation expected non-empty for PY_ENV_ACCESS")
	}
	if decoded.CodeFindings[1].Remediation == "" {
		t.Errorf("code_findings[1].remediation expected non-empty for GO_EXEC_COMMAND")
	}
}
