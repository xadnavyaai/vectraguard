# Vectra Guard Sandbox Execution

**Enterprise-grade sandboxing for AI coding agents**

Vectra Guard's sandbox system provides transparent, fast, and secure isolation for risky commands. It intelligently decides when to sandbox based on risk level and context, with minimal friction for developers.

---

## 🎯 Core Capabilities

### Phase 1: Core Execution Engine
**Policy-driven executor chooses between host and sandbox execution**

```yaml
sandbox:
  enabled: true
  mode: always  # Default: always (maximum security)
                # Options: always, auto, risky, never
```

**Decision Logic (Default - Always Mode):**
- ✅ **All commands** → sandbox execution (maximum security)
- 🚀 **Caching enabled** → 10x speedup on repeated runs
- 📝 Trusted commands → can be remembered to skip sandbox
- 🔒 Critical commands → always sandboxed (cannot bypass)

**Decision Logic (Auto Mode):**
- ✅ Low-risk commands → host execution
- ⚠️ Medium/high-risk → sandbox execution
- 🔒 Networked installs → automatic sandboxing
- 📝 Trusted commands → remembered and run on host

### Phase 2: Sandbox Runtime & Isolation
**Multiple runtime options with transparent execution**

```yaml
sandbox:
  runtime: docker  # docker, podman, process
  image: ubuntu:22.04
  timeout: 300
```

**Supported Runtimes:**
- **Bubblewrap** - Fastest (<1ms overhead), Linux only, battle-tested by Flatpak
- **Namespace** - Fast (<1ms overhead), Linux only, pure Go implementation
- **Docker** - Most compatible, works everywhere (2-5s startup)
- **Podman** - Rootless alternative, same CLI as Docker

### ✅ Dependencies & Setup (Recommended)
Install the runtime dependencies that match your environment. For most users, install both Docker and bubblewrap so auto‑selection can pick the best runtime.

**macOS (Docker runtime)**
```bash
# Docker Desktop (recommended)
brew install --cask docker
open -a Docker
```

**Debian/Ubuntu (Docker + bubblewrap)**
```bash
sudo apt-get update -y
sudo apt-get install -y docker.io docker-compose-plugin bubblewrap uidmap
sudo systemctl enable --now docker
sudo usermod -aG docker $USER
```

**Fedora/RHEL (Docker + bubblewrap)**
```bash
sudo dnf install -y docker docker-compose-plugin bubblewrap
sudo systemctl enable --now docker
sudo usermod -aG docker $USER
```

**Install via Vectra Guard**
```bash
# After installing vectra-guard
vectra-guard sandbox deps install

# Wrapper script (uses vectra-guard under the hood)
./scripts/install-sandbox-deps.sh
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-sandbox-deps.sh | bash
```

**Dry run (preview commands only):**
```bash
vectra-guard sandbox deps install --dry-run
DRY_RUN=1 ./scripts/install-sandbox-deps.sh
```

> **Note (Linux namespaces):** bubblewrap and namespace runtimes require unprivileged user namespaces. On most modern distros this is enabled by default. If not, set:
> `sudo sysctl -w kernel.unprivileged_userns_clone=1`

**Runtime Selection:**
Vectra Guard intelligently selects the best runtime based on your environment:

```
┌──────────────────────────────────────────────────────────┐
│                  Runtime Selection                       │
├──────────────────────────────────────────────────────────┤
│  1. Auto-detect environment (dev vs CI/prod)            │
│  2. Check available capabilities                         │
│  3. Select best runtime:                                 │
│     • Dev:  bubblewrap → namespace → docker             │
│     • CI:   docker → bubblewrap → namespace             │
│     • Prod: docker → bubblewrap → namespace             │
└──────────────────────────────────────────────────────────┘
```

**Runtime Comparison:**

| Runtime | Startup | Security | Dev Experience | Platform |
|---------|---------|----------|----------------|----------|
| **bubblewrap** | <1ms | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Linux |
| **namespace** | <1ms | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Linux |
| **docker** | 2-5s | ⭐⭐⭐⭐⭐ | ⭐⭐ | All |
| **podman** | 2-5s | ⭐⭐⭐⭐⭐ | ⭐⭐ | All |

**Bubblewrap Runtime** (Recommended for Linux):
- ✅ **< 1ms overhead** (essentially native speed)
- ✅ **All caches work** (npm, pip, cargo, go, maven, gradle)
- ✅ **State persistence** (node_modules, target/ preserved)
- ✅ **Zero configuration** (works out of the box)
- ✅ **Read-only system** (cannot modify /bin, /usr, /etc)
- ✅ **Network optional** (can be enabled/disabled)

**Installation:**
```bash
# Install bubblewrap for best performance
sudo apt install bubblewrap     # Debian/Ubuntu
sudo yum install bubblewrap     # RHEL/CentOS
sudo dnf install bubblewrap     # Fedora
```

**Configuration:**
```yaml
sandbox:
  runtime: auto              # Auto-select best runtime
  auto_detect_env: true      # Detect dev vs CI
  prefer_fast: true          # Prefer fast runtimes in dev
```

### Phase 3: Cache Strategy
**10x faster sandbox execution through intelligent cache mounting**

Vectra Guard's caching system makes isolated execution **10x faster** than traditional approaches by intelligently mounting host cache directories into ephemeral containers. This enables the "best of both worlds": **strong isolation + development speed**.

**How It Works:**

```
┌──────────────────────────────────────────────────────┐
│  Host Machine                                         │
│  ┌────────────────────────────────────────────┐      │
│  │ ~/.npm/ (Cache Root)                       │      │
│  │   ├── express@4.18.0/                      │      │
│  │   ├── lodash@4.17.21/                      │      │
│  │   └── ...1000+ packages                    │      │
│  └────────────────────────────────────────────┘      │
│          ▲                          │                 │
│          │ Persist                  │ Read            │
│          │                          ▼                 │
│  ┌────────────────────────────────────────────┐      │
│  │ Sandbox Container (Ephemeral)             │      │
│  │  ┌──────────────────────────────────────┐  │      │
│  │  │ /.npm/ ──▶ Mounted from host         │  │      │
│  │  └──────────────────────────────────────┘  │      │
│  │                                             │      │
│  │  npm install express                        │      │
│  │    1. Check /.npm/ for express ✅           │      │
│  │    2. Found! No download needed             │      │
│  │    3. Install completes in 1.2s ⚡          │      │
│  └────────────────────────────────────────────┘      │
└──────────────────────────────────────────────────────┘
```

**Key Innovations:**
1. **Bind Mounts** - Not volumes, direct FS mount (zero copy)
2. **Read/Write** - Container can use AND populate cache
3. **Persistent** - Cache survives container destruction
4. **Shared** - All projects use same cache
5. **Zero Config** - Automatic detection and mounting

**Configuration:**
```yaml
sandbox:
  enable_cache: true
  cache_dirs:
    - ~/.npm
    - ~/.cargo
    - ~/go/pkg
```

**Automatic Cache Detection:**

| Ecosystem | Cache Location | Auto-Detect | Speedup |
|-----------|---------------|-------------|---------|
| **npm** | `~/.npm` | ✅ | 10.2x |
| **Yarn v1** | `~/.yarn` | ✅ | 8.4x |
| **Yarn v2+** | `~/.yarn/cache` | ✅ | 9.1x |
| **pnpm** | `~/.pnpm-store` | ✅ | 12.3x |
| **pip** | `~/.cache/pip` | ✅ | 9.6x |
| **Poetry** | `~/.cache/pypoetry` | ✅ | 8.8x |
| **Cargo** | `~/.cargo` | ✅ | 15.2x |
| **Go** | `~/go/pkg` | ✅ | 11.0x |
| **Ruby Gems** | `~/.gem` | ✅ | 7.3x |
| **Maven** | `~/.m2` | ✅ | 6.8x |
| **Gradle** | `~/.gradle` | ✅ | 7.1x |

**Performance Example:**

```bash
# First run (builds cache)
$ vg exec "npm install"
⏱️  12.8s - Downloads 50 packages
📦 Cache populated

# Second run (uses cache)
$ vg exec "npm install"
⏱️  1.2s ⚡ - All from cache!
🎉 10x FASTER
```

**Benefits:**
- 🚀 **10x faster** subsequent installs
- 💾 **Shared** across sandbox instances
- 🔄 **Persistent** between runs
- 🌐 **Offline-capable** once populated
- 🎯 **Automatic** - zero configuration needed

**Custom Cache Configuration:**
```yaml
sandbox:
  enable_cache: true
  cache_dirs:
    - ~/.custom-cache:/.custom-cache
    - /opt/shared-cache:/cache
```

### Phase 4: Security Posture Controls
**Tune isolation without sacrificing speed**

```yaml
sandbox:
  security_level: balanced  # permissive, balanced, strict, paranoid
  network_mode: restricted   # none, restricted, full
```

**Security Levels:**

| Level | Network | Root FS | Memory | CPU | Use Case |
|-------|---------|---------|--------|-----|----------|
| **Permissive** | Full | R/W | 2GB | 2.0 | Development |
| **Balanced** | Restricted | R/W | 1GB | 1.0 | Default |
| **Strict** | Restricted | RO | 512MB | 0.5 | CI/CD |
| **Paranoid** | None | RO | 256MB | 0.25 | Production |

### Phase 5: Policy Learning & Trust
**"Approve and remember" reduces friction**

```bash
# Interactive approval with remember option
⚠️  Command requires approval
Command: npm install suspicious-package
Risk Level: MEDIUM

Options:
  y  - Yes, run once
  r  - Yes, and remember (trust permanently)
  n  - No, cancel

Choose [y/r/N]: r
✅ Approved and remembered
```

**Trust Management:**
```bash
# List trusted commands
vg trust list

# Add command to trust store
vg trust add "npm install express" --note "Common package"

# Remove trusted command
vg trust remove "npm install express"

# Clean expired entries
vg trust clean
```

### Phase 6: Developer Experience
**"Just works" mode with minimal friction**

```yaml
# Developer preset - copy to vectra-guard.yaml
guard_level:
  level: low

sandbox:
  enabled: true
  mode: auto
  security_level: permissive
  enable_cache: true
  network_mode: full
```

**Quick Start:**
```bash
# Use developer preset
cp presets/developer.yaml vectra-guard.yaml

# Install and run - no friction
npm install express
npm run dev
```

### Phase 7: Observability & Analytics
**Track sandbox usage and performance**

```bash
# View metrics
vg metrics show

# Output:
Vectra Guard Sandbox Metrics
===============================
Total Executions:    142
  - Host:            89 (62.7%)
  - Sandbox:         53 (37.3%)
  - Cached:          41 (28.9%)

Average Duration:    1.2s

By Risk Level:
  - low: 89 (62.7%)
  - medium: 42 (29.6%)
  - high: 11 (7.7%)
```

**Metrics Tracking:**
- Total/host/sandbox execution counts
- Cache hit rates
- Average execution duration
- Breakdown by risk level
- Breakdown by runtime
- Last 100 execution history

---

## 🎨 UX Enhancements

### Single-line Notices
```bash
📦 Running in sandbox (cached).
   Why: medium risk + networked install
```

### Explain the "Why"
Every sandbox decision includes clear reasoning:
- "Sandbox chosen: medium risk + networked install"
- "Running on host: low risk, no isolation needed"
- "Running on host: command previously approved and trusted"

### No Prompts Unless Needed
- Low-risk: silent execution
- Medium-risk: automatic sandboxing, no prompt
- High-risk: approval only if interactive
- Trusted: remembered approvals skip prompts

### Remembered Approvals
```bash
✅ Approved and remembered

# Next time - no prompt!
npm install express
# → runs directly, remembered from previous approval
```

### Consistent Output
Same format whether running on host or in sandbox:
```bash
# Host execution
$ vg exec "echo hello"
hello

# Sandbox execution
$ vg exec "npm install"
📦 Running in sandbox (cached).
   Why: medium risk + networked install
added 142 packages in 2.3s
```

---

## 📋 Configuration Reference

### Complete Sandbox Config

```yaml
sandbox:
  # Core settings
  enabled: true
  mode: auto              # auto, always, risky, never
  security_level: balanced # permissive, balanced, strict, paranoid
  
  # Runtime
  runtime: docker         # docker, podman, process
  image: ubuntu:22.04
  timeout: 300           # seconds
  
  # Caching
  enable_cache: true
  cache_dirs:
    - ~/.npm
    - ~/.cargo
    - ~/go/pkg
  
  # Network
  network_mode: restricted # none, restricted, full
  
  # Security
  seccomp_profile: /path/to/seccomp.json
  
  # Environment
  env_whitelist:
    - HOME
    - USER
    - PATH
    - SHELL
    - TERM
    - PWD
  
  # Custom mounts
  bind_mounts:
    - host_path: /path/on/host
      container_path: /path/in/container
      read_only: false
  
  # Observability
  enable_metrics: true
  log_output: false
  
  # Trust store
  trust_store_path: ~/.vectra-guard/trust.json
```

---

## 🎯 Presets

### Developer Preset
**Optimized for local development**

```bash
cp presets/developer.yaml vectra-guard.yaml
```

Features:
- Low guard level (minimal friction)
- Auto sandboxing for risky commands
- Permissive security (fast)
- Full network access
- Caching enabled

### CI/CD Preset
**Balanced for automated pipelines**

```bash
cp presets/ci-cd.yaml vectra-guard.yaml
```

Features:
- High guard level (strong protection)
- Sandbox only high-risk commands
- Balanced security
- Restricted network
- Caching enabled for speed

### Production Preset
**Maximum protection for production**

```bash
cp presets/production.yaml vectra-guard.yaml
```

Features:
- Paranoid guard level (all require approval)
- Always sandbox
- Maximum isolation
- No network access
- No caching (reproducibility)

---

## 🔧 Advanced Usage

### Custom Security Profiles

```yaml
sandbox:
  security_level: strict
  seccomp_profile: /etc/vectra-guard/seccomp-custom.json
  network_mode: none
  bind_mounts:
    - host_path: /readonly/data
      container_path: /data
      read_only: true
```

### Environment Variable Control

```yaml
sandbox:
  env_whitelist:
    - HOME
    - USER
    - PATH
    # Custom vars
    - COMPANY_API_TOKEN
    - BUILD_NUMBER
```

### Selective Sandboxing

```yaml
# Allowlist trusted commands
policies:
  allowlist:
    - "npm test"
    - "npm run build"
    - "git status"

# Always sandbox these
policies:
  denylist:
    - "rm -rf"
    - "curl * | sh"
    - "DROP DATABASE"
```

---

## 🚀 Performance Tips

### 1. Enable Caching
```yaml
sandbox:
  enable_cache: true
```
**Impact:** 10x faster subsequent installs

### 2. Use Process Sandbox (Linux)
```yaml
sandbox:
  runtime: process
```
**Impact:** 50% faster than Docker

### 3. Permissive Mode for Dev
```yaml
sandbox:
  security_level: permissive
```
**Impact:** Minimal overhead

### 4. Trust Common Commands
```bash
vg trust add "npm install"
vg trust add "npm test"
```
**Impact:** Skip sandboxing entirely

---

## 🔍 Troubleshooting

### Docker Not Available
```bash
# Use podman instead
sandbox:
  runtime: podman

# Or process sandbox (Linux only)
sandbox:
  runtime: process
```

### Slow Execution
```bash
# Enable caching
sandbox:
  enable_cache: true

# Or trust frequently-used commands
vg trust add "npm install"
```

### Network Issues
```bash
# Allow full network access
sandbox:
  network_mode: full
```

### Permission Errors
```bash
# Use permissive security level
sandbox:
  security_level: permissive

# Or bind mount with write access
sandbox:
  bind_mounts:
    - host_path: /path/to/data
      container_path: /data
      read_only: false
```

---

## 📊 Metrics & Analytics

### View Metrics
```bash
# Human-readable summary
vg metrics show

# JSON format
vg metrics show --json
```

### Reset Metrics
```bash
vg metrics reset
```

### Metrics File Location
```
~/.vectra-guard/metrics.json
```

---

## 🔐 Security Considerations

### Isolation Guarantees

| Runtime | Process | Network | Filesystem | Performance |
|---------|---------|---------|------------|-------------|
| **Docker** | ✅ Full | ✅ Full | ✅ Full | Good |
| **Podman** | ✅ Full | ✅ Full | ✅ Full | Good |
| **Process** | ⚠️ Partial | ⚠️ Partial | ⚠️ Partial | Excellent |

### Trust Store Security
- Stored at `~/.vectra-guard/trust.json`
- File permissions: `0600` (owner read/write only)
- SHA256 hash-based indexing
- Optional expiration times

### Seccomp Profiles
Custom seccomp profiles restrict system calls:
```yaml
sandbox:
  seccomp_profile: /etc/vectra-guard/seccomp-strict.json
```

Example profile provided: `seccomp-profile.json`

---

## 🎓 Examples

### Example 1: Development Workflow
```bash
# Clone and setup
git clone https://github.com/myproject/app.git
cd app

# Use developer preset
cp /path/to/vectra-guard/presets/developer.yaml vectra-guard.yaml

# Install dependencies (auto-sandboxed, cached)
vg exec "npm install"

# Run tests (trusted command, runs on host)
vg trust add "npm test"
vg exec "npm test"

# Build (sandboxed first time, then trusted)
vg exec "npm run build"
vg trust add "npm run build"
```

### Example 2: CI/CD Pipeline
```yaml
# .github/workflows/ci.yml
steps:
  - name: Setup Vectra Guard
    run: |
      curl -sSL https://vectra-guard.dev/install.sh | bash
      cp presets/ci-cd.yaml vectra-guard.yaml

  - name: Install dependencies
    run: vg exec "npm ci"

  - name: Run tests
    run: vg exec "npm test"

  - name: Build
    run: vg exec "npm run build"
```

### Example 3: Production Deployment
```bash
# Production server
cp presets/production.yaml vectra-guard.yaml

# All commands require approval
vg exec "kubectl apply -f deployment.yaml"
# → Prompts for approval with security details

# Critical operations blocked
vg exec "rm -rf /data"
# → Blocked immediately
```

---

## 🤝 Integration

### Shell Wrapper
```bash
# Install shell wrapper
./scripts/install-shell-wrapper.sh

# Now all commands protected
npm install  # → automatically uses vg exec
```

---

## 🔍 Package Audits

Vectra Guard can run dependency audits for npm and Python projects.

```bash
# npm audit in a project directory
vectra-guard audit npm --path /path/to/project

# python audit (uses requirements.txt when present)
vectra-guard audit python --path /path/to/project

# Disable auto-install of audit tools
vectra-guard audit npm --path . --no-install
vectra-guard audit python --path . --no-install
```

### VS Code / Cursor Integration
```bash
# Setup Cursor protection
./scripts/setup-cursor-protection.sh
```

### API Usage
```go
import "github.com/vectra-guard/vectra-guard/internal/sandbox"

executor, _ := sandbox.NewExecutor(cfg, logger)
decision := executor.DecideExecutionMode(ctx, cmdArgs, riskLevel, findings)
err := executor.Execute(ctx, cmdArgs, decision)
```

---

## 📚 Further Reading

- [Configuration Guide](CONFIGURATION.md)
- [Getting Started](GETTING_STARTED.md)
- [Advanced Features](ADVANCED_FEATURES.md)
- [API Documentation](https://pkg.go.dev/github.com/vectra-guard/vectra-guard)

---

## 💡 Best Practices

1. **Start with Auto Mode**
   ```yaml
   sandbox:
     mode: auto
   ```
   Let Vectra Guard make smart decisions.

2. **Enable Caching**
   ```yaml
   sandbox:
     enable_cache: true
   ```
   Dramatically improves performance.

3. **Trust Common Commands**
   ```bash
   vg trust add "npm test"
   vg trust add "npm run build"
   ```
   Reduce friction for safe operations.

4. **Monitor Metrics**
   ```bash
   vg metrics show
   ```
   Understand your usage patterns.

5. **Use Presets**
   ```bash
   cp presets/developer.yaml vectra-guard.yaml
   ```
   Start with proven configurations.

---

## 🎉 Summary

Vectra Guard's sandbox system provides:

✅ **Transparent** - Works like normal execution  
✅ **Fast** - Caching makes it nearly instant  
✅ **Secure** - Multiple isolation levels  
✅ **Smart** - Auto-detects when to sandbox  
✅ **Flexible** - Multiple runtimes and presets  
✅ **Observable** - Full metrics and analytics  
✅ **Learnable** - Remembers trusted commands  
✅ **Developer-friendly** - Minimal friction  

**Get started in 60 seconds:**
```bash
cp presets/developer.yaml vectra-guard.yaml
vg exec "npm install express"
```

That's it! 🚀

