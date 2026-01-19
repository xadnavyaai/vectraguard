# CI/CD Guide for Vectra Guard

This guide explains the continuous integration and deployment setup for Vectra Guard.

## 📚 Table of Contents

- [Overview](#overview)
- [Workflows](#workflows)
- [Pre-commit Checklist](#pre-commit-checklist)
- [Pull Request Process](#pull-request-process)
- [Release Process](#release-process)
- [Troubleshooting](#troubleshooting)

## Overview

Vectra Guard uses GitHub Actions for CI/CD with three main workflows:

1. **PR Quick Check**: Fast feedback on pull requests (~2-5 minutes)
2. **CI/CD Pipeline**: Comprehensive testing (~10-20 minutes)
3. **Release**: Automated binary builds and GitHub releases

## Workflows

### 1. PR Quick Check (`pr-quick-check.yml`)

**Triggers**: On every PR open, update, or reopen

**What it does**:
- ✅ Checks Go code formatting
- ✅ Runs static analysis (`go vet`)
- ✅ Verifies build succeeds
- ✅ Runs fast unit tests

**Why it exists**: Provides rapid feedback to developers without waiting for full CI suite.

### 2. CI/CD Pipeline (`ci.yml`)

**Triggers**: On PR creation/update AND pushes to `main` branch

**Jobs**:

1. **Unit Tests**
   - Runs all Go unit tests: `make test`
   - Internal package tests: `make test-internal`
   - CVE tests: `make test-cve`

2. **Build Verification**
   - Builds binary: `make build`
   - Verifies binary exists and runs

3. **Docker Tests**
   - Extended tests in Docker
   - E2E tests in Docker
   - Comprehensive Docker test suite: `make test-docker-pr`

4. **Multi-platform Build**
   - Tests build on Ubuntu and macOS
   - Ensures cross-platform compatibility

5. **Lint & Code Quality**
   - Code formatting verification
   - Static analysis (`go vet`, `staticcheck`)

6. **Test Coverage**
   - Generates coverage report
   - Uploads coverage artifact

7. **CI Success**
   - Summary job that requires all previous jobs to pass

### 3. Release Workflow (`release.yml`)

**Triggers**: When version tag is pushed (format: `v*.*.*`, e.g., `v0.3.0`)

**Process**:

1. **Pre-release Tests**
   - Runs unit and internal tests
   - Must pass before building

2. **Build Binaries**
   - Builds for 4 platforms:
     - `linux-amd64`
     - `linux-arm64`
     - `darwin-amd64` (Intel Mac)
     - `darwin-arm64` (Apple Silicon)
   - Embeds version in binary
   - Generates SHA256 checksums

3. **Create GitHub Release**
   - Creates release with installation instructions
   - Uploads all binaries
   - Includes checksums file

4. **Homebrew Update** (optional)
   - Placeholder for Homebrew formula update

## Pre-commit Checklist

Before pushing code, run these checks locally:

### Quick Checks (2-3 minutes)

```bash
# Check code formatting
gofmt -s -l .

# If files are listed, format them
go fmt ./...

# Run static analysis
go vet ./...

# Build project
make build

# Run fast tests
go test -short -v ./...
```

### Comprehensive Tests (5-15 minutes)

```bash
# Run all unit tests
make test

# Run internal tests
make test-internal

# Run CVE tests
make test-cve

# Optional: Run Docker tests (requires Docker)
make test-docker-pr

# Optional: Run comprehensive test suite
make test-all-quick
```

## Pull Request Process

### Step 1: Create PR

1. Push your branch to GitHub
2. Open a Pull Request against `main`
3. Fill in PR description with:
   - What changes were made
   - Why the changes are needed
   - Any testing performed locally

### Step 2: Automated Checks

**Quick Check runs immediately** (~2-5 minutes):
- If it fails, fix the issues quickly (usually formatting)
- Push fixes, which will re-trigger the check

**CI/CD Pipeline runs in parallel** (~10-20 minutes):
- Runs comprehensive test suite
- Must pass for PR to be mergeable

### Step 3: Review

- Wait for CI checks to pass (green checkmarks)
- Address any review comments
- Push additional commits if needed (CI re-runs automatically)

### Step 4: Merge

- Once all checks pass and review is approved
- Merge the PR into `main`
- CI/CD runs again on `main` to ensure stability

## Release Process

### Preparing for Release

1. **Ensure `main` is stable**
   ```bash
   # All CI checks should be passing on main
   # View at: https://github.com/<org>/vectra-guard/actions
   ```

2. **Update version references** (if needed)
   - Update `CHANGELOG.md` or release notes
   - Ensure version-specific docs are current

3. **Local validation**
   ```bash
   # Run comprehensive tests
   make test-all-comprehensive
   
   # Build and test locally
   make build
   ./vectra-guard version
   ```

### Creating Release

1. **Create version tag**
   ```bash
   # Use semantic versioning: v<major>.<minor>.<patch>
   git tag -a v0.3.0 -m "Release v0.3.0: <brief description>"
   
   # Example with more detail
   git tag -a v0.3.0 -m "Release v0.3.0

   - Added feature X
   - Fixed bug Y
   - Improved performance Z"
   ```

2. **Push tag to trigger release**
   ```bash
   git push origin v0.3.0
   ```

3. **GitHub Actions automatically**:
   - Runs pre-release tests
   - Builds binaries for all platforms
   - Creates GitHub Release
   - Uploads binaries and checksums

4. **Monitor release workflow**
   ```bash
   # Visit: https://github.com/<org>/vectra-guard/actions
   # Click on "Release" workflow
   # Monitor progress (~10-15 minutes)
   ```

### Post-release

1. **Verify release**
   - Check GitHub Releases page
   - Download a binary and test it
   - Verify checksum matches

2. **Test installation**
   ```bash
   # Test the install script
   curl -fsSL https://raw.githubusercontent.com/<org>/vectra-guard/main/install.sh | bash
   
   # Verify version
   vectra-guard version
   ```

3. **Announce release** (optional)
   - Update documentation
   - Post to social media/blog
   - Notify users

## Troubleshooting

### Common Issues

#### 1. Formatting Check Fails

**Error**: "Go code is not formatted"

**Solution**:
```bash
# Format all Go files
go fmt ./...

# Check what will be formatted
gofmt -s -l .

# Commit and push
git add .
git commit -m "Fix formatting"
git push
```

#### 2. Go Vet Fails

**Error**: `go vet` reports issues

**Solution**:
```bash
# Run locally to see errors
go vet ./...

# Fix reported issues
# Common issues:
# - Unused variables
# - Incorrect printf formatting
# - Shadowed variables
# - Unreachable code

# Commit fixes
git add .
git commit -m "Fix vet issues"
git push
```

#### 3. Tests Fail

**Error**: Unit tests fail in CI

**Solution**:
```bash
# Run tests locally
make test

# Run specific test
go test -v ./path/to/package -run TestName

# Run with verbose output
go test -v ./...

# Fix failing tests
# Commit and push
```

#### 4. Docker Tests Fail

**Error**: Docker-based tests fail

**Solution**:
```bash
# Ensure Docker is running
docker ps

# Run Docker tests locally
make test-docker-pr

# Check Docker Compose config
docker-compose -f docker-compose.test.yml config

# Fix issues and push
```

#### 5. Build Fails

**Error**: Build fails in CI

**Solution**:
```bash
# Test build locally
make build

# Check for:
# - Missing dependencies
# - Syntax errors
# - Import issues

# Update dependencies
go mod tidy
go mod download

# Rebuild
make build
```

#### 6. Release Build Fails

**Error**: Release workflow fails

**Solution**:
- Ensure tag follows format: `v0.0.0`
- Check that all tests pass on main before tagging
- Verify Go version in workflow matches `go.mod`
- Check workflow logs for specific errors

### Getting Help

If you encounter issues:

1. **Check workflow logs**
   - Go to Actions tab in GitHub
   - Click on failed workflow
   - Review logs for each job

2. **Run locally first**
   - Always test locally before pushing
   - Use the same commands as CI

3. **Ask for help**
   - Open an issue
   - Include error messages and logs
   - Mention what you've already tried

## Best Practices

### For Contributors

1. **Always format code**: Run `go fmt ./...` before committing
2. **Run tests locally**: Don't rely on CI to catch basic errors
3. **Write tests**: Add tests for new features
4. **Keep PRs focused**: One feature/fix per PR
5. **Update docs**: Keep documentation in sync with code

### For Maintainers

1. **Review CI logs**: Check for flaky tests
2. **Monitor coverage**: Ensure test coverage doesn't decrease
3. **Keep dependencies updated**: Regularly update Go and dependencies
4. **Test releases**: Always test release binaries before announcing
5. **Document changes**: Keep CHANGELOG.md up to date

## Configuration

### Updating Go Version

To update the Go version used in CI:

1. Update `go.mod`:
   ```go
   go 1.26.0
   ```

2. Update workflow files:
   ```yaml
   env:
     GO_VERSION: '1.26.0'
   ```

3. Files to update:
   - `.github/workflows/ci.yml`
   - `.github/workflows/release.yml`
   - `.github/workflows/pr-quick-check.yml`

### Modifying Workflows

To modify workflow behavior:

1. Edit workflow files in `.github/workflows/`
2. Test changes in a feature branch
3. Create PR to review changes
4. Merge after verification

### Adding New Tests

To add new test categories:

1. Add test target to `Makefile`
2. Update CI workflow to run new tests
3. Document new tests in this guide

## Metrics and Monitoring

### CI Performance

Typical run times:
- PR Quick Check: 2-5 minutes
- CI/CD Pipeline: 10-20 minutes
- Release: 10-15 minutes

### Test Coverage

- Coverage reports available as artifacts
- Download from workflow run page
- View with: `go tool cover -html=coverage.out`

### Success Rates

Monitor workflow success rates:
- Go to Insights → Actions in GitHub
- Review workflow runs
- Identify patterns in failures

## Related Documentation

- [Workflows README](.github/workflows/README.md) - Detailed workflow documentation
- [Makefile](../Makefile) - Test targets and commands
- [FEATURES.md](../FEATURES.md) - Feature documentation
- [GETTING_STARTED.md](../GETTING_STARTED.md) - User guide

---

**Maintained by**: Vectra Guard Team
**Last Updated**: January 2026
**Questions?** Open an issue or discussion on GitHub
