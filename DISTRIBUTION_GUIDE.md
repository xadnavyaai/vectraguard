# Distribution Guide: Making Vectra Guard Easy to Install

This guide shows you how to package and distribute Vectra Guard so customers can install with one command.

---

## 🎯 Goal: One-Command Installation

**We want customers to run:**
```bash
# Option 1: Homebrew (macOS)
brew install vectra-guard

# Option 2: Install script (all platforms)
curl -fsSL https://vectra-guard.sh | bash

# Option 3: Go install
go install github.com/xadnavyaai/vectra-guard@latest
```

---

## 📦 Distribution Methods

### 1. Homebrew (EASIEST for macOS users) ⭐

**Why**: 90% of macOS developers use Homebrew. This is the gold standard.

#### Step 1: Create Homebrew Formula

Create `vectra-guard.rb`:

```ruby
class VectraGuard < Formula
  desc "Security guard for AI coding agents"
  homepage "https://github.com/xadnavyaai/vectra-guard"
  url "https://github.com/xadnavyaai/vectra-guard/archive/v1.0.0.tar.gz"
  sha256 "PUT_SHA256_HERE"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./main.go"
    
    # Install scripts
    bin.install "vectra-guard"
    prefix.install "scripts"
    
    # Post-install message
    ohai "Vectra Guard installed successfully!"
    ohai "Run: vectra-guard init to get started"
  end

  test do
    assert_match "usage:", shell_output("#{bin}/vectra-guard 2>&1", 1)
  end
end
```

#### Step 2: Calculate SHA256

```bash
# Create archive
git archive --format=tar.gz --prefix=vectra-guard-1.0.0/ v1.0.0 > vectra-guard-1.0.0.tar.gz

# Get SHA256
shasum -a 256 vectra-guard-1.0.0.tar.gz
```

#### Step 3: Create Homebrew Tap

```bash
# Create tap repository
mkdir -p homebrew-tap
cd homebrew-tap
git init

# Add formula
cp ../vectra-guard.rb Formula/vectra-guard.rb
git add .
git commit -m "Add vectra-guard formula"

# Push to GitHub as homebrew-tap repository
# Repository name MUST be: homebrew-tap
git remote add origin https://github.com/xadnavyaai/homebrew-tap.git
git push -u origin main
```

#### Step 4: Customers Install

```bash
# Add tap
brew tap xadnavyaai/tap

# Install
brew install vectra-guard

# Use
vectra-guard init
```

**Time to set up**: 30 minutes  
**Customer install time**: 30 seconds

---

### 2. Install Script (ALL PLATFORMS) ⭐⭐

**Why**: Works on macOS, Linux, any Unix system. No dependencies.

#### Create `install.sh`:

```bash
#!/bin/bash
# Vectra Guard Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash

set -e

REPO="xadnavyaai/vectra-guard"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="vectra-guard"

echo "🛡️  Installing Vectra Guard..."
echo ""

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    *)
        echo "❌ Unsupported OS: $OS"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Get latest release
echo "📦 Downloading latest release..."
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}-${OS}-${ARCH}"

# Download
if command -v curl &> /dev/null; then
    curl -fsSL "$DOWNLOAD_URL" -o "$BINARY_NAME"
elif command -v wget &> /dev/null; then
    wget -q "$DOWNLOAD_URL" -O "$BINARY_NAME"
else
    echo "❌ Need curl or wget to download"
    exit 1
fi

# Make executable
chmod +x "$BINARY_NAME"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    echo "📝 Need sudo to install to $INSTALL_DIR"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
fi

# Verify
if command -v vectra-guard &> /dev/null; then
    echo ""
    echo "✅ Vectra Guard installed successfully!"
    echo ""
    echo "Get started:"
    echo "  vectra-guard init"
    echo ""
    echo "Or install universal protection:"
    echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-universal-shell-protection.sh | bash"
else
    echo "❌ Installation failed"
    exit 1
fi
```

#### Host on GitHub

```bash
# Add install.sh to repository root
git add install.sh
git commit -m "Add installation script"
git push
```

#### Customers Install

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**One command! ✨**

---

### 3. GitHub Releases with Binaries ⭐

**Why**: Direct downloads, no dependencies. Works offline.

#### Step 1: Build Binaries for All Platforms

```bash
#!/bin/bash
# build-release.sh

VERSION="v1.0.0"

# Create dist directory
mkdir -p dist

# Build for each platform
GOOS=darwin GOARCH=amd64 go build -o dist/vectra-guard-darwin-amd64 main.go
GOOS=darwin GOARCH=arm64 go build -o dist/vectra-guard-darwin-arm64 main.go
GOOS=linux GOARCH=amd64 go build -o dist/vectra-guard-linux-amd64 main.go
GOOS=linux GOARCH=arm64 go build -o dist/vectra-guard-linux-arm64 main.go
GOOS=windows GOARCH=amd64 go build -o dist/vectra-guard-windows-amd64.exe main.go

# Create archives
cd dist
for binary in vectra-guard-*; do
    tar czf "${binary}.tar.gz" "$binary"
done
cd ..

echo "✅ Binaries built in dist/"
```

#### Step 2: Upload to GitHub Release

1. Go to: https://github.com/xadnavyaai/vectra-guard/releases/tag/v1.0.0
2. Click "Edit"
3. Drag and drop binaries from `dist/` folder
4. Click "Update release"

#### Step 3: Customers Download

```bash
# macOS ARM (M1/M2/M3)
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-darwin-arm64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/

# macOS Intel
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-darwin-amd64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/

# Linux
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-linux-amd64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/
```

---

### 4. Docker Hub (FOR CONTAINER USERS)

**Why**: Users can pull pre-built image without building locally.

#### Step 1: Build and Tag

```bash
# Build image
docker build -t xadnavyaai/vectra-guard:v1.0.0 .
docker tag xadnavyaai/vectra-guard:v1.0.0 xadnavyaai/vectra-guard:latest
```

#### Step 2: Push to Docker Hub

```bash
# Login
docker login

# Push
docker push xadnavyaai/vectra-guard:v1.0.0
docker push xadnavyaai/vectra-guard:latest
```

#### Step 3: Customers Use

```bash
# Pull and run
docker pull xadnavyaai/vectra-guard:latest
docker run -it xadnavyaai/vectra-guard:latest

# Or use docker-compose
docker-compose up
```

---

### 5. Go Install (FOR GO DEVELOPERS)

**Already works!** No setup needed.

```bash
go install github.com/xadnavyaai/vectra-guard@latest
```

---

## 🎯 Recommended Setup (Do These 3)

### Priority 1: Install Script (30 min)
**Most universal, easiest for customers**

1. Create `install.sh` (see above)
2. Add to repository root
3. Test it
4. Add to README

**Customer command**:
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

---

### Priority 2: GitHub Release Binaries (1 hour)
**Direct downloads, good for all platforms**

1. Create `build-release.sh` (see above)
2. Run it to build binaries
3. Upload to GitHub release
4. Update README with download links

---

### Priority 3: Homebrew Tap (2 hours)
**Gold standard for macOS developers**

1. Create formula
2. Create tap repository
3. Test installation
4. Add to README

**Customer command**:
```bash
brew install xadnavyaai/tap/vectra-guard
```

---

## 📝 Update README Installation Section

Replace current installation with:

```markdown
## 🚀 Installation

### Quick Install (Recommended)

**macOS (Homebrew)**:
```bash
brew install xadnavyaai/tap/vectra-guard
```

**All Platforms (Install Script)**:
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Go Developers**:
```bash
go install github.com/xadnavyaai/vectra-guard@latest
```

### Manual Installation

Download pre-built binary from [Releases](https://github.com/xadnavyaai/vectra-guard/releases):

```bash
# macOS ARM64 (M1/M2/M3)
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-darwin-arm64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/

# macOS AMD64 (Intel)
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-darwin-amd64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/

# Linux AMD64
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-linux-amd64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/xadnavyaai/vectra-guard.git
cd vectra-guard
go build -o vectra-guard main.go
sudo mv vectra-guard /usr/local/bin/
```
```

---

## 🤖 Automation with GitHub Actions

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [linux, darwin, windows]
        goarch: [amd64, arm64]
        exclude:
          - goos: windows
            goarch: arm64
    
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        run: |
          OUTPUT="vectra-guard-${{ matrix.goos }}-${{ matrix.goarch }}"
          if [ "${{ matrix.goos }}" = "windows" ]; then
            OUTPUT="${OUTPUT}.exe"
          fi
          go build -o "$OUTPUT" main.go
      
      - name: Upload
        uses: softprops/action-gh-release@v1
        with:
          files: vectra-guard-*
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Now**: Every time you push a tag, binaries are built and uploaded automatically!

---

## 📊 Comparison

| Method | Ease (Customer) | Ease (Setup) | Updates | Platforms |
|--------|----------------|--------------|---------|-----------|
| **Install Script** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Manual | All Unix |
| **Homebrew** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | Auto | macOS/Linux |
| **GitHub Release** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Manual | All |
| **Go Install** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Auto | All (with Go) |
| **Docker Hub** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Auto | All (Docker) |

**Recommendation**: Do Install Script + GitHub Release + Homebrew for maximum reach.

---

## 🎯 Action Plan

### Week 1: Essential (Do Now)
1. ✅ Create `install.sh` script
2. ✅ Build and upload release binaries
3. ✅ Update README with new install methods
4. ✅ Test installations on different platforms

### Week 2: Professional (Do Next)
1. ⬜ Create Homebrew tap
2. ⬜ Set up GitHub Actions for auto-build
3. ⬜ Push to Docker Hub
4. ⬜ Create installation documentation

### Week 3: Polish
1. ⬜ Add checksums for security
2. ⬜ Create update mechanism
3. ⬜ Add installation analytics
4. ⬜ Marketing materials

---

## ✅ Success Metrics

After setup, customers can:
- ✅ Install with one command
- ✅ No manual steps needed
- ✅ Works on their platform
- ✅ Auto-complete from package manager
- ✅ Easy to update

---

## 🎉 Result

**Before**:
```bash
# Customer needs to:
git clone ...
cd ...
go build ...
sudo cp ...
# 5 commands, 5 minutes
```

**After**:
```bash
# Customer runs:
brew install vectra-guard
# 1 command, 30 seconds ✨
```

**That's a 10x improvement in user experience!**

