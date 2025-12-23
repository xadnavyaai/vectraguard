# How to Create GitHub Release and Upload Binaries

## 🎯 Quick Steps

### Step 1: Create a New Release

1. **Go to**: https://github.com/xadnavyaai/vectra-guard/releases

2. **Click**: The green **"Create a new release"** button (top right)

3. **Fill in the form**:
   - **Choose a tag**: Select `v1.0.0` from dropdown
   - **Release title**: `Vectra Guard v1.0.0 - Universal AI Security Platform`
   - **Description**: Copy from `RELEASE_NOTES_v1.0.0.md` (see below)
   - **Attach files**: Drag and drop binaries from `dist/` folder

4. **Upload these 6 files** from your `dist/` folder:
   - `vectra-guard-darwin-amd64.tar.gz`
   - `vectra-guard-darwin-arm64.tar.gz`
   - `vectra-guard-linux-amd64.tar.gz`
   - `vectra-guard-linux-arm64.tar.gz`
   - `vectra-guard-windows-amd64.exe.zip`
   - `checksums-archives.txt` (rename to `checksums.txt`)

5. **Click**: **"Publish release"** (green button at bottom)

---

## 📝 Release Description (Copy This)

```markdown
# Vectra Guard v1.0.0

> **Security Guard for AI Coding Agents & Development Workflows**

## 🎉 First Stable Release!

Vectra Guard is a comprehensive security platform that protects systems from risky shell commands and AI agent activities. This release includes universal shell protection, container isolation, and complete audit capabilities.

## ⚡ Quick Install

**One-line installation**:
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Or download the appropriate binary below for your platform.**

## ✨ Key Features

- ✅ **Script Validation** - Analyze shell scripts for security risks
- ✅ **Agent Session Tracking** - Monitor all AI agent activities  
- ✅ **Universal Shell Protection** - Automatic protection for Bash, Zsh, Fish
- ✅ **Execution Control** - Interactive approval for risky operations
- ✅ **Container Isolation** - Docker-based sandboxing with seccomp
- ✅ **Audit Trails** - Complete logging for compliance

## 🚀 Getting Started

After installation:

```bash
# Initialize configuration
vectra-guard init

# Install universal protection (recommended)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-universal-shell-protection.sh | bash

# Validate a script
vectra-guard validate your-script.sh

# Execute safely
vectra-guard exec "npm install"
```

## 📦 Installation Options

### Install Script (Recommended)
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

### Go Install
```bash
go install github.com/xadnavyaai/vectra-guard@latest
```

### Download Binary
Choose the appropriate archive below:
- **macOS M1/M2/M3**: `vectra-guard-darwin-arm64.tar.gz`
- **macOS Intel**: `vectra-guard-darwin-amd64.tar.gz`
- **Linux 64-bit**: `vectra-guard-linux-amd64.tar.gz`
- **Linux ARM**: `vectra-guard-linux-arm64.tar.gz`
- **Windows**: `vectra-guard-windows-amd64.exe.zip`

Then extract and install:
```bash
# macOS/Linux
tar xzf vectra-guard-*.tar.gz
chmod +x vectra-guard-*
sudo mv vectra-guard-* /usr/local/bin/vectra-guard

# Windows
# Extract the zip and run vectra-guard.exe
```

## 🔐 Security

All binaries include SHA256 checksums in `checksums.txt`. Verify before installation:

```bash
shasum -a 256 -c checksums.txt
```

## 📚 Documentation

- [README](https://github.com/xadnavyaai/vectra-guard#readme) - Complete documentation
- [Getting Started Guide](https://github.com/xadnavyaai/vectra-guard/blob/main/GETTING_STARTED.md) - Step-by-step tutorial
- [Distribution Guide](https://github.com/xadnavyaai/vectra-guard/blob/main/DISTRIBUTION_GUIDE.md) - Advanced deployment

## 🎯 What's Included

### Core Features
- Script validation engine with pattern detection
- Session management for agent tracking
- Real-time command execution control
- Risk scoring and violation tracking
- Structured logging (JSON/text)

### Enforcement Options
- Opt-in execution wrapper
- Universal shell integration
- Container-based sandboxing
- Seccomp syscall filtering

### Platform Support
- macOS (Intel & ARM)
- Linux (AMD64 & ARM64)
- Windows (AMD64)

## 🐛 Known Issues

None! This is a stable release.

## 📝 License

Apache 2.0 - See [LICENSE](https://github.com/xadnavyaai/vectra-guard/blob/main/LICENSE)

## 🤝 Contributing

Contributions welcome! See issues for current priorities.

## 📧 Support

- **Issues**: https://github.com/xadnavyaai/vectra-guard/issues
- **Discussions**: https://github.com/xadnavyaai/vectra-guard/discussions

---

**Full Changelog**: Initial release v1.0.0
```

---

## 🖼️ Visual Guide

### Where to Go:
```
GitHub Repository → Releases Tab → "Create a new release" button
```

### What You'll See:

```
┌─────────────────────────────────────────────────────────┐
│  Create a new release                                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Choose a tag:  [v1.0.0 ▼]  or create new tag          │
│                                                          │
│  Release title: Vectra Guard v1.0.0 - Universal AI...  │
│                                                          │
│  Description:   [Your release notes here]              │
│                                                          │
│  Attach binaries by dragging & dropping files          │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Drop files here or click to browse              │  │
│  └──────────────────────────────────────────────────┘  │
│                                                          │
│  [ ] This is a pre-release                              │
│  [ ] Set as latest release                              │
│                                                          │
│  [Publish release]                                      │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## ✅ Checklist

- [ ] Go to https://github.com/xadnavyaai/vectra-guard/releases
- [ ] Click "Create a new release"
- [ ] Select tag: `v1.0.0`
- [ ] Add release title
- [ ] Paste release description (from above)
- [ ] Upload 6 files from `dist/` folder
- [ ] Check "Set as latest release"
- [ ] Click "Publish release"
- [ ] Test installer:
      ```bash
      curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
      ```

---

## 🎉 After Publishing

Your package will be available at:
- **Release page**: https://github.com/xadnavyaai/vectra-guard/releases/tag/v1.0.0
- **Latest release**: https://github.com/xadnavyaai/vectra-guard/releases/latest

Customers can install with:
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**That's it!** 🚀

