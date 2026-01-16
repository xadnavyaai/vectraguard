# Vectra Guard Agent Guidance (Claude)

## Use Vectra Guard for execution
- Run shell commands via `vectra-guard exec -- <command>` when changes are involved.
- Avoid destructive commands or anything targeting system paths.

## Config & Sandbox
- Config lives in `vectra-guard.yaml` (or `.vectra-guard/config.yaml` with `--local`).
- Sandbox is enabled by default. Prefer `mode: always` for risky commands.
- Cache-optimized sandbox:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Setup helpers
- `vectra-guard init --local`
- `vectra-guard sandbox deps install`
- `vectra-guard roadmap add --title "..." --summary "..." --tags "agent,plan"`

