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

# Always build from main branch (latest code)
echo "📦 Building Vectra Guard from main branch..."
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Cannot build from source."
    echo ""
    echo "   Please install Go first:"
    echo "   - macOS: brew install go"
    echo "   - Linux: https://go.dev/doc/install"
    echo ""
    echo "   Or download a pre-built binary from:"
    echo "   https://github.com/${REPO}/releases"
    exit 1
fi

# Create temp file for binary
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

# Create temp directory for building
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

echo "📥 Cloning repository from main branch..."
if ! git clone --depth 1 --branch main "https://github.com/${REPO}.git" "$BUILD_DIR" 2>/dev/null; then
    echo "❌ Failed to clone repository"
    echo "   Please check your internet connection or install manually"
    exit 1
fi

echo "🔨 Building binary..."
cd "$BUILD_DIR"
if ! go build -o "$TEMP_FILE" .; then
    echo "❌ Build failed"
    echo "   Please report this issue: https://github.com/${REPO}/issues"
    exit 1
fi

echo "✅ Build successful!"
echo ""

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

