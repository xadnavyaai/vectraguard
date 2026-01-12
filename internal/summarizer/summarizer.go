package summarizer

import (
	"sort"
	"strings"
)

type scoredLine struct {
	index int
	text  string
	score float64
}

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "but": {}, "by": {},
	"for": {}, "from": {}, "has": {}, "have": {}, "if": {}, "in": {}, "into": {}, "is": {},
	"it": {}, "its": {}, "of": {}, "on": {}, "or": {}, "such": {}, "that": {}, "the": {},
	"their": {}, "then": {}, "there": {}, "these": {}, "they": {}, "this": {}, "to": {},
	"was": {}, "will": {}, "with": {}, "you": {}, "your": {},
}

// SummarizeText extracts the most relevant sentences using a lightweight frequency heuristic.
func SummarizeText(text string, maxSentences int) []string {
	if maxSentences <= 0 {
		return nil
	}
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return nil
	}

	wordFreq := make(map[string]int)
	for _, sentence := range sentences {
		for _, token := range tokenize(sentence) {
			wordFreq[token]++
		}
	}

	scored := make([]scoredLine, 0, len(sentences))
	for i, sentence := range sentences {
		score := 0.0
		for _, token := range tokenize(sentence) {
			score += float64(wordFreq[token])
		}
		scored = append(scored, scoredLine{index: i, text: sentence, score: score})
	}

	return selectTop(scored, maxSentences)
}

// SummarizeCode extracts high-signal lines from code using simple structural heuristics.
func SummarizeCode(text string, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	lines := strings.Split(text, "\n")
	scored := make([]scoredLine, 0, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		score := scoreCodeLine(trimmed)
		scored = append(scored, scoredLine{index: i, text: trimmed, score: score})
	}

	return selectTop(scored, maxLines)
}

func scoreCodeLine(line string) float64 {
	switch {
	case strings.HasPrefix(line, "func "):
		return 3.0
	case strings.HasPrefix(line, "type "):
		return 2.6
	case strings.HasPrefix(line, "//"):
		return 2.2
	case strings.HasPrefix(line, "/*"):
		return 2.0
	case strings.HasPrefix(line, "const "):
		return 1.8
	case strings.HasPrefix(line, "var "):
		return 1.6
	case strings.HasPrefix(line, "package "):
		return 1.2
	case strings.HasPrefix(line, "import "):
		return 1.1
	default:
		return 0.5
	}
}

func selectTop(scored []scoredLine, maxItems int) []string {
	if len(scored) == 0 {
		return nil
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].index < scored[j].index
		}
		return scored[i].score > scored[j].score
	})

	if maxItems > len(scored) {
		maxItems = len(scored)
	}

	selected := scored[:maxItems]
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].index < selected[j].index
	})

	result := make([]string, 0, len(selected))
	for _, item := range selected {
		result = append(result, item.text)
	}
	return result
}

func splitSentences(text string) []string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	var sentences []string
	var current strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			isEnd := i == len(runes)-1
			nextIsSpace := !isEnd && (runes[i+1] == ' ' || runes[i+1] == '\t')
			if isEnd || nextIsSpace {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}
	if tail := strings.TrimSpace(current.String()); tail != "" {
		sentences = append(sentences, tail)
	}
	return sentences
}

func tokenize(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		token := strings.Trim(field, "\"'()[]{}<>.,;:!?*`~")
		if token == "" {
			continue
		}
		if _, found := stopWords[token]; found {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}
