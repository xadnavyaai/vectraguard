# Vectra Guard Agent Instructions

Use these guidelines for agentic coding tools and IDE assistants.

## Safe Execution
- Run shell commands via `vectra-guard exec -- <command>` when the command could modify system state.
- Avoid destructive commands (e.g., `rm -rf /`, `rm -rf /bin`, `mkfs`, `dd if=`). These are blocked by design.
- If you need a sandboxed environment, use `vectra-guard sandbox deps install` and keep sandbox enabled.

## Recommended Setup
- Initialize a repo-local config and cache:
  - `vectra-guard init --local`
- Default config file: `vectra-guard.yaml` (or `.vectra-guard/config.yaml` for local mode).

## Sandboxing
- Default behavior is safe-by-default; sandbox can be set to `auto` or `always`.
- Use cache-optimized sandboxing for builds:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Shortcuts
- `vg` is an alias for `vectra-guard` when shell integration is installed.

