# Vectra Guard Security Guide

## Overview

This document provides comprehensive security information for Vectra Guard, including the security model, improvements, and testing procedures.

---

## Security Model

### Core Principles

#### 1. **Mandatory Pre-Execution Assessment**

**Every command must be assessed BEFORE execution**, not after. The assessment includes:

- **Pattern Analysis**: Detects dangerous command patterns (not just exact matches)
- **Path Analysis**: Identifies operations targeting root, system directories, or user home
- **Risk Classification**: Categorizes commands as low/medium/high/critical
- **Permission Check**: Determines if command requires sandboxing or approval

**Key Rule**: If assessment fails or is inconclusive, **default to BLOCK**.

#### 2. **True Sandboxing (Cannot Be Bypassed)**

For **critical** commands, sandboxing is **mandatory** and **cannot be bypassed** by:
- Trust store entries
- Allowlist patterns
- Configuration settings
- User bypass mechanisms

**Critical commands that require mandatory sandboxing:**
- `DANGEROUS_DELETE_ROOT` - Any rm command targeting root or system directories
- `DANGEROUS_DELETE_HOME` - Recursive delete of home directory
- `FORK_BOMB` - Commands that can crash the system
- `SENSITIVE_ENV_ACCESS` - Access to credentials/secrets
- `DOTENV_FILE_READ` - Reading .env files

#### 3. **Enhanced Pattern Detection**

The analyzer detects **all variations** of destructive commands:

```bash
# All of these are now detected:
rm -rf /          # Original pattern
rm -r /           # Without force flag
rm -rf /*         # With wildcard
rm -r /*          # Without force, with wildcard
rm -rf / *        # Space between / and *
rm -rf /bin       # System directories
rm -rf /usr       # System directories
```

#### 4. **Defense in Depth**

Multiple layers of protection:
1. **Pattern Detection** - Catches dangerous patterns
2. **Path Analysis** - Identifies risky file operations
3. **Risk Classification** - Assigns severity levels
4. **Mandatory Sandboxing** - Enforces isolation for critical commands
5. **Pre-Execution Block** - Prevents execution if sandbox unavailable

---

## Development vs Production Security

### Development Configuration

**Goals:**
- **Productivity**: Don't block legitimate development work
- **Safety**: Prevent catastrophic mistakes
- **Learning**: Allow developers to understand risks

```yaml
guard_level:
  level: medium  # Blocks critical + high severity
  allow_user_bypass: true
  require_approval_above: medium

sandbox:
  enabled: true
  mode: auto  # Auto-sandbox medium+ risk
  security_level: balanced
```

**Behavior:**
- Low risk → Run on host
- Medium risk → Sandbox automatically
- High risk → Mandatory sandbox + approval
- Critical → BLOCKED or mandatory sandbox (no bypass)

### Production Configuration

**Goals:**
- **Maximum Protection**: Prevent any destructive operations
- **Zero Tolerance**: No bypasses, no exceptions
- **Audit Trail**: Log everything for compliance

```yaml
guard_level:
  level: paranoid  # Everything requires approval
  allow_user_bypass: false  # NO bypasses in production

sandbox:
  enabled: true
  mode: always  # ALWAYS sandbox in production
  security_level: paranoid
  network_mode: none  # No network access
```

**Behavior:**
- ALL commands → Mandatory sandbox + approval
- Critical commands → BLOCKED (cannot execute)
- No network access
- Read-only root filesystem

---

## Security Improvements

### Incident Response

**What Happened:**
- A tool executed `rm -r /*` during development
- The command was **not detected** by the analyzer
- It **bypassed sandboxing** and ran directly on the host
- Result: **Entire OS corrupted**

**Root Causes:**
1. Pattern detection was too narrow (only checked for `rm -rf /`)
2. Sandboxing could be bypassed by trust store or config
3. No mandatory pre-execution assessment
4. Critical commands could fallback to host execution

### Improvements Implemented

#### 1. Enhanced Destructive Command Detection ✅

**Solution:** Comprehensive pattern matching that detects all variations:
- `rm -rf /` (original)
- `rm -r /` (without force)
- `rm -rf /*` (with wildcard)
- `rm -r /*` (without force, with wildcard)
- System directory targets (`/bin`, `/usr`, `/etc`, etc.)
- Home directory wildcards (`~/`, `$HOME/*`)

#### 2. Mandatory Pre-Execution Assessment ✅

**Solution:** Added mandatory assessment layer that:
- Runs **BEFORE** any command execution
- Checks for critical command codes
- Enforces sandbox requirement for critical commands
- Blocks execution if sandbox unavailable (no fallback)

#### 3. Mandatory Sandboxing for Critical Commands ✅

**Solution:** Implemented mandatory sandboxing that:
- **Cannot be bypassed** by trust store
- **Cannot be bypassed** by allowlist
- **Cannot be bypassed** by configuration
- **Cannot be bypassed** by user bypass mechanism

#### 4. Path-Based Risk Assessment ✅

**Solution:** Added path analysis that:
- Detects operations on root (`/`)
- Detects operations on system directories (`/bin`, `/usr`, etc.)
- Detects wildcard usage in dangerous contexts
- Identifies home directory operations

---

## Security Testing

### Running Tests

#### Unit Tests

```bash
# Test enhanced destructive command detection
go test ./internal/analyzer/... -run TestEnhancedDestructiveCommandDetection

# Test mandatory sandboxing
go test ./internal/sandbox/... -run TestMandatorySandboxingForCriticalCommands

# Test pre-execution assessment
go test ./cmd/... -run TestPreExecutionAssessment

# Test regression (incident scenario)
go test ./cmd/... -run TestSecurityImprovementsRegression
```

#### Integration Tests

```bash
# Full test suite
./scripts/test-security.sh

# Quick mode (only critical tests)
./scripts/test-security.sh --quick

# Verbose output
./scripts/test-security.sh --verbose
```

### Test Coverage

#### 1. Enhanced Destructive Command Detection

**Tests:** `TestEnhancedDestructiveCommandDetection`

**What it tests:**
- Detection of `rm -r /*` (the incident scenario)
- Detection of all variations
- System directory targets
- Safe operations should NOT trigger

#### 2. Mandatory Sandboxing

**Tests:** `TestMandatorySandboxingForCriticalCommands`

**What it tests:**
- Critical commands cannot be bypassed by trust store
- Critical commands cannot be bypassed by allowlist
- Critical commands cannot be bypassed by configuration

#### 3. Pre-Execution Assessment

**Tests:** `TestPreExecutionAssessment`

**What it tests:**
- Critical commands are blocked if sandbox unavailable
- No fallback to host execution for critical commands

### Test Scenarios

#### Scenario 1: The Incident (rm -r /*)

**Command:** `rm -r /*`

**Expected Behavior:**
1. ✅ Detected as `DANGEROUS_DELETE_ROOT` (critical)
2. ✅ Pre-execution assessment blocks if sandbox unavailable
3. ✅ Mandatory sandboxing enforced (cannot bypass)
4. ✅ Runs in isolated container

#### Scenario 2: Trust Store Bypass Attempt

**Command:** `rm -r /*` (previously trusted)

**Expected Behavior:**
1. ✅ Detected as critical
2. ✅ Trust store entry ignored
3. ✅ Still requires mandatory sandboxing

#### Scenario 3: Sandbox Disabled

**Command:** `rm -r /*` (sandbox disabled in config)

**Expected Behavior:**
1. ✅ Detected as critical
2. ✅ Pre-execution assessment blocks execution
3. ✅ Error: "Sandbox required for critical commands"

---

## Security Guarantees

### What We Guarantee

1. ✅ **Critical commands** (`rm -r /*`, etc.) are **ALWAYS blocked** or **MANDATORY sandboxed**
2. ✅ **No bypass possible** for critical commands (even with user bypass)
3. ✅ **Sandbox required** for critical commands (no fallback to host)
4. ✅ **Enhanced detection** catches all variations of destructive commands
5. ✅ **Pre-execution assessment** happens before any command runs

### What We Don't Guarantee

1. **100% detection** - New attack patterns may not be caught
2. **Sandbox escape prevention** - Advanced attackers may escape
3. **Performance** - Security comes with overhead
4. **False positive elimination** - Some safe commands may be flagged

---

## Best Practices

### Development

1. ✅ Use `medium` guard level
2. ✅ Enable sandboxing with `auto` mode
3. ✅ Allow user bypass for emergencies
4. ✅ Review findings to learn
5. ✅ Test commands in sandbox first

### Production

1. ✅ Use `paranoid` guard level
2. ✅ Disable all bypasses
3. ✅ Always sandbox (`always` mode)
4. ✅ Enable audit logging
5. ✅ Review logs regularly
6. ✅ Test in staging first

---

## Troubleshooting

### Command Being Blocked Unnecessarily

**Dev:**
- Lower guard level to `low`
- Add to allowlist (with review)
- Use bypass for one-time operations

**Prod:**
- Review why it's blocked
- Consider if it's truly safe
- May need to adjust pattern detection

### Sandbox Not Available

**Dev:**
- Install Docker/Podman
- Check sandbox config
- Fallback to host (for non-critical)

**Prod:**
- **CRITICAL commands cannot execute without sandbox**
- Must fix sandbox before proceeding
- No fallback allowed

---

**Stay Safe. Code Fearlessly.** 🛡️

