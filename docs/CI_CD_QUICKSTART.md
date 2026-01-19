# CI/CD Quick Start

Get up and running with Vectra Guard's automated testing and releases in 5 minutes.

## 📦 What You Get

✅ **Automated Testing**: Every PR runs comprehensive tests  
✅ **Fast Feedback**: Quick checks in 2-5 minutes  
✅ **Multi-platform Builds**: Linux, macOS (amd64, arm64)  
✅ **Automated Releases**: One command to build and publish  
✅ **Code Quality**: Formatting and linting checks  

## 🚀 For Contributors (PRs)

### Step 1: Before You Push

```bash
# Format your code
go fmt ./...

# Run tests
make test

# Build to ensure it compiles
make build
```

### Step 2: Create Your PR

```bash
git push origin your-feature-branch
```

Then open a PR on GitHub.

### Step 3: Watch CI Run

Two checks run automatically:
1. **Quick Check** (~2-5 min): Fast validation
2. **CI Pipeline** (~10-20 min): Comprehensive tests

Both must pass ✅ before merge.

### Step 4: If CI Fails

**Most common issues:**

```bash
# Formatting error? Fix with:
go fmt ./...
git commit -am "Fix formatting"
git push

# Test failures? Run locally:
make test
# Fix the failing tests, then:
git commit -am "Fix tests"
git push

# Build errors? Check syntax:
go build ./...
```

CI re-runs automatically on every push.

## 📦 For Maintainers (Releases)

### Creating a New Release (3 steps)

**Step 1: Ensure main is stable**

```bash
# Check CI status on main branch
# Visit: https://github.com/YOUR_ORG/vectra-guard/actions
```

**Step 2: Create and push tag**

```bash
# Create annotated tag
git tag -a v0.3.0 -m "Release v0.3.0

Added features:
- Feature 1
- Feature 2

Bug fixes:
- Fix 1
"

# Push the tag
git push origin v0.3.0
```

**Step 3: Wait for automation**

GitHub Actions will automatically:
- ✅ Run pre-release tests
- ✅ Build binaries for 4 platforms
- ✅ Generate checksums
- ✅ Create GitHub Release
- ✅ Upload all assets

Time: ~10-15 minutes

### Verify Release

```bash
# After release is published, test it:
curl -L https://github.com/YOUR_ORG/vectra-guard/releases/download/v0.3.0/vectra-guard-linux-amd64 -o vectra-guard
chmod +x vectra-guard
./vectra-guard version
```

## 🔍 Monitoring

### Check CI Status

**Via GitHub:**
- Go to repository → Actions tab
- See all workflow runs
- Click any run for detailed logs

**Via Badge:**
- README shows CI status badge
- Green = passing, Red = failing

### Download Artifacts

After CI runs:
1. Go to Actions → Select workflow run
2. Scroll to "Artifacts" section
3. Download coverage reports or logs

## 🛠️ Common Tasks

### Run Tests Locally (Same as CI)

```bash
# Quick tests
make test

# All internal tests
make test-internal

# Docker tests (requires Docker)
make test-docker-pr

# Comprehensive suite
make test-all-quick
```

### Check Code Quality

```bash
# Format check
gofmt -s -l .

# Should output nothing (no files need formatting)

# Vet check
go vet ./...

# Build check
make build
```

### Generate Coverage Report

```bash
# Generate coverage
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out
```

## 📋 Workflow Triggers

| Event | Workflows That Run |
|-------|-------------------|
| Open PR | Quick Check + CI Pipeline |
| Update PR | Quick Check + CI Pipeline |
| Merge to main | CI Pipeline |
| Push tag `v*.*.*` | Release |

## ⚡ Speed Tips

### Make CI Faster

1. **Run quick tests locally first**
   ```bash
   go test -short ./...
   ```

2. **Format before pushing**
   ```bash
   go fmt ./...
   ```

3. **Cache is your friend**
   - Go modules are cached
   - Subsequent runs are faster

### Make Local Testing Faster

```bash
# Run only changed tests
go test ./path/to/changed/package

# Skip slow tests
go test -short ./...

# Run in parallel
go test -parallel 4 ./...
```

## 🐛 Troubleshooting

### "Formatting check failed"

```bash
go fmt ./...
git commit -am "Fix formatting"
git push
```

### "Tests failed"

```bash
# Run locally to debug
make test

# Run specific test
go test -v ./package -run TestName

# See detailed output
go test -v ./...
```

### "Docker tests failed"

```bash
# Ensure Docker is running
docker ps

# Run Docker tests locally
make test-docker-pr
```

### "Build failed"

```bash
# Check for syntax errors
go build ./...

# Update dependencies
go mod tidy
go mod download

# Try building again
make build
```

## 📚 More Information

- **Comprehensive Guide**: [CI_CD_GUIDE.md](CI_CD_GUIDE.md)
- **Setup Summary**: [CI_CD_SETUP_SUMMARY.md](CI_CD_SETUP_SUMMARY.md)
- **Workflows README**: [../.github/workflows/README.md](../.github/workflows/README.md)

## 🎯 Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│                    CI/CD Quick Reference                    │
├─────────────────────────────────────────────────────────────┤
│ Before PR:        go fmt ./... && make test                 │
│ Create PR:        git push origin branch                    │
│ Fix formatting:   go fmt ./... && git push                  │
│ Fix tests:        make test, fix, git push                  │
│ Create release:   git tag -a v0.3.0 -m "..." && git push   │
│ Monitor CI:       GitHub → Actions tab                      │
│ View coverage:    Download from workflow artifacts          │
└─────────────────────────────────────────────────────────────┘
```

## ✅ Success Checklist

Before merging PR:
- [ ] Quick Check passed ✅
- [ ] CI Pipeline passed ✅
- [ ] All tests green ✅
- [ ] Code reviewed ✅
- [ ] Ready to merge 🎉

Before creating release:
- [ ] Main branch stable ✅
- [ ] All CI checks passed ✅
- [ ] Version tag created ✅
- [ ] Tag pushed to GitHub ✅
- [ ] Release workflow running ✅
- [ ] Binaries published 🎉

---

**Need Help?** Check the [comprehensive CI/CD guide](CI_CD_GUIDE.md) or open an issue.

**Questions?** See the [workflows README](../.github/workflows/README.md) for detailed docs.
