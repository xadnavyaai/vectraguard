package softdelete

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func setupTestManagerForGit(t *testing.T, cfg config.SoftDeleteConfig) (*Manager, func()) {
	tmpDir, err := os.MkdirTemp("", "vectra-guard-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cfg.BackupDir = filepath.Join(tmpDir, "backups")

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create manager: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return mgr, cleanup
}

func TestIsGitFile(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = true

	mgr, cleanup := setupTestManagerForGit(t, cfg)
	defer cleanup()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{".git directory", ".git", true},
		{".git directory with path", "/path/to/.git", true},
		{".gitignore file", ".gitignore", true},
		{".gitattributes file", ".gitattributes", true},
		{".gitconfig file", ".gitconfig", true},
		{".gitmodules file", ".gitmodules", true},
		{".gitkeep file", ".gitkeep", true},
		{"file inside .git", ".git/config", true},
		{"file inside .git subdir", ".git/refs/heads/main", true},
		{"normal file", "test.txt", false},
		{"normal directory", "testdir", false},
		{"file in subdirectory", "subdir/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mgr.isGitFile(tt.path)
			if result != tt.expected {
				t.Errorf("isGitFile(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsGitFile_Disabled(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = false // Git protection disabled

	mgr, cleanup := setupTestManagerForGit(t, cfg)
	defer cleanup()

	// Should return false when protection is disabled
	if mgr.isGitFile(".git") {
		t.Fatal("should return false when git protection is disabled")
	}

	if mgr.isGitFile(".gitignore") {
		t.Fatal("should return false when git protection is disabled")
	}
}

func TestIsGitProtected(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = true

	mgr, cleanup := setupTestManagerForGit(t, cfg)
	defer cleanup()

	if !mgr.IsGitProtected(".git") {
		t.Fatal(".git should be protected")
	}

	if !mgr.IsGitProtected(".gitignore") {
		t.Fatal(".gitignore should be protected")
	}

	if mgr.IsGitProtected("normal.txt") {
		t.Fatal("normal file should not be protected")
	}
}

func TestShouldBlockGitDeletion(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = true

	mgr, cleanup := setupTestManagerForGit(t, cfg)
	defer cleanup()

	// Test with git files
	paths := []string{".git", "test.txt"}
	shouldBlock, gitPaths := mgr.ShouldBlockGitDeletion(paths)

	if shouldBlock {
		t.Fatal("should not block (soft delete is allowed)")
	}

	if len(gitPaths) != 1 || gitPaths[0] != ".git" {
		t.Fatalf("expected 1 git path, got %v", gitPaths)
	}

	// Test with no git files
	paths = []string{"test.txt", "other.txt"}
	shouldBlock, gitPaths = mgr.ShouldBlockGitDeletion(paths)

	if shouldBlock {
		t.Fatal("should not block")
	}

	if len(gitPaths) != 0 {
		t.Fatalf("expected 0 git paths, got %v", gitPaths)
	}
}

func TestShouldBlockGitDeletion_Disabled(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = false

	mgr, cleanup := setupTestManagerForGit(t, cfg)
	defer cleanup()

	paths := []string{".git", ".gitignore"}
	shouldBlock, gitPaths := mgr.ShouldBlockGitDeletion(paths)

	if shouldBlock {
		t.Fatal("should not block")
	}

	if len(gitPaths) != 0 {
		t.Fatalf("expected 0 git paths when protection disabled, got %v", gitPaths)
	}
}

func TestGetGitProtectedPaths(t *testing.T) {
	protectedPaths := GetGitProtectedPaths()

	expectedPaths := []string{
		".git/",
		".gitignore",
		".gitattributes",
		".gitconfig",
		".gitmodules",
		".gitkeep",
		".git/HEAD",
		".git/config",
		".git/hooks/",
		".git/refs/",
		".git/objects/",
		".git/index",
		".git/logs/",
	}

	if len(protectedPaths) != len(expectedPaths) {
		t.Fatalf("expected %d protected paths, got %d", len(expectedPaths), len(protectedPaths))
	}

	// Check that all expected paths are present
	pathMap := make(map[string]bool)
	for _, path := range protectedPaths {
		pathMap[path] = true
	}

	for _, expected := range expectedPaths {
		if !pathMap[expected] {
			t.Errorf("expected protected path %q not found", expected)
		}
	}
}

func TestSoftDelete_GitFiles(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = true

	mgr, cleanup := setupTestManagerForGit(t, cfg)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "vectra-guard-test-files-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create .gitignore file
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	gitignoreContent := "*.log\n*.tmp\n"
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	// Soft delete .gitignore
	backup, err := mgr.SoftDelete(ctx, []string{"rm", gitignorePath}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Verify it's marked as git backup
	if !backup.IsGitBackup {
		t.Fatal("backup should be marked as git backup")
	}

	// Verify .gitignore file is detected
	gitFileFound := false
	for _, file := range backup.Files {
		if file.IsGitFile {
			gitFileFound = true
			break
		}
	}

	if !gitFileFound {
		t.Fatal(".gitignore should be detected as git file")
	}
}

func TestSoftDelete_GitDirectory(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = true

	mgr, cleanup := setupTestManagerForGit(t, cfg)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "vectra-guard-test-files-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create .git directory structure
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}

	// Create files inside .git
	configFile := filepath.Join(gitDir, "config")
	if err := os.WriteFile(configFile, []byte("[core]"), 0644); err != nil {
		t.Fatalf("failed to create git config: %v", err)
	}

	headFile := filepath.Join(gitDir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main"), 0644); err != nil {
		t.Fatalf("failed to create HEAD: %v", err)
	}

	// Soft delete .git directory
	backup, err := mgr.SoftDelete(ctx, []string{"rm", "-rf", gitDir}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Verify it's marked as git backup
	if !backup.IsGitBackup {
		t.Fatal("backup should be marked as git backup")
	}

	// Verify all git files are detected
	gitFileCount := 0
	for _, file := range backup.Files {
		if file.IsGitFile {
			gitFileCount++
		}
	}

	if gitFileCount == 0 {
		t.Fatal("git files should be detected")
	}
}
