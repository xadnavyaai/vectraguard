# Control Panel & AI Agent Deployment Security

This document maps real-world incidents (e.g. exposed AI control panels like Moltbot/Clawdbot) to **Vectra Guard** capabilities, provides a **securing checklist**, and documents **scan-security** rule codes for code and config.

## Table of Contents

- [Reference: Moltbot/Clawdbot Security Alert](#reference-moltbotclawdbot-security-alert)
- [What Vectra Guard Can Do Today](#what-vectra-guard-can-do-today)
- [Securing AI Agent Platforms: Checklist](#securing-ai-agent-platforms-checklist)
- [CI Example: Security and Config Scans](#ci-example-security-and-config-scans)
- [Scan Security Rule Reference](#scan-security-rule-reference)
- [Testing: Release vs Local & Similar Tools](#testing-release-vs-local--similar-tools)
- [Extensions and Future Work](#extensions-and-future-work)

---

## Reference: Moltbot/Clawdbot Security Alert

In brief, the incident involved:

- **Exposed control panels** — Hundreds of internet-facing admin dashboards reachable without proper auth.
- **Data exposure** — Configuration data, API keys, full conversation histories from private chats.
- **Agent autonomy risk** — Agents can send messages, run tools, execute commands (Telegram, Slack, Discord); attackers could impersonate operators and siphon data.
- **Worst case** — Some instances allowed **unauthenticated command execution** on the host with elevated privileges.
- **Root cause** — **Misconfiguration**: localhost trust assumptions + reverse proxy setups caused internet connections to be treated as local and auto-approved.
- **Architectural concentration** — AI agents read messages, store secrets, execute actions; when misconfigured, multiple security boundaries collapse at once.

---

## What Vectra Guard Can Do Today

| Article issue | Vectra Guard today | How |
|---------------|--------------------|-----|
| **Exposed dashboards / bind 0.0.0.0** | **Predetect** | `vg scan-security` flags `BIND_ALL_INTERFACES` in Go, Python, C, and **config** (YAML/JSON). Our own dashboard (`vg serve`) binds only to `127.0.0.1`. |
| **Secrets in code / config** | **Predetect** | `vg scan-secrets` finds API keys, tokens, and high-entropy strings in repo (optional allowlist). |
| **Risky agent instructions** | **Predetect** | `vg validate-agent` statically checks agent workflows for dangerous patterns. |
| **Prompt injection** | **Control** | `vg prompt-firewall` blocks or scores malicious prompt content. |
| **Uncontrolled command execution** | **Control** | `vg exec` runs commands in a sandbox; `vg lockdown` blocks execution when enabled. |
| **Risky code patterns (HTTP, env, subprocess)** | **Predetect** | `vg scan-security` flags external HTTP, env access, subprocess/system calls in Go/Python/C. |
| **Config/deployment misconfig** | **Predetect** | `vg scan-security --languages config` scans YAML/JSON for bind 0.0.0.0, trust-proxy, and auth: false. |
| **Localhost trust + reverse proxy** | **Predetect** | Config scan flags `LOCALHOST_TRUST_PROXY` (trustProxy, X-Forwarded-For). |

**Summary:** Vectra Guard helps **predetect** (secrets, risky code, bind-all, agent scripts, **and** deployment/config misconfigs) and **control** (sandbox, lockdown, prompt firewall). Use `--languages go,python,c,config` for full coverage including control-panel configs.

---

## Securing AI Agent Platforms: Checklist

Use this workflow to reduce control-panel and agent deployment risks:

1. **Secrets**
   - Run `vg scan-secrets --path .` in CI; fix or allowlist real secrets.
   - Keep `.env.example` and baseline files out of production; allowlist them if needed.

2. **Code and config**
   - Run `vg scan-security --path . --languages go,python,c,config`.
   - Fix or justify every `BIND_ALL_INTERFACES`, `LOCALHOST_TRUST_PROXY`, and `UNAUTHENTICATED_ACCESS` in configs.
   - Ensure dashboards and control panels use auth and TLS when bound to 0.0.0.0.

3. **Agent workflows**
   - Run `vg validate-agent <path-to-agent-workflows>` (e.g. `.agent` or `scripts`).
   - Review any risky patterns before deploying.

4. **Execution and prompts**
   - Use `vg exec --` for agent-triggered commands so they run in the sandbox.
   - Use `vg lockdown enable` when no one should run sensitive commands; `vg lockdown disable` when deploying.
   - Pipe user-facing prompt content through `vg prompt-firewall` to block or score injection.

5. **Operational**
   - Never expose admin dashboards on 0.0.0.0 without authentication and TLS.
   - If using a reverse proxy, ensure auth is enforced **before** trusting `X-Forwarded-For` or localhost.

---

## CI Example: Security and Config Scans

```yaml
# Example: run in CI (e.g. .github/workflows/security.yml)
- name: Scan secrets
  run: vg scan-secrets --path .

- name: Scan security (code + config)
  run: vg scan-security --path . --languages go,python,c,config

- name: Validate agent workflows
  run: vg validate-agent .agent
```

`vg scan-security` exits with code 2 when any finding is reported; use that to fail the job or gate merges.

---

## Scan Security Rule Reference

### Go (`--languages go`)

| Code | Severity | Description |
|------|----------|-------------|
| `GO_EXEC_COMMAND` | high | Use of `exec.Command`; validate and sandbox inputs. |
| `GO_DANGEROUS_SHELL` | critical | Dangerous shell pattern (e.g. `rm -rf /`, `curl \| sh`). |
| `GO_NET_HTTP` | medium | Use of `net/http`; ensure remote calls are authenticated/sanitized. |
| `GO_ENV_READ` | medium | Environment variable access; avoid leaking credentials. |
| `GO_SYSTEM_WRITE` | high | Writing to `/etc`, `/var`, `/usr`; review for safety. |
| `GO_EXTERNAL_HTTP` | medium | Non-localhost HTTP(S) URL; SSRF risk with untrusted input. |
| `BIND_ALL_INTERFACES` | medium | Binding to 0.0.0.0; ensure auth and TLS. |

### Python (`--languages python`)

| Code | Severity | Description |
|------|----------|-------------|
| `PY_EVAL` | high | Use of `eval()`; avoid untrusted input. |
| `PY_EXEC` | high | Use of `exec()`; avoid untrusted input. |
| `PY_SUBPROCESS` | medium | Use of subprocess/os.system; validate and sandbox commands. |
| `PY_REMOTE_HTTP` | medium | Remote HTTP (e.g. requests); validate URLs and responses. |
| `PY_ENV_ACCESS` | medium | Environment or .env access; avoid exposing secrets. |
| `PY_EXTERNAL_HTTP` | medium | Non-localhost HTTP(S) URL; SSRF risk. |
| `BIND_ALL_INTERFACES` | medium | Binding to 0.0.0.0; ensure auth and TLS. |

### C (`--languages c`)

| Code | Severity | Description |
|------|----------|-------------|
| `C_SHELL_EXEC` | high | Use of system/popen/exec*; avoid untrusted input. |
| `C_GETS` | critical | Use of `gets()`; inherently unsafe. |
| `C_UNSAFE_STRING` | high | strcpy/strcat; buffer overflow risk. |
| `C_MEMCPY` | medium | memcpy; validate bounds. |
| `C_RAW_SOCKET` | medium | Raw socket use; review for abuse. |
| `BIND_ALL_INTERFACES` | medium | Binding to 0.0.0.0; ensure auth and TLS. |

### Config / deployment (`--languages config`)

Scans `.yaml`, `.yml`, `.json` when `config` is included in `--languages`.

| Code | Severity | Description |
|------|----------|-------------|
| `BIND_ALL_INTERFACES` | medium | Config binds to 0.0.0.0; control panels need auth and TLS. |
| `LOCALHOST_TRUST_PROXY` | medium | trustProxy / X-Forwarded-For; ensure auth is not bypassed. |
| `UNAUTHENTICATED_ACCESS` | high | auth/secure set to false or disabled; require authentication. |

---

## Testing: Release vs Local & Similar Tools

### Was the release binary used?

The Moltbot scan in this repo was run with **local source** (`go run . scan-security` and `go run . audit repo`), not with the published release binary. To validate the **release**:

1. **Build locally:** `./scripts/build-release.sh v0.4.0` then run `dist/vectra-guard-<os>-<arch> scan-security --path <repo>` and `audit repo --path <repo>`.
2. **Use the release script:** `./scripts/test-release.sh v0.4.0` runs the built binary from `dist/` (version, help, and other checks).
3. **Install from GitHub:** Download the binary for your platform from [Releases](https://github.com/xadnavyaai/vectra-guard/releases), put `vg` on PATH, then run `vg scan-security --path <repo>` and `vg audit repo --path <repo>`.

Using the release binary ensures you are testing the same code that users install.

### Testing on similar AI agent tools

You can run the same scans on other open-source AI agents that run locally and execute code (similar to Moltbot/Clawdbot):

| Project | Repo | Notes |
|--------|------|--------|
| **Moltbot** | [moltbot/moltbot](https://github.com/moltbot/moltbot) | Python/TS; control-panel–style risks. |
| **Open Interpreter** | [openinterpreter/open-interpreter](https://github.com/openinterpreter/open-interpreter) | Python; runs code locally; good for PY_* and bind/subprocess checks. |
| **AutoGPT** | [Significant-Gravitas/Auto-GPT](https://github.com/Significant-Gravitas/Auto-GPT) | Agent platform; Python/JS; useful for env, HTTP, exec patterns. |
| **Huginn** | [huginn/huginn](https://github.com/huginn/huginn) | Agent framework; local execution; MIT. |
| **LocalAGI** | [mudler/LocalAGI](https://github.com/mudler/LocalAGI) | Local agent platform; mix of languages. |
| **Aider** | [Aider-AI/aider](https://github.com/Aider-AI/aider) | AI pair programming; terminal, local code execution; Python. |

**Clone all and scan:** `scripts/test-similar-agent-repos.sh` clones the full list above into `test-workspaces/` and runs scan-security + audit repo on each.

```bash
# One-time: clone all similar-agent repos into test-workspaces/
./scripts/test-similar-agent-repos.sh clone-all

# Scan all repos (use release binary or local build)
VG_CMD="./dist/vectra-guard-darwin-arm64" ./scripts/test-similar-agent-repos.sh
# Or: vg on PATH, or VG_CMD="go run ." from repo root
./scripts/test-similar-agent-repos.sh
```

**One-off commands** (use release `vg` or `go run .`):

```bash
# From repo root; VG=vg or VG="go run ."
VG="${VG:-vg}"
$VG scan-security --path test-workspaces/moltbot --languages go,python,c,config
$VG audit repo --path test-workspaces/moltbot --no-install
```

**Single-repo clone:** `REPO_URL=https://github.com/openinterpreter/open-interpreter REPO_NAME=open-interpreter ./scripts/test-similar-agent-repos.sh clone`

---

## Extensions and Future Work

- **Stronger validate-agent** — Warn when agent workflows can execute shell or broad filesystem access without constraints; flag “disable auth when local” in instructions.
- **Unauthenticated-exec detector** — In code scan, flag HTTP handlers that run shell/exec with no obvious auth check (heuristic).
- **Allowlisting** — Allow per-repo or per-file allowlists for scan-security findings (e.g. documented false positives).

**Related docs:** [FEATURES.md](../FEATURES.md) (overview), [CONFIGURATION.md](../CONFIGURATION.md) (config file), [CVE awareness](cve-awareness.md).
