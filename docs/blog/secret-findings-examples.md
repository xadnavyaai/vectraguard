# What the 200K+ “Key-Like” Findings Actually Are

When we say **200K+ key-like strings flagged as potential secrets** across the six AI agent repos, we mean: the scanner reported that many *candidates*—strings that *look like* secrets (long, high-entropy, or matching known patterns). Most are **not** real credentials; they’re paths, IDs, config keys, or docs. This doc shows representative examples and how we classify them.

---

## How the scanner works (after FP improvements)

1. **Known-pattern rules** (high confidence): `api_key=...`, `token=...`, `secret=...`, AWS key shapes, `-----BEGIN PRIVATE KEY-----`. These are **true issues** until proven allowlisted or placeholder.
2. **ENTROPY_CANDIDATE** (medium): Only when **the line has secret context** (e.g. contains `token`, `api_key`, `secret`, `password`, `credential`, `auth`). Then we look for 20+ char high-entropy substrings and **filter out** UUIDs, paths (contain `/`), slugs (e.g. `1-eliminating-waterfalls`), URL fragments (`com/`, `org/`, `http`), and code identifiers (snake_case/CamelCase with no digits).

Lockfiles are **skipped by default**. These changes (context requirement + FP filters) dramatically reduce noise from paths, docs, and config keys.

---

## Example 1: Paths and repo/URL fragments (false positive)

The scanner sees long path-like strings and flags them as ENTROPY_CANDIDATE.

| File | Line | Match | Why flagged |
|------|------|--------|-------------|
| `.branchlet.json` | 8 | `autogpt_platform/backend/` | Long path segment, high entropy |
| `.branchlet.json` | 32 | `autogpt_platform/frontend` | Same |
| `.github/ISSUE_TEMPLATE/1.bug.yml` | 18 | `com/Significant-Gravitas/AutoGPT/issues` | URL path |
| `.github/ISSUE_TEMPLATE/1.bug.yml` | 19 | `com/Significant-Gravitas/AutoGPT/wiki/Contributing` | URL path |
| `.claude/skills/.../AGENTS.md` | 191 | `com/shuding/better-all` | GitHub-style path |

**Explanation:** These are directory paths or URL fragments. They’re 20+ chars and use `/` and mixed case, so entropy is high. They are **not** secrets; the scanner can’t tell that without context.

---

## Example 2: Doc slugs and checklist text (false positive)

Docs and markdown often have long hyphenated slugs or section IDs.

| File | Line | Match | Why flagged |
|------|------|--------|-------------|
| `.claude/skills/.../AGENTS.md` | 23 | `1-eliminating-waterfalls` | Long slug, entropy ≥ 3.5 |
| `.claude/skills/.../AGENTS.md` | 38 | `33-parallel-data-fetching-with-component-composition` | Section/slug |
| `.claude/skills/.../AGENTS.md` | 62 | `72-build-index-maps-for-repeated-lookups` | Same |
| `.claude/skills/.../AGENTS.md` | 410 | `com/blog/how-we-optimized-package-imports-in-next-js` | URL path in doc |

**Explanation:** Numbered list items and doc URLs are long and mixed-case. They match the entropy rule but are not credentials.

---

## Example 3: Config keys and identifiers (false positive)

JSON/config keys and code identifiers often look “random” enough to trigger ENTROPY_CANDIDATE.

| File | Line | Match | Why flagged |
|------|------|--------|-------------|
| `.branchlet.json` | 2 | `worktreeCopyPatterns` | CamelCase config key |
| `.branchlet.json` | 36 | `deleteBranchWithWorktree` | Function/config name |
| `.github/dependabot.yml` | 8 | `open-pull-requests-limit` | Config key |
| `.github/ISSUE_TEMPLATE/1.bug.yml` | 17 | `com/channels/1092243196446249134/1092275629602394184` | Discord-style channel ID in template |

**Explanation:** Keys and IDs are fixed strings; the scanner doesn’t know they’re not secrets. Channel IDs and numeric IDs in URLs are especially noisy.

---

## Example 4: Known-pattern findings (true issues — review required)

These match **GENERIC_API_KEY** (or AWS/private-key rules). The *pattern* is “secret-like”; the *value* may be placeholder or real.

| File | Line | Pattern | Match | Context |
|------|------|---------|--------|---------|
| `.github/workflows/platform-frontend-ci.yml` | 125 | GENERIC_API_KEY | Token | `projectToken: chpt_9e7c1a76478c9c8...` |
| `autogpt_platform/.env.default` | 7 | GENERIC_API_KEY | SECRET | `JWT_SECRET=your-super-secret-jwt-token-with-at-least-32-...` |
| `autogpt_platform/.env.default` | 113 | GENERIC_API_KEY | API_KEY | `LOGFLARE_LOGGER_BACKEND_API_KEY=your-super-secret-and-lon...` |
| `autogpt_platform/.../config_test.py` | 17 | GENERIC_API_KEY | secret | Test fixture with `secret=...` |
| `autogpt_platform/.../oauth_test.py` | 921 | GENERIC_API_KEY | token | Test with `token=...` |

**Explanation:** The scanner correctly flags “key name + long value” as a potential secret. Some are **placeholders** (`.env.default`, tests); some (e.g. `projectToken` in CI) need to be confirmed as allowlisted or rotated. These are the ones that **should** be reviewed or allowlisted.

---

## Why counts dropped after improvements

- **Before (one line, many matches):** A single line can contain several substrings that each match the entropy rule (e.g. multiple paths or slugs). Each is one finding.
- **After:** ENTROPY_CANDIDATE only runs when the line has secret context, and we filter UUIDs, paths, slugs, URL fragments, and code identifiers. Counts are much lower and focus on lines that look like credential assignments.

Previously **200K+** meant “this many substrings looked like potential secrets.” A small fraction are **true issues** (known-pattern); the rest are **candidates** that warrant triage (allowlist, ignore path, or fix).

---

## What to do with findings

- **Known-pattern (GENERIC_API_KEY, AWS_*, PRIVATE_KEY):** Review every one. Replace with env/secrets manager or add to allowlist if placeholder/test.
- **ENTROPY_CANDIDATE in paths/docs/config:** Usually false positives. Use `--allowlist` or ignore paths (e.g. `docs/`, `.claude/`) if you want to focus on app code.
- **Run verification:** `go run scripts/verify-secret-findings.go` samples findings per repo and labels TRUE_ISSUE vs FP; use it to sanity-check the scanner.

---

## Security scan (HTTP, bind, etc.)

We also reduced false positives in **security scanning** (`vg scan-security`):

- **Comment-only lines** are skipped for Python (`#`), Go (`//`, `/*`), and config (`#`). So URLs or bind examples in comments no longer produce PY_EXTERNAL_HTTP or BIND_ALL_INTERFACES findings.

For rule reference and control-panel security, see [Control panel & deployment security](../control-panel-security.md).
