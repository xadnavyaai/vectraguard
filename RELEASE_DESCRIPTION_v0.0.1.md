# Vectra Guard v0.0.1 - Initial Release 🎉

**Security guard for AI coding agents and development workflows**

## 🚀 Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

Or download the binary for your platform below.

---

## ✨ What is Vectra Guard?

A security platform that protects your system from risky shell commands and AI agent activities. It validates scripts, monitors command execution, tracks agent sessions, and enforces security policies with real-time protection.

---

## 🎯 Key Features

- ✅ **Script Validation** - Analyze shell scripts for security risks before execution
- ✅ **AI Agent Tracking** - Monitor all agent activities with unique session IDs
- ✅ **Universal Shell Protection** - Automatic integration with Bash, Zsh, Fish
- ✅ **Real-Time Control** - Interactive approval for risky operations
- ✅ **Container Isolation** - Docker-based sandboxing with seccomp profiles
- ✅ **Audit Trails** - Complete logging for compliance and debugging

---

## 🚀 Getting Started

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash

# Initialize
vectra-guard init

# Install universal protection (recommended)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-universal-shell-protection.sh | bash

# Restart terminal
source ~/.zshrc
```

---

## 💻 Manual Installation

Download the appropriate binary for your platform:

### macOS
- **Apple Silicon (M1/M2/M3)**: `vectra-guard-darwin-arm64`
- **Intel**: `vectra-guard-darwin-amd64`

### Linux
- **AMD64**: `vectra-guard-linux-amd64`
- **ARM64**: `vectra-guard-linux-arm64`

### Windows
- **AMD64**: `vectra-guard-windows-amd64.exe`

Then install:
```bash
chmod +x vectra-guard-*
sudo mv vectra-guard-* /usr/local/bin/vectra-guard
```

---

## 📖 Usage

### Validate Scripts
```bash
vectra-guard validate deploy.sh
vectra-guard explain setup.sh
```

### Execute Safely
```bash
vectra-guard exec "npm install"
vectra-guard exec --interactive "sudo apt upgrade"
```

### Manage Sessions
```bash
vectra-guard session list
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

---

## 🔐 Security

### Zero-Trust Model
- All commands analyzed before execution
- Unknown patterns require approval
- Critical operations blocked by default

### Risk Levels
- **Low**: Standard operations (ls, cd, echo)
- **Medium**: File modifications, installations
- **High**: Sudo commands, system changes
- **Critical**: Destructive operations

---

## 📦 What's Included

### Binaries
- macOS (Intel & ARM)
- Linux (AMD64 & ARM64)
- Windows (AMD64)

### Features
- CLI interface with 5 commands (init, validate, explain, exec, session)
- Universal shell integration
- Docker support
- Comprehensive documentation

---

## 🔗 Resources

- **Documentation**: [README.md](https://github.com/xadnavyaai/vectra-guard#readme)
- **Getting Started**: [GETTING_STARTED.md](https://github.com/xadnavyaai/vectra-guard/blob/main/GETTING_STARTED.md)
- **Full Release Notes**: [RELEASE_NOTES_v0.0.1.md](https://github.com/xadnavyaai/vectra-guard/blob/main/RELEASE_NOTES_v0.0.1.md)

---

## 🐛 Known Limitations (v0.0.1)

- Session persistence uses JSON files (not database yet)
- Shell integration requires manual terminal restart
- Container isolation requires Docker
- CLI only (no GUI yet)

See [roadmap.md](https://github.com/xadnavyaai/vectra-guard/blob/main/roadmap.md) for planned improvements.

---

## 🤝 Contributing

This is an initial release! Contributions, feedback, and bug reports are very welcome.

**Issues**: https://github.com/xadnavyaai/vectra-guard/issues

---

## 📝 License

Apache License 2.0

---

## 🎯 What's Next?

Check our [roadmap](https://github.com/xadnavyaai/vectra-guard/blob/main/roadmap.md) for upcoming features in v0.1.0 and beyond!

---

**Secure your AI agent workflows today!** 🛡️

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

---

**Full Changelog**: Initial release v0.0.1

