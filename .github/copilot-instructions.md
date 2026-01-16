# Vectra Guard Copilot Instructions

## Execution Safety
- Prefer `vectra-guard exec -- <command>` for commands that modify files or system state.
- Do not suggest destructive commands (e.g., `rm -rf /`, `mkfs`, `dd if=`). These are blocked and unsafe.

## Configuration
- Default config: `vectra-guard.yaml` at repo root.
- Repo-local config: `.vectra-guard/config.yaml` created via `vectra-guard init --local`.

## Sandboxing
- Default sandboxing is enabled. Use `auto` or `always` modes.
- Cache-optimized sandbox example:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Useful Commands
- `vectra-guard init --local`
- `vectra-guard sandbox deps install`
- `vectra-guard exec -- <command>`

