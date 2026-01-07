#!/bin/bash
# Vectra Guard Quick Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash

set -e

REPO="xadnavyaai/vectra-guard"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="vectra-guard"

echo "🛡️  Vectra Guard Installer"
echo "=========================="
echo ""

# Check if already installed
if command -v vectra-guard &> /dev/null; then
    INSTALLED_VERSION=$(vectra-guard --help 2>&1 | head -1 || echo "unknown")
    echo "ℹ️  Vectra Guard is already installed"
    echo "   Location: $(which vectra-guard)"
    echo "   Version: $INSTALLED_VERSION"
    echo ""
    
    # Prompt for upgrade (use /dev/tty if piped through curl | bash)
    if [ ! -t 0 ]; then
        # Check if we have access to /dev/tty (real terminal)
        if [ ! -c /dev/tty ]; then
            echo "⚡ Non-interactive environment - auto-upgrading..."
            UPGRADE=true
        else
            # Read directly from /dev/tty when piped
            read -p "Upgrade to latest version? [Y/n] " -n 1 -r < /dev/tty
            echo
            if [[ $REPLY =~ ^[Nn]$ ]]; then
                echo "❌ Installation cancelled"
                exit 0
            fi
            UPGRADE=true
        fi
    else
        read -p "Upgrade to latest version? [Y/n] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            echo "❌ Installation cancelled"
            exit 0
        fi
        UPGRADE=true
    fi
    echo ""
fi

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    *)
        echo "❌ Unsupported OS: $OS"
        echo "   Please install manually from: https://github.com/${REPO}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        echo "   Please install manually from: https://github.com/${REPO}"
        exit 1
        ;;
esac

echo "📋 System: $OS $ARCH"
echo ""

# Get latest release
echo "📦 Downloading Vectra Guard..."
BINARY_FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_FILENAME}"

# Create temp file
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

# Download
if command -v curl &> /dev/null; then
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
        echo "❌ Download failed. Release may not exist yet."
        echo "   Trying to build from source instead..."
        echo ""
        echo "   To build from source:"
        echo "   git clone https://github.com/${REPO}.git"
        echo "   cd vectra-guard"
        echo "   go build -o vectra-guard ."
        echo "   sudo cp vectra-guard /usr/local/bin/"
        rm -f "$TEMP_FILE"
        exit 1
    fi
elif command -v wget &> /dev/null; then
    if ! wget -q "$DOWNLOAD_URL" -O "$TEMP_FILE"; then
        echo "❌ Download failed. Release may not exist yet."
        echo "   Trying to build from source instead..."
        echo ""
        echo "   To build from source:"
        echo "   git clone https://github.com/${REPO}.git"
        echo "   cd vectra-guard"
        echo "   go build -o vectra-guard ."
        echo "   sudo cp vectra-guard /usr/local/bin/"
        rm -f "$TEMP_FILE"
        exit 1
    fi
else
    echo "❌ Need curl or wget to download"
    echo "   Please install curl or wget, or build from source"
    exit 1
fi

# Make executable
chmod +x "$TEMP_FILE"

# Install
echo "📝 Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
    echo "   (requires sudo)"
    sudo mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

# Verify installation
if command -v vectra-guard &> /dev/null; then
    NEW_VERSION=$(vectra-guard version 2>&1 | head -1 || vectra-guard --help 2>&1 | head -1 || echo "installed")
    echo ""
    
    if [ "${UPGRADE:-false}" = true ]; then
        echo "✅ Vectra Guard upgraded successfully!"
        echo ""
        if [ -n "${INSTALLED_VERSION:-}" ]; then
            echo "   Old: $INSTALLED_VERSION"
        fi
        echo "   New: $NEW_VERSION"
    else
        echo "✅ Vectra Guard installed successfully!"
        echo "   Version: $NEW_VERSION"
    fi
    
    echo ""
    echo "🚀 Quick Start:"
    echo ""
    echo "   1. Validate a script (safe - never executes):"
    echo "      vectra-guard validate my-script.sh"
    echo ""
    echo "   2. Execute commands safely:"
    echo "      vectra-guard exec -- npm install"
    echo ""
    echo "   3. Get security explanations:"
    echo "      vectra-guard explain risky-script.sh"
    echo ""
    echo "   4. Optional - Initialize configuration:"
    echo "      vectra-guard init"
    echo ""
    echo "   5. Optional - Enable Universal Shell Protection (recommended):"
    echo "      curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-universal-shell-protection.sh | bash"
    echo ""
    echo "📚 Full Documentation: https://github.com/${REPO}"
    echo "🗑️  Uninstall: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/uninstall.sh | bash"
    echo ""
else
    echo ""
    echo "❌ Installation failed - binary not found in PATH"
    echo "   Installed to: $INSTALL_DIR/$BINARY_NAME"
    echo "   Please ensure $INSTALL_DIR is in your PATH"
    echo "   Or try: export PATH=\"$INSTALL_DIR:\$PATH\""
    exit 1
fi

