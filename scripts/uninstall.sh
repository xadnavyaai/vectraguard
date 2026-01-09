#!/bin/bash
# Vectra Guard Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/uninstall.sh | bash

set -e

echo "🗑️  Vectra Guard Uninstaller"
echo "============================"
echo ""

BINARY_NAME="vectra-guard"
INSTALL_DIR="/usr/local/bin"

# Check if installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Vectra Guard is not installed"
    exit 0
fi

echo "Found Vectra Guard at: $(which vectra-guard)"
echo ""

# Confirm uninstall
# Use /dev/tty to read from terminal even when piped through curl | bash
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
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Uninstall cancelled"
    exit 0
fi

echo ""
echo "📋 Uninstalling components..."
echo ""

# 1. Remove binary
echo "1/4: Removing binary..."
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    if [ -w "$INSTALL_DIR" ]; then
        rm -f "$INSTALL_DIR/$BINARY_NAME"
    else
        sudo rm -f "$INSTALL_DIR/$BINARY_NAME"
    fi
    echo "  ✅ Binary removed"
else
    echo "  ℹ️  Binary not found"
fi

# 2. Remove shell integration and restore from backups
echo ""
echo "2/4: Removing shell integration..."
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
        if grep -q "Vectra Guard Integration" "$shell_rc_expanded" 2>/dev/null; then
            # If backup exists, restore from backup
            if [ -f "$backup_file" ]; then
                echo "  ✅ Restoring $(basename $shell_rc_expanded) from backup"
                cp "$backup_file" "$shell_rc_expanded"
                # Remove the backup file after successful restore
                rm -f "$backup_file"
                RESTORED_FROM_BACKUP=true
                REMOVED_INTEGRATION=true
            else
                # No backup, just remove the integration section
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    # macOS (BSD sed)
                    sed -i '' '/# ====.*Vectra Guard Integration/,/# End Vectra Guard Integration/d' "$shell_rc_expanded"
                else
                    # Linux (GNU sed)
                    sed -i '/# ====.*Vectra Guard Integration/,/# End Vectra Guard Integration/d' "$shell_rc_expanded"
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
echo "3/4: Cleaning up session data..."
if [ -f ~/.vectra-guard-session ]; then
    rm -f ~/.vectra-guard-session
    echo "  ✅ Session file removed"
else
    echo "  ℹ️  No session file found"
fi

# 4. Ask about data directory
echo ""
echo "4/4: Configuration and session data..."
if [ -d ~/.vectra-guard ]; then
    echo "  ℹ️  Data directory found: ~/.vectra-guard"
    echo "  ℹ️  Keeping data directory (remove manually if desired: rm -rf ~/.vectra-guard)"
else
    echo "  ℹ️  No data directory found"
fi

# 5. Check for remaining backups (should be none if restored)
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

if [ -d ~/.vectra-guard ]; then
    echo "ℹ️  Data directory preserved at: ~/.vectra-guard"
    echo "   Remove manually if desired: rm -rf ~/.vectra-guard"
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

