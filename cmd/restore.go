package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/softdelete"
)

func runRestore(ctx context.Context, args []string) error {
	cfg := config.FromContext(ctx)

	if !cfg.SoftDelete.Enabled {
		return fmt.Errorf("soft delete is disabled. Enable it in config: soft_delete.enabled: true")
	}

	if len(args) == 0 {
		return runRestoreList(ctx)
	}

	subcommand := args[0]
	switch subcommand {
	case "list", "ls":
		return runRestoreList(ctx)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: vg restore show <backup-id>")
		}
		return runRestoreShow(ctx, args[1])
	case "clean":
		olderThan := 0
		if len(args) >= 2 {
			// Parse --older-than flag
			for i, arg := range args[1:] {
				if strings.HasPrefix(arg, "--older-than=") {
					fmt.Sscanf(strings.TrimPrefix(arg, "--older-than="), "%d", &olderThan)
				} else if arg == "--older-than" && i+1 < len(args)-1 {
					fmt.Sscanf(args[i+2], "%d", &olderThan)
				}
			}
		}
		return runRestoreClean(ctx, olderThan)
	case "stats":
		return runRestoreStats(ctx)
	case "delete", "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: vg restore delete <backup-id>")
		}
		return runRestoreDelete(ctx, args[1])
	case "auto-delete":
		return runRestoreAutoDelete(ctx)
	default:
		// Assume it's a backup ID to restore
		backupID := subcommand
		targetPath := ""
		if len(args) >= 2 {
			// Check for --to flag
			for i, arg := range args[1:] {
				if strings.HasPrefix(arg, "--to=") {
					targetPath = strings.TrimPrefix(arg, "--to=")
				} else if arg == "--to" && i+1 < len(args)-1 {
					targetPath = args[i+2]
				}
			}
		}
		return runRestoreBackup(ctx, backupID, targetPath)
	}
}

func runRestoreList(ctx context.Context) error {
	cfg := config.FromContext(ctx)

	mgr, err := softdelete.NewManager(ctx, cfg.SoftDelete)
	if err != nil {
		return fmt.Errorf("failed to initialize soft delete manager: %w", err)
	}

	backups, err := mgr.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	// Print header
	fmt.Printf("%-8s  %-19s  %-30s  %-6s  %-10s\n", "ID", "Timestamp", "Command", "Files", "Size")
	fmt.Println(strings.Repeat("-", 80))

	// Print backups
	for _, backup := range backups {
		timestamp := backup.Timestamp.Format("2006-01-02 15:04:05")
		command := backup.OriginalCommand
		if len(command) > 30 {
			command = command[:27] + "..."
		}
		filesCount := len(backup.Files)
		size := formatSize(backup.TotalSize)
		
		gitMark := ""
		if backup.IsGitBackup {
			gitMark = " ⚠️ GIT"
		}
		
		fmt.Printf("%-8s  %-19s  %-30s  %-6d  %-10s%s\n",
			backup.ID[:8], timestamp, command, filesCount, size, gitMark)
	}

	return nil
}

func runRestoreShow(ctx context.Context, backupID string) error {
	cfg := config.FromContext(ctx)

	mgr, err := softdelete.NewManager(ctx, cfg.SoftDelete)
	if err != nil {
		return fmt.Errorf("failed to initialize soft delete manager: %w", err)
	}

	backup, err := mgr.GetBackup(backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	fmt.Printf("Backup ID: %s\n", backup.ID)
	fmt.Printf("Timestamp: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Command: %s\n", backup.OriginalCommand)
	if backup.SessionID != "" {
		fmt.Printf("Session: %s\n", backup.SessionID)
	}
	if backup.Agent != "" {
		fmt.Printf("Agent: %s\n", backup.Agent)
	}
	fmt.Printf("Total Size: %s\n", formatSize(backup.TotalSize))
	fmt.Printf("Files: %d\n", len(backup.Files))
	if backup.IsGitBackup {
		fmt.Printf("⚠️  Git Backup: Yes\n")
	}
	
	fmt.Println("\nFiles:")
	for i, file := range backup.Files {
		if i >= 20 {
			fmt.Printf("  ... and %d more files\n", len(backup.Files)-20)
			break
		}
		fileType := "file"
		if file.IsDir {
			fileType = "dir"
		}
		gitMark := ""
		if file.IsGitFile {
			gitMark = " (git)"
		}
		fmt.Printf("  [%s] %s%s (%s)\n", fileType, file.OriginalPath, gitMark, formatSize(file.Size))
	}

	return nil
}

func runRestoreBackup(ctx context.Context, backupID string, targetPath string) error {
	cfg := config.FromContext(ctx)

	mgr, err := softdelete.NewManager(ctx, cfg.SoftDelete)
	if err != nil {
		return fmt.Errorf("failed to initialize soft delete manager: %w", err)
	}

	backup, err := mgr.GetBackup(backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	fmt.Printf("Restoring backup %s...\n", backupID)
	
	if err := mgr.Restore(ctx, backupID, targetPath); err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	fmt.Printf("✅ Restored %d files from backup %s\n", len(backup.Files), backupID)
	if targetPath != "" {
		fmt.Printf("   Restored to: %s\n", targetPath)
	} else {
		fmt.Printf("   Restored to original locations\n")
	}

	return nil
}

func runRestoreClean(ctx context.Context, olderThanDays int) error {
	cfg := config.FromContext(ctx)

	mgr, err := softdelete.NewManager(ctx, cfg.SoftDelete)
	if err != nil {
		return fmt.Errorf("failed to initialize soft delete manager: %w", err)
	}

	// Temporarily override max age if specified
	if olderThanDays > 0 {
		originalMaxAge := cfg.SoftDelete.MaxAgeDays
		cfg.SoftDelete.MaxAgeDays = olderThanDays
		defer func() {
			cfg.SoftDelete.MaxAgeDays = originalMaxAge
		}()
	}

	fmt.Println("Cleaning old backups...")
	
	if err := mgr.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	fmt.Println("✅ Cleanup completed")

	return nil
}

func runRestoreStats(ctx context.Context) error {
	cfg := config.FromContext(ctx)

	mgr, err := softdelete.NewManager(ctx, cfg.SoftDelete)
	if err != nil {
		return fmt.Errorf("failed to initialize soft delete manager: %w", err)
	}

	backups, err := mgr.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	var totalSize int64
	var totalFiles int
	var gitBackups int
	now := time.Now()
	var oldBackupsCount int
	var oldBackupsSize int64

	for _, backup := range backups {
		totalSize += backup.TotalSize
		totalFiles += len(backup.Files)
		if backup.IsGitBackup {
			gitBackups++
		}
		
		// Count backups that would be auto-deleted
		if cfg.SoftDelete.AutoDelete && cfg.SoftDelete.AutoDeleteAfterDays > 0 {
			age := now.Sub(backup.Timestamp)
			threshold := time.Duration(cfg.SoftDelete.AutoDeleteAfterDays) * 24 * time.Hour
			if age > threshold {
				oldBackupsCount++
				oldBackupsSize += backup.TotalSize
			}
		}
	}

	fmt.Printf("Backup Statistics:\n")
	fmt.Printf("  Total Backups: %d\n", len(backups))
	fmt.Printf("  Total Files: %d\n", totalFiles)
	fmt.Printf("  Total Size: %s\n", formatSize(totalSize))
	fmt.Printf("  Git Backups: %d\n", gitBackups)
	fmt.Printf("\nRotation Policy:\n")
	fmt.Printf("  Max Age: %d days\n", cfg.SoftDelete.MaxAgeDays)
	fmt.Printf("  Max Backups: %d\n", cfg.SoftDelete.MaxBackups)
	fmt.Printf("  Max Size: %s\n", formatSize(int64(cfg.SoftDelete.MaxSizeMB)*1024*1024))
	fmt.Printf("\nAuto-Delete:\n")
	if cfg.SoftDelete.AutoDelete {
		fmt.Printf("  Enabled: Yes\n")
		fmt.Printf("  Delete After: %d days\n", cfg.SoftDelete.AutoDeleteAfterDays)
		if oldBackupsCount > 0 {
			fmt.Printf("  ⚠️  %d backups eligible for auto-deletion (%s)\n", oldBackupsCount, formatSize(oldBackupsSize))
		}
	} else {
		fmt.Printf("  Enabled: No\n")
	}

	return nil
}

func runRestoreDelete(ctx context.Context, backupID string) error {
	cfg := config.FromContext(ctx)

	mgr, err := softdelete.NewManager(ctx, cfg.SoftDelete)
	if err != nil {
		return fmt.Errorf("failed to initialize soft delete manager: %w", err)
	}

	fmt.Printf("Deleting backup %s...\n", backupID)
	
	if err := mgr.DeleteBackup(ctx, backupID); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	fmt.Printf("✅ Backup %s deleted\n", backupID)

	return nil
}

func runRestoreAutoDelete(ctx context.Context) error {
	cfg := config.FromContext(ctx)

	if !cfg.SoftDelete.AutoDelete {
		return fmt.Errorf("auto-delete is disabled. Enable it in config: soft_delete.auto_delete: true")
	}

	mgr, err := softdelete.NewManager(ctx, cfg.SoftDelete)
	if err != nil {
		return fmt.Errorf("failed to initialize soft delete manager: %w", err)
	}

	fmt.Println("Running auto-delete for old backups...")
	
	if err := mgr.AutoDeleteOldBackups(ctx); err != nil {
		return fmt.Errorf("failed to auto-delete: %w", err)
	}

	fmt.Println("✅ Auto-delete completed")

	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
