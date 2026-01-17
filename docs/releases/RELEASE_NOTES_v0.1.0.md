# Vectra Guard v0.1.0 Release Notes

## 🎉 First Stable Release

This is the first release focused on **macOS** and **Debian Linux** (x86_64, arm64) with a secure-by-default command guard, sandboxing, and shell tracking.

---

## ✅ Supported Platforms

- **macOS** (x86_64, arm64)
- **Debian Linux** (x86_64, arm64)

---

## 🛡️ Security & Safety

- **Critical command blocking**: hard-blocks destructive commands such as `rm -rf /` and `rm -rf ~/`
- **Protected directory detection**: detects direct access to protected system paths
- **Guard levels**: `off`, `low`, `medium`, `high`, `paranoid`, `auto`
- **Trust store**: approve-and-remember safe commands

---

## 🧪 Sandboxing & Execution

- **Sandboxed execution** with intelligent cache mounting
- **Auto mode** chooses host vs sandbox based on risk
- **Mandatory sandboxing for critical commands**

---

## 🧰 Shell Tracker

- **Bash and Zsh integration** for automatic pre-exec validation
- **Session tracking** across shells
- **`vg` alias** for faster workflows

---

---
## 📦 Installation

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

---

## 🧪 Testing

```bash
# Internal tests (safe)
make test-internal

# Docker-based execution tests
make test-docker-pr
```

---

## 🔐 Verification

Checksums are provided in `checksums.txt` for the release assets.

---

## 📚 Documentation

- [Getting Started](https://github.com/xadnavyaai/vectra-guard/blob/main/GETTING_STARTED.md)
- [Configuration Guide](https://github.com/xadnavyaai/vectra-guard/blob/main/CONFIGURATION.md)
- [Sandbox Guide](https://github.com/xadnavyaai/vectra-guard/blob/main/SANDBOX.md)

