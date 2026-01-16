# Vectra Guard v0.1.1 Release Notes

Release Date: 2026-01-16

## ✅ Highlights
- Fix Debian/Ubuntu sandbox dependency installation when `docker-compose-plugin` is not available.
- Improve Linux namespace sandbox setup by relaxing AppArmor restrictions when needed.
- Updated sandbox documentation for Ubuntu user namespace requirements.

## 🔧 Fixes
### Sandbox Dependencies (Linux)
- Install core Docker + bubblewrap dependencies first.
- Attempt `docker-compose-plugin`, then fall back to `docker-compose` without failing the install.
- Enable `kernel.apparmor_restrict_unprivileged_userns=0` when supported to allow process/namespace sandboxing.

## 📦 Installation
Recommended one-liner:

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-all.sh | bash
```

## 🔒 Security Note
Critical commands (e.g., `rm -rf /`) remain **non-bypassable** even if sandboxing is disabled.

