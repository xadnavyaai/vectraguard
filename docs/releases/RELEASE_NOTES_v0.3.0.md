# Vectra Guard v0.3.0 Release Notes

Release Date: 2026-01-23

## 🎉 Major Features

### Soft Delete with Auto-Delete
Vectra Guard now includes a comprehensive soft delete feature that provides a safety net for file deletions!

- **Automatic Backup**: Intercepts `rm` commands and backs up files instead of permanently deleting
- **Rotating Backups**: Configurable retention policies based on age, count, and size
- **Auto-Delete**: Optionally automatically delete old backups permanently (configurable threshold)
- **Git Protection**: Enhanced protection for `.git`, `.gitignore`, `.gitattributes`, and other git files
- **Full Restore**: Restore deleted files to original or custom locations

#### New Commands
```bash
# List all backups
vg restore list

# Show backup details
vg restore show <backup-id>

# Restore a deleted file/directory
vg restore <backup-id>

# Restore to a different location
vg restore <backup-id> --to /path/to/restore

# Clean old backups (rotation)
vg restore clean

# Manually trigger auto-delete
vg restore auto-delete

# Show backup statistics
vg restore stats

# Permanently delete a backup
vg restore delete <backup-id>
```

#### Configuration
```yaml
soft_delete:
  enabled: true
  backup_dir: "~/.vectra-guard/backups"
  max_age_days: 30
  max_backups: 100
  max_size_mb: 1024
  auto_cleanup: true
  auto_delete: false
  auto_delete_after_days: 90
  protect_git: true
  git_backup_copies: 3
  rotation_policy: "age_and_count"
```

#### Features
- **Safe Deletion**: Files deleted via `rm` are automatically backed up
- **Git Protection**: Git files get 2x protection threshold when auto-delete is enabled
- **Rotation Policies**: Age-based, count-based, size-based, or combined
- **Auto-Cleanup**: Automatically rotates old backups based on policy
- **Auto-Delete**: Permanently deletes backups older than threshold (optional)
- **Comprehensive CLI**: Full backup management via command line

## 🔧 Improvements

- **Agent Templates**: Updated all agent templates (AGENTS.md, CLAUDE.md, CODEX.md) with soft delete documentation
- **Test Coverage**: Added 28 comprehensive tests for soft delete feature
- **Documentation**: Complete feature documentation in `docs/soft-delete.md`

## 📊 Statistics

- **New Files**: 7 files (manager, restore, git protection, tests)
- **New Commands**: 8 restore commands
- **Test Coverage**: 28 new tests
- **Lines Added**: ~3,000 lines of code

## 🛡️ Safety Features

- Critical deletions (like `rm -rf /`) are still blocked
- Soft delete only applies to safe deletions
- Git files get enhanced protection
- Configurable retention policies prevent disk space issues

## 🔄 Migration

No migration required. Feature is opt-in via configuration.

## 📝 Breaking Changes

None - feature is opt-in and backward compatible.

## 🐛 Bug Fixes

- Fixed code formatting issues for CI/CD compliance
- Improved git file detection logic

## 📚 Documentation

- Added comprehensive soft delete documentation
- Updated agent templates with feature usage
- Added configuration examples

## 🙏 Acknowledgments

This release adds a major safety feature for AI agents and developers working with file operations.
