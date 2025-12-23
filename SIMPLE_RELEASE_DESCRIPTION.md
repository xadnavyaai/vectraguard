# Vectra Guard v1.0.0

**Security Guard for AI Coding Agents & Development Workflows**

## 🎉 First Stable Release

Vectra Guard protects systems from risky shell commands and AI agent activities with universal shell protection, container isolation, and complete audit capabilities.

## ⚡ Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

Or download the binary for your platform below.

## ✨ Key Features

- ✅ **Script Validation** - Analyze shell scripts for security risks
- ✅ **Agent Session Tracking** - Monitor all AI agent activities  
- ✅ **Universal Shell Protection** - Automatic protection for Bash, Zsh, Fish
- ✅ **Execution Control** - Interactive approval for risky operations
- ✅ **Container Isolation** - Docker-based sandboxing
- ✅ **Audit Trails** - Complete logging for compliance

## 🚀 Getting Started

```bash
# Initialize
vectra-guard init

# Install universal protection (recommended)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-universal-shell-protection.sh | bash

# Validate a script
vectra-guard validate script.sh

# Execute safely
vectra-guard exec "npm install"
```

## 📦 Manual Installation

Download the appropriate binary:
- **macOS M1/M2/M3**: `vectra-guard-darwin-arm64.tar.gz`
- **macOS Intel**: `vectra-guard-darwin-amd64.tar.gz`
- **Linux 64-bit**: `vectra-guard-linux-amd64.tar.gz`
- **Linux ARM**: `vectra-guard-linux-arm64.tar.gz`
- **Windows**: `vectra-guard-windows-amd64.exe.zip`

Extract and install:
```bash
tar xzf vectra-guard-*.tar.gz
chmod +x vectra-guard-*
sudo mv vectra-guard-* /usr/local/bin/vectra-guard
```

## 🔐 Security

Verify downloads with checksums:
```bash
shasum -a 256 -c checksums.txt
```

## 📚 Documentation

- [Complete README](https://github.com/xadnavyaai/vectra-guard#readme)
- [Getting Started Guide](https://github.com/xadnavyaai/vectra-guard/blob/main/GETTING_STARTED.md)
- [Distribution Guide](https://github.com/xadnavyaai/vectra-guard/blob/main/DISTRIBUTION_GUIDE.md)

## 📝 License

Apache 2.0

---

**Full Changelog**: Initial release v1.0.0

