package promptfw

import (
	"math"
	"regexp"
	"strings"
)

// Result is the classification result for a prompt.
type Result struct {
	RiskLevel string   `json:"risk_level"` // low, medium, high
	Reasons   []string `json:"reasons"`
	Score     float64  `json:"score"` // aggregate numeric score for debugging
}

var (
	// Regexes for common prompt-injection / jailbreak patterns.
	injectionPatterns = []struct {
		id     string
		re     *regexp.Regexp
		weight float64
	}{
		{
			id:     "IGNORE_INSTRUCTIONS",
			re:     regexp.MustCompile(`(?i)ignore (all )?(previous|prior)? (system )?(instructions|messages)`),
			weight: 3.0,
		},
		{
			id:     "OVERRIDE_SAFETY",
			re:     regexp.MustCompile(`(?i)(disable|bypass|override).*(safety|safe mode|guardrails|filters?)`),
			weight: 2.5,
		},
		{
			id:     "REVEAL_SECRETS",
			re:     regexp.MustCompile(`(?i)(reveal|show|exfiltrate|dump).*(secret|credential|password|token|api key)`),
			weight: 3.0,
		},
		{
			id:     "SYSTEM_PROMPT_INJECTION",
			re:     regexp.MustCompile(`(?i)(you are (now)?|forget you are|act as).*(root|developer|system|prompt)`),
			weight: 2.0,
		},
	}

	highEntropyTokenRe = regexp.MustCompile(`([A-Za-z0-9+/=_-]{32,})`)

	// A tiny baseline 3-gram histogram for \"benign\" English-ish prompts.
	// This is intentionally simple; we only care about how many n-grams are
	// completely unknown relative to this baseline.
	baselineTrigrams = map[string]int{
		" th": 50, "he ": 40, "ing": 35, "er ": 30, "re ": 25,
		" to": 25, "ou ": 20, "in ": 40, "de ": 15, "on ": 20,
	}
)

// Analyze classifies a prompt into low/medium/high risk using:
// - regex rules for injection phrases
// - entropy-based detection of suspicious tokens
// - simple trigram anomaly scoring
func Analyze(prompt string) Result {
	text := strings.TrimSpace(prompt)
	if text == "" {
		return Result{RiskLevel: "low", Reasons: []string{"empty prompt"}, Score: 0}
	}

	var (
		score   float64
		reasons []string
	)

	lower := strings.ToLower(text)

	// 1) Regex-based injection patterns.
	for _, pat := range injectionPatterns {
		if pat.re.MatchString(lower) {
			score += pat.weight
			reasons = append(reasons, "pattern:"+pat.id)
		}
	}

	// 2) High-entropy segments (base64 / random strings).
	tokens := highEntropyTokenRe.FindAllString(text, -1)
	for _, tok := range tokens {
		h := shannonEntropy(tok)
		if h >= 4.0 {
			score += 1.5
			reasons = append(reasons, "high_entropy_segment")
			break
		}
	}

	// 3) Trigram anomaly score.
	anomaly := trigramAnomaly(lower)
	if anomaly > 0.3 {
		// Only treat as signal when sufficiently unusual.
		score += anomaly * 2.0
		reasons = append(reasons, "ngram_anomaly")
	}

	// Map score to discrete risk level.
	switch {
	case score >= 4.5:
		return Result{RiskLevel: "high", Reasons: reasons, Score: score}
	case score >= 2.0:
		return Result{RiskLevel: "medium", Reasons: reasons, Score: score}
	default:
		return Result{RiskLevel: "low", Reasons: reasons, Score: score}
	}
}

func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	freq := make(map[rune]int)
	for _, r := range s {
		freq[r]++
	}
	l := float64(len([]rune(s)))
	var entropy float64
	for _, count := range freq {
		p := float64(count) / l
		entropy -= p * math.Log2(p)
	}
	if math.IsNaN(entropy) {
		return 0
	}
	return entropy
}

func trigramAnomaly(text string) float64 {
	if len(text) < 3 {
		return 0
	}
	text = strings.ReplaceAll(text, "\n", " ")
	text = " " + text + " "

	var total, unknown int
	for i := 0; i+3 <= len(text); i++ {
		tg := text[i : i+3]
		if strings.TrimSpace(tg) == "" {
			continue
		}
		total++
		if _, ok := baselineTrigrams[tg]; !ok {
			unknown++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(unknown) / float64(total)
}
