# Scanning 6 AI Agent Repos for Security Risks: What We Found

*How we ran Vectra Guard on Moltbot, Open Interpreter, AutoGPT, Huginn, LocalAGI, and Aider—and what it means for securing local AI agents.*

---

Local AI agents are having a moment. Tools like Moltbot (formerly Clawdbot), Open Interpreter, and AutoGPT give users assistants that run code, hit APIs, and automate tasks on their machines. That power also creates real risk: exposed dashboards, secrets in configs, and unchecked subprocess or HTTP usage can turn a productivity tool into an attack surface—as recent headlines about Moltbot showed.

We wanted to see how common these patterns are across the ecosystem. So we ran **Vectra Guard** (our open-source CLI for sandboxing, secret scanning, and static security checks) on six popular AI agent repos. Here’s what we did and what we found.

## What We Scanned

We cloned six public repos into a test workspace and ran:

- **Static security scan** — `vg scan-security` for Go, Python, C, and config (YAML/JSON), looking for things like bind-to-all-interfaces, env/API access, remote HTTP, subprocess use, and auth-off patterns.
- **Repo audit** — `vg audit repo` for the same code findings plus secret detection and package audits (npm/pip) where applicable.

**Repos:**

| Project | What it does |
|--------|----------------|
| **Moltbot** | Personal AI agent; messaging, local execution, control panel. |
| **Open Interpreter** | Natural language interface that runs code locally. |
| **AutoGPT** | Autonomous agent platform; tasks, tools, API. |
| **Huginn** | Agent framework for monitoring and acting on your behalf. |
| **LocalAGI** | Self-hosted local AI agent platform. |
| **Aider** | AI pair programming in the terminal; edits code, runs locally. |

All are open source and run or trigger code locally—exactly the kind of surface we care about.

## The Numbers

| Repo | Code findings | Severity mix | Secrets (candidates) | Package issues |
|------|----------------|-------------|----------------------|----------------|
| AutoGPT | 1,298 | Medium | 5,651 | 14 (python) |
| Open Interpreter | 247 | 5 high, 242 medium | 10 | 14 (python) |
| Aider | 435 | Medium | 170 | — |
| LocalAGI | 68 | 2 high, 66 medium | 13 | 14 (python) |
| Moltbot | 11 | Medium | 1,546 | npm 0, python 14 |
| Huginn | 3 (scan) / 0 (audit) | — | 104 | 14 (python) |

*Code findings* = static patterns (e.g. env access, external HTTP, bind 0.0.0.0, subprocess); comment-only lines are skipped. *Secrets* = key-like strings in source/config with secret context (token/api_key/secret etc.) and high-entropy values; lockfiles are skipped and paths/slugs/identifiers are filtered to reduce false positives (~7.5K total across six repos). See [Secret findings examples](secret-findings-examples.md) and [Findings analysis](../reports/findings-analysis.md). *Package issues* = pip-audit / npm audit where we ran it.

The table above reflects a run with **improved detection** (context-based secret flagging and comment-line skip for security rules); earlier runs showed higher secret counts before these FP reductions.

## What Showed Up Most

From the full scan we extracted counts by rule code (see [similar-agent-findings.json](../reports/similar-agent-findings.json) (vectra-guard tool output format) and [findings-analysis.md](../reports/findings-analysis.md) for reference):

- **External and remote HTTP** — `PY_EXTERNAL_HTTP` (1,245) and `PY_REMOTE_HTTP` (175) dominate. Agents call APIs and web services; the risk is SSRF and leaking URLs or responses. Validate and restrict what gets passed into requests.
- **Environment and API usage** — `PY_ENV_ACCESS` (459) and Go’s `GO_ENV_READ` (37). Env vars hold keys and config; the risk is leaking them in logs or responses. Don’t echo env in user-facing or external channels.
- **Subprocess and exec** — `PY_SUBPROCESS` (175) and a few `PY_EXEC`/`PY_EVAL`. Running shell or dynamic code is powerful; it should be validated and sandboxed. The small number of exec/eval hits are high-impact if fed by untrusted input.
- **Binding to all interfaces** — `BIND_ALL_INTERFACES` (22). Binding to `0.0.0.0` is fine only with auth and TLS; otherwise it’s the same class of misconfiguration that led to exposed Moltbot panels.
- **Config and deployment** — `UNAUTHENTICATED_ACCESS` (5) in config. Auth disabled or weakened in config, combined with bind-all or proxy trust, leads to unauthenticated access. Fix or justify every one.

**Numbers cross-checked:** Rule-code counts above (e.g. PY_EXTERNAL_HTTP 1,245; PY_REMOTE_HTTP 175; BIND_ALL_INTERFACES 22) were validated against [similar-agent-scan-raw.txt](../reports/similar-agent-scan-raw.txt) from the full six-repo scan.

None of this is to say these projects are “unsafe”—they’re complex, actively developed, and many findings are in tests or optional features. The takeaway is that **these patterns are widespread**. Tools that flag them before deployment can help maintainers and operators lock down instances. Findings are stored in **vectra-guard tool output format** (same as `vg audit repo --output json`: `file`, `line`, `severity`, `code`, `description`, `remediation`). For a deeper discussion of each pattern and how to address it, see [Findings analysis](../reports/findings-analysis.md).

## How to Run It (Using the Release Artifact)

Use the **release binary** from [GitHub Releases](https://github.com/xadnavyaai/vectra-guard/releases) and the scan script via **curl** from the repo. No clone or build required.

**1. Create a directory and download the script:**

```bash
mkdir -p vg-scan/scripts
curl -sSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/test-similar-agent-repos.sh -o vg-scan/scripts/test-similar-agent-repos.sh
chmod +x vg-scan/scripts/test-similar-agent-repos.sh
cd vg-scan
```

**2. Download the binary for your platform** (e.g. v0.4.0 — pick the latest). Examples:

```bash
# Linux (amd64)
curl -sSL https://github.com/xadnavyaai/vectra-guard/releases/download/v0.4.0/vectra-guard-linux-amd64 -o vectra-guard
chmod +x vectra-guard

# Linux (arm64)
curl -sSL https://github.com/xadnavyaai/vectra-guard/releases/download/v0.4.0/vectra-guard-linux-arm64 -o vectra-guard
chmod +x vectra-guard

# macOS (Apple Silicon)
curl -sSL https://github.com/xadnavyaai/vectra-guard/releases/download/v0.4.0/vectra-guard-darwin-arm64 -o vectra-guard
chmod +x vectra-guard

# macOS (Intel)
curl -sSL https://github.com/xadnavyaai/vectra-guard/releases/download/v0.4.0/vectra-guard-darwin-amd64 -o vectra-guard
chmod +x vectra-guard
```

**3. Clone the six AI agent repos** into `test-workspaces/`:

```bash
./scripts/test-similar-agent-repos.sh clone-all
```

**4. Run the scan** (use the binary you downloaded in this directory):

```bash
VG_CMD="./vectra-guard" ./scripts/test-similar-agent-repos.sh
```

Optional: move `vectra-guard` to your PATH (e.g. `sudo mv vectra-guard /usr/local/bin/`) and run `./scripts/test-similar-agent-repos.sh` with no `VG_CMD`; the script defaults to `vg` on PATH.

The script and repo list live in the [Vectra Guard repo](https://github.com/xadnavyaai/vectra-guard); the control-panel security doc there maps Moltbot-style incidents to our checks and lists the rule codes.

## What You Can Do With This

- **Maintainers** — Run `vg scan-security --path . --languages go,python,c,config` and `vg audit repo` in CI to catch bind-all, env/HTTP misuse, and config issues before they hit production.
- **Operators** — Before exposing any agent dashboard, fix or justify every `BIND_ALL_INTERFACES` and `UNAUTHENTICATED_ACCESS`; use auth and TLS when binding to 0.0.0.0.
- **Contributors** — Use `vg exec --` for agent-triggered commands so they run in a sandbox, and consider `vg prompt-firewall` for user-facing prompt content.

Local AI agents are here to stay. So are the risks that come with broad system access. Scanning early and often—and sandboxing execution—is one way to keep the upside without the headline.

---

*Vectra Guard is open source: [GitHub](https://github.com/xadnavyaai/vectra-guard). For rule reference and a securing checklist, see [Control panel & deployment security](https://github.com/xadnavyaai/vectra-guard/blob/main/docs/control-panel-security.md).*
