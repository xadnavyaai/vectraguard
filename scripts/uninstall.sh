#!/bin/bash
# Vectra Guard Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/uninstall.sh | bash

set -e

echo "🗑️  Vectra Guard Uninstaller"
echo "============================"
echo ""

BINARY_NAME="vectra-guard"
BIN_PATH="$(command -v vectra-guard 2>/dev/null || true)"
INSTALL_DIR="${BIN_PATH%/*}"

# Check if installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Vectra Guard is not installed"
    exit 0
fi

echo "Found Vectra Guard at: ${BIN_PATH}"
echo ""

# Confirm uninstall
# Use /dev/tty to read from terminal even when piped through curl | bash
if [ -n "${VECTRAGUARD_UNINSTALL_AUTO:-}" ]; then
    REPLY="y"
else
    if [ ! -t 0 ]; then
        # Check if we have access to /dev/tty (real terminal)
        if [ ! -c /dev/tty ]; then
            echo "⚠️  Running in non-interactive environment"
            echo "   Uninstall cancelled (requires interactive terminal)"
            exit 0
        fi
        # Read directly from /dev/tty when piped
        read -p "Remove Vectra Guard? [y/N] " -n 1 -r < /dev/tty
    else
        read -p "Remove Vectra Guard? [y/N] " -n 1 -r
    fi
fi
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Uninstall cancelled"
    exit 0
fi

echo ""
echo "📋 Uninstalling components..."
echo ""

# 1. Remove binary
echo "1/6: Removing binary..."
if [ -n "$BIN_PATH" ] && [ -f "$BIN_PATH" ]; then
    if [ -w "$INSTALL_DIR" ]; then
        rm -f "$BIN_PATH"
    else
        sudo rm -f "$BIN_PATH"
    fi
    echo "  ✅ Binary removed"
else
    echo "  ℹ️  Binary not found"
fi

DATA_REMOVED=false

# 2. Remove shell integration and restore from backups
echo ""
echo "2/6: Removing shell integration..."
REMOVED_INTEGRATION=false
RESTORED_FROM_BACKUP=false

# Process each shell config file
for shell_rc in ~/.bashrc ~/.zshrc ~/.config/fish/config.fish; do
    # Expand tilde to actual path
    shell_rc_expanded=$(eval echo "$shell_rc")
    
    # Determine backup file path
    case "$shell_rc" in
        ~/.bashrc)
            backup_file="$HOME/.bashrc.vectra-backup"
            ;;
        ~/.zshrc)
            backup_file="$HOME/.zshrc.vectra-backup"
            ;;
        ~/.config/fish/config.fish)
            backup_file="$HOME/.config/fish/config.fish.vectra-backup"
            ;;
        *)
            backup_file=""
            ;;
    esac
    
    if [ -f "$shell_rc_expanded" ]; then
        # Check if integration exists
        if grep -q "Vectra Guard Integration" "$shell_rc_expanded" 2>/dev/null || \
           grep -q "vectra-guard/tracker" "$shell_rc_expanded" 2>/dev/null; then
            # If backup exists, restore from backup
            if [ -f "$backup_file" ]; then
                echo "  ✅ Restoring $(basename $shell_rc_expanded) from backup"
                cp "$backup_file" "$shell_rc_expanded"
                # Remove the backup file after successful restore
                rm -f "$backup_file"
                RESTORED_FROM_BACKUP=true
                REMOVED_INTEGRATION=true
            else
                # No backup, just remove the integration section and safety aliases
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    # macOS (BSD sed)
                    sed -i '' '/# ====.*Vectra Guard Integration/,/# End Vectra Guard Integration/d' "$shell_rc_expanded"
                    sed -i '' '/# Vectra Guard Safety Aliases/,/fi/d' "$shell_rc_expanded"
                    sed -i '' '/# Vectra Guard Safety Aliases/,/end/d' "$shell_rc_expanded"
                    sed -i '' '/vectra-guard\/tracker\.bash/d' "$shell_rc_expanded"
                    sed -i '' '/vectra-guard\/tracker\.zsh/d' "$shell_rc_expanded"
                    sed -i '' '/vectra-guard\/tracker\.fish/d' "$shell_rc_expanded"
                else
                    # Linux (GNU sed)
                    sed -i '/# ====.*Vectra Guard Integration/,/# End Vectra Guard Integration/d' "$shell_rc_expanded"
                    sed -i '/# Vectra Guard Safety Aliases/,/fi/d' "$shell_rc_expanded"
                    sed -i '/# Vectra Guard Safety Aliases/,/end/d' "$shell_rc_expanded"
                    sed -i '/vectra-guard\/tracker\.bash/d' "$shell_rc_expanded"
                    sed -i '/vectra-guard\/tracker\.zsh/d' "$shell_rc_expanded"
                    sed -i '/vectra-guard\/tracker\.fish/d' "$shell_rc_expanded"
                fi
                echo "  ✅ Removed from $(basename $shell_rc_expanded)"
                REMOVED_INTEGRATION=true
            fi
        fi
    elif [ -f "$backup_file" ]; then
        # Config file doesn't exist but backup does - restore it
        echo "  ✅ Restoring $(basename $shell_rc_expanded) from backup"
        # Ensure directory exists
        mkdir -p "$(dirname "$shell_rc_expanded")"
        cp "$backup_file" "$shell_rc_expanded"
        rm -f "$backup_file"
        RESTORED_FROM_BACKUP=true
    fi
done

if [ "$REMOVED_INTEGRATION" = false ]; then
    echo "  ℹ️  No shell integration found"
fi

# 3. Remove session file
echo ""
echo "3/6: Cleaning up session data..."
if [ -f ~/.vectra-guard-session ]; then
    rm -f ~/.vectra-guard-session
    echo "  ✅ Session file removed"
else
    echo "  ℹ️  No session file found"
fi

# 4. Remove configs and data
echo ""
echo "4/6: Configuration and data cleanup..."
if [ -n "${VECTRAGUARD_UNINSTALL_AUTO:-}" ]; then
    if [ -n "${VECTRAGUARD_UNINSTALL_PURGE:-}" ]; then
        REPLY="y"
    else
        REPLY="n"
    fi
else
    if [ -c /dev/tty ]; then
        read -p "Remove all Vectra Guard configs/data (~/.config/vectra-guard, ~/.vectra-guard, ~/.vectra-guard-session, ~/vectra-guard.yaml)? [Y/n] " -n 1 -r < /dev/tty
    else
        read -p "Remove all Vectra Guard configs/data (~/.config/vectra-guard, ~/.vectra-guard, ~/.vectra-guard-session, ~/vectra-guard.yaml)? [Y/n] " -n 1 -r
    fi
fi
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    if [ -d ~/.vectra-guard ]; then
        rm -rf ~/.vectra-guard
        echo "  ✅ Removed data directory: ~/.vectra-guard"
    else
        echo "  ℹ️  No data directory found"
    fi
    if [ -d ~/.config/vectra-guard ]; then
        rm -rf ~/.config/vectra-guard
        echo "  ✅ Removed config directory: ~/.config/vectra-guard"
    else
        echo "  ℹ️  No config directory found"
    fi
    if [ -f ~/vectra-guard.yaml ]; then
        rm -f ~/vectra-guard.yaml
        echo "  ✅ Removed user config: ~/vectra-guard.yaml"
    fi
    if [ -f ~/.vectra-guard-session ]; then
        rm -f ~/.vectra-guard-session
        echo "  ✅ Removed session file: ~/.vectra-guard-session"
    fi
    if [ -f ~/.vectra-guard/session-index.json ]; then
        rm -f ~/.vectra-guard/session-index.json
        echo "  ✅ Removed session index: ~/.vectra-guard/session-index.json"
    fi
    DATA_REMOVED=true
else
    echo "  ℹ️  Kept configs/data (remove manually if desired)."
    if [ -n "${VECTRAGUARD_UNINSTALL_AUTO:-}" ]; then
        if [ -n "${VECTRAGUARD_UNINSTALL_CACHE_ONLY:-}" ]; then
            REPLY="y"
        else
            REPLY="n"
        fi
    else
        if [ -c /dev/tty ]; then
            read -p "Remove cache/metrics only (~/.vectra-guard/cache, ~/.vectra-guard/metrics.json, ~/.vectra-guard/sessions)? [y/N] " -n 1 -r < /dev/tty
        else
            read -p "Remove cache/metrics only (~/.vectra-guard/cache, ~/.vectra-guard/metrics.json, ~/.vectra-guard/sessions)? [y/N] " -n 1 -r
        fi
    fi
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf ~/.vectra-guard/cache ~/.vectra-guard/sessions
        rm -f ~/.vectra-guard/metrics.json
        echo "  ✅ Cache/metrics removed"
    fi
fi

# 5. Remove tracker files (if any remain)
echo ""
echo "5/6: Removing shell tracker files..."
TRACKER_REMOVED=false
for tracker in ~/.vectra-guard/tracker.bash ~/.vectra-guard/tracker.zsh ~/.vectra-guard/tracker.fish; do
    if [ -f "$tracker" ]; then
        rm -f "$tracker"
        echo "  ✅ Removed $(basename "$tracker")"
        TRACKER_REMOVED=true
    fi
done
if [ "$TRACKER_REMOVED" = false ]; then
    echo "  ℹ️  No tracker files found"
fi

# 6. Remove pre-commit hook (current repo only if present)
echo ""
echo "6/6: Removing git pre-commit hook (current repo if present)..."
if [ -d .git ] && [ -f .git/hooks/pre-commit ]; then
    if grep -q "Vectra Guard Pre-commit Hook" .git/hooks/pre-commit 2>/dev/null; then
        rm -f .git/hooks/pre-commit
        echo "  ✅ Removed Vectra Guard pre-commit hook from .git/hooks/pre-commit"
    else
        echo "  ℹ️  Pre-commit hook exists but not created by Vectra Guard (left untouched)"
    fi
else
    echo "  ℹ️  No git pre-commit hook found in current directory"
fi

# 6. Check for remaining backups (should be none if restored)
echo ""
echo "Checking for remaining backups..."
FOUND_BACKUPS=false

for backup in ~/.bashrc.vectra-backup ~/.zshrc.vectra-backup ~/.config/fish/config.fish.vectra-backup; do
    if [ -f "$backup" ]; then
        echo "  ℹ️  Backup still exists: $backup"
        echo "     (This backup was not used - config may have been manually modified)"
        FOUND_BACKUPS=true
    fi
done

if [ "$FOUND_BACKUPS" = false ] && [ "$RESTORED_FROM_BACKUP" = true ]; then
    echo "  ✅ All backups restored and cleaned up"
elif [ "$FOUND_BACKUPS" = false ]; then
    echo "  ℹ️  No backups found"
fi

echo ""
echo "=========================================="
echo "✅ Vectra Guard Uninstalled"
echo "=========================================="
echo ""

if [ "$REMOVED_INTEGRATION" = true ]; then
    echo "⚠️  Shell integration was removed."
    echo "   Please restart your terminal or run:"
    echo "   source ~/.bashrc  # or ~/.zshrc"
    echo ""
fi

if [ "$DATA_REMOVED" = true ]; then
    echo "✅ Config/data removed (~/.config/vectra-guard, ~/.vectra-guard, ~/.vectra-guard-session, ~/vectra-guard.yaml)"
    echo ""
else
    echo "ℹ️  Config/data kept. To remove manually:"
    echo "   rm -rf ~/.config/vectra-guard ~/.vectra-guard ~/.vectra-guard-session ~/vectra-guard.yaml"
    echo ""
fi

if [ "$RESTORED_FROM_BACKUP" = true ]; then
    echo "✅ Shell configs restored from backups"
    echo ""
elif [ "$FOUND_BACKUPS" = true ]; then
    echo "ℹ️  Some backup files still exist (configs may have been manually modified)"
    echo "   These backups were not automatically restored"
    echo ""
fi

echo "To reinstall: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash"
echo ""

