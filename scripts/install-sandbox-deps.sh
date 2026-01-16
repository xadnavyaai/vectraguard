#!/bin/bash
# Install sandbox runtime dependencies (Docker/Podman + bubblewrap)
set -e

echo "🛡️  Vectra Guard Sandbox Dependencies"
echo "===================================="
echo ""

DRY_RUN="${DRY_RUN:-0}"

run_cmd() {
    if [ "$DRY_RUN" = "1" ]; then
        echo "→ $*"
        return 0
    fi
    "$@"
}

OS="$(uname -s)"

if [ "$OS" = "Darwin" ]; then
    if ! command -v brew &> /dev/null; then
        echo "❌ Homebrew not found."
        echo "   Install Homebrew first: https://brew.sh/"
        exit 1
    fi

    # Docker Desktop cask may conflict with existing hub-tool binary
    if [ -e /usr/local/bin/hub-tool ]; then
        echo "⚠️  Detected /usr/local/bin/hub-tool (Docker Desktop conflict)."
        echo "   Skipping Docker Desktop install to avoid forcing changes."
        echo "   You can install Docker Desktop manually if desired."
        exit 0
    fi

    echo "📦 Installing Docker Desktop..."
    run_cmd brew install --cask docker
    echo ""
    echo "✅ Docker Desktop installed."
    echo "   Open Docker.app once to finish setup."
    exit 0
fi

if [ "$OS" = "Linux" ]; then
    if [ -f /etc/os-release ]; then
        # shellcheck source=/dev/null
        . /etc/os-release
    fi

    if command -v apt-get &> /dev/null; then
        echo "📦 Installing Docker + bubblewrap (Debian/Ubuntu)..."
        run_cmd sudo apt-get update -y
        run_cmd sudo apt-get install -y docker.io docker-compose-plugin bubblewrap uidmap
        run_cmd sudo systemctl enable --now docker
        run_cmd sudo usermod -aG docker "$USER" || true
    elif command -v dnf &> /dev/null; then
        echo "📦 Installing Podman + bubblewrap (Fedora/RHEL)..."
        run_cmd sudo dnf install -y podman podman-docker bubblewrap
    elif command -v yum &> /dev/null; then
        echo "📦 Installing Podman + bubblewrap (CentOS/RHEL)..."
        run_cmd sudo yum install -y podman podman-docker bubblewrap
    else
        echo "❌ Unsupported Linux package manager."
        echo "   Please install Docker/Podman and bubblewrap manually."
        exit 1
    fi

    # Best-effort enable unprivileged user namespaces
    if sysctl -n kernel.unprivileged_userns_clone &> /dev/null; then
        run_cmd sudo sysctl -w kernel.unprivileged_userns_clone=1 || true
    fi

    echo ""
    echo "✅ Sandbox dependencies installed."
    echo "   Log out and back in to apply docker group changes."
    exit 0
fi

echo "❌ Unsupported OS: $OS"
exit 1
