# Vectra Guard v0.0.1 - Initial Release

**Release Date**: December 23, 2025

## 🎉 First Public Release

Vectra Guard is a security platform designed to protect systems from risky shell commands and AI agent activities. This initial release provides comprehensive monitoring, validation, and enforcement capabilities for development environments.

---

## ✨ Key Features

### 🔍 Script Validation & Analysis
- Analyze shell scripts for security risks before execution
- Detect dangerous patterns (fork bombs, rm -rf /, privilege escalation)
- **NEW**: Detect SQL/NoSQL database operations (MySQL, PostgreSQL, MongoDB, Redis, Cassandra, DynamoDB, and more)
- **NEW**: Production/staging environment detection with escalated warnings
- **NEW**: Smart severity escalation for destructive database operations in production (CRITICAL alerts)
- Policy-based validation with allow/deny lists
- Risk scoring (low, medium, high, critical)
- Human-readable explanations of security issues

### 🎭 AI Agent Session Management
- Track AI agent activities with unique session IDs
- Monitor command execution with timestamps and exit codes
- Record file operations and system changes
- Risk scoring per session
- Violation tracking and reporting
- Structured audit logs (JSON/text formats)

### 🛡️ Universal Shell Protection
- Automatic integration with Bash, Zsh, and Fish shells
- Transparent command interception
- Works across all IDEs (Cursor, VSCode, etc.)
- Works in terminals, SSH sessions, and scripts
- One installation protects all environments
- **NEW**: Automatically installs `vg` alias for convenience (`vg validate`, `vg exec`, etc.)

### ⚡ Real-Time Execution Control
- Interactive approval for risky commands
- Auto-approve known-safe operations
- Auto-block critical security violations
- Context-aware risk assessment
- Session-based tracking

### 🐳 Container Isolation
- Docker-based sandboxing for maximum security
- Seccomp profiles for syscall filtering
- Multiple security profiles (dev, prod, sandbox)
- Deny-all filesystem policies
- Network isolation options

---

## 🚀 Getting Started

### Quick Install

```bash
# One-line installer (recommended)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

### Or Install Manually

Download the appropriate binary for your platform:
- **macOS M1/M2/M3**: `vectra-guard-darwin-arm64`
- **macOS Intel**: `vectra-guard-darwin-amd64`
- **Linux 64-bit**: `vectra-guard-linux-amd64`
- **Linux ARM**: `vectra-guard-linux-arm64`
- **Windows**: `vectra-guard-windows-amd64.exe`

```bash
# Make executable and install
chmod +x vectra-guard-*
sudo mv vectra-guard-* /usr/local/bin/vectra-guard
```

### Initial Setup

```bash
# 1. Initialize configuration
vectra-guard init

# 2. Install universal shell protection (recommended)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-universal-shell-protection.sh | bash

# 3. Restart your terminal
source ~/.zshrc  # or ~/.bashrc
```

---

## 📚 Usage Examples

### Script Validation

```bash
# Validate a script before running
vectra-guard validate deploy.sh

# Get detailed risk explanation
vectra-guard explain setup.sh
```

### Secure Command Execution

```bash
# Execute with validation
vectra-guard exec "npm install"

# Interactive approval for risky commands
vectra-guard exec --interactive "sudo rm -rf /tmp/old-data"
```

### Session Management

```bash
# View active sessions
vectra-guard session list

# Show session details
vectra-guard session show $VECTRAGUARD_SESSION_ID

# End a session
vectra-guard session end $VECTRAGUARD_SESSION_ID
```

---

## 🏗️ Architecture

### Components

1. **Core Analyzer** - Pattern detection and risk assessment
2. **Session Manager** - Activity tracking and persistence
3. **CLI Interface** - User-facing commands
4. **Shell Integration** - Transparent command interception
5. **Container Runtime** - Isolated execution environments

### Storage

- **Sessions**: `~/.vectra-guard/sessions/`
- **Config**: `vectra-guard.yaml` (project or `~/.config/vectra-guard/`)
- **Logs**: Stdout/stderr (structured JSON or text)

---

## 🔐 Security Model

### Zero-Trust Defaults
- All commands are analyzed before execution
- Unknown patterns require approval
- Critical operations are blocked by default

### Risk Levels
- **Low**: Standard operations (ls, cd, echo)
- **Medium**: File modifications, installations
- **High**: Sudo commands, system changes
- **Critical**: Destructive operations, root deletion

### Enforcement Options
1. **Opt-in**: Use `vectra-guard exec` explicitly
2. **Shell Integration**: Automatic for all commands
3. **Container Isolation**: Kernel-enforced sandboxing

---

## 📦 Distribution

### Available Formats
- **Direct Download**: Raw binaries from GitHub Releases
- **One-Line Installer**: `curl | bash` installer
- **Go Install**: `go install github.com/xadnavyaai/vectra-guard@latest`
- **Docker**: Pre-built images with docker-compose configurations

### Platform Support
- macOS (Intel & ARM/Apple Silicon)
- Linux (AMD64 & ARM64)
- Windows (AMD64)

---

## 🔧 Configuration

### Default Policy

```yaml
policy:
  mode: "enforce"
  allowlist:
    - "^ls"
    - "^cd"
    - "^pwd"
    - "^echo"
  denylist:
    - "rm -rf /"
    - "mkfs"
    - "dd if=/dev/zero"
```

### Logging

```yaml
logging:
  level: "info"
  format: "json"
  output: "stdout"
```

---

## 🧪 Testing

All core components include unit tests:

```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...
```

---

## 📖 Documentation

- **README.md**: Complete project documentation
- **GETTING_STARTED.md**: Step-by-step tutorial
- **Project.md**: Technical design and architecture
- **DISTRIBUTION_GUIDE.md**: Packaging and deployment

---

## 🐛 Known Limitations

### v0.0.1 Limitations
- Session persistence uses JSON files (not database)
- Shell integration requires manual restart after install
- Container isolation requires Docker installation
- No GUI or web interface (CLI only)

### Planned Improvements
See `roadmap.md` for upcoming features.

---

## 🤝 Contributing

Contributions welcome! This is an initial release and we're actively developing new features.

### Priority Areas
- Additional shell support (PowerShell, Nushell)
- Database backend for sessions
- Web UI for monitoring
- VS Code extension
- Enhanced policy engine

---

## 📝 License

Apache License 2.0 - See [LICENSE](LICENSE) file for details.

---

## 🔗 Links

- **Repository**: https://github.com/xadnavyaai/vectra-guard
- **Issues**: https://github.com/xadnavyaai/vectra-guard/issues
- **Releases**: https://github.com/xadnavyaai/vectra-guard/releases

---

## 📊 Technical Details

### Dependencies
- Go 1.21+
- Docker (optional, for container isolation)
- Bash/Zsh/Fish (for shell integration)

### Binary Sizes
- macOS ARM64: ~2.3 MB
- macOS AMD64: ~2.3 MB
- Linux AMD64: ~2.4 MB
- Linux ARM64: ~2.4 MB
- Windows AMD64: ~2.5 MB

### Checksums
See `checksums.txt` in release assets for SHA256 verification.

---

## 🎯 What's Next?

This is just the beginning! Check out our [roadmap](roadmap.md) for planned features in v0.1.0 and beyond.

---

## 🙏 Acknowledgments

Built for the AI development community. Special thanks to early testers and contributors.

---

**Install now and secure your AI agent workflows!** 🛡️🚀

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

