# Vectra Guard v0.0.1 Release Notes

## 🎉 Initial Release

Vectra Guard is a security tool that protects your development and production environments from dangerous commands by analyzing scripts before execution.

---

## 🚀 Core Features

### 🛡️ Command Analysis
- Pre-execution risk assessment
- Pattern-based detection of dangerous commands
- Configurable security policies

### 🔒 Security Policies
- Allowlist/denylist support
- Guard levels: `off`, `low`, `medium`, `high`, `paranoid`, `auto`
- Git operations monitoring
- Production environment detection
- SQL operation detection

### 🐳 Sandboxing
- Docker-based sandbox execution
- Isolated command execution
- Configurable sandbox modes

### 📊 Trust Store
- Remember approved commands
- User bypass mechanism
- Secure command approval

---

## 🔍 Detection Capabilities

### Critical Threats
- Root directory deletion (`rm -rf /`, `rm -r /*`)
- Home directory deletion (`rm -rf ~/*`)
- Fork bombs
- Reverse shells
- Sensitive environment access

### File System Operations
- Dangerous file deletions
- System path modifications
- Permission changes on system directories

### Network Security
- Piped script downloads (`curl | bash`)
- Network script execution
- Command injection attempts

### Database Operations
- Destructive SQL commands
- Database deletion attempts

### Git Operations
- Force push detection
- Destructive Git commands

---

## 📦 Installation

```bash
# Download for your platform
# See GitHub Releases: https://github.com/xadnavyaai/vectra-guard/releases

# Verify checksum
shasum -a 256 vectra-guard-<platform> | grep <checksum>

# Make executable
chmod +x vectra-guard-<platform>
```

---

## 🚀 Quick Start

```bash
# Validate a script
vectra-guard validate script.sh

# Execute with protection
vectra-guard exec -- command args

# Check version
vectra-guard version
```

---

## 📝 Configuration

Vectra Guard supports configuration via YAML files. See [CONFIGURATION.md](CONFIGURATION.md) for details.

**Example configuration:**
```yaml
guard_level:
  level: high
  allow_user_bypass: false

sandbox:
  enabled: true
  mode: auto
  runtime: docker

policies:
  monitor_git_ops: true
  block_force_git: true
  detect_prod_env: true
```

---

## 🧪 Testing

```bash
# Run tests
make test

# Run extended tests in Docker
make test-extended-docker
```

---

## 📚 Documentation

- [Getting Started](GETTING_STARTED.md)
- [Configuration Guide](CONFIGURATION.md)
- [Advanced Features](ADVANCED_FEATURES.md)
- [Security Model](SECURITY_MODEL.md)
- [Docker Testing](DOCKER_TESTING.md)

---

## 🙏 Acknowledgments

Special thanks to the security community for feedback and contributions.

---

**Full Changelog**: Initial release

