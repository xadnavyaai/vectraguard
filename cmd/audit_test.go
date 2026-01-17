package cmd

import (
	"testing"

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
