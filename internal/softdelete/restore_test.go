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

func TestRestore_SingleFile(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create and soft delete a file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, Restore!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backup, err := mgr.SoftDelete(ctx, []string{"rm", testFile}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}

	// Restore file
	if err := mgr.Restore(ctx, backup.ID, ""); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify file is restored
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("file should be restored")
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(content) != testContent {
		t.Fatalf("content mismatch: expected %q, got %q", testContent, string(content))
	}
}

func TestRestore_Directory(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create directory structure
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	file1 := filepath.Join(testDir, "file1.txt")
	file2 := filepath.Join(testDir, "subdir", "file2.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(file2), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Soft delete directory
	backup, err := mgr.SoftDelete(ctx, []string{"rm", "-rf", testDir}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Verify directory is deleted
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatal("directory should be deleted")
	}

	// Restore directory
	if err := mgr.Restore(ctx, backup.ID, ""); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify directory is restored
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatal("directory should be restored")
	}

	// Verify files are restored
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Fatal("file1 should be restored")
	}

	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Fatal("file2 should be restored")
	}

	// Verify content
	content1, _ := os.ReadFile(file1)
	if string(content1) != "content1" {
		t.Fatalf("file1 content mismatch")
	}

	content2, _ := os.ReadFile(file2)
	if string(content2) != "content2" {
		t.Fatalf("file2 content mismatch")
	}
}

func TestRestore_ToDifferentLocation(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create and soft delete a file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, Restore!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backup, err := mgr.SoftDelete(ctx, []string{"rm", testFile}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Restore to different location
	targetPath := filepath.Join(tmpDir, "restored.txt")
	if err := mgr.Restore(ctx, backup.ID, targetPath); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify file is restored to new location
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("file should be restored to target location")
	}

	// Verify content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(content) != testContent {
		t.Fatalf("content mismatch: expected %q, got %q", testContent, string(content))
	}
}

func TestRestore_NonExistentBackup(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Try to restore non-existent backup
	err := mgr.Restore(ctx, "non-existent-id", "")
	if err == nil {
		t.Fatal("should fail for non-existent backup")
	}
}

func TestDeleteBackup(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create a backup
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backup, err := mgr.SoftDelete(ctx, []string{"rm", testFile}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Verify backup exists
	backupDir := filepath.Join(mgr.backupDir, backup.ID)
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Fatal("backup directory should exist")
	}

	// Delete backup
	if err := mgr.DeleteBackup(ctx, backup.ID); err != nil {
		t.Fatalf("delete backup failed: %v", err)
	}

	// Verify backup is deleted
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatal("backup directory should be deleted")
	}

	// Verify backup is removed from metadata
	_, err = mgr.GetBackup(backup.ID)
	if err == nil {
		t.Fatal("backup should not be retrievable after deletion")
	}
}

func TestDeleteBackup_NonExistent(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Try to delete non-existent backup
	err := mgr.DeleteBackup(ctx, "non-existent-id")
	if err == nil {
		t.Fatal("should fail for non-existent backup")
	}
}
