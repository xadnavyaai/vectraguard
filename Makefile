.PHONY: build build-version test clean version

# Default version for development
VERSION ?= dev

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

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f vectra-guard
	rm -rf dist

# Show current version (if binary exists)
version:
	@./vectra-guard version 2>/dev/null || echo "Binary not built. Run 'make build' first."

# Dockerized testing
test-docker:
	./scripts/test-docker.sh

test-docker-quick:
	./scripts/test-docker.sh --quick

test-docker-security:
	./scripts/test-docker.sh --security

test-docker-destructive:
	./scripts/test-docker.sh --destructive

test-docker-shell:
	./scripts/test-docker.sh --shell

test-docker-clean:
	./scripts/test-docker.sh --clean

# Destructive testing (attack vectors)
test-destructive:
	./scripts/test-destructive.sh

test-destructive-quick:
	./scripts/test-destructive.sh --quick

test-destructive-docker:
	docker-compose -f docker-compose.test.yml run --rm --no-deps test-destructive

# Extended testing (comprehensive)
# DEFAULT: Docker mode (safe, isolated)
test-extended:
	@echo "⚠️  WARNING: Local testing uses 'validate' only (static analysis, never executes)"
	@echo "⚠️  For safer testing, use: make test-extended-docker"
	@echo ""
	@read -p "Continue with local testing? (y/N) " -n 1 -r; \
	echo; \
	if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Aborted. Use 'make test-extended-docker' for safe Docker testing."; \
		exit 1; \
	fi
	./scripts/test-extended.sh --local

# SAFE: Docker mode (recommended - all tests in isolated containers)
test-extended-docker:
	docker-compose -f docker-compose.test.yml run --rm --no-deps test-extended

# SAFE: Local tests in Docker (simulates dev local run - detection only)
# Runs in Docker but uses --local mode (only validate, never executes)
# Perfect for testing detection without execution risks
test-extended-local-docker:
	docker-compose -f docker-compose.test.yml run --rm --no-deps test-extended-local

# Release testing - tests the built release binaries in Docker
# Uses actual release artifacts from dist/ folder
# Tests Linux binaries inside Docker container (safe, isolated)
test-release-docker:
	@if [ ! -d "dist" ] || [ -z "$$(ls -A dist/vectra-guard-linux-* 2>/dev/null)" ]; then \
		echo "❌ ERROR: Release binaries not found in dist/ folder"; \
		echo "   Run: ./scripts/build-release.sh v0.0.1"; \
		exit 1; \
	fi
	docker-compose -f docker-compose.test.yml run --rm --no-deps test-release

# Alias for convenience
test-docker-extended: test-extended-docker
test-local-docker: test-extended-local-docker
test-release: test-release-docker

test-extended-quick:
	@echo "⚠️  WARNING: Local testing uses 'validate' only (static analysis, never executes)"
	@echo "⚠️  For safer testing, use: make test-extended-docker"
	@echo ""
	@read -p "Continue with local quick testing? (y/N) " -n 1 -r; \
	echo; \
	if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Aborted. Use 'make test-extended-docker' for safe Docker testing."; \
		exit 1; \
	fi
	./scripts/test-extended.sh --local --quick

# SAFE: Full two-phase testing (Docker only - no local execution)
test-extended-full:
	@echo "Step 1: Testing in Docker (execution verification)..."
	@make test-extended-docker || (echo "Docker tests failed! Fix issues before proceeding." && exit 1)
	@echo ""
	@echo "✅ All Docker tests passed!"
	@echo "⚠️  Local testing skipped for safety. Use 'make test-extended-docker' for all testing."

# Dev mode setup
dev-mode:
	./scripts/dev-mode.sh

dev-mode-force:
	./scripts/dev-mode.sh --force

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build          - Build binary (version: dev)"
	@echo "  build-version  - Build with version (requires VERSION=v0.0.2)"
	@echo ""
	@echo "Testing:"
	@echo "  test          - Run all tests"
	@echo "  test-docker   - Run tests in Docker container"
	@echo "  test-docker-quick - Run quick tests in Docker"
	@echo "  test-docker-security - Run security tests in Docker"
	@echo "  test-docker-destructive - Run destructive tests in Docker"
	@echo "  test-docker-shell - Interactive shell in test container"
	@echo "  test-docker-clean - Clean up test containers/images"
	@echo ""
	@echo "Destructive Testing (Attack Vectors):"
	@echo "  test-destructive - Run destructive test suite locally"
	@echo "  test-destructive-quick - Run quick destructive tests"
	@echo "  test-destructive-docker - Run destructive tests in Docker"
	@echo "  test-extended - Run extended destructive tests locally (requires confirmation)"
	@echo "  test-extended-docker - Run extended tests in Docker (execution + detection)"
	@echo "  test-extended-local-docker - Run local tests in Docker (detection only, safe)"
	@echo "  test-docker-extended - Alias for test-extended-docker"
	@echo "  test-local-docker - Alias for test-extended-local-docker"
	@echo ""
	@echo "Development:"
	@echo "  dev-mode      - Setup sandbox-based dev mode (easy setup)"
	@echo "  dev-mode-force - Force overwrite existing config"
	@echo ""
	@echo "Utilities:"
	@echo "  clean         - Remove build artifacts"
	@echo "  version       - Show current version"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make build-version VERSION=v0.0.1"
	@echo "  make test"
	@echo "  make test-docker"
	@echo "  make test-docker-security"
	@echo "  make test-destructive-docker"
	@echo "  make dev-mode"
