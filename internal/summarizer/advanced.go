package summarizer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"
	"sync"
)

type advancedFuncSummary struct {
	name      string
	signature string
	doc       string
	calls     []string
	startLine int
	endLine   int
}

type funcGraphStats struct {
	inDegree  int
	outDegree int
	callers   []string
}

type advancedCacheEntry struct {
	scored []scoredLine
}

var advancedCache sync.Map

// SummarizeCodeAdvanced builds a function-level summary using Go AST heuristics.
func SummarizeCodeAdvanced(text string, maxItems int) []string {
	if maxItems <= 0 {
		return nil
	}
	cacheKey := hashContent(text)
	if cached, ok := advancedCache.Load(cacheKey); ok {
		entry := cached.(advancedCacheEntry)
		return selectTop(entry.scored, maxItems)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "input.go", text, parser.ParseComments)
	if err != nil {
		return SummarizeCode(text, maxItems)
	}

	summaries := buildAdvancedSummaries(fset, file)
	if len(summaries) == 0 {
		return SummarizeCode(text, maxItems)
	}

	graphStats := buildGraphStats(summaries)
	scored := make([]scoredLine, 0, len(summaries))
	for i, summary := range summaries {
		stats := graphStats[summary.name]
		score := scoreFuncSummary(summary, stats)
		scored = append(scored, scoredLine{index: i, text: formatFuncSummary(summary, stats), score: score})
	}

	advancedCache.Store(cacheKey, advancedCacheEntry{scored: scored})
	return selectTop(scored, maxItems)
}

func buildAdvancedSummaries(fset *token.FileSet, file *ast.File) []advancedFuncSummary {
	var summaries []advancedFuncSummary
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		summary := advancedFuncSummary{
			name:      fn.Name.Name,
			signature: formatFuncSignature(fset, fn),
			doc:       strings.TrimSpace(extractDoc(fn)),
			calls:     collectCalls(fn),
			startLine: fset.Position(fn.Pos()).Line,
			endLine:   fset.Position(fn.End()).Line,
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func formatFuncSignature(fset *token.FileSet, fn *ast.FuncDecl) string {
	buf := &bytes.Buffer{}
	if fn.Recv != nil {
		_ = printer.Fprint(buf, fset, fn.Recv)
		buf.WriteString(" ")
	}
	buf.WriteString(fn.Name.Name)
	_ = printer.Fprint(buf, fset, fn.Type.Params)
	if fn.Type.Results != nil {
		buf.WriteString(" ")
		_ = printer.Fprint(buf, fset, fn.Type.Results)
	}
	return buf.String()
}

func extractDoc(fn *ast.FuncDecl) string {
	if fn.Doc == nil {
		return ""
	}
	var lines []string
	for _, comment := range fn.Doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "/*"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "*/"))
		if text == "" {
			continue
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, " ")
}

func collectCalls(fn *ast.FuncDecl) []string {
	calls := map[string]struct{}{}
	if fn.Body == nil {
		return nil
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			calls[fun.Name] = struct{}{}
		case *ast.SelectorExpr:
			calls[fun.Sel.Name] = struct{}{}
		}
		return true
	})

	if len(calls) == 0 {
		return nil
	}
	result := make([]string, 0, len(calls))
	for name := range calls {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func scoreFuncSummary(summary advancedFuncSummary, stats funcGraphStats) float64 {
	score := 2.0
	if summary.doc != "" {
		score += 1.2
	}
	lines := summary.endLine - summary.startLine
	if lines > 0 {
		score += float64(min(lines/10, 3))
	}
	score += float64(min(stats.outDegree, 4)) * 0.25
	score += float64(min(stats.inDegree, 4)) * 0.35
	return score
}

func formatFuncSummary(summary advancedFuncSummary, stats funcGraphStats) string {
	parts := []string{fmt.Sprintf("func %s", summary.signature)}
	if summary.doc != "" {
		parts = append(parts, fmt.Sprintf("doc: %s", summary.doc))
	}
	if len(summary.calls) > 0 {
		parts = append(parts, fmt.Sprintf("calls: %s", strings.Join(summary.calls, ", ")))
	}
	if stats.inDegree > 0 {
		parts = append(parts, fmt.Sprintf("callers: %d", stats.inDegree))
	}
	return strings.Join(parts, " | ")
}

func buildGraphStats(summaries []advancedFuncSummary) map[string]funcGraphStats {
	defined := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		defined[summary.name] = struct{}{}
	}

	stats := make(map[string]funcGraphStats, len(summaries))
	for _, summary := range summaries {
		stats[summary.name] = funcGraphStats{}
	}

	for _, summary := range summaries {
		outSet := make(map[string]struct{})
		for _, call := range summary.calls {
			if _, ok := defined[call]; !ok {
				continue
			}
			outSet[call] = struct{}{}
		}

		if len(outSet) == 0 {
			continue
		}

		entry := stats[summary.name]
		entry.outDegree = len(outSet)
		stats[summary.name] = entry

		for callee := range outSet {
			target := stats[callee]
			target.inDegree++
			target.callers = append(target.callers, summary.name)
			stats[callee] = target
		}
	}

	for name, entry := range stats {
		if len(entry.callers) > 1 {
			sort.Strings(entry.callers)
		}
		stats[name] = entry
	}

	return stats
}

func hashContent(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
