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

# Run namespace tests (local, safe - only detection logic)
test-namespace:
	go test -v ./internal/sandbox/namespace/... ./internal/sandbox/runtime_test.go

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
	@echo "  test-namespace - Run namespace tests (local, safe)"
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
	@echo "  make test-docker          # Recommended: All tests in Docker"
	@echo "  make test-local-quick     # Quick local validation"
	@echo "  make test-local-extensive # Full local validation"
	@echo "  make test-context         # Test context summarize features"
	@echo "  make test-context-docker  # Test context summarize in Docker"