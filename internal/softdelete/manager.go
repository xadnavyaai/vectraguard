package softdelete

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// Manager handles soft delete operations (backup/restore).
type Manager struct {
	config       config.SoftDeleteConfig
	logger       *logging.Logger
	backupDir    string
	metadataPath string
}

// BackupEntry represents a single backup entry.
type BackupEntry struct {
	ID              string     `json:"id"`
	Timestamp       time.Time  `json:"timestamp"`
	OriginalCommand string     `json:"original_command"`
	Files           []FileInfo `json:"files"`
	SessionID       string     `json:"session_id,omitempty"`
	Agent           string     `json:"agent,omitempty"`
	TotalSize       int64      `json:"total_size"`
	IsGitBackup     bool       `json:"is_git_backup"`
}

// FileInfo represents a file in a backup.
type FileInfo struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path"`
	Size         int64  `json:"size"`
	IsGitFile    bool   `json:"is_git_file"`
	IsDir        bool   `json:"is_dir"`
}

// Metadata stores all backup entries.
type Metadata struct {
	Backups []BackupEntry `json:"backups"`
}

// NewManager creates a new soft delete manager.
func NewManager(ctx context.Context, cfg config.SoftDeleteConfig) (*Manager, error) {
	logger := logging.FromContext(ctx)

	backupDir := cfg.BackupDir
	if backupDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		backupDir = filepath.Join(home, ".vectra-guard", "backups")
	}

	// Expand ~ in path
	if strings.HasPrefix(backupDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		backupDir = strings.Replace(backupDir, "~", home, 1)
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	metadataPath := filepath.Join(backupDir, "metadata.json")

	m := &Manager{
		config:       cfg,
		logger:       logger,
		backupDir:    backupDir,
		metadataPath: metadataPath,
	}

	// Load existing metadata (ignore error, will create fresh if needed)
	_, _ = m.loadMetadata()

	return m, nil
}

// SoftDelete performs a soft delete operation (moves files to backup).
func (m *Manager) SoftDelete(ctx context.Context, cmdArgs []string, sessionID, agent string) (*BackupEntry, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("soft delete is disabled")
	}

	// Parse rm command to extract file paths
	paths, err := m.parseRmCommand(cmdArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rm command: %w", err)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no files to delete")
	}

	// Generate backup ID
	backupID := m.generateBackupID()
	backupEntry := BackupEntry{
		ID:              backupID,
		Timestamp:       time.Now(),
		OriginalCommand: strings.Join(cmdArgs, " "),
		Files:           []FileInfo{},
		SessionID:       sessionID,
		Agent:           agent,
		TotalSize:       0,
		IsGitBackup:     false,
	}

	backupDir := filepath.Join(m.backupDir, backupID)
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Process each path
	for _, path := range paths {
		fileInfos, err := m.backupPath(ctx, path, backupDir, &backupEntry)
		if err != nil {
			m.logger.Warn("failed to backup path", map[string]any{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}
		backupEntry.Files = append(backupEntry.Files, fileInfos...)
	}

	if len(backupEntry.Files) == 0 {
		// Clean up empty backup directory
		os.RemoveAll(backupDir)
		return nil, fmt.Errorf("no files were backed up")
	}

	// Calculate total size
	for _, f := range backupEntry.Files {
		backupEntry.TotalSize += f.Size
	}

	// Check if this is a git backup
	for _, f := range backupEntry.Files {
		if f.IsGitFile {
			backupEntry.IsGitBackup = true
			break
		}
	}

	// Save metadata
	if err := m.addBackupEntry(backupEntry); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	m.logger.Info("soft delete completed", map[string]any{
		"backup_id":   backupID,
		"files_count": len(backupEntry.Files),
		"total_size":  backupEntry.TotalSize,
		"is_git":      backupEntry.IsGitBackup,
	})

	// Auto-cleanup if enabled
	if m.config.AutoCleanup {
		if err := m.Cleanup(ctx); err != nil {
			m.logger.Warn("auto-cleanup failed", map[string]any{
				"error": err.Error(),
			})
		}
	}

	// Auto-delete old backups if enabled (permanent deletion)
	if m.config.AutoDelete {
		if err := m.AutoDeleteOldBackups(ctx); err != nil {
			m.logger.Warn("auto-delete failed", map[string]any{
				"error": err.Error(),
			})
		}
	}

	return &backupEntry, nil
}

// backupPath backs up a single path (file or directory).
func (m *Manager) backupPath(ctx context.Context, path, backupDir string, entry *BackupEntry) ([]FileInfo, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, skip
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var files []FileInfo

	if info.IsDir() {
		// Backup directory recursively
		err := filepath.Walk(absPath, func(filePath string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(absPath, filePath)
			if err != nil {
				return err
			}

			backupPath := filepath.Join(backupDir, "files", relPath)

			if fileInfo.IsDir() {
				// Create directory in backup
				if err := os.MkdirAll(backupPath, fileInfo.Mode()); err != nil {
					return err
				}
				files = append(files, FileInfo{
					OriginalPath: filePath,
					BackupPath:   backupPath,
					Size:         0,
					IsGitFile:    m.isGitFile(filePath),
					IsDir:        true,
				})
			} else {
				// Copy file to backup
				if err := m.copyFile(filePath, backupPath, fileInfo.Mode()); err != nil {
					return err
				}
				files = append(files, FileInfo{
					OriginalPath: filePath,
					BackupPath:   backupPath,
					Size:         fileInfo.Size(),
					IsGitFile:    m.isGitFile(filePath),
					IsDir:        false,
				})
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}

		// Remove original directory
		if err := os.RemoveAll(absPath); err != nil {
			return nil, fmt.Errorf("failed to remove original directory: %w", err)
		}
	} else {
		// Backup single file
		relPath := filepath.Base(absPath)
		backupPath := filepath.Join(backupDir, "files", relPath)

		if err := os.MkdirAll(filepath.Dir(backupPath), 0700); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}

		if err := m.copyFile(absPath, backupPath, info.Mode()); err != nil {
			return nil, fmt.Errorf("failed to copy file: %w", err)
		}

		files = append(files, FileInfo{
			OriginalPath: absPath,
			BackupPath:   backupPath,
			Size:         info.Size(),
			IsGitFile:    m.isGitFile(absPath),
			IsDir:        false,
		})

		// Remove original file
		if err := os.Remove(absPath); err != nil {
			return nil, fmt.Errorf("failed to remove original file: %w", err)
		}
	}

	return files, nil
}

// copyFile copies a file preserving permissions.
func (m *Manager) copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dst, data, mode); err != nil {
		return err
	}

	return nil
}

// generateBackupID generates a unique backup ID.
func (m *Manager) generateBackupID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// parseRmCommand parses rm command arguments to extract file paths.
func (m *Manager) parseRmCommand(args []string) ([]string, error) {
	var paths []string

	for i, arg := range args {
		// Skip flags
		if strings.HasPrefix(arg, "-") {
			continue
		}

		// Skip if it's the command name itself
		if i == 0 && (arg == "rm" || strings.HasSuffix(arg, "/rm")) {
			continue
		}

		// This should be a path
		if !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)
		}
	}

	return paths, nil
}

// loadMetadata loads backup metadata from disk.
func (m *Manager) loadMetadata() (*Metadata, error) {
	data, err := os.ReadFile(m.metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Metadata{Backups: []BackupEntry{}}, nil
		}
		return nil, err
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// saveMetadata saves backup metadata to disk.
func (m *Manager) saveMetadata(metadata *Metadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.metadataPath, data, 0600)
}

// addBackupEntry adds a backup entry to metadata.
func (m *Manager) addBackupEntry(entry BackupEntry) error {
	metadata, err := m.loadMetadata()
	if err != nil {
		return err
	}

	metadata.Backups = append(metadata.Backups, entry)

	return m.saveMetadata(metadata)
}

// GetBackup retrieves a backup entry by ID.
func (m *Manager) GetBackup(id string) (*BackupEntry, error) {
	metadata, err := m.loadMetadata()
	if err != nil {
		return nil, err
	}

	for i := range metadata.Backups {
		if metadata.Backups[i].ID == id {
			return &metadata.Backups[i], nil
		}
	}

	return nil, fmt.Errorf("backup not found: %s", id)
}

// ListBackups returns all backup entries.
func (m *Manager) ListBackups() ([]BackupEntry, error) {
	metadata, err := m.loadMetadata()
	if err != nil {
		return nil, err
	}

	return metadata.Backups, nil
}

// Cleanup removes old backups based on rotation policy.
func (m *Manager) Cleanup(ctx context.Context) error {
	metadata, err := m.loadMetadata()
	if err != nil {
		return err
	}

	now := time.Now()
	var toKeep []BackupEntry
	var totalSize int64

	// Sort by timestamp (newest first)
	backups := metadata.Backups
	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].Timestamp.Before(backups[j].Timestamp) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Apply rotation policy
	for _, backup := range backups {
		shouldKeep := true

		// Check age
		if m.config.MaxAgeDays > 0 {
			age := now.Sub(backup.Timestamp)
			if age > time.Duration(m.config.MaxAgeDays)*24*time.Hour {
				shouldKeep = false
			}
		}

		// Check count
		if shouldKeep && m.config.MaxBackups > 0 {
			if len(toKeep) >= m.config.MaxBackups {
				shouldKeep = false
			}
		}

		// Check size
		if shouldKeep && m.config.MaxSizeMB > 0 {
			if totalSize+backup.TotalSize > int64(m.config.MaxSizeMB)*1024*1024 {
				shouldKeep = false
			}
		}

		if shouldKeep {
			toKeep = append(toKeep, backup)
			totalSize += backup.TotalSize
		} else {
			// Delete backup
			backupDir := filepath.Join(m.backupDir, backup.ID)
			if err := os.RemoveAll(backupDir); err != nil {
				m.logger.Warn("failed to remove backup directory", map[string]any{
					"backup_id": backup.ID,
					"error":     err.Error(),
				})
			}
		}
	}

	// Update metadata
	metadata.Backups = toKeep
	return m.saveMetadata(metadata)
}

// AutoDeleteOldBackups permanently deletes backups older than the configured threshold.
func (m *Manager) AutoDeleteOldBackups(ctx context.Context) error {
	if !m.config.AutoDelete {
		return nil
	}

	if m.config.AutoDeleteAfterDays <= 0 {
		return nil // Auto-delete disabled or invalid threshold
	}

	metadata, err := m.loadMetadata()
	if err != nil {
		return err
	}

	now := time.Now()
	threshold := time.Duration(m.config.AutoDeleteAfterDays) * 24 * time.Hour
	var toKeep []BackupEntry
	deletedCount := 0
	deletedSize := int64(0)

	for _, backup := range metadata.Backups {
		age := now.Sub(backup.Timestamp)

		// Check if backup should be auto-deleted
		shouldDelete := age > threshold

		// Special protection: Never auto-delete git backups if protect_git is enabled
		if shouldDelete && m.config.ProtectGit && backup.IsGitBackup {
			// Keep git backups longer (2x the threshold)
			if age <= threshold*2 {
				shouldDelete = false
			}
		}

		if shouldDelete {
			// Permanently delete backup
			backupDir := filepath.Join(m.backupDir, backup.ID)
			if err := os.RemoveAll(backupDir); err != nil {
				m.logger.Warn("failed to remove backup directory during auto-delete", map[string]any{
					"backup_id": backup.ID,
					"error":     err.Error(),
				})
				// Keep in metadata if deletion failed
				toKeep = append(toKeep, backup)
			} else {
				deletedCount++
				deletedSize += backup.TotalSize
				m.logger.Info("auto-deleted old backup", map[string]any{
					"backup_id": backup.ID,
					"age_days":  int(age.Hours() / 24),
					"size":      backup.TotalSize,
				})
			}
		} else {
			toKeep = append(toKeep, backup)
		}
	}

	if deletedCount > 0 {
		metadata.Backups = toKeep
		if err := m.saveMetadata(metadata); err != nil {
			return err
		}

		m.logger.Info("auto-delete completed", map[string]any{
			"deleted_count":  deletedCount,
			"deleted_size":   deletedSize,
			"threshold_days": m.config.AutoDeleteAfterDays,
		})
	}

	return nil
}
