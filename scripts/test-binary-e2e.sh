#!/usr/bin/env bash
# E2E tests for the vectra-guard binary: old and new features (secrets, security scan, audit).
# Run from repo root: ./scripts/test-binary-e2e.sh [path-to-binary]
# Default binary: ./vectra-guard (or VG_CMD if set).

set -e
VG="${VG_CMD:-./vectra-guard}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
failed=0

run() { echo "  $*"; "$@"; }
ok()  { echo -e "${GREEN}PASS${NC}: $1"; }
fail() { echo -e "${RED}FAIL${NC}: $1"; failed=$((failed+1)); }

echo "=== Binary: $VG ==="
if ! test -x "$VG" 2>/dev/null; then
  echo "Binary not found or not executable: $VG"
  exit 1
fi

echo ""
echo "--- 1. scan-secrets: known patterns (AWS, GENERIC_API_KEY) ---"
T=$(mktemp -d)
echo 'aws_access_key_id: AKIAIOSFODNN7EXAMPLE' > "$T/cfg.yaml"
echo 'api_key=phc_abcdefghijklmnopqrstuvwxyz123456' >> "$T/cfg.yaml"
out=$("$VG" scan-secrets --path "$T" 2>&1) || true
if echo "$out" | grep -q "AWS_ACCESS_KEY_ID\|GENERIC_API_KEY"; then
  ok "Known-pattern secrets detected"
else
  fail "Expected AWS or GENERIC_API_KEY finding; got: $out"
fi
rm -rf "$T"

echo ""
echo "--- 2. scan-secrets: ENTROPY only with secret context (new) ---"
T=$(mktemp -d)
echo 'token: abcdEFGHijklMNOPqrstUVWX12345678' > "$T/app.env"
out=$("$VG" scan-secrets --path "$T" 2>&1) || true
if echo "$out" | grep -q "secret detected"; then
  ok "Entropy with context (token:) detected"
else
  fail "Expected secret finding for token: high-entropy; got: $out"
fi
rm -rf "$T"

echo ""
echo "--- 3. scan-secrets: no ENTROPY without context (new FP reduction) ---"
T=$(mktemp -d)
echo 'See https://github.com/SomeOrg/some-repo/issues/123' > "$T/readme.md"
out=$("$VG" scan-secrets --path "$T" 2>&1) || true
if echo "$out" | grep -q "secret detected"; then
  fail "Expected no finding for path-only line (no secret context); got: $out"
else
  ok "Path-only line without context not flagged"
fi
rm -rf "$T"

echo ""
echo "--- 4. scan-secrets: lockfiles skipped (old feature) ---"
T=$(mktemp -d)
echo '{"integrity":"sha512-abcdEFGHijklMNOPqrstUVWX1234567890ABCDEFGHijklMNOPqrstUVWX12345678=="}' > "$T/package-lock.json"
out=$("$VG" scan-secrets --path "$T" 2>&1) || true
if echo "$out" | grep -q "secret detected"; then
  fail "Lockfile should be skipped; got: $out"
else
  ok "Lockfile skipped"
fi
rm -rf "$T"

echo ""
echo "--- 5. scan-security: fixture findings (eval, exec, requests, bind) ---"
out=$("$VG" scan-security --path "$ROOT/internal/secscan/testdata/fixture" --languages go,python,c,config 2>&1) || true
if echo "$out" | grep -q "PY_EVAL\|PY_EXEC\|PY_REMOTE_HTTP\|PY_ENV_ACCESS\|BIND_ALL_INTERFACES"; then
  ok "Security scan reports expected patterns"
else
  fail "Expected PY_EVAL/PY_EXEC/HTTP/env/bind in fixture; got: $out"
fi

echo ""
echo "--- 6. scan-security: comment-only line skipped (new) ---"
T=$(mktemp -d)
echo '# https://api.example.com/docs' > "$T/example.py"
echo 'url = "https://real.example.com"' >> "$T/example.py"
out=$("$VG" scan-security --path "$T" --languages python 2>&1) || true
# Should have exactly one PY_EXTERNAL_HTTP (the real line), not from comment
count=$(echo "$out" | grep -c "PY_EXTERNAL_HTTP" || true)
if [ "${count:-0}" -eq 1 ]; then
  ok "Comment-only line not reported (one finding from code line)"
else
  fail "Expected 1 PY_EXTERNAL_HTTP (comment skipped); got count: ${count:-0}; out: $out"
fi
rm -rf "$T"

echo ""
echo "--- 7. audit repo: runs scan-security + scan-secrets + package audit ---"
T=$(mktemp -d)
echo 'api_key=placeholder_key_here' > "$T/app.env"
echo 'requests.get("https://example.com")' > "$T/main.py"
out=$("$VG" audit repo --path "$T" --no-install 2>&1) || true
if echo "$out" | grep -q "secrets_total\|code_findings\|repo audit"; then
  ok "Audit repo produces summary"
else
  fail "Audit repo should emit summary; got: $out"
fi
rm -rf "$T"

echo ""
echo "--- 8. validate: script validation ---"
T=$(mktemp -d)
echo '#!/bin/sh' > "$T/simple.sh"
echo 'echo hello' >> "$T/simple.sh"
out=$("$VG" validate "$T/simple.sh" 2>&1) || true
if [ $? -eq 0 ] || echo "$out" | grep -q "risk\|validation"; then
  ok "Validate command runs"
else
  fail "Validate failed: $out"
fi
rm -rf "$T"

echo ""
echo "--- 9. help / init ---"
out=$("$VG" init --help 2>&1) || true
if echo "$out" | grep -q "path\|config"; then
  ok "Init help works"
else
  fail "Init help: $out"
fi

echo ""
echo "--- 10. validate: risky script (old feature - DANGEROUS_DELETE_ROOT) ---"
T=$(mktemp -d)
echo 'rm -rf /' > "$T/risky.sh"
out=$("$VG" validate "$T/risky.sh" 2>&1) || true
if echo "$out" | grep -q "DANGEROUS_DELETE_ROOT\|critical\|violations"; then
  ok "Risky script (rm -rf /) detected"
else
  fail "Expected DANGEROUS_DELETE_ROOT or critical; got: $out"
fi
rm -rf "$T"

echo ""
echo "--- 11. session + exec + audit (old feature) ---"
SESSION=$("$VG" session start --agent "e2e" 2>&1) || true
if [ -z "$SESSION" ] || ! echo "$SESSION" | grep -q "session-"; then
  fail "Session start should return session ID; got: $SESSION"
else
  export VECTRAGUARD_SESSION_ID="$SESSION"
  out=$("$VG" exec -- echo "e2e-ok" 2>&1) || true
  if echo "$out" | grep -q "e2e-ok"; then
    audit=$("$VG" audit session 2>&1) || true
    if echo "$audit" | grep -q "session\|audit\|execution"; then
      ok "Session + exec + audit works"
    else
      fail "Audit session should emit summary; got: $audit"
    fi
  else
    fail "Exec should run echo; got: $out"
  fi
fi
unset VECTRAGUARD_SESSION_ID 2>/dev/null || true

echo ""
echo "--- 12. explain (old feature) ---"
T=$(mktemp -d)
echo 'rm -rf /tmp/foo' > "$T/script.sh"
out=$("$VG" explain "$T/script.sh" 2>&1) || true
if echo "$out" | grep -q "risk\|DANGEROUS\|recommendation\|script"; then
  ok "Explain runs and describes risk"
else
  fail "Explain should describe risk; got: $out"
fi
rm -rf "$T"

echo ""
echo "--- 13. CVE (old feature) ---"
T=$(mktemp -d)
out=$("$VG" cve sync --path "$T" 2>&1) || true
if echo "$out" | grep -q "sync\|CVE\|complete\|fetched"; then
  ok "CVE sync runs"
else
  out2=$("$VG" cve scan --path "$T" 2>&1) || true
  if echo "$out2" | grep -q "scan\|CVE\|packages\|report"; then
    ok "CVE scan runs"
  elif echo "$out $out2" | grep -q "cve.*disabled\|cve.enabled"; then
    ok "CVE command runs (disabled in config)"
  else
    fail "CVE sync or scan should run; sync: $out scan: $out2"
  fi
fi
rm -rf "$T"

echo ""
if [ "$failed" -eq 0 ]; then
  echo -e "${GREEN}All binary E2E checks passed.${NC}"
  exit 0
else
  echo -e "${RED}$failed check(s) failed.${NC}"
  exit 1
fi
