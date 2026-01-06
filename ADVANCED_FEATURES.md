# Vectra Guard - Advanced Features

This document describes the advanced features and enhancements added to Vectra Guard for more robust security control.

## Table of Contents

1. [Guard Levels](#guard-levels)
2. [Git Operations Monitoring](#git-operations-monitoring)
3. [Refined SQL Detection](#refined-sql-detection)
4. [Production Environment Detection](#production-environment-detection)
5. [User Bypass Mechanism](#user-bypass-mechanism)
6. [Configuration Examples](#configuration-examples)

---

## Guard Levels

This document focuses on **advanced behaviors and examples**; the canonical reference for guard levels and their semantics lives in **[CONFIGURATION.md](CONFIGURATION.md#guard-level-auto-detection)**.

At a high level:
- `off` → no protection (testing only, not recommended)
- `low` / `medium` / `high` → progressively stricter blocking of risky commands
- `paranoid` → everything requires approval

Use this file for recipes (e.g. combining guard levels with git monitoring, SQL detection, and production detection); use `CONFIGURATION.md` when you need the full option matrix and default values.

---

## Git Operations Monitoring

Vectra Guard now monitors and controls risky Git operations that could damage repositories or cause data loss.

### Monitored Git Operations

| Operation | Risk Level | Description |
|-----------|------------|-------------|
| `git push --force` / `-f` | High → Critical | Can overwrite remote history, elevated to critical in production |
| `git push --force-with-lease` | Medium | Safer alternative to force push |
| `git reset --hard` | Medium | Discards local changes |
| `git clean -fd` | Medium | Deletes untracked files |
| `git branch -D` | Medium | Force deletes branches |
| `git rebase` | Low | Rewrites commit history |
| `git filter-branch` | High | Rewrites entire repository history |
| `git filter-repo` | High | Rewrites entire repository history |
| `git reflog expire` | High | Permanently deletes commit references |
| `git gc --aggressive` | Medium | Aggressive garbage collection |
| `git update-ref -d` | High | Direct ref manipulation |

### Configuration

```yaml
policies:
  monitor_git_ops: true    # Enable git operations monitoring
  block_force_git: true    # Block force push operations
```

### Examples

```bash
# This will be blocked or require approval:
vectra-guard exec git push --force origin main

# Safer alternative suggested:
vectra-guard exec git push --force-with-lease origin main

# Force push to production is CRITICAL:
vectra-guard exec git push -f origin production  # Blocked!
```

---

## Refined SQL Detection

SQL command detection has been refined to **only flag destructive operations** by default, reducing false positives.

### Destructive SQL Operations Detected

- `DROP DATABASE` / `DROP TABLE` / `DROP SCHEMA` / `DROP INDEX`
- `TRUNCATE TABLE` / `TRUNCATE`
- `DELETE FROM` / `DELETE`
- `UPDATE` (data modification)
- `ALTER TABLE` / `ALTER DATABASE` (schema changes)
- `GRANT ALL` / `REVOKE` (permission changes)

### Safe Operations (Not Flagged by Default)

- `SELECT` queries
- `SHOW` commands
- `DESCRIBE` / `EXPLAIN`
- Connection commands
- Read-only operations

### Configuration

```yaml
policies:
  only_destructive_sql: true  # Default: only flag destructive operations
  # Set to false to flag ALL database operations
```

### Examples

```bash
# Safe - will NOT be flagged:
vectra-guard exec mysql -e "SELECT * FROM users"
vectra-guard exec psql -c "SHOW TABLES"

# Destructive - WILL be flagged:
vectra-guard exec mysql -e "DROP TABLE users"      # HIGH severity
vectra-guard exec psql -c "DELETE FROM logs"        # HIGH severity

# Destructive in production - CRITICAL:
vectra-guard exec mysql -h prod-db.com -e "TRUNCATE users"  # CRITICAL!
```

---

## Production Environment Detection

Vectra Guard now intelligently detects when commands target production, staging, or live environments and requires additional approval.

### Detection Patterns

Default patterns (customizable):
- `prod`, `production`, `prd`
- `live`
- `staging`, `stg`

### Detection Contexts

The tool looks for production patterns in:
- **URLs**: `https://api.prod.example.com`
- **Hostnames**: `ssh deploy@prod-server-01`
- **Environment variables**: `export ENV=production`
- **File paths**: `/etc/nginx/sites-available/prod.conf`
- **Configuration files**: `kubectl apply -f production-config.yaml`
- **Database hosts**: `mysql -h prod-db.example.com`
- **Deployment commands**: `ansible-playbook deploy-production.yml`
- **Git branches**: `git push origin production`

### Configuration

```yaml
policies:
  detect_prod_env: true
  prod_env_patterns:
    - prod
    - production
    - prd
    - live
    - staging
    - stg
    - uat  # Add custom patterns
```

### Examples

```bash
# These will trigger production warnings:
vectra-guard exec curl https://api.prod.example.com/deploy
vectra-guard exec kubectl apply -f production-deployment.yaml
vectra-guard exec ssh deploy@prod-server.com
vectra-guard exec mysql -h prod-db-01.example.com

# Combined with destructive operations = CRITICAL:
vectra-guard exec git push --force origin production  # CRITICAL!
vectra-guard exec mysql -h prod-db.com -e "DROP TABLE users"  # CRITICAL!
```

---

## User Bypass Mechanism

A secure bypass mechanism allows real users to bypass protection while preventing AI agents from doing so automatically.

### How It Works

The bypass requires:
1. **Setting an environment variable** with a non-trivial value (≥10 characters)
2. **Value must not contain common agent patterns** like "bypass", "agent", "ai", "automated", etc.
3. **Agents unlikely to guess** the correct format

### Setting Up Bypass

```bash
# Method 1: Simple user-identifiable bypass
export VECTRAGUARD_BYPASS="i-am-human-$(whoami)"

# Method 2: Time-based bypass (more secure)
export VECTRAGUARD_BYPASS="$(date +%s | shasum | head -c 16)"

# Method 3: Custom memorable phrase
export VECTRAGUARD_BYPASS="my-secure-phrase-123"

# Now commands execute without protection:
vectra-guard exec git push --force origin main  # Executes directly
```

### Why Agents Can't Easily Bypass

The bypass detection blocks values containing:
- `bypass`, `agent`, `ai`, `automated`, `script`
- `cursor`, `copilot`, `gpt`, `claude`
- `true`, `yes`, `1`, `enabled`
- Values shorter than 10 characters

### Configuration

```yaml
guard_level:
  allow_user_bypass: true              # Enable bypass mechanism
  bypass_env_var: VECTRAGUARD_BYPASS   # Custom env var name
```

### Disabling Bypass for Maximum Security

```yaml
guard_level:
  allow_user_bypass: false  # No bypass allowed
```

---

## Configuration Examples

### Example 1: Development Environment (Relaxed)

```yaml
guard_level:
  level: low  # Only critical issues
  allow_user_bypass: true

policies:
  monitor_git_ops: true
  block_force_git: false  # Allow force push in dev
  detect_prod_env: false  # No prod detection needed
  only_destructive_sql: true

  allowlist:
    - "echo *"
    - "npm *"
    - "git *"
    - "docker-compose *"

  denylist:
    - "rm -rf /"
    - ":(){ :|:& };:"
```

### Example 2: Production Environment (Strict)

```yaml
guard_level:
  level: high  # Block critical, high, and medium
  allow_user_bypass: false  # No bypass allowed
  require_approval_above: medium

policies:
  monitor_git_ops: true
  block_force_git: true
  detect_prod_env: true
  only_destructive_sql: true
  
  prod_env_patterns:
    - prod
    - production
    - live
    - staging

  allowlist:
    - "git status"
    - "git diff"
    - "git log"
    - "ls *"
    - "cat *"

  denylist:
    - "rm -rf"
    - "sudo *"
    - "dd if="
    - "mkfs"
    - "git push --force"
    - "DROP DATABASE"
    - "TRUNCATE"
```

### Example 3: Paranoid Mode (Maximum Security)

```yaml
guard_level:
  level: paranoid  # Everything requires approval
  allow_user_bypass: false
  require_approval_above: low

policies:
  monitor_git_ops: true
  block_force_git: true
  detect_prod_env: true
  only_destructive_sql: true
  
  prod_env_patterns:
    - prod
    - production
    - prd
    - live
    - staging
    - stg
    - uat
    - preprod

  allowlist: []  # Nothing auto-allowed

  denylist:
    - "rm *"
    - "sudo *"
    - "git push"
    - "DELETE"
    - "DROP"
    - "TRUNCATE"
    - "curl *"
    - "wget *"
```

### Example 4: Balanced Team Environment

```yaml
guard_level:
  level: medium  # Default, good balance
  allow_user_bypass: true
  bypass_env_var: VECTRAGUARD_BYPASS

policies:
  monitor_git_ops: true
  block_force_git: true
  detect_prod_env: true
  only_destructive_sql: true
  
  prod_env_patterns:
    - prod
    - production
    - staging

  allowlist:
    - "echo *"
    - "cat *"
    - "ls *"
    - "npm install"
    - "npm test"
    - "npm run build"
    - "git status"
    - "git diff"
    - "git log"

  denylist:
    - "rm -rf /"
    - "sudo rm"
    - "sudo dd"
    - ":(){ :|:& };:"
    - "curl * | sh"
    - "wget * | bash"
```

---

## Testing Your Configuration

### Test Guard Levels

```bash
# Test different guard levels
vectra-guard exec git push --force origin main  # Should be blocked or require approval

# With bypass
export VECTRAGUARD_BYPASS="i-am-human-$(whoami)"
vectra-guard exec git push --force origin main  # Should execute directly
unset VECTRAGUARD_BYPASS
```

### Test Git Monitoring

```bash
# Should be flagged as risky
vectra-guard exec git push --force origin main
vectra-guard exec git reset --hard HEAD~1
vectra-guard exec git branch -D feature-branch

# Should be critical in production
vectra-guard exec git push -f origin production
```

### Test SQL Detection

```bash
# Should NOT be flagged (safe queries)
vectra-guard exec mysql -e "SELECT * FROM users"
vectra-guard exec psql -c "SHOW TABLES"

# Should be flagged (destructive)
vectra-guard exec mysql -e "DROP TABLE test_data"
vectra-guard exec psql -c "DELETE FROM logs WHERE created_at < '2023-01-01'"

# Should be CRITICAL (destructive + production)
vectra-guard exec mysql -h prod-db.example.com -e "TRUNCATE sessions"
```

### Test Production Detection

```bash
# Should trigger production warnings
vectra-guard exec curl https://api.prod.example.com/deploy
vectra-guard exec kubectl apply -f production-config.yaml
vectra-guard exec ansible-playbook -i production deploy.yml
```

---

## Best Practices

1. **Start with `medium` guard level** - It provides good protection without being too restrictive
2. **Enable git monitoring** - Prevents accidental destructive git operations
3. **Enable production detection** - Critical for preventing accidents in production
4. **Use bypass sparingly** - Only for urgent situations or trusted operations
5. **Review logs regularly** - Check what operations are being flagged
6. **Customize prod patterns** - Add your organization's specific environment names
7. **Test configuration changes** - Always test in dev before applying to production
8. **Use interactive mode** - Add `--interactive` flag for risky operations you need to approve

---

## Troubleshooting

### Commands Being Blocked Unnecessarily

**Solution 1**: Lower guard level
```yaml
guard_level:
  level: low  # Only blocks critical issues
```

**Solution 2**: Add to allowlist
```yaml
policies:
  allowlist:
    - "your-safe-command *"
```

**Solution 3**: Use bypass for one-time operations
```bash
export VECTRAGUARD_BYPASS="i-am-human-$(whoami)"
vectra-guard exec your-command
unset VECTRAGUARD_BYPASS
```

### False Positives for Production Detection

**Solution**: Adjust production patterns to be more specific
```yaml
policies:
  prod_env_patterns:
    - "prod."      # Only match if followed by dot
    - "production"
    - "-prod-"     # Only match if surrounded by dashes
```

### SQL Queries Being Flagged Incorrectly

**Solution**: Ensure `only_destructive_sql` is enabled
```yaml
policies:
  only_destructive_sql: true  # Only flag dangerous operations
```

---

## Migration Guide

### Upgrading from Older Versions

If you have an existing `vectra-guard.yaml`, you can add the new settings:

```yaml
# Add to existing config
guard_level:
  level: medium
  allow_user_bypass: true

policies:
  # ... existing policies ...
  monitor_git_ops: true
  block_force_git: true
  detect_prod_env: true
  only_destructive_sql: true
  prod_env_patterns:
    - prod
    - production
    - staging
```

All new features have sensible defaults, so they work out of the box without configuration.

---

## Security Considerations

1. **Bypass mechanism** - While designed to prevent agent abuse, sophisticated actors could still discover it. For maximum security, disable bypass in production.

2. **Production detection** - Pattern matching can have false positives/negatives. Review patterns and adjust for your environment.

3. **Guard levels** - Higher levels are more secure but may impact productivity. Find the right balance for your team.

4. **Git operations** - Blocking force pushes can prevent emergencies. Ensure team knows how to use bypass for legitimate needs.

5. **SQL detection** - Only detects patterns in command line. Doesn't inspect SQL files or scripts run from within applications.

---

## Getting Help

- **Documentation**: See main [README.md](README.md)
- **Issues**: Report bugs on GitHub
- **Questions**: Check existing issues or open a new one

---

**Stay Safe. Code Fearlessly.** 🛡️
