# Vectra Guard Windsurf Rules

## Command Execution
- Use `vectra-guard exec -- <command>` for commands that change files or state.
- Never propose destructive commands (`rm -rf /`, `mkfs`, `dd if=`).

## Repo Setup
- Prefer `vectra-guard init --local` for repo-local config and cache.
- Config paths: `vectra-guard.yaml` or `.vectra-guard/config.yaml`.

## Sandboxing
- Default sandboxing is enabled. Use `mode: always` for high-risk operations.
- Cache-optimized sandbox example:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Roadmap
- Capture plans and progress with `vectra-guard roadmap add/list/status/log`.
