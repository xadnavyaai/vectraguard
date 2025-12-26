# Vectra Guard v0.0.3 Release Notes

## 🎉 Major Security Enhancements & Python Command Parsing

This release introduces intelligent Python command parsing, enhanced detection patterns, and a comprehensive test suite with 100% attack blocking success rate.

---

## 🚀 What's New

### 🔍 Smart Python Command Parsing
**Extract and analyze shell commands from Python code**

- ✅ Intelligent extraction from `os.system()`, `subprocess.*()`, `os.popen()`
- ✅ Handles nested quotes and complex Python one-liners
- ✅ Recursive analysis of extracted commands
- ✅ Detects dangerous commands hidden in Python wrappers

**Example:**
```bash
# Now detects: python -c 'import os; os.system("rm -rf /")'
# Extracts: "rm -rf /" → Detected as DANGEROUS_DELETE_ROOT
```

### 🛡️ Enhanced Detection Patterns
**Comprehensive coverage of destructive operations**

- ✅ **Disk Operations**: `wipefs`, `vgremove`, `cryptsetup luksFormat`
- ✅ **Dangerous Permissions**: Recursive `chmod`/`chown` on system paths
- ✅ **Container Operations**: `docker system prune`, `docker rm -f`
- ✅ **Kubernetes**: `kubectl delete --all`, destructive namespace operations
- ✅ **Cloud Storage**: `aws s3 rm --recursive`, `gsutil rm -r`, `az storage blob delete-batch`
- ✅ **Infrastructure**: `terraform destroy`, `pulumi destroy`, `helm uninstall`
- ✅ **Package Removal**: `apt-get remove`, `yum remove`, `dnf remove`
- ✅ **Enhanced System Paths**: Detection for `/lib64`, `/usr/local`, `/home`, `/srv`, etc.

### 🧪 Comprehensive Test Suite
**184 attack vectors, 100% success rate**

- ✅ Extended test suite covering all attack categories
- ✅ Docker-based testing (safe, isolated execution)
- ✅ Local tests in Docker (simulates dev environment)
- ✅ Two-phase testing: detection + execution verification
- ✅ Python reverse shell detection
- ✅ Bypass attempt detection

**Test Coverage:**
- File system destruction (18 attacks)
- Disk operations (6 attacks)
- Process/system attacks (13 attacks)
- Network attacks (10 attacks)
- Privilege escalation (6 attacks)
- Database operations (8 attacks)
- Git operations (3 attacks)
- Command injection (10 attacks)
- Bypass attempts (9 attacks)
- Safe commands verification (6 tests)

### 🔒 Security Improvements
**Mandatory protections for critical commands**

- ✅ **Mandatory Sandboxing**: Critical commands cannot bypass sandbox
- ✅ **Pre-execution Assessment**: Blocks critical commands if sandbox unavailable
- ✅ **Enhanced Reverse Shell Detection**: Detects `/bin/sh -i` in subprocess calls
- ✅ **Improved Risk Assessment**: Better filtering of findings by guard level

### 🐳 Docker Testing Infrastructure
**Safe, isolated testing environment**

- ✅ `test-extended-docker`: Full tests with execution verification
- ✅ `test-extended-local-docker`: Local tests in Docker (detection only)
- ✅ Docker-first approach (all tests isolated)
- ✅ Safety confirmations for local testing

---

## 📊 Test Results

**Extended Test Suite:**
- ✅ **184 attacks blocked** (100% success rate)
- ✅ **0 attacks escaped**
- ✅ **2 tests skipped** (Windows commands on Linux - expected)

**Categories:**
- File System Destruction: ✅ 100%
- Disk Operations: ✅ 100%
- Process/System Attacks: ✅ 100%
- Network Attacks: ✅ 100%
- Privilege Escalation: ✅ 100%
- Database Operations: ✅ 100%
- Git Operations: ✅ 100%
- Command Injection: ✅ 100%
- Bypass Attempts: ✅ 100%

---

## 🔧 Technical Details

### Python Command Parsing
- Extracts commands from `python -c '...'` invocations
- Handles nested quotes (single, double, triple)
- Parses Python arrays/tuples in subprocess calls
- Recursively analyzes extracted commands

### Enhanced Analyzer
- New detection codes: `DISK_WIPE`, `DANGEROUS_PERMISSIONS`, `DESTRUCTIVE_CONTAINER_OP`, `DESTRUCTIVE_K8S_OP`, `DESTRUCTIVE_CLOUD_STORAGE`, `DESTRUCTIVE_INFRA`, `DESTRUCTIVE_PACKAGE_REMOVAL`
- Improved system path detection
- Better reverse shell pattern matching

### Testing Infrastructure
- New Docker service: `test-extended-local` (simulates dev environment)
- Makefile targets: `test-extended-local-docker`, `test-local-docker`
- Safety confirmations for local testing
- Comprehensive test reporting

---

## 🐛 Bug Fixes

- Fixed unbound variable error in test script
- Fixed merge conflicts in test files
- Improved error handling in Python parser
- Enhanced quote handling in command extraction

---

## 📝 Configuration

No breaking changes. Existing configurations continue to work.

**New optional features:**
- Python command parsing (automatic, no config needed)
- Enhanced detection patterns (automatic, no config needed)

---

## 🚀 Upgrade Guide

**No action required** - this is a drop-in upgrade with enhanced security.

**Recommended:**
1. Update to v0.0.3
2. Run `make test-extended-docker` to verify
3. Review new detection patterns if needed

---

## 📦 Installation

```bash
# Download for your platform
# See GitHub Releases: https://github.com/xadnavyaai/vectra-guard/releases

# Verify checksum
shasum -a 256 vectra-guard-<platform> | grep <checksum>

# Make executable
chmod +x vectra-guard-<platform>
```

---

## 🙏 Acknowledgments

Special thanks to the security community for feedback and testing.

---

## 📚 Documentation

- [Security Model](SECURITY_MODEL.md)
- [Advanced Features](ADVANCED_FEATURES.md)
- [Testing Guide](TESTING_EXTENDED.md)
- [Docker Testing](DOCKER_TESTING.md)

---

**Full Changelog**: Compare with [v0.0.2](https://github.com/xadnavyaai/vectra-guard/compare/v0.0.2...v0.0.3)

