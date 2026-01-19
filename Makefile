.PHONY: build build-version test clean version help

# Default version for development
VERSION ?= dev

# ============================================================================
# BUILD
# ============================================================================

# Build without version (shows "dev")
build:
	go build -o vectra-guard .

# Build with version
build-version:
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "Usage: make build-version VERSION=v0.0.2"; \
		exit 1; \
	fi
	go build -ldflags "-X github.com/vectra-guard/vectra-guard/cmd.Version=$(VERSION)" -o vectra-guard .

# ============================================================================
# TESTING - DOCKER (Recommended - All Extended Tests)
# ============================================================================

# Detect docker compose command (newer: "docker compose", older: "docker-compose")
DOCKER_COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; else echo ""; fi)

# Run all extended tests in Docker (SAFE - isolated, comprehensive)
# Includes: extended tests, e2e tests, all Go tests
test-docker:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "❌ Docker is not installed"; \
		echo ""; \
		echo "Install Docker:"; \
		echo "  macOS: brew install --cask docker"; \
		echo "  Or download from: https://www.docker.com/products/docker-desktop"; \
		exit 1; \
	fi
	@if [ -z "$(DOCKER_COMPOSE)" ]; then \
		echo "❌ docker-compose is not available"; \
		echo ""; \
		echo "Install docker-compose:"; \
		echo "  macOS: brew install docker-compose"; \
		echo "  Or use Docker Desktop (includes compose)"; \
		echo "  Download: https://www.docker.com/products/docker-desktop"; \
		exit 1; \
	fi
	@echo "🚀 Running all extended tests in Docker..."
	@echo "ℹ️  Using: $(DOCKER_COMPOSE)"
	$(DOCKER_COMPOSE) -f docker-compose.test.yml run --rm --no-deps test-extended
	$(DOCKER_COMPOSE) -f docker-compose.test.yml run --rm --no-deps test-e2e
	$(DOCKER_COMPOSE) -f docker-compose.test.yml run --rm test-all

# Run Docker-based tests for PR changes (isolated execution tests)
# Tests runtime execution, timeouts, and sandbox behavior in Docker
test-docker-pr:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "❌ Docker is not installed"; \
		exit 1; \
	fi
	@if [ -z "$(DOCKER_COMPOSE)" ]; then \
		echo "❌ docker-compose is not available"; \
		exit 1; \
	fi
	@echo "🐳 Running Docker-based tests for PR changes..."
	@echo "Testing extended tests in Docker (PR #8 & #9)..."
	$(DOCKER_COMPOSE) -f docker-compose.test.yml run --rm --no-deps test-extended
	@echo "Testing e2e tests in Docker..."
	$(DOCKER_COMPOSE) -f docker-compose.test.yml run --rm --no-deps test-e2e
	@echo "Testing all Go tests in Docker..."
	$(DOCKER_COMPOSE) -f docker-compose.test.yml run --rm test-all
	@echo "✅ Docker-based tests completed"

# ============================================================================
# TESTING - LOCAL MODE
# ============================================================================

# Quick local tests (fast, basic validation)
# SAFE: Only uses 'validate' (static analysis, never executes)
test-local-quick:
	@echo "⚡ Running quick local tests (detection only, safe)..."
	./scripts/test-extended.sh --local --quick

# Extensive local tests (comprehensive validation)
# SAFE: Only uses 'validate' (static analysis, never executes)
test-local-extensive:
	@echo "🔍 Running extensive local tests (detection only, safe)..."
	./scripts/test-extended.sh --local

# ============================================================================
# BASIC TESTS
# ============================================================================

# Run Go unit tests (local)
test:
	go test -v ./...

# CVE test suite (unit tests only)
test-cve:
	go test -v ./internal/cve/... ./cmd -run TestParsePackageArg\|TestResolveCVECachePath\|TestShortSummary

# Run internal tests (safe - no execution, unit tests only)
# Tests: daemon, sandbox, runtime, analyzer, config
test-internal:
	@echo "🧪 Running internal Go unit tests (safe, no execution)..."
	@echo ""
	@echo "Testing daemon (PR #8 changes)..."
	go test -v ./internal/daemon/...
	@echo ""
	@echo "Testing sandbox (PR #8 & #9 changes)..."
	go test -v ./internal/sandbox/...
	@echo ""
	@echo "Testing runtime (PR #9 changes)..."
	go test -v ./internal/sandbox/... -run TestRuntime
	@echo ""
	@echo "Testing analyzer..."
	go test -v ./internal/analyzer/...
	@echo ""
	@echo "Testing config..."
	go test -v ./internal/config/...
	@echo ""
	@echo "Testing CVE..."
	@$(MAKE) test-cve
	@echo ""
	@echo "Testing other internal packages..."
	go test -v ./internal/logging/... ./internal/session/...
	@echo ""
	@echo "✅ Internal tests completed"

# Run namespace tests (local, safe - only detection logic)
test-namespace:
	go test -v ./internal/sandbox/namespace/... ./internal/sandbox/runtime_test.go

# Test PR #8 specific changes (daemon validation, seccomp)
test-pr8:
	@echo "🔍 Testing PR #8: Command validation and sandboxing..."
	go test -v ./internal/daemon/... -run TestCommand
	go test -v ./internal/sandbox/... -run TestDecideExecutionMode
	go test -v ./internal/sandbox/namespace/... -run TestSeccomp

# Test PR #9 specific changes (runtime execution, timeouts)
test-pr9:
	@echo "🔍 Testing PR #9: Runtime execution and timeouts..."
	go test -v ./internal/sandbox/runtime_test.go -run TestRuntime
	go test -v ./internal/sandbox/... -run TestExecute
	go test -v ./internal/sandbox/... -run TestTimeout

# ============================================================================
# CONTEXT SUMMARIZE TESTS
# ============================================================================

# Test context summarize features (local, quick)
test-context:
	@echo "🧪 Testing context summarize features..."
	@./scripts/test-context-summarize.sh --quick

# Test context summarize features (local, extensive)
test-context-extensive:
	@echo "🧪 Running extensive context summarize tests..."
	@./scripts/test-context-summarize.sh

# Test context summarize features (Docker, isolated)
test-context-docker:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "❌ Docker is not installed"; \
		exit 1; \
	fi
	@if [ -z "$(DOCKER_COMPOSE)" ]; then \
		echo "❌ docker-compose is not available"; \
		exit 1; \
	fi
	@echo "🚀 Running context summarize tests in Docker..."
	$(DOCKER_COMPOSE) -f docker-compose.test.yml run --rm --no-deps test-context-summarize

# ============================================================================
# COMPREHENSIVE TEST SUITE (All Tests in Order)
# ============================================================================

# Run all tests in the specified order:
# 1. Internal tests (Go unit tests - safe, no execution)
# 2. Docker-based tests (isolated execution tests)
# 3. Local quick tests (detection only)
# 4. Extensive tests (comprehensive validation)
test-all-comprehensive:
	@echo "╔═══════════════════════════════════════════════════════════╗"
	@echo "║  Comprehensive Test Suite - PR #8 & #9 Review           ║"
	@echo "╚═══════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Step 1/4: Running internal Go unit tests (safe, no execution)..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@$(MAKE) test-internal || (echo "❌ Internal tests failed!" && exit 1)
	@echo ""
	@echo "Step 2/4: Running Docker-based tests (isolated execution)..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@$(MAKE) test-docker-pr || (echo "❌ Docker tests failed!" && exit 1)
	@echo ""
	@echo "Step 3/4: Running local quick tests (detection only)..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@$(MAKE) test-local-quick || (echo "❌ Local quick tests failed!" && exit 1)
	@echo ""
	@echo "Step 4/4: Running extensive tests (comprehensive validation)..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@$(MAKE) test-local-extensive || (echo "❌ Extensive tests failed!" && exit 1)
	@echo ""
	@echo "╔═══════════════════════════════════════════════════════════╗"
	@echo "║  ✅ All Tests Completed Successfully!                    ║"
	@echo "╚═══════════════════════════════════════════════════════════╝"

# Quick comprehensive test (skips extensive tests)
test-all-quick:
	@echo "╔═══════════════════════════════════════════════════════════╗"
	@echo "║  Quick Test Suite - PR #8 & #9 Review                    ║"
	@echo "╚═══════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Step 1/3: Running internal Go unit tests..."
	@$(MAKE) test-internal || (echo "❌ Internal tests failed!" && exit 1)
	@echo ""
	@echo "Step 2/3: Running Docker-based tests..."
	@$(MAKE) test-docker-pr || (echo "❌ Docker tests failed!" && exit 1)
	@echo ""
	@echo "Step 3/3: Running local quick tests..."
	@$(MAKE) test-local-quick || (echo "❌ Local quick tests failed!" && exit 1)
	@echo ""
	@echo "✅ Quick test suite completed!"

# ============================================================================
# UTILITIES
# ============================================================================

# Clean build artifacts
clean:
	rm -f vectra-guard
	rm -rf dist

# Show current version (if binary exists)
version:
	@./vectra-guard version 2>/dev/null || echo "Binary not built. Run 'make build' first."

# Dev mode setup
dev-mode:
	./scripts/dev-mode.sh

dev-mode-force:
	./scripts/dev-mode.sh --force

# ============================================================================
# HELP
# ============================================================================

help:
	@echo "Vectra Guard - Makefile"
	@echo ""
	@echo "Build:"
	@echo "  build          - Build binary (version: dev)"
	@echo "  build-version  - Build with version (requires VERSION=v0.0.2)"
	@echo ""
	@echo "Testing - Docker (Recommended):"
	@echo "  test-docker    - Run ALL extended tests in Docker (comprehensive, safe)"
	@echo "                    Includes: extended tests + e2e tests + all Go tests"
	@echo ""
	@echo "Testing - Local Mode:"
	@echo "  test-local-quick      - Quick local tests (detection only, safe)"
	@echo "  test-local-extensive  - Extensive local tests (detection only, safe)"
	@echo ""
	@echo "Context Summarize Tests:"
	@echo "  test-context          - Quick context summarize tests (local)"
	@echo "  test-context-extensive - Extensive context summarize tests (local)"
	@echo "  test-context-docker   - Context summarize tests in Docker (isolated)"
	@echo ""
	@echo "Basic Testing:"
	@echo "  test          - Run Go unit tests (local)"
	@echo "  test-cve      - Run CVE unit tests (cache + scan helpers)"
	@echo "  test-internal - Run internal Go unit tests (safe, no execution)"
	@echo "  test-namespace - Run namespace tests (local, safe)"
	@echo "  test-pr8      - Test PR #8 specific changes (daemon, seccomp)"
	@echo "  test-pr9      - Test PR #9 specific changes (runtime, timeouts)"
	@echo ""
	@echo "Comprehensive Testing (PR Review):"
	@echo "  test-all-comprehensive - Run ALL tests in order (internal → docker → quick → extensive)"
	@echo "  test-all-quick        - Run quick test suite (internal → docker → quick)"
	@echo "  test-docker-pr        - Run Docker-based tests for PR changes"
	@echo ""
	@echo "Development:"
	@echo "  dev-mode      - Setup sandbox-based dev mode"
	@echo "  dev-mode-force - Force overwrite existing config"
	@echo ""
	@echo "Utilities:"
	@echo "  clean         - Remove build artifacts"
	@echo "  version       - Show current version"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test-all-comprehensive  # Full test suite (PR review)"
	@echo "  make test-all-quick          # Quick test suite (PR review)"
	@echo "  make test-internal           # Internal Go unit tests only"
	@echo "  make test-docker             # All tests in Docker"
	@echo "  make test-docker-pr          # Docker tests for PR changes"
	@echo "  make test-local-quick        # Quick local validation"
	@echo "  make test-local-extensive    # Full local validation"
	@echo "  make test-context            # Test context summarize features"
	@echo "  make test-context-docker     # Test context summarize in Docker"