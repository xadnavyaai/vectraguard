# Vectra Guard - Simple Usage Guide

> **Protect your system from risky commands with one simple tool**

## 🚀 Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

That's it! The tool is now installed at `/usr/local/bin/vectra-guard`.

## 📖 How to Use

### 1. Validate Scripts (Safe - Never Executes)

Check if a script is safe before running it:

```bash
vectra-guard validate my-script.sh
```

**Example:**
```bash
$ vectra-guard validate deploy.sh
[WARN] finding code=PROTECTED_DIRECTORY_ACCESS severity=critical
Command targets protected directory: /etc
BLOCKED: Operations on /etc are not allowed.
```

### 2. Execute Commands Safely

Run commands with automatic protection:

```bash
vectra-guard exec -- npm install
vectra-guard exec -- rm -rf /tmp/old-files
```

**What happens:**
- ✅ Safe commands → Execute normally
- ⚠️ Risky commands → Blocked or sandboxed
- 🔒 Protected directories → Always blocked

### 3. Explain Security Risks

Get detailed explanation of what's risky:

```bash
vectra-guard explain risky-script.sh
```

## 🎯 Common Use Cases

### Use Case 1: Validate Before Running

```bash
# Check a script first
vectra-guard validate deploy.sh

# If safe, run it
./deploy.sh
```

### Use Case 2: Protect Command Execution

```bash
# This will be blocked (targets protected directory)
vectra-guard exec -- rm -rf /etc

# This will work (safe directory)
vectra-guard exec -- rm -rf /tmp/test
```

### Use Case 3: Check AI-Generated Commands

```bash
# AI suggests a command? Validate it first!
echo "rm -rf node_modules" | vectra-guard validate /dev/stdin
```

## ⚙️ Configuration (Optional)

Create a config file for custom settings:

```bash
vectra-guard init
```

This creates `vectra-guard.yaml` with:
- Protected directories (17 system directories by default)
- Security levels (low/medium/high/paranoid)
- Sandbox settings

## 🛡️ What Gets Protected?

**By default, these directories are protected:**
- `/`, `/bin`, `/sbin`, `/usr`, `/etc`, `/var`
- `/lib`, `/lib64`, `/opt`, `/boot`, `/root`
- `/sys`, `/proc`, `/dev`, `/home`, `/srv`

**Risky commands are detected:**
- `rm -rf /` (root deletion)
- `sudo` commands
- Database operations
- Network installs (`curl | sh`)
- And 200+ more patterns

## 📚 More Information

- **Full Documentation**: See [README.md](README.md) for complete details
- **Configuration**: See [CONFIGURATION.md](CONFIGURATION.md)
- **Advanced Features**: See [ADVANCED_FEATURES.md](ADVANCED_FEATURES.md)

## 🆘 Quick Help

```bash
vectra-guard --help           # Show all commands
vectra-guard validate --help  # Help for validate
vectra-guard exec --help      # Help for exec
```

---

**That's it!** You're now protected. The tool works automatically - just use `vectra-guard exec` for risky commands, or `vectra-guard validate` to check scripts first.

