#!/bin/bash
# One-command installer: sandbox deps + vectra-guard release binary
set -e

REPO="xadnavyaai/vectra-guard"
INSTALL_DEPS="${INSTALL_DEPS:-1}"
DRY_RUN="${DRY_RUN:-0}"

echo "🛡️  Vectra Guard Full Installer"
echo "================================"
echo ""

if [ "$DRY_RUN" = "1" ]; then
    echo "⚠️  DRY_RUN enabled - no system changes will be made."
    echo ""
fi

echo "🚀 Installing vectra-guard..."
if [ "$DRY_RUN" = "1" ]; then
    echo "→ curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin} bash"
else
    curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/install.sh" | INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}" bash
fi

if [ "$INSTALL_DEPS" = "1" ]; then
    echo ""
    echo "📦 Installing sandbox dependencies..."
    if [ "$DRY_RUN" = "1" ]; then
        DRY_RUN=1 vectra-guard sandbox deps install
    else
        vectra-guard sandbox deps install
    fi
    echo ""
fi
