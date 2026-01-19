#!/bin/bash

# Script to verify CI/CD setup
# This checks that all required files exist and are properly configured

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║  Vectra Guard - CI/CD Setup Verification                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check counter
CHECKS_PASSED=0
CHECKS_FAILED=0

# Function to check file exists
check_file() {
    local file=$1
    local description=$2
    
    if [ -f "$PROJECT_ROOT/$file" ]; then
        echo -e "${GREEN}✓${NC} $description"
        ((CHECKS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $description (file not found: $file)"
        ((CHECKS_FAILED++))
        return 1
    fi
}

# Function to check directory exists
check_dir() {
    local dir=$1
    local description=$2
    
    if [ -d "$PROJECT_ROOT/$dir" ]; then
        echo -e "${GREEN}✓${NC} $description"
        ((CHECKS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $description (directory not found: $dir)"
        ((CHECKS_FAILED++))
        return 1
    fi
}

# Function to check file contains string
check_content() {
    local file=$1
    local pattern=$2
    local description=$3
    
    if [ -f "$PROJECT_ROOT/$file" ]; then
        if grep -q "$pattern" "$PROJECT_ROOT/$file"; then
            echo -e "${GREEN}✓${NC} $description"
            ((CHECKS_PASSED++))
            return 0
        else
            echo -e "${RED}✗${NC} $description (pattern not found)"
            ((CHECKS_FAILED++))
            return 1
        fi
    else
        echo -e "${RED}✗${NC} $description (file not found)"
        ((CHECKS_FAILED++))
        return 1
    fi
}

echo "1. Checking GitHub Actions workflows..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
check_dir ".github/workflows" "Workflows directory exists"
check_file ".github/workflows/ci.yml" "Main CI/CD pipeline workflow"
check_file ".github/workflows/release.yml" "Release workflow"
check_file ".github/workflows/pr-quick-check.yml" "PR quick check workflow"
check_file ".github/workflows/README.md" "Workflows documentation"
echo ""

echo "2. Checking workflow configurations..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
check_content ".github/workflows/ci.yml" "GO_VERSION" "CI workflow has Go version configured"
check_content ".github/workflows/ci.yml" "make test" "CI workflow runs tests"
check_content ".github/workflows/ci.yml" "make build" "CI workflow builds binary"
check_content ".github/workflows/release.yml" "v\*\.\*\.\*" "Release workflow triggers on version tags"
check_content ".github/workflows/pr-quick-check.yml" "pull_request" "PR quick check triggers on PRs"
echo ""

echo "3. Checking documentation..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
check_file "docs/CI_CD_GUIDE.md" "CI/CD comprehensive guide"
check_file "docs/CI_CD_SETUP_SUMMARY.md" "CI/CD setup summary"
check_content "README.md" "CI Status" "README has CI status badge"
echo ""

echo "4. Checking Makefile test targets..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
check_content "Makefile" "test:" "Makefile has test target"
check_content "Makefile" "test-internal:" "Makefile has test-internal target"
check_content "Makefile" "test-docker-pr:" "Makefile has test-docker-pr target"
check_content "Makefile" "build:" "Makefile has build target"
echo ""

echo "5. Checking Go configuration..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
check_file "go.mod" "Go module file"
check_file "go.sum" "Go dependencies file"
echo ""

echo "6. Validating YAML syntax..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check if yamllint is available
if command -v yamllint &> /dev/null; then
    for workflow in .github/workflows/*.yml; do
        if [ -f "$PROJECT_ROOT/$workflow" ]; then
            if yamllint "$PROJECT_ROOT/$workflow" &> /dev/null; then
                echo -e "${GREEN}✓${NC} YAML syntax valid: $workflow"
                ((CHECKS_PASSED++))
            else
                echo -e "${RED}✗${NC} YAML syntax invalid: $workflow"
                yamllint "$PROJECT_ROOT/$workflow" || true
                ((CHECKS_FAILED++))
            fi
        fi
    done
else
    echo -e "${YELLOW}⚠${NC} yamllint not installed (optional)"
    echo "  Install with: pip install yamllint"
    echo "  Skipping YAML syntax validation"
fi
echo ""

echo "7. Checking test infrastructure..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
check_file "docker-compose.test.yml" "Docker test configuration"

# Check if test files exist
if find "$PROJECT_ROOT" -name "*_test.go" -type f | grep -q .; then
    echo -e "${GREEN}✓${NC} Go test files found"
    ((CHECKS_PASSED++))
else
    echo -e "${RED}✗${NC} No Go test files found"
    ((CHECKS_FAILED++))
fi
echo ""

# Summary
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║  Verification Summary                                     ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo -e "  ${GREEN}Passed:${NC} $CHECKS_PASSED"
echo -e "  ${RED}Failed:${NC} $CHECKS_FAILED"
echo ""

if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ CI/CD setup verification complete - all checks passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Commit and push the changes"
    echo "  2. Create a pull request to test the workflows"
    echo "  3. Verify CI checks run automatically"
    echo ""
    exit 0
else
    echo -e "${RED}❌ CI/CD setup verification failed - please fix the issues above${NC}"
    echo ""
    exit 1
fi
