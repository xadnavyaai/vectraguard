package softdelete

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func setupTestManager(t *testing.T, cfg config.SoftDeleteConfig) (*Manager, string, func()) {
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

	return mgr, tmpDir, cleanup
}

func TestNewManager(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	if mgr == nil {
		t.Fatal("manager is nil")
	}

	if mgr.backupDir == "" {
		t.Fatal("backupDir is empty")
	}

	// Check backup directory exists
	if _, err := os.Stat(mgr.backupDir); os.IsNotExist(err) {
		t.Fatalf("backup directory was not created: %s", mgr.backupDir)
	}

	// Check metadata file exists or can be created
	if _, err := os.Stat(mgr.metadataPath); os.IsNotExist(err) {
		// That's okay, it will be created on first use
	}
}

func TestSoftDelete_SingleFile(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Perform soft delete
	backup, err := mgr.SoftDelete(ctx, []string{"rm", testFile}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	if backup == nil {
		t.Fatal("backup entry is nil")
	}

	if backup.ID == "" {
		t.Fatal("backup ID is empty")
	}

	if len(backup.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(backup.Files))
	}

	// Check original file is deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatal("original file should be deleted")
	}

	// Check backup file exists
	backupFile := backup.Files[0].BackupPath
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Fatalf("backup file does not exist: %s", backupFile)
	}

	// Verify backup content
	backupContent, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	if string(backupContent) != testContent {
		t.Fatalf("backup content mismatch: expected %q, got %q", testContent, string(backupContent))
	}
}

func TestSoftDelete_Directory(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Create test directory structure
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

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Perform soft delete
	backup, err := mgr.SoftDelete(ctx, []string{"rm", "-rf", testDir}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Check original directory is deleted
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatal("original directory should be deleted")
	}

	// Check all files are backed up
	if len(backup.Files) < 3 { // directory + 2 files
		t.Fatalf("expected at least 3 entries (dir + 2 files), got %d", len(backup.Files))
	}

	// Verify backup files exist
	for _, file := range backup.Files {
		if !file.IsDir {
			if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
				t.Fatalf("backup file does not exist: %s", file.BackupPath)
			}
		}
	}
}

func TestSoftDelete_GitProtection(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.ProtectGit = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Create a .git directory
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}

	gitFile := filepath.Join(gitDir, "config")
	if err := os.WriteFile(gitFile, []byte("[core]"), 0644); err != nil {
		t.Fatalf("failed to create git config: %v", err)
	}

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Perform soft delete
	backup, err := mgr.SoftDelete(ctx, []string{"rm", "-rf", gitDir}, "session-1", "test-agent")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// Check git backup flag
	if !backup.IsGitBackup {
		t.Fatal("backup should be marked as git backup")
	}

	// Check git files are detected
	gitFileFound := false
	for _, file := range backup.Files {
		if file.IsGitFile {
			gitFileFound = true
			break
		}
	}

	if !gitFileFound {
		t.Fatal("git files should be detected")
	}
}

func TestSoftDelete_Disabled(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = false

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Should fail when disabled
	_, err := mgr.SoftDelete(ctx, []string{"rm", testFile}, "session-1", "test-agent")
	if err == nil {
		t.Fatal("soft delete should fail when disabled")
	}
}

func TestCleanup_AgeBased(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.AutoCleanup = false // Manual cleanup
	cfg.MaxAgeDays = 1      // Keep for 1 day
	cfg.MaxBackups = 100
	cfg.MaxSizeMB = 1024

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create old backup manually
	oldBackupID := "old-backup-1"
	oldBackupDir := filepath.Join(mgr.backupDir, oldBackupID)
	if err := os.MkdirAll(oldBackupDir, 0755); err != nil {
		t.Fatalf("failed to create old backup dir: %v", err)
	}

	// Create metadata with old backup
	metadata := &Metadata{
		Backups: []BackupEntry{
			{
				ID:        oldBackupID,
				Timestamp: time.Now().Add(-2 * 24 * time.Hour), // 2 days ago
				TotalSize: 100,
			},
		},
	}

	if err := mgr.saveMetadata(metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// Run cleanup
	if err := mgr.Cleanup(ctx); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Check old backup is removed
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	for _, backup := range backups {
		if backup.ID == oldBackupID {
			t.Fatal("old backup should be removed")
		}
	}
}

func TestCleanup_CountBased(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.AutoCleanup = false
	cfg.MaxAgeDays = 30
	cfg.MaxBackups = 3 // Keep only 3 backups
	cfg.MaxSizeMB = 1024

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create 5 backups
	var backups []BackupEntry
	for i := 0; i < 5; i++ {
		backupID := filepath.Join(mgr.backupDir, "backup-"+fmt.Sprintf("%d", i))
		if err := os.MkdirAll(backupID, 0755); err != nil {
			t.Fatalf("failed to create backup dir: %v", err)
		}

		backups = append(backups, BackupEntry{
			ID:        "backup-" + fmt.Sprintf("%d", i),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			TotalSize: 100,
		})
	}

	metadata := &Metadata{Backups: backups}
	if err := mgr.saveMetadata(metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// Run cleanup
	if err := mgr.Cleanup(ctx); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Check only 3 backups remain
	remaining, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	if len(remaining) > 3 {
		t.Fatalf("expected max 3 backups, got %d", len(remaining))
	}
}

func TestAutoDeleteOldBackups(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.AutoDelete = true
	cfg.AutoDeleteAfterDays = 1 // Delete after 1 day
	cfg.ProtectGit = true

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create old backup (2 days ago)
	oldBackupID := "old-backup-1"
	oldBackupDir := filepath.Join(mgr.backupDir, oldBackupID)
	if err := os.MkdirAll(oldBackupDir, 0755); err != nil {
		t.Fatalf("failed to create old backup dir: %v", err)
	}

	// Create recent backup (1 hour ago)
	recentBackupID := "recent-backup-1"
	recentBackupDir := filepath.Join(mgr.backupDir, recentBackupID)
	if err := os.MkdirAll(recentBackupDir, 0755); err != nil {
		t.Fatalf("failed to create recent backup dir: %v", err)
	}

	// Create metadata
	metadata := &Metadata{
		Backups: []BackupEntry{
			{
				ID:        oldBackupID,
				Timestamp: time.Now().Add(-2 * 24 * time.Hour), // 2 days ago
				TotalSize: 100,
			},
			{
				ID:        recentBackupID,
				Timestamp: time.Now().Add(-1 * time.Hour), // 1 hour ago
				TotalSize: 100,
			},
		},
	}

	if err := mgr.saveMetadata(metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// Run auto-delete
	if err := mgr.AutoDeleteOldBackups(ctx); err != nil {
		t.Fatalf("auto-delete failed: %v", err)
	}

	// Check old backup is deleted
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	oldFound := false
	recentFound := false
	for _, backup := range backups {
		if backup.ID == oldBackupID {
			oldFound = true
		}
		if backup.ID == recentBackupID {
			recentFound = true
		}
	}

	if oldFound {
		t.Fatal("old backup should be deleted")
	}

	if !recentFound {
		t.Fatal("recent backup should remain")
	}
}

func TestAutoDelete_GitProtection(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.AutoDelete = true
	cfg.AutoDeleteAfterDays = 1 // Delete after 1 day
	cfg.ProtectGit = true

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create old git backup (1.5 days ago, should be protected - within 2x threshold)
	gitBackupID := "git-backup-1"
	gitBackupDir := filepath.Join(mgr.backupDir, gitBackupID)
	if err := os.MkdirAll(gitBackupDir, 0755); err != nil {
		t.Fatalf("failed to create git backup dir: %v", err)
	}

	// Create old non-git backup (2 days ago, should be deleted)
	normalBackupID := "normal-backup-1"
	normalBackupDir := filepath.Join(mgr.backupDir, normalBackupID)
	if err := os.MkdirAll(normalBackupDir, 0755); err != nil {
		t.Fatalf("failed to create normal backup dir: %v", err)
	}

	// Create metadata
	metadata := &Metadata{
		Backups: []BackupEntry{
			{
				ID:          gitBackupID,
				Timestamp:   time.Now().Add(-36 * time.Hour), // 1.5 days ago (within 2x threshold)
				TotalSize:   100,
				IsGitBackup: true, // Git backup
			},
			{
				ID:          normalBackupID,
				Timestamp:   time.Now().Add(-2 * 24 * time.Hour), // 2 days ago
				TotalSize:   100,
				IsGitBackup: false,
			},
		},
	}

	if err := mgr.saveMetadata(metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// Run auto-delete
	if err := mgr.AutoDeleteOldBackups(ctx); err != nil {
		t.Fatalf("auto-delete failed: %v", err)
	}

	// Check git backup is protected (should remain)
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	gitFound := false
	normalFound := false
	for _, backup := range backups {
		if backup.ID == gitBackupID {
			gitFound = true
		}
		if backup.ID == normalBackupID {
			normalFound = true
		}
	}

	if !gitFound {
		t.Fatal("git backup should be protected and remain")
	}

	if normalFound {
		t.Fatal("normal old backup should be deleted")
	}
}

func TestAutoDelete_Disabled(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true
	cfg.AutoDelete = false // Disabled

	mgr, _, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create old backup
	oldBackupID := "old-backup-1"
	oldBackupDir := filepath.Join(mgr.backupDir, oldBackupID)
	if err := os.MkdirAll(oldBackupDir, 0755); err != nil {
		t.Fatalf("failed to create old backup dir: %v", err)
	}

	metadata := &Metadata{
		Backups: []BackupEntry{
			{
				ID:        oldBackupID,
				Timestamp: time.Now().Add(-100 * 24 * time.Hour), // Very old
				TotalSize: 100,
			},
		},
	}

	if err := mgr.saveMetadata(metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// Run auto-delete (should do nothing)
	if err := mgr.AutoDeleteOldBackups(ctx); err != nil {
		t.Fatalf("auto-delete failed: %v", err)
	}

	// Check backup still exists
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	found := false
	for _, backup := range backups {
		if backup.ID == oldBackupID {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("backup should remain when auto-delete is disabled")
	}
}

func TestGetBackup(t *testing.T) {
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

	// Get backup by ID
	retrieved, err := mgr.GetBackup(backup.ID)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}

	if retrieved.ID != backup.ID {
		t.Fatalf("backup ID mismatch: expected %s, got %s", backup.ID, retrieved.ID)
	}

	// Try to get non-existent backup
	_, err = mgr.GetBackup("non-existent")
	if err == nil {
		t.Fatal("should fail for non-existent backup")
	}
}

func TestListBackups(t *testing.T) {
	cfg := config.DefaultConfig().SoftDelete
	cfg.Enabled = true

	mgr, tmpDir, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	// Create multiple backups
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tmpDir, "test"+fmt.Sprintf("%d", i)+".txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if _, err := mgr.SoftDelete(ctx, []string{"rm", testFile}, "session-1", "test-agent"); err != nil {
			t.Fatalf("soft delete failed: %v", err)
		}
	}

	// List backups
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	if len(backups) != 3 {
		t.Fatalf("expected 3 backups, got %d", len(backups))
	}
}
