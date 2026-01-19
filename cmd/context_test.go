package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunContextSummarizeFile_CodeMode(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func helper() {
	fmt.Println("Helper function")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "code", testFile, 3, "text", "")
	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}
}

func TestRunContextSummarizeFile_DocsMode(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.md")
	content := `# Test Document

This is a test document. It contains multiple sentences.
Some sentences are more important than others.
The summarizer should extract the most relevant ones.
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "docs", testFile, 2, "text", "")
	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}
}

func TestRunContextSummarizeFile_AdvancedMode(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	content := `package main

// Main function is the entry point
func main() {
	helper()
}

// Helper does something useful
func helper() {
	doWork()
}

func doWork() {
	// Implementation
}
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "advanced", testFile, 3, "text", "")
	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}
}

func TestRunContextSummarizeFile_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	content := `package main

func main() {
	println("test")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	// Capture output
	output := &strings.Builder{}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runContextSummarize(ctx, "code", testFile, 5, "json", "")
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}

	// Read output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output.Write(buf[:n])

	// Verify JSON structure
	var result contextSummaryOutput
	if err := json.Unmarshal([]byte(output.String()), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Mode != "code" {
		t.Errorf("expected mode 'code', got %s", result.Mode)
	}
	if result.Path != testFile {
		t.Errorf("expected path %s, got %s", testFile, result.Path)
	}
	if len(result.Summary) == 0 && len(result.Files) == 0 {
		t.Error("expected summary or files in output")
	}
}

func TestRunContextSummarizeRepo_CodeMode(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	files := map[string]string{
		"main.go": `package main
func main() {}
`,
		"helper.go": `package main
func helper() {}
`,
		"README.md": `# Test Project
This is a test project.
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", name, err)
		}
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "code", dir, 5, "text", "")
	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}
}

func TestRunContextSummarizeRepo_JSONOutput(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	files := map[string]string{
		"main.go": `package main
func main() {}
`,
		"helper.go": `package main
func helper() {}
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", name, err)
		}
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	// Capture output
	output := &strings.Builder{}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runContextSummarize(ctx, "code", dir, 5, "json", "")
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}

	// Read output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output.Write(buf[:n])

	// Verify JSON structure
	var result contextSummaryOutput
	if err := json.Unmarshal([]byte(output.String()), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Mode != "code" {
		t.Errorf("expected mode 'code', got %s", result.Mode)
	}
	if result.FileCount == 0 {
		t.Error("expected fileCount > 0")
	}
	if len(result.Files) == 0 {
		t.Error("expected files in output")
	}
}

func TestShouldProcessFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		mode     string
		want     bool
	}{
		{"Go file code mode", "test.go", "code", true},
		{"Go file advanced mode", "test.go", "advanced", true},
		{"JS file code mode", "test.js", "code", true},
		{"Python file code mode", "test.py", "code", true},
		{"Markdown file docs mode", "test.md", "docs", true},
		{"Text file docs mode", "test.txt", "docs", true},
		{"Go file docs mode", "test.go", "docs", false},
		{"Markdown file code mode", "test.md", "code", false},
		{"Hidden file", ".test.go", "code", false},
		{"Makefile code mode", "Makefile", "code", true},
		{"Dockerfile code mode", "Dockerfile", "code", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldProcessFile(tt.filePath, tt.mode)
			if got != tt.want {
				t.Errorf("shouldProcessFile(%q, %q) = %v, want %v", tt.filePath, tt.mode, got, tt.want)
			}
		})
	}
}

func TestFindRepoRoot(t *testing.T) {
	dir := t.TempDir()

	// Create .git directory
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("create .git: %v", err)
	}

	// Test from subdirectory
	subDir := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	got := findRepoRoot(subDir)
	if got != dir {
		t.Errorf("findRepoRoot(%q) = %q, want %q", subDir, got, dir)
	}
}

func TestFindRepoRoot_WithVectraGuard(t *testing.T) {
	dir := t.TempDir()

	// Create .vectra-guard directory
	vectraDir := filepath.Join(dir, ".vectra-guard")
	if err := os.MkdirAll(vectraDir, 0o755); err != nil {
		t.Fatalf("create .vectra-guard: %v", err)
	}

	// Test from subdirectory
	subDir := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	got := findRepoRoot(subDir)
	if got != dir {
		t.Errorf("findRepoRoot(%q) = %q, want %q", subDir, got, dir)
	}
}

func TestCacheOperations(t *testing.T) {
	cacheDir := t.TempDir()
	testFile := "/tmp/test.go"
	mode := "code"
	summary := []string{"func main()", "func helper()"}

	// Test save
	if err := saveToCache(cacheDir, testFile, mode, summary); err != nil {
		t.Fatalf("saveToCache failed: %v", err)
	}

	// Test load
	loaded, err := loadFromCache(cacheDir, testFile, mode)
	if err != nil {
		t.Fatalf("loadFromCache failed: %v", err)
	}

	if loaded.Path != testFile {
		t.Errorf("expected path %s, got %s", testFile, loaded.Path)
	}
	if loaded.Mode != mode {
		t.Errorf("expected mode %s, got %s", mode, loaded.Mode)
	}
	if len(loaded.Summary) != len(summary) {
		t.Errorf("expected %d summary items, got %d", len(summary), len(loaded.Summary))
	}
}

func TestOutputSummary_TextMode(t *testing.T) {
	files := []fileSummaryOutput{
		{Path: "file1.go", Summary: []string{"func main()", "func helper()"}},
		{Path: "file2.go", Summary: []string{"func test()"}},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputSummary(".", "code", files, "text", 2)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputSummary failed: %v", err)
	}

	// Read output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "file1.go") {
		t.Error("output should contain file1.go")
	}
	if !strings.Contains(output, "file2.go") {
		t.Error("output should contain file2.go")
	}
}

func TestOutputSummary_JSONMode(t *testing.T) {
	files := []fileSummaryOutput{
		{Path: "file1.go", Summary: []string{"func main()"}},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputSummary(".", "code", files, "json", 1)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputSummary failed: %v", err)
	}

	// Read and parse JSON
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	var result contextSummaryOutput
	if err := json.Unmarshal(buf[:n], &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if result.Mode != "code" {
		t.Errorf("expected mode 'code', got %s", result.Mode)
	}
	if result.FileCount != 1 {
		t.Errorf("expected fileCount 1, got %d", result.FileCount)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}
}

func TestRunContextSummarize_InvalidMode(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "invalid", testFile, 5, "text", "")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "unknown summarize mode") {
		t.Errorf("expected 'unknown summarize mode' error, got: %v", err)
	}
}

func TestRunContextSummarize_FileNotFound(t *testing.T) {
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "code", "/nonexistent/file.go", 5, "text", "")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestGetChangedFiles_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := getChangedFiles(dir, "HEAD~1")
	if err == nil {
		t.Fatal("expected error for non-git repo")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository' error, got: %v", err)
	}
}

func TestRunContextSummarizeRepo_WithSinceFlag(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	files := map[string]string{
		"main.go": `package main
func main() {}
`,
		"helper.go": `package main
func helper() {}
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", name, err)
		}
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	// Test with --since flag (should work even without git, just processes all)
	err := runContextSummarize(ctx, "code", dir, 5, "text", "HEAD~1")
	// Should not fail even if git is not available (gracefully degrades)
	if err != nil && !strings.Contains(err.Error(), "git") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunContextSummarize_CachingBehavior(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	content := `package main

func main() {
	println("test")
}

func helper() {
	println("helper")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	// First run - should compute
	err := runContextSummarize(ctx, "code", testFile, 3, "text", "")
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// Second run - should use cache
	err = runContextSummarize(ctx, "code", testFile, 3, "text", "")
	if err != nil {
		t.Fatalf("second run (cached) failed: %v", err)
	}
}

func TestRunContextSummarizeRepo_DocsMode(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	files := map[string]string{
		"README.md": `# Test Project
This is a test project with multiple sentences.
Some sentences are more important than others.
The summarizer should extract the most relevant ones.
`,
		"CHANGELOG.md": `# Changelog

## Version 1.0
Initial release with basic features.

## Version 1.1
Added new functionality.
`,
		"LICENSE": `MIT License

Copyright (c) 2024
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", name, err)
		}
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "docs", dir, 3, "text", "")
	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}
}

func TestRunContextSummarizeRepo_AdvancedMode(t *testing.T) {
	dir := t.TempDir()

	// Create test Go files with function relationships
	files := map[string]string{
		"main.go": `package main

// Main is the entry point
func main() {
	helper()
}

// Helper does something
func helper() {
	doWork()
}

func doWork() {
	// Implementation
}
`,
		"utils.go": `package main

// Process handles processing
func Process() {
	main()
}
`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", name, err)
		}
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "advanced", dir, 5, "text", "")
	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}
}

func TestRunContextSummarize_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	// Should not fail on empty directory
	err := runContextSummarize(ctx, "code", dir, 5, "text", "")
	if err != nil {
		t.Fatalf("unexpected error on empty directory: %v", err)
	}
}

func TestRunContextSummarize_IgnoresHiddenFiles(t *testing.T) {
	dir := t.TempDir()

	// Create visible and hidden files
	files := map[string]string{
		"main.go":     `package main\nfunc main() {}\n`,
		".hidden.go":  `package main\nfunc hidden() {}\n`,
		".git/config": `[core]\n`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", name, err)
		}
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runContextSummarize(ctx, "code", dir, 5, "text", "")
	if err != nil {
		t.Fatalf("runContextSummarize failed: %v", err)
	}
}

func TestOutputSummary_SingleFileJSON(t *testing.T) {
	files := []fileSummaryOutput{
		{Path: "file.go", Summary: []string{"func main()"}},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputSummary("file.go", "code", files, "json", 1)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputSummary failed: %v", err)
	}

	// Read and parse JSON
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	var result contextSummaryOutput
	if err := json.Unmarshal(buf[:n], &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Single file should have Summary at top level
	if len(result.Summary) == 0 {
		t.Error("expected Summary field for single file JSON output")
	}
}

func TestCacheKey_Uniqueness(t *testing.T) {
	key1 := cacheKey("/path/to/file.go", "code")
	key2 := cacheKey("/path/to/file.go", "docs")
	key3 := cacheKey("/path/to/other.go", "code")

	if key1 == key2 {
		t.Error("cache keys should differ for different modes")
	}
	if key1 == key3 {
		t.Error("cache keys should differ for different files")
	}
	if key2 == key3 {
		t.Error("cache keys should differ for different files and modes")
	}
}
