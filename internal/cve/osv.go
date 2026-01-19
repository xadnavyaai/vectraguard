package cve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const osvEndpoint = "https://api.osv.dev/v1/query"

type osvQueryRequest struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Version string `json:"version,omitempty"`
}

type osvResponse struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvVuln struct {
	ID         string         `json:"id"`
	Summary    string         `json:"summary"`
	Details    string         `json:"details"`
	Aliases    []string       `json:"aliases"`
	Severity   []osvSeverity  `json:"severity"`
	References []osvReference `json:"references"`
	Published  string         `json:"published"`
	Modified   string         `json:"modified"`
}

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type osvReference struct {
	URL string `json:"url"`
}

// FetchOSVVulns queries OSV for a package/version and normalizes results.
func FetchOSVVulns(ctx context.Context, ref PackageRef) ([]Vulnerability, error) {
	reqBody := osvQueryRequest{}
	reqBody.Package.Name = ref.Name
	reqBody.Package.Ecosystem = ref.Ecosystem
	reqBody.Version = ref.Version

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("encode osv request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, osvEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build osv request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osv request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read osv response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("osv error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed osvResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse osv response: %w", err)
	}

	vulns := make([]Vulnerability, 0, len(parsed.Vulns))
	for _, v := range parsed.Vulns {
		cvssScore := parseOSVScore(v.Severity)
		vulns = append(vulns, Vulnerability{
			ID:         v.ID,
			Summary:    v.Summary,
			Details:    v.Details,
			Severity:   severityFromScore(cvssScore),
			CVSS:       cvssScore,
			Aliases:    append([]string{}, v.Aliases...),
			References: extractReferences(v.References),
			Published:  parseOSVTime(v.Published),
			Modified:   parseOSVTime(v.Modified),
		})
	}

	return vulns, nil
}

func parseOSVScore(severities []osvSeverity) float64 {
	for _, sev := range severities {
		if sev.Score == "" {
			continue
		}
		score, err := strconv.ParseFloat(strings.TrimSpace(sev.Score), 64)
		if err == nil {
			return score
		}
	}
	return 0
}

func severityFromScore(score float64) string {
	switch {
	case score >= 9:
		return "critical"
	case score >= 7:
		return "high"
	case score >= 4:
		return "medium"
	case score > 0:
		return "low"
	default:
		return "unknown"
	}
}

func extractReferences(refs []osvReference) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.URL != "" {
			out = append(out, ref.URL)
		}
	}
	return out
}

func parseOSVTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
