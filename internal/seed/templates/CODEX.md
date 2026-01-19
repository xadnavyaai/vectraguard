# Vectra Guard Codex Instructions

## Execution Safety
- Prefer `vectra-guard exec -- <command>` for commands that modify files or system state.
- Never propose destructive commands (`rm -rf /`, `mkfs`, `dd if=`).

## CVE Scanning (Security First)
- **Always check for vulnerabilities before installing:**
  ```bash
  vg cve sync --path .      # Update vulnerability database
  vg cve scan --path .      # Scan project dependencies
  vg exec -- npm install    # Install if clean
  ```
- Enable CVE in config: `cve: { enabled: true, sources: ["osv"] }`
- **Check specific package:**
  ```bash
  vg cve explain lodash@4.17.20 --ecosystem npm
  ```

## Config & Sandbox
- Config file: `vectra-guard.yaml` (or `.vectra-guard/config.yaml` for local mode).
- Sandbox is enabled by default; use `auto` or `always`.
- Cache-optimized sandbox example:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Helpful Commands
- Local install (no sudo): `INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash`
- Ensure `~/.local/bin` is on `PATH`
- `vectra-guard init --local`
- `vectra-guard sandbox deps install`
- `vectra-guard exec -- <command>`
- `vectra-guard roadmap add --title "..." --summary "..." --tags "agent,plan"`
## Security Practices (Lightweight)
- Prefer user-space installs and avoid `sudo`.
- Avoid `curl | bash`; download and review scripts first.
- Keep secrets out of logs and outputs.

