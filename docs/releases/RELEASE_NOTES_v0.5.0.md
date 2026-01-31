# Vectra Guard v0.5.0 Release Notes

Release Date: 2026-01-28

## Highlights

This release adds **reports in Vectra Guard tool format**, **improved secret scanning** with fewer false positives, **verification tooling**, and refreshed **documentation and blog** content.

## New Features & Improvements

### Reports and Tool-Format Output
- **Vectra Guard output format** for stored findings: `vg audit repo --path <dir> --output json` schema is documented and used for reports (path, code_findings, code_by_severity, secrets_total, package_audits).
- **similar-agent-findings.json** and **similar-agent-scan-raw.txt** support: scripts and docs for scanning similar AI agent repos and storing results in tool format.
- **docs/reports**: README and cross-validation notes for rule/code counts and secret totals.

### Secret Scanning
- **Improved accuracy**: Skip lockfiles and apply context (token/api_key/secret, etc.) plus FP filters (paths, slugs, UUIDs) so ENTROPY_CANDIDATE is more actionable.
- **Verification script**: `scripts/verify-secret-findings.go` to sample findings per repo, verify line matches, and classify TRUE_ISSUE vs false positive (requires test-workspaces with cloned repos).
- **Documentation**: Clarified feature impact, detection behavior, and usage for secret scanning.

### Documentation & Content
- **Blog**: “Scanning 6 AI Agent Repos for Security Risks: What We Found” with updated secret counts, cross-validation, and verification approach.
- **Docs**: Rerun six-repo scan and refresh reports; enhanced docs for secret scanning and reports.

### Testing & Quality
- **E2E validation**: End-to-end tests for risky scripts and session management.
- **Lint**: Fixes for S1008/S1009 and removal of draft LinkedIn content.

## Installation

See [GETTING_STARTED.md](../GETTING_STARTED.md) or download binaries from the [Releases](https://github.com/xadnavyaai/vectra-guard/releases) page.

## Full Changelog

- feat: reports in vectra-guard tool format, similar-agent scan script and docs
- feat: enhance secret scanning (skip lockfiles, FP filters, verification script)
- docs: blog post with updated secret counts and cross-validation
- docs: clarify feature impact and secret scanning behavior
- test: end-to-end validation for risky scripts and session management
- fix: lint S1008/S1009 and cleanup
