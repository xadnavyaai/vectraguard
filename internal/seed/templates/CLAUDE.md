# Vectra Guard Agent Guidance (Claude)

## Use Vectra Guard for execution
- Run shell commands via `vectra-guard exec -- <command>` when changes are involved.
- Avoid destructive commands or anything targeting system paths.

## CVE Scanning Workflow
- **Before installing dependencies, scan for vulnerabilities:**
  ```bash
  vg cve sync --path .
  vg cve scan --path .
  vg cve explain <package>@<version> --ecosystem <npm|pypi|go>
  ```
- Enable in config: `cve: { enabled: true, sources: ["osv"] }`
- **Example workflow:**
  ```bash
  # Sync CVE database
  vg cve sync --path .
  
  # Scan manifest files (package.json, requirements.txt, go.mod)
  vg cve scan --path .
  
  # If clean, install safely
  vg exec -- npm install
  ```

## Config & Sandbox
- Config lives in `vectra-guard.yaml` (or `.vectra-guard/config.yaml` with `--local`).
- Sandbox is enabled by default. Prefer `mode: always` for risky commands.
- Cache-optimized sandbox:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Setup helpers
- Local install (no sudo): `INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash`
- Ensure `~/.local/bin` is on `PATH`
- `vectra-guard init --local`
- `vectra-guard sandbox deps install`
- `vectra-guard roadmap add --title "..." --summary "..." --tags "agent,plan"`

## Security Practices (Lightweight)
- Prefer user-space installs and avoid `sudo`.
- Avoid `curl | bash`; download and review scripts first.
- Keep secrets out of logs and outputs.
