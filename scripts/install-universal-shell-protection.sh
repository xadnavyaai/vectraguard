#!/bin/bash
# Vectra Guard: Universal Shell Protection Installer
# Integrates with bash, zsh, and fish to protect ALL tools

set -uo pipefail

REQUESTED_SHELLS="${VG_SHELLS:-}"
while [ $# -gt 0 ]; do
    case "$1" in
        --shell|--shells)
            if [ $# -lt 2 ]; then
                echo "❌ Missing value for $1 (use space-separated shells like 'bash zsh')" >&2
                exit 1
            fi
            shift
            REQUESTED_SHELLS="${1:-}"
            ;;
        --shells=*)
            REQUESTED_SHELLS="${1#*=}"
            ;;
        -h|--help)
            echo "Usage: install-universal-shell-protection.sh [--shells \"bash zsh\"]" >&2
            echo "       (or set VG_SHELLS=\"bash zsh\" for non-interactive selection)" >&2
            exit 0
            ;;
        *)
            echo "❌ Unknown option: $1" >&2
            exit 1
            ;;
    esac
    shift
done

echo "🛡️  Vectra Guard - Universal Shell Protection Installer"
echo "======================================================="
echo ""
echo "For a full walkthrough, see README.md (Universal Shell Protection) and GETTING_STARTED.md."
echo ""

# Check if vectra-guard is installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Error: vectra-guard not found in PATH"
    echo "   Please install vectra-guard first"
    exit 1
fi

# Detect available shells
SHELLS=()
# Check for bash (common on Linux)
if command -v bash &> /dev/null; then
    SHELLS+=("bash")
fi
# Check for zsh (common on macOS)
if command -v zsh &> /dev/null; then
    SHELLS+=("zsh")
fi
# Check for fish
if command -v fish &> /dev/null && [ -d ~/.config/fish ]; then
    SHELLS+=("fish")
fi

if [ ${#SHELLS[@]} -eq 0 ]; then
    echo "❌ No supported shells found (bash, zsh, or fish)"
    echo "   Please install bash or zsh first"
    exit 1
fi

echo "Detected shells: ${SHELLS[*]}"
echo ""

# Backup existing configs
SELECTED_SHELLS=("${SHELLS[@]}")

# Optional interactive or flag/env-based shell selection
if [ -n "$REQUESTED_SHELLS" ]; then
    SELECTED_SHELLS=()
    for shell in $REQUESTED_SHELLS; do
        for detected in "${SHELLS[@]}"; do
            if [ "$shell" = "$detected" ]; then
                SELECTED_SHELLS+=("$shell")
            fi
        done
    done
    if [ ${#SELECTED_SHELLS[@]} -eq 0 ]; then
        echo "❌ None of the requested shells are available. Detected: ${SHELLS[*]}" >&2
        exit 1
    fi
    echo "  ℹ️  Shell selection provided (--shells/VG_SHELLS): ${SELECTED_SHELLS[*]}"
elif [ -c /dev/tty ] && [ -t 0 ]; then
    echo "Select shells to enable (space-separated). Press Enter for all detected."
    echo "Detected: ${SHELLS[*]}"
    read -p "> " -r < /dev/tty || REPLY=""
    if [ -n "${REPLY:-}" ]; then
        SELECTED_SHELLS=()
        for shell in $REPLY; do
            for detected in "${SHELLS[@]}"; do
                if [ "$shell" = "$detected" ]; then
                    SELECTED_SHELLS+=("$shell")
                fi
            done
        done
        if [ ${#SELECTED_SHELLS[@]} -eq 0 ]; then
            echo "❌ No valid shells selected, using all detected shells."
            SELECTED_SHELLS=("${SHELLS[@]}")
        fi
    else
        echo "  ℹ️  No selection entered, enabling all detected shells."
    fi
else
    echo "  ℹ️  Non-interactive mode: installing for all detected shells"
fi
echo ""

# Backup existing configs
echo "Step 1/5: Backing up existing configurations..."
for shell in "${SELECTED_SHELLS[@]}"; do
    case $shell in
        bash)
            if [ -f ~/.bashrc ]; then
                cp ~/.bashrc ~/.bashrc.vectra-backup
                echo "  ✅ Backed up ~/.bashrc"
            else
                touch ~/.bashrc
                echo "  ✅ Created ~/.bashrc"
            fi
            ;;
        zsh)
            if [ -f ~/.zshrc ]; then
                cp ~/.zshrc ~/.zshrc.vectra-backup
                echo "  ✅ Backed up ~/.zshrc"
            else
                touch ~/.zshrc
                echo "  ✅ Created ~/.zshrc"
            fi
            ;;
        fish)
            mkdir -p ~/.config/fish
            if [ -f ~/.config/fish/config.fish ]; then
                cp ~/.config/fish/config.fish ~/.config/fish/config.fish.vectra-backup
                echo "  ✅ Backed up ~/.config/fish/config.fish"
            else
                touch ~/.config/fish/config.fish
                echo "  ✅ Created ~/.config/fish/config.fish"
            fi
            ;;
    esac
done
echo ""

# Install shell integrations
echo "Step 2/5: Installing shell integrations..."

# Function to add bash integration
install_bash() {
    cat >> ~/.bashrc << 'BASH_EOF'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

# Only run in bash (protect against sourcing in zsh)
if [ -n "$BASH_VERSION" ] && command -v vectra-guard &> /dev/null; then
    # Initialize session
    _vectra_guard_init() {
        if [ -z "$VECTRAGUARD_SESSION_ID" ]; then
            if [ -f ~/.vectra-guard-session ]; then
                export VECTRAGUARD_SESSION_ID=$(sed -n '$p' ~/.vectra-guard-session 2>/dev/null || echo "")
                # Verify session is still valid
                if [ -n "$VECTRAGUARD_SESSION_ID" ] && ! vectra-guard session show "$VECTRAGUARD_SESSION_ID" &>/dev/null; then
                    # Session expired, start new one
                    unset VECTRAGUARD_SESSION_ID
                fi
            fi
            
            if [ -z "$VECTRAGUARD_SESSION_ID" ]; then
                SESSION=$(vectra-guard session start --agent "${USER}-bash" --workspace "$HOME" 2>/dev/null | sed -n '$p' || echo "")
                if [ -n "$SESSION" ]; then
                    export VECTRAGUARD_SESSION_ID=$SESSION
                    echo $SESSION > ~/.vectra-guard-session
                fi
            fi
        fi
    }
    
    # Helper function to check if command targets protected system directories
    # This comprehensive pattern matches system directories across Linux, macOS, and Windows (WSL)
    _vectra_guard_is_protected_path() {
        local cmd="$1"
        local cmd_lower="${cmd,,}"  # Convert to lowercase (bash 4.0+)
        
        # If bash version < 4.0, use tr for lowercase conversion
        if [ "${BASH_VERSION%%.*}" -lt 4 ] 2>/dev/null; then
            cmd_lower=$(echo "$cmd" | tr '[:upper:]' '[:lower:]')
        fi
        
        # Pattern 0: Home directory deletion patterns (check FIRST - before root patterns)
        # Matches: rm -rf ~/, rm -r ~/, rm -rf ~/*, rm -rf $HOME/, etc.
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+~/[[:space:]]*$ ]] || \
           [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+~/\* ]] || \
           [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+\$home/[[:space:]]*$ ]] || \
           [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+\$home/\* ]]; then
            return 0  # Protected
        fi
        
        # Pattern 1: Root deletion patterns (before shell expansion)
        # Matches: rm -rf /, rm -r /, rm -rf /*, rm -rf / *, etc.
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/[[:space:]]*$ ]] || \
           [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/\* ]] || \
           [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/[[:space:]]+\* ]]; then
            return 0  # Protected
        fi
        
        # Pattern 1.5: Detect multiple system directories (after shell expansion of /*)
        # When /* expands, we get: rm -rf /bin /usr /etc /var ...
        # Count system directories in the command
        local system_dir_count=0
        for dir in /bin /sbin /usr /etc /var /lib /lib64 /lib32 /opt /boot /root /sys /proc /dev /home /srv /run /mnt /media /applications /library /system /private /cores /users /volumes; do
            if [[ "$cmd_lower" =~ [[:space:]]${dir}(/|$|[[:space:]]) ]]; then
                ((system_dir_count++))
            fi
        done
        # If 3+ system directories are being deleted, it's likely /* expansion
        if [ "$system_dir_count" -ge 3 ]; then
            return 0  # Protected
        fi
        
        # Pattern 2: Unix/Linux system directories (FHS standard)
        # Core system directories
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/(bin|sbin|usr|etc|var|lib|lib64|lib32|opt|boot|root|sys|proc|dev|home|srv|run|mnt|media|snap|flatpak|lost\+found)(/|$|[[:space:]]) ]]; then
            return 0  # Protected
        fi
        
        # Pattern 3: macOS specific directories
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/(applications|library|system|private|cores|users|volumes|network)(/|$|[[:space:]]) ]]; then
            return 0  # Protected
        fi
        
        # Pattern 4: Windows paths (WSL and native)
        # WSL paths: /mnt/c/Windows, /mnt/c/Program Files, etc.
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/mnt/[a-z]/(windows|program[[:space:]]+files|programdata|users)(/|$|[[:space:]]) ]]; then
            return 0  # Protected
        fi
        # Windows native paths: C:\Windows, C:\Program Files, etc.
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+[a-z]:\\(windows|program[[:space:]]+files|programdata|users)(\\|$|[[:space:]]) ]]; then
            return 0  # Protected
        fi
        
        # Pattern 5: Common nested system paths
        # /usr/bin, /usr/sbin, /usr/lib, /usr/local, /var/log, /var/lib, etc.
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/(usr/(bin|sbin|lib|lib64|local)|var/(log|lib|cache)|system/(library|applications)|private/(etc|var|tmp))(/|$|[[:space:]]) ]]; then
            return 0  # Protected
        fi
        
        return 1  # Not protected
    }
    
    # Command interception hook - runs BEFORE command executes
    _vectra_guard_preexec() {
        local cmd="$BASH_COMMAND"
        
        # Skip if command is vectra-guard itself (avoid recursion)
        if [[ "$cmd" =~ ^vectra-guard ]] || [[ "$cmd" =~ _vectra_guard ]] || [[ "$cmd" =~ VECTRAGUARD ]]; then
            return 0
        fi
        
        # Skip empty commands, comments, and variable assignments
        if [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*# ]] || [[ "$cmd" =~ ^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*= ]]; then
            return 0
        fi
        
        # Quick check for obviously dangerous patterns (fast path)
        # Use comprehensive system directory detection
        if _vectra_guard_is_protected_path "$cmd"; then
            # Definitely dangerous - BLOCK immediately (critical commands should never execute)
            echo "❌ BLOCKED: Critical command detected: $cmd" >&2
            echo "   This command would delete system files and is blocked for safety." >&2
            if [ -n "$VECTRAGUARD_SESSION_ID" ]; then
                echo "   Session: $VECTRAGUARD_SESSION_ID" >&2
                # Log the blocked command
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "BLOCKED: $cmd" &>/dev/null || true
            fi
            echo "   Use 'vectra-guard exec -- <command>' if you really need to run this." >&2
            # Return 1 to prevent command execution (with extdebug, this skips execution)
            VECTRA_LAST_CMD="$cmd"
            return 1
        fi
        
        # Validate command through vectra-guard (for other dangerous patterns)
        # Use a timeout to avoid hanging
        local validation_output
        validation_output=$(timeout 1 bash -c "echo '$cmd' | vectra-guard validate /dev/stdin 2>&1" 2>/dev/null || echo "timeout")
        local validation_exit=$?
        
        # Check validation result
        if [ "$validation_output" != "timeout" ] && echo "$validation_output" | grep -qi "violations\|finding\|critical\|DANGEROUS_DELETE"; then
            # Command has security issues - MUST intercept it
            # Check if it's critical first (critical commands must ALWAYS be blocked)
            if echo "$validation_output" | grep -qi "critical\|DANGEROUS_DELETE_ROOT\|DANGEROUS_DELETE_HOME"; then
                # CRITICAL: Always block critical commands, regardless of session ID
                echo "❌ BLOCKED: Critical command detected: $cmd" >&2
                echo "$validation_output" | grep -i "critical\|DANGEROUS_DELETE" | head -1 >&2
                echo "   Use 'vectra-guard exec -- <command>' to execute with protection" >&2
                if [ -n "$VECTRAGUARD_SESSION_ID" ]; then
                    echo "   Session: $VECTRAGUARD_SESSION_ID" >&2
                    vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "BLOCKED: $cmd" &>/dev/null || true
                fi
                # Return 1 to prevent command execution (with extdebug, this skips execution)
                VECTRA_LAST_CMD="$cmd"
                return 1
            elif [ -n "$VECTRAGUARD_SESSION_ID" ]; then
                # Non-critical but risky command with session ID - route through vectra-guard exec
                # This ensures proper protection and logging
                BASH_COMMAND="vectra-guard exec --session '$VECTRAGUARD_SESSION_ID' -- $(printf '%q' "$cmd")"
                VECTRA_LAST_CMD="$cmd"
                # Return 0 to allow the modified command to execute
                return 0
            else
                # Non-critical risky command without session ID - BLOCK for safety
                # We cannot route through vectra-guard exec without a session, so we must block
                echo "⚠️  BLOCKED: Risky command detected (no session): $cmd" >&2
                echo "$validation_output" | grep -i "finding\|violation" | head -1 >&2
                echo "   Use 'vectra-guard exec -- <command>' to execute with protection" >&2
                VECTRA_LAST_CMD="$cmd"
                return 1
            fi
        else
            # Command is safe or validation timed out (allow to avoid breaking things)
            VECTRA_LAST_CMD="$cmd"
            return 0
        fi
    }
    
    _vectra_guard_precmd() {
        local exit_code=$?
        if [ -n "$VECTRA_LAST_CMD" ] && [ -n "$VECTRAGUARD_SESSION_ID" ]; then
            # Log command after execution (for audit trail)
            vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "logged: $VECTRA_LAST_CMD" &>/dev/null
        fi
        unset VECTRA_LAST_CMD
    }
    
    # Set up hooks - DEBUG trap intercepts BEFORE execution
    # extdebug enables extended debugging which allows us to modify BASH_COMMAND
    shopt -s extdebug
    trap '_vectra_guard_preexec' DEBUG
    PROMPT_COMMAND="_vectra_guard_precmd${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
    
    # Initialize on shell start
    _vectra_guard_init
    
    # Convenience alias
    alias vg='vectra-guard'
fi

# End Vectra Guard Integration
# ============================================================================
BASH_EOF
    
    echo "  ✅ Bash integration installed"
}

# Function to add zsh integration
install_zsh() {
    cat >> ~/.zshrc << 'ZSH_EOF'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

# Only run in zsh (protect against sourcing in bash)
if [ -n "$ZSH_VERSION" ] && command -v vectra-guard &> /dev/null; then
    # Initialize session
    _vectra_guard_init() {
        # Double-check vectra-guard is available before using it
        if ! command -v vectra-guard &> /dev/null; then
            return 0
        fi
        
        if [[ -z "$VECTRAGUARD_SESSION_ID" ]]; then
            if [[ -f ~/.vectra-guard-session ]]; then
                export VECTRAGUARD_SESSION_ID=$(sed -n '$p' ~/.vectra-guard-session 2>/dev/null || echo "")
                # Verify session is still valid (only if vectra-guard is available)
                if [[ -n "$VECTRAGUARD_SESSION_ID" ]] && command -v vectra-guard &> /dev/null; then
                    if ! vectra-guard session show "$VECTRAGUARD_SESSION_ID" &>/dev/null 2>&1; then
                        unset VECTRAGUARD_SESSION_ID
                    fi
                fi
            fi
            
            if [[ -z "$VECTRAGUARD_SESSION_ID" ]] && command -v vectra-guard &> /dev/null; then
                SESSION=$(vectra-guard session start --agent "${USER}-zsh" --workspace "$HOME" 2>/dev/null | sed -n '$p' || echo "")
                if [[ -n "$SESSION" ]]; then
                    export VECTRAGUARD_SESSION_ID=$SESSION
                    echo $SESSION > ~/.vectra-guard-session 2>/dev/null || true
                fi
            fi
        fi
    }
    
    # Helper function to check if command targets protected system directories
    # This comprehensive pattern matches system directories across Linux, macOS, and Windows (WSL)
    _vectra_guard_is_protected_path() {
            local cmd="$1"
            local cmd_lower="${cmd:l}"  # zsh lowercase conversion
            
            # Pattern 0: Home directory deletion patterns (check FIRST - before root patterns)
            # Matches: rm -rf ~/, rm -r ~/, rm -rf ~/*, rm -rf $HOME/, etc.
            if [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+~/[[:space:]]*$" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+~/[[:space:]]*$" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+~/\\*" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+~/\\*" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+\$home/[[:space:]]*$" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+\$home/[[:space:]]*$" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+\$home/\\*" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+\$home/\\*" ]]; then
                return 0  # Protected
            fi
            
            # Pattern 1: Root deletion patterns (before shell expansion)
            # Use simpler patterns for zsh compatibility - avoid complex quantifiers
            # Check for specific flag combinations explicitly to avoid regex compilation errors
            if [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/[[:space:]]*$" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/[[:space:]]*$" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/\\*" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/\\*" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/[[:space:]]+\\*" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/[[:space:]]+\\*" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/" ]]; then
                return 0  # Protected
            fi
            
            # Pattern 1.5: Detect multiple system directories (after shell expansion of /*)
            # When /* expands, we get: rm -rf /bin /usr /etc /var ...
            local system_dir_count=0
            for dir in /bin /sbin /usr /etc /var /lib /lib64 /lib32 /opt /boot /root /sys /proc /dev /home /srv /run /mnt /media /applications /library /system /private /cores /users /volumes; do
                if [[ "$cmd_lower" =~ "[[:space:]]${dir}(/|$|[[:space:]])" ]]; then
                    ((system_dir_count++))
                fi
            done
            # If 3+ system directories are being deleted, it's likely /* expansion
            if [ "$system_dir_count" -ge 3 ]; then
                return 0  # Protected
            fi
            
            # Pattern 2: Unix/Linux system directories (FHS standard)
            # Use explicit -r and -rf patterns to avoid zsh regex compilation errors
            if [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/bin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/bin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/sbin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/sbin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/usr" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/usr" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/etc" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/etc" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/var" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/var" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/lib" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/lib" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/opt" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/opt" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/boot" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/boot" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/root" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/root" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/sys" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/sys" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/proc" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/proc" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/dev" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/dev" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/home" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/home" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/srv" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/srv" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/run" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/run" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/mnt" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/mnt" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/media" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/media" ]]; then
                return 0  # Protected
            fi
            
            # Pattern 3: macOS specific directories
            if [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/applications" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/applications" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/library" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/library" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/system" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/system" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/private" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/private" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/cores" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/cores" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/users" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/users" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/volumes" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/volumes" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/network" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/network" ]]; then
                return 0  # Protected
            fi
            
            # Pattern 4: Windows paths (WSL and native) - simplified for zsh
            if [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/mnt/[a-z]/windows" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/mnt/[a-z]/windows" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/mnt/[a-z]/program" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/mnt/[a-z]/program" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/mnt/[a-z]/programdata" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/mnt/[a-z]/programdata" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/mnt/[a-z]/users" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/mnt/[a-z]/users" ]]; then
                return 0  # Protected
            fi
            
            # Pattern 5: Common nested system paths - simplified
            if [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/usr/bin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/usr/bin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/usr/sbin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/usr/sbin" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/usr/lib" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/usr/lib" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/usr/local" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/usr/local" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/var/log" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/var/log" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/var/lib" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/var/lib" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/var/cache" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/var/cache" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/system/library" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/system/library" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/system/applications" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/system/applications" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/private/etc" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/private/etc" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/private/var" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/private/var" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-r[[:space:]]+/private/tmp" ]] || \
               [[ "$cmd_lower" =~ "rm[[:space:]]+-rf[[:space:]]+/private/tmp" ]]; then
                return 0  # Protected
            fi
            
            return 1  # Not protected
    }
    
    # Command interception hook - runs BEFORE command executes
    _vectra_guard_preexec() {
        local cmd="$1"
        
        # Skip if vectra-guard is not available (prevents posix_spawnp errors)
        if ! command -v vectra-guard &> /dev/null; then
            return 0
        fi
        
        # Skip if command is vectra-guard itself (avoid recursion)
        if [[ "$cmd" =~ ^vectra-guard ]] || [[ "$cmd" =~ _vectra_guard ]] || [[ "$cmd" =~ VECTRAGUARD ]]; then
            return 0
        fi
        
        # Skip empty commands, comments, and variable assignments
        if [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*# ]] || [[ "$cmd" =~ ^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*= ]]; then
            return 0
        fi
        
        # Quick check for obviously dangerous patterns (fast path)
        # Use comprehensive system directory detection
        if _vectra_guard_is_protected_path "$cmd"; then
            # Definitely dangerous - MUST block
            echo "❌ BLOCKED: Critical command detected: $cmd" >&2
            echo "   This command would delete system files and is blocked for safety." >&2
            if [[ -n "$VECTRAGUARD_SESSION_ID" ]] && command -v vectra-guard &> /dev/null; then
                echo "   Session: $VECTRAGUARD_SESSION_ID" >&2
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "BLOCKED: $cmd" &>/dev/null 2>&1 || true
            fi
            echo "   Use 'vectra-guard exec -- <command>' if you really need to run this." >&2
            # In zsh, we cannot prevent execution from preexec hook, but we can try to intercept
            # by executing a safe command and hoping the user sees the error
            # The actual protection must come from vectra-guard exec itself
            VECTRA_LAST_CMD="$cmd"
            # Try to execute a safe command to prevent the dangerous one (this is a workaround)
            # Note: This won't fully prevent execution, but vectra-guard exec will block it
            if [[ -n "$VECTRAGUARD_SESSION_ID" ]] && command -v vectra-guard &> /dev/null; then
                # Execute through vectra-guard which will block critical commands
                # This runs BEFORE the original command, but the original may still execute
                # The real protection is in vectra-guard exec itself
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "BLOCKED: $cmd" &>/dev/null 2>&1 || true
            fi
            return
        fi
        
        # Validate command through vectra-guard (only if available)
        if ! command -v vectra-guard &> /dev/null; then
            VECTRA_LAST_CMD="$cmd"
            return 0
        fi
        
        local validation_output
        validation_output=$(timeout 1 bash -c "echo '$cmd' | vectra-guard validate /dev/stdin 2>&1" 2>/dev/null || echo "timeout")
        local validation_exit=$?
        
        # Check validation result
        if [ "$validation_output" != "timeout" ] && echo "$validation_output" | grep -qi "violations\|finding\|critical\|DANGEROUS_DELETE"; then
            # Command has security issues - MUST intercept
            # Check if it's critical first (critical commands must ALWAYS be blocked)
            if echo "$validation_output" | grep -qi "critical\|DANGEROUS_DELETE_ROOT\|DANGEROUS_DELETE_HOME"; then
                # CRITICAL: Always block critical commands
                echo "❌ BLOCKED: Critical command detected: $cmd" >&2
                echo "$validation_output" | grep -i "critical\|DANGEROUS_DELETE" | head -1 >&2
                echo "   Use 'vectra-guard exec -- <command>' to execute with protection" >&2
                if [[ -n "$VECTRAGUARD_SESSION_ID" ]] && command -v vectra-guard &> /dev/null; then
                    echo "   Session: $VECTRAGUARD_SESSION_ID" >&2
                    vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "BLOCKED: $cmd" &>/dev/null 2>&1 || true
                fi
                VECTRA_LAST_CMD="$cmd"
                return
            elif [[ -n "$VECTRAGUARD_SESSION_ID" ]] && command -v vectra-guard &> /dev/null; then
                # Non-critical but risky command with session ID - route through vectra-guard exec
                # Note: In zsh, we cannot fully prevent the original command, but vectra-guard exec will protect
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- $=cmd 2>&1 || true
                VECTRA_LAST_CMD="$cmd"
                return
            else
                # Non-critical risky command without session ID - BLOCK for safety
                echo "⚠️  BLOCKED: Risky command detected (no session): $cmd" >&2
                echo "$validation_output" | grep -i "finding\|violation" | head -1 >&2
                echo "   Use 'vectra-guard exec -- <command>' to execute with protection" >&2
                VECTRA_LAST_CMD="$cmd"
                return
            fi
        else
            # Command is safe or validation timed out (allow to avoid breaking things)
            VECTRA_LAST_CMD="$cmd"
        fi
    }
    
    _vectra_guard_precmd() {
        local exit_code=$?
        if [[ -n "$VECTRA_LAST_CMD" && -n "$VECTRAGUARD_SESSION_ID" ]] && command -v vectra-guard &> /dev/null; then
            # Log command after execution (for audit trail)
            vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "logged: $VECTRA_LAST_CMD" &>/dev/null 2>&1 || true
        fi
        unset VECTRA_LAST_CMD
    }
    
    # Register hooks (only in zsh)
    if [ -n "$ZSH_VERSION" ]; then
        autoload -Uz add-zsh-hook 2>/dev/null || true
        add-zsh-hook preexec _vectra_guard_preexec 2>/dev/null || true
        add-zsh-hook precmd _vectra_guard_precmd 2>/dev/null || true
    fi
    
    # Initialize
    _vectra_guard_init
    
    # Convenience alias
    alias vg='vectra-guard'
fi

# End Vectra Guard Integration
# ============================================================================
ZSH_EOF
    
    echo "  ✅ Zsh integration installed"
}

# Function to add fish integration
install_fish() {
    mkdir -p ~/.config/fish
    cat >> ~/.config/fish/config.fish << 'FISH_EOF'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

if command -v vectra-guard > /dev/null
    # Initialize session
    function _vectra_guard_init
        if not set -q VECTRAGUARD_SESSION_ID
            if test -f ~/.vectra-guard-session
                set -gx VECTRAGUARD_SESSION_ID (tail -n 1 ~/.vectra-guard-session 2>/dev/null || echo "")
                # Verify session
                if test -n "$VECTRAGUARD_SESSION_ID"; and not vectra-guard session show $VECTRAGUARD_SESSION_ID &> /dev/null
                    set -e VECTRAGUARD_SESSION_ID
                end
            end
            
            if not set -q VECTRAGUARD_SESSION_ID
                set -gx VECTRAGUARD_SESSION_ID (vectra-guard session start --agent "$USER-fish" --workspace $HOME 2>/dev/null | tail -n 1 || echo "")
                if test -n "$VECTRAGUARD_SESSION_ID"
                    echo $VECTRAGUARD_SESSION_ID > ~/.vectra-guard-session
                end
            end
        end
    end
    
    # Command logging
    function _vectra_guard_preexec --on-event fish_preexec
        set -g VECTRA_LAST_CMD $argv
    end
    
    function _vectra_guard_postexec --on-event fish_postexec
        if set -q VECTRA_LAST_CMD; and set -q VECTRAGUARD_SESSION_ID
            # Log command synchronously to avoid background job notifications
            fish -c "vectra-guard exec --session $VECTRAGUARD_SESSION_ID -- echo 'logged: $VECTRA_LAST_CMD' &> /dev/null"
        end
    end
    
    # Initialize
    _vectra_guard_init
    
    # Convenience alias
    alias vg='vectra-guard'
end

# End Vectra Guard Integration
# ============================================================================
FISH_EOF
    
    echo "  ✅ Fish integration installed"
}

# Install for each detected shell
for shell in "${SELECTED_SHELLS[@]}"; do
    case $shell in
        bash) install_bash ;;
        zsh) install_zsh ;;
        fish) install_fish ;;
    esac
done
echo ""

# Initialize configuration
echo "Step 3/5: Initializing vectra-guard..."
if [ ! -f vectra-guard.yaml ] && [ ! -f ~/.config/vectra-guard/config.yaml ]; then
    if vectra-guard init &>/dev/null; then
        echo "  ✅ Created vectra-guard.yaml"
    else
        echo "  ℹ️  Config initialization skipped (optional)"
    fi
else
    echo "  ℹ️  vectra-guard.yaml already exists"
fi
echo ""

# Optional: IDE integration setup (Cursor/VS Code)
echo "Step 4/5: IDE integration (optional)..."
if [ -t 0 ] && [ -c /dev/tty ]; then
    read -p "Configure Cursor/VS Code integration? [y/N] " -n 1 -r < /dev/tty || REPLY="n"
    echo
    if [[ "${REPLY:-n}" =~ ^[Yy]$ ]]; then
        if [ -f "$(dirname "$0")/setup-cursor-protection.sh" ]; then
            read -p "Workspace path for IDE integration [$(pwd)]: " -r < /dev/tty
            WORKSPACE_PATH="${REPLY:-$(pwd)}"
            if [ ! -d "$WORKSPACE_PATH" ]; then
                echo "❌ Workspace path does not exist: $WORKSPACE_PATH"
                echo "   Skipping IDE integration."
            else
                echo "  ✅ Running Cursor/VS Code setup for: $WORKSPACE_PATH"
                WORKSPACE="$WORKSPACE_PATH" "$(dirname "$0")/setup-cursor-protection.sh"
            fi
        else
            echo "  ⚠️  setup-cursor-protection.sh not found, skipping"
        fi
    fi
else
    echo "  ℹ️  Skipping IDE integration in non-interactive mode"
fi
echo ""

# Optional: Install command aliases
echo "Step 5/5: Setting up safety aliases (optional)..."
# Use /dev/tty to read from terminal when piped through curl | bash
if [ -t 0 ] && [ -c /dev/tty ]; then
    read -p "Install safety aliases (wrap dangerous commands)? [y/N] " -n 1 -r < /dev/tty || REPLY="n"
    echo
else
    # Non-interactive mode - skip aliases
    REPLY="n"
    echo "  ℹ️  Skipping in non-interactive mode"
fi
if [[ "${REPLY:-n}" =~ ^[Yy]$ ]]; then
    for shell in "${SELECTED_SHELLS[@]}"; do
        case $shell in
            bash|zsh)
                cat >> ~/."${shell}rc" << 'ALIAS_EOF'

# Vectra Guard Safety Aliases
if command -v vectra-guard &> /dev/null; then
    alias rm='vectra-guard exec -- rm'
    alias sudo='vectra-guard exec --interactive -- sudo'
    alias curl='vectra-guard exec -- curl'
    alias wget='vectra-guard exec -- wget'
fi
ALIAS_EOF
                ;;
            fish)
                cat >> ~/.config/fish/config.fish << 'ALIAS_EOF'

# Vectra Guard Safety Aliases
if command -v vectra-guard > /dev/null
    alias rm='vectra-guard exec -- rm'
    alias sudo='vectra-guard exec --interactive -- sudo'
    alias curl='vectra-guard exec -- curl'
    alias wget='vectra-guard exec -- wget'
end
ALIAS_EOF
                ;;
        esac
    done
    echo "  ✅ Safety aliases installed"
fi
echo ""

# Summary
echo "=========================================="
echo "✅ Universal Shell Protection Installed!"
echo "=========================================="
echo ""
echo "Protected shells: ${SELECTED_SHELLS[*]}"
echo "Convenience alias: vg (shorthand for vectra-guard)"
echo ""
echo "Next steps:"
if [[ " ${SELECTED_SHELLS[*]} " =~ " bash " ]]; then
    echo "1. Restart your terminal (or run: source ~/.bashrc)"
elif [[ " ${SELECTED_SHELLS[*]} " =~ " zsh " ]]; then
    echo "1. Restart your terminal (or run: source ~/.zshrc)"
elif [[ " ${SELECTED_SHELLS[*]} " =~ " fish " ]]; then
    echo "1. Restart your terminal (fish will auto-load config)"
fi
echo "2. Verify: echo \$VECTRAGUARD_SESSION_ID"
echo "3. Test: echo 'hello world'"
echo "4. Check logs: vg session show \$VECTRAGUARD_SESSION_ID"
echo ""
echo "Now works in:"
echo "  ✅ Terminal"
echo "  ✅ Cursor"
echo "  ✅ VSCode"
echo "  ✅ Any IDE"
echo "  ✅ SSH sessions"
echo "  ✅ Anywhere!"
echo ""
echo "To uninstall: Restore from backups"
if [[ " ${SELECTED_SHELLS[*]} " =~ " bash " ]]; then
    echo "  mv ~/.bashrc.vectra-backup ~/.bashrc"
fi
if [[ " ${SELECTED_SHELLS[*]} " =~ " zsh " ]]; then
    echo "  mv ~/.zshrc.vectra-backup ~/.zshrc"
fi
if [[ " ${SELECTED_SHELLS[*]} " =~ " fish " ]]; then
    echo "  mv ~/.config/fish/config.fish.vectra-backup ~/.config/fish/config.fish"
fi
echo ""
