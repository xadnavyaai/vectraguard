# Vectra Guard VS Code Instructions

## Execution Safety
- Prefer `vectra-guard exec -- <command>` for commands that modify files or system state.
- Avoid destructive commands (`rm -rf /`, `mkfs`, `dd if=`).

## Config & Sandbox
- Config file: `vectra-guard.yaml` (or `.vectra-guard/config.yaml` for local mode).
- Sandbox is enabled by default; use `auto` or `always`.
- Cache-optimized sandbox example:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Helpful Commands
- `vectra-guard init --local`
- `vectra-guard sandbox deps install`
- `vectra-guard exec -- <command>`

