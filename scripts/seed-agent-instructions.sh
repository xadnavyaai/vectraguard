#!/bin/bash
# Seed agent-instruction files into a target repository.
set -e

TARGET_DIR="."
FORCE=0

usage() {
    echo "Usage: seed-agent-instructions.sh [--target <path>] [--force]"
    echo ""
    echo "Examples:"
    echo "  ./scripts/seed-agent-instructions.sh --target /path/to/repo"
    echo "  ./scripts/seed-agent-instructions.sh --force"
}

while [ $# -gt 0 ]; do
    case "$1" in
        --target)
            shift
            TARGET_DIR="${1:-}"
            ;;
        --force)
            FORCE=1
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "❌ Unknown option: $1"
            usage
            exit 1
            ;;
    esac
    shift
done

if [ -z "$TARGET_DIR" ]; then
    echo "❌ Missing --target value"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

sources=(
    "$REPO_ROOT/AGENTS.md"
    "$REPO_ROOT/CLAUDE.md"
    "$REPO_ROOT/CODEX.md"
    "$REPO_ROOT/.github/copilot-instructions.md"
    "$REPO_ROOT/.cursor/rules/vectra-guard.md"
    "$REPO_ROOT/.windsurf/rules.md"
    "$REPO_ROOT/.vscode/vectra-guard.instructions.md"
)

targets=(
    "$TARGET_DIR/AGENTS.md"
    "$TARGET_DIR/CLAUDE.md"
    "$TARGET_DIR/CODEX.md"
    "$TARGET_DIR/.github/copilot-instructions.md"
    "$TARGET_DIR/.cursor/rules/vectra-guard.md"
    "$TARGET_DIR/.windsurf/rules.md"
    "$TARGET_DIR/.vscode/vectra-guard.instructions.md"
)

echo "🧭 Seeding agent instructions into: $TARGET_DIR"

for i in "${!sources[@]}"; do
    src="${sources[$i]}"
    dst="${targets[$i]}"

    if [ ! -f "$src" ]; then
        echo "⚠️  Missing template: $src (skipping)"
        continue
    fi

    if [ -f "$dst" ] && [ "$FORCE" -ne 1 ]; then
        echo "↷ Exists, skipping: $dst"
        continue
    fi

    mkdir -p "$(dirname "$dst")"
    cp "$src" "$dst"
    echo "✅ Wrote: $dst"
done

echo "Done."
