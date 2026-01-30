#!/usr/bin/env bash
# Run Vectra Guard scan-security and audit repo on similar AI agent repos (Moltbot-style).
# Use release binary (vg) or local build: VG_CMD="go run ." ./scripts/test-similar-agent-repos.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKSPACES="${ROOT}/test-workspaces"

# Curated list of Moltbot-like AI agent repos (owner/repo). Dir name = last path segment.
SIMILAR_AGENT_REPOS=(
  moltbot/moltbot
  openinterpreter/open-interpreter
  Significant-Gravitas/Auto-GPT
  huginn/huginn
  mudler/LocalAGI
  Aider-AI/aider
)

# VG_CMD: "vg" (release on PATH) or "go run ." (local from repo root)
VG_CMD="${VG_CMD:-vg}"
[[ "$VG_CMD" == *"go run"* ]] && cd "$ROOT"

# Clone all repos from SIMILAR_AGENT_REPOS into test-workspaces. Skip if dir exists.
clone_all() {
  mkdir -p "$WORKSPACES"
  for entry in "${SIMILAR_AGENT_REPOS[@]}"; do
    local name="${entry##*/}"
    if [[ -d "$WORKSPACES/$name" ]]; then
      echo "Already exists: $WORKSPACES/$name (skip)"
      continue
    fi
    echo "Cloning https://github.com/$entry -> $WORKSPACES/$name"
    git clone --depth 1 "https://github.com/$entry" "$WORKSPACES/$name" || true
  done
}

# Optional: clone one repo into test-workspaces (usage: REPO_URL=url REPO_NAME=name, or pass as args)
# Example: ./scripts/test-similar-agent-repos.sh clone
# Example: REPO_URL=https://github.com/openinterpreter/open-interpreter REPO_NAME=open-interpreter ./scripts/test-similar-agent-repos.sh clone
clone_repo() {
  local url="${1:-}"
  local name="${2:-}"
  if [[ -z "$url" ]]; then
    url="${REPO_URL:-}"
    name="${REPO_NAME:-open-interpreter}"
  fi
  if [[ -z "$url" ]]; then
    echo "Usage: REPO_URL=<git-url> REPO_NAME=<dir> $0 clone"
    echo "Example: REPO_URL=https://github.com/openinterpreter/open-interpreter REPO_NAME=open-interpreter $0 clone"
    return 1
  fi
  mkdir -p "$WORKSPACES"
  if [[ -d "$WORKSPACES/$name" ]]; then
    echo "Already exists: $WORKSPACES/$name (pull to update)"
    return 0
  fi
  echo "Cloning $url -> $WORKSPACES/$name"
  git clone --depth 1 "$url" "$WORKSPACES/$name"
}

# When VG_OUTPUT=json, run_scan_json writes one audit JSON object to a temp file (tool format).
# Strip leading logger lines so the file is valid JSON (tool may log to stdout before emitting JSON).
run_scan_json() {
  local dir="$1"
  local outfile="$2"
  $VG_CMD audit repo --path "$dir" --no-install --output json 2>/dev/null | sed -n '/^{/,$p' > "$outfile" || true
}

run_scan() {
  local dir="$1"
  local name="$(basename "$dir")"
  if [[ "${VG_OUTPUT:-}" == "json" ]]; then
    return 0
  fi
  echo "--- scan-security: $name ---"
  $VG_CMD scan-security --path "$dir" --languages go,python,c,config 2>&1 || true
  echo ""
  echo "--- audit repo: $name ---"
  $VG_CMD audit repo --path "$dir" --no-install 2>&1 || true
  echo ""
}

main() {
  if [[ "${1:-}" == "clone-all" ]]; then
    clone_all
    exit 0
  fi
  if [[ "${1:-}" == "clone" ]]; then
    clone_repo "$2" "$3"
    exit 0
  fi

  if [[ ! -d "$WORKSPACES" ]]; then
    echo "No test-workspaces directory. Create it and add repos, or run:"
    echo "  mkdir -p $WORKSPACES"
    echo "  git clone --depth 1 https://github.com/moltbot/moltbot $WORKSPACES/moltbot"
    exit 1
  fi

  echo "Using VG_CMD: $VG_CMD"
  echo "Target: $WORKSPACES"
  if [[ "${VG_OUTPUT:-}" == "json" ]]; then
    echo "Output: vectra-guard tool format (audit repo --output json)"
    echo ""
    OUTPUT_JSON_FILE="${OUTPUT_JSON_FILE:-$ROOT/docs/reports/similar-agent-findings.json}"
    mkdir -p "$(dirname "$OUTPUT_JSON_FILE")"
    TMPDIR="${TMPDIR:-/tmp}"
    tmpdir="$(mktemp -d "$TMPDIR/vg-audit-XXXXXX")"
    i=0
    for dir in "$WORKSPACES"/*; do
      [[ -d "$dir" ]] || continue
      run_scan_json "$dir" "$tmpdir/repo-$i.json"
      ((i++)) || true
    done
    if command -v jq &>/dev/null && [[ "$i" -gt 0 ]]; then
      jq -s '{ repos: . }' "$tmpdir"/repo-*.json > "$OUTPUT_JSON_FILE" 2>/dev/null || cat "$tmpdir"/repo-*.json > "$OUTPUT_JSON_FILE"
    else
      cat "$tmpdir"/repo-*.json > "$OUTPUT_JSON_FILE" 2>/dev/null || true
    fi
    rm -rf "$tmpdir"
    echo "Wrote: $OUTPUT_JSON_FILE"
    return 0
  fi

  for dir in "$WORKSPACES"/*; do
    [[ -d "$dir" ]] || continue
    run_scan "$dir"
  done
}

main "$@"
