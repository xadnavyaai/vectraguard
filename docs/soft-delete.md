# Soft Delete Feature Design

## Overview

Vectra Guard's Soft Delete feature provides a safety net for file deletions by moving files to a rotating backup system instead of permanently deleting them. This allows for easy rollback and recovery, especially important for AI agents that might accidentally delete important files.

## Goals

1. **Prevent Permanent Data Loss**: Intercept `rm` commands and move files to backup instead
2. **Git Protection**: Protect `.git` directory and git config files to allow rollbacks
3. **Rotating Backups**: Automatically manage backup storage with rotation policies
4. **Easy Recovery**: Simple CLI commands to restore deleted files
5. **Agent Integration**: Make agents automatically use soft delete

## Architecture

### Components

1. **Soft Delete Manager** (`internal/softdelete/`)
   - Backup storage management
   - File tracking and metadata
   - Rotation policies
   - Restore operations

2. **Command Interception** (`cmd/exec.go`)
   - Detect `rm` commands
   - Replace with soft delete operation
   - Preserve command semantics

3. **Git Protection** (`internal/softdelete/git.go`)
   - Detect git-related files
   - Enhanced protection for `.git/`, `.gitignore`, `.gitconfig`, etc.
   - Prevent accidental git history loss

4. **CLI Commands** (`cmd/restore.go`)
   - `vg restore <backup-id>` - Restore a deleted file
   - `vg restore list` - List all deleted files
   - `vg restore clean` - Clean old backups
   - `vg restore show <backup-id>` - Show backup details

## Design Details

### Backup Storage Structure

```
~/.vectra-guard/backups/
├── metadata.json          # Index of all backups
├── 2026-01-19_14-30-15_abc123/
│   ├── original_path.txt  # Original file path
│   ├── deleted_at.txt     # Timestamp
│   ├── command.txt        # Original command
│   └── files/             # Actual backed up files
│       ├── file1.txt
│       └── dir1/
│           └── file2.txt
└── 2026-01-19_15-45-22_def456/
    └── ...
```

### Metadata Format

```json
{
  "backups": [
    {
      "id": "abc123",
      "timestamp": "2026-01-19T14:30:15Z",
      "original_command": "rm -rf test/",
      "files": [
        {
          "original_path": "/home/user/project/test/file.txt",
          "backup_path": "backups/abc123/files/test/file.txt",
          "size": 1024,
          "is_git_file": false
        }
      ],
      "session_id": "session-123",
      "agent": "cursor-ai"
    }
  ]
}
```

### Rotation Policy

- **Max Age**: Keep backups for N days (default: 30)
- **Max Count**: Keep maximum N backups (default: 100)
- **Max Size**: Maximum total backup size (default: 1GB)
- **Auto-cleanup**: Run cleanup on sync/restore operations (moves out of rotation)
- **Auto-delete**: Permanently delete backups older than N days (default: disabled)
  - When enabled, backups older than `auto_delete_after_days` are permanently deleted
  - Git backups get extra protection (2x threshold) if `protect_git` is enabled
  - Runs automatically after each backup operation

### Git Protection

Protected git files/patterns:
- `.git/` (entire directory)
- `.gitignore`
- `.gitattributes`
- `.gitconfig`
- `.gitmodules`
- `.gitkeep`
- `.git/HEAD`
- `.git/config`
- `.git/hooks/`
- `.git/refs/`
- `.git/objects/`

**Enhanced Protection**:
- Git files require explicit confirmation even with soft delete
- Multiple backup copies for git files
- Special restore workflow for git files

### Command Interception Flow

```
User/Agent: rm -rf test/
    ↓
Vectra Guard Analyzer: Detects rm command
    ↓
Soft Delete Interceptor: 
  - Parse rm command arguments
  - Resolve file paths
  - Check git protection
  - Create backup entry
  - Move files to backup
  - Log operation
    ↓
Return success (files "deleted" but backed up)
```

### Integration Points

1. **exec.go**: Intercept `rm` commands before execution
2. **analyzer.go**: Detect deletion operations
3. **session.go**: Track deletions in session logs
4. **config.go**: Add soft delete configuration

## Configuration

```yaml
soft_delete:
  enabled: true
  backup_dir: "~/.vectra-guard/backups"
  max_age_days: 30
  max_backups: 100
  max_size_mb: 1024
  auto_cleanup: true
  auto_delete: false  # Auto-delete old backups permanently
  auto_delete_after_days: 90  # Delete backups older than 90 days (if auto_delete enabled)
  protect_git: true
  git_backup_copies: 3  # Extra copies for git files
  rotation_policy: "age_and_count"  # age, count, size, age_and_count
```

## CLI Commands

### Restore Command

```bash
# List all deleted files
vg restore list

# Show details of a backup
vg restore show <backup-id>

# Restore a specific backup
vg restore <backup-id>

# Restore to a different location
vg restore <backup-id> --to /path/to/restore

# Clean old backups
vg restore clean

# Clean backups older than N days
vg restore clean --older-than 7

# Show backup statistics
vg restore stats
```

### Output Examples

```bash
$ vg restore list
ID       Timestamp              Command              Files    Size
abc123   2026-01-19 14:30:15   rm -rf test/         5        2.1 MB
def456   2026-01-19 15:45:22   rm file.txt          1        4.2 KB
ghi789   2026-01-19 16:20:10   rm -rf .git/         42       15.3 MB ⚠️ GIT

$ vg restore show abc123
Backup ID: abc123
Timestamp: 2026-01-19 14:30:15
Command: rm -rf test/
Session: session-123
Agent: cursor-ai
Files:
  - /home/user/project/test/file1.txt (1.2 MB)
  - /home/user/project/test/file2.txt (0.9 MB)
  - /home/user/project/test/subdir/file3.txt (0.1 MB)
Total: 5 files, 2.1 MB

$ vg restore abc123
✅ Restored 5 files from backup abc123
   Original location: /home/user/project/test/
   Restored to: /home/user/project/test/
```

## Agent Integration

Agents will automatically use soft delete when:
1. Running `rm` commands through `vg exec`
2. Soft delete is enabled in config
3. Files are moved to backup instead of deleted

Agent templates will be updated to:
- Mention soft delete feature
- Show how to restore files
- Explain backup rotation

## Security Considerations

1. **Permissions**: Backup directory should have restricted permissions (700)
2. **Sensitive Files**: Option to exclude sensitive files from backups
3. **Disk Space**: Monitor backup size and warn when approaching limits
4. **Git Protection**: Extra safeguards for git files

## Implementation Phases

### Phase 1: Core Soft Delete
- [x] Design document
- [ ] Configuration structure
- [ ] Backup storage system
- [ ] Basic file backup/restore
- [ ] Command interception

### Phase 2: Git Protection
- [ ] Git file detection
- [ ] Enhanced git protection
- [ ] Git-specific restore workflow

### Phase 3: Rotation & Management
- [ ] Rotation policies
- [ ] Auto-cleanup
- [ ] Statistics and monitoring

### Phase 4: CLI & Integration
- [ ] Restore commands
- [ ] Agent template updates
- [ ] Documentation

### Phase 5: Testing & Polish
- [ ] Comprehensive tests
- [ ] Edge case handling
- [ ] Performance optimization

## Future Enhancements

1. **Cloud Backup**: Optional cloud storage for backups
2. **Compression**: Compress old backups to save space
3. **Encryption**: Encrypt sensitive file backups
4. **Search**: Search backups by file name, date, etc.
5. **Web UI**: Web interface for backup management
6. **Notifications**: Notify when important files are deleted
