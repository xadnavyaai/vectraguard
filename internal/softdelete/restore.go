package softdelete

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// Restore restores files from a backup.
func (m *Manager) Restore(ctx context.Context, backupID string, targetPath string) error {
	logger := logging.FromContext(ctx)

	backup, err := m.GetBackup(backupID)
	if err != nil {
		return err
	}

	restoredCount := 0
	skippedCount := 0

	// Group files by top-level directory to restore directories as a whole
	topLevelDirs := make(map[string]bool)
	topLevelFiles := make(map[string]FileInfo)

	for _, file := range backup.Files {
		// Check if this file is inside a top-level directory we're restoring
		isInsideTopLevel := false
		for topDir := range topLevelDirs {
			if strings.HasPrefix(file.OriginalPath, topDir+"/") || strings.HasPrefix(file.OriginalPath, topDir+"\\") {
				isInsideTopLevel = true
				break
			}
		}

		if isInsideTopLevel {
			continue // Will be restored as part of directory
		}

		// Check if this is a top-level directory
		isTopLevel := true
		for _, otherFile := range backup.Files {
			if otherFile.OriginalPath != file.OriginalPath &&
				(strings.HasPrefix(file.OriginalPath, otherFile.OriginalPath+"/") ||
					strings.HasPrefix(file.OriginalPath, otherFile.OriginalPath+"\\")) {
				isTopLevel = false
				break
			}
		}

		if file.IsDir && isTopLevel {
			topLevelDirs[file.OriginalPath] = true
		} else if !file.IsDir && isTopLevel {
			topLevelFiles[file.OriginalPath] = file
		}
	}

	// Restore top-level directories
	for dirPath := range topLevelDirs {
		// Find all files in this directory
		var dirFiles []FileInfo
		for _, file := range backup.Files {
			if file.OriginalPath == dirPath ||
				strings.HasPrefix(file.OriginalPath, dirPath+"/") ||
				strings.HasPrefix(file.OriginalPath, dirPath+"\\") {
				dirFiles = append(dirFiles, file)
			}
		}

		if len(dirFiles) == 0 {
			continue
		}

		// Find the root backup path for this directory
		var rootBackupPath string
		for _, file := range dirFiles {
			if file.OriginalPath == dirPath {
				rootBackupPath = file.BackupPath
				break
			}
		}

		if rootBackupPath == "" && len(dirFiles) > 0 {
			// Use the first file's backup path and go up to find the directory
			rootBackupPath = filepath.Dir(dirFiles[0].BackupPath)
		}

		var restorePath string
		if targetPath != "" {
			restorePath = filepath.Join(targetPath, filepath.Base(dirPath))
		} else {
			restorePath = dirPath
		}

		// Check if target already exists
		if _, err := os.Stat(restorePath); err == nil {
			logger.Warn("target already exists, skipping", map[string]any{
				"path": restorePath,
			})
			skippedCount++
			continue
		}

		// Restore directory
		if err := m.restoreDirectory(rootBackupPath, restorePath); err != nil {
			logger.Warn("failed to restore directory", map[string]any{
				"source": rootBackupPath,
				"target": restorePath,
				"error":  err.Error(),
			})
			continue
		}

		restoredCount++
	}

	// Restore top-level files
	for originalPath, file := range topLevelFiles {
		sourcePath := file.BackupPath

		var restorePath string
		if targetPath != "" {
			if len(topLevelFiles) == 1 && len(topLevelDirs) == 0 {
				restorePath = targetPath
			} else {
				restorePath = filepath.Join(targetPath, filepath.Base(originalPath))
			}
		} else {
			restorePath = originalPath
		}

		// Check if target already exists
		if _, err := os.Stat(restorePath); err == nil {
			logger.Warn("target already exists, skipping", map[string]any{
				"path": restorePath,
			})
			skippedCount++
			continue
		}

		// Restore file
		if err := m.restoreFile(sourcePath, restorePath); err != nil {
			logger.Warn("failed to restore file", map[string]any{
				"source": sourcePath,
				"target": restorePath,
				"error":  err.Error(),
			})
			continue
		}

		restoredCount++
	}

	logger.Info("restore completed", map[string]any{
		"backup_id":      backupID,
		"restored_count": restoredCount,
		"skipped_count":  skippedCount,
	})

	if restoredCount == 0 {
		return fmt.Errorf("no files were restored")
	}

	return nil
}

// restoreFile restores a single file.
func (m *Manager) restoreFile(source, target string) error {
	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	// Read source file
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	// Get file mode from source
	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	// Write to target
	if err := os.WriteFile(target, data, info.Mode()); err != nil {
		return err
	}

	return nil
}

// restoreDirectory restores a directory recursively.
func (m *Manager) restoreDirectory(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(target, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return m.restoreFile(path, targetPath)
	})
}

// DeleteBackup permanently deletes a backup.
func (m *Manager) DeleteBackup(ctx context.Context, backupID string) error {
	logger := logging.FromContext(ctx)

	// Check if backup exists
	_, err := m.GetBackup(backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Remove backup directory
	backupDir := filepath.Join(m.backupDir, backupID)
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("failed to remove backup directory: %w", err)
	}

	// Remove from metadata
	metadata, err := m.loadMetadata()
	if err != nil {
		return err
	}

	var updated []BackupEntry
	for _, backup := range metadata.Backups {
		if backup.ID != backupID {
			updated = append(updated, backup)
		}
	}

	metadata.Backups = updated
	if err := m.saveMetadata(metadata); err != nil {
		return err
	}

	logger.Info("backup deleted", map[string]any{
		"backup_id": backupID,
	})

	return nil
}
