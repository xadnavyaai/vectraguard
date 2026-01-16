package cmd

import "testing"

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
