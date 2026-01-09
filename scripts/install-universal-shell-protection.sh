#!/bin/bash
# Vectra Guard: Universal Shell Protection Installer
# Integrates with bash, zsh, and fish to protect ALL tools

set -uo pipefail

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
echo "Step 1/4: Backing up existing configurations..."
for shell in "${SHELLS[@]}"; do
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
echo "Step 2/4: Installing shell integrations..."

# Function to add bash integration
install_bash() {
    cat >> ~/.bashrc << 'BASH_EOF'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

if command -v vectra-guard &> /dev/null; then
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
        
        # Pattern 1: Root deletion patterns
        # Matches: rm -rf /, rm -r /, rm -rf /*, rm -rf / *, etc.
        if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/[[:space:]]*$ ]] || \
           [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/\* ]] || \
           [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/[[:space:]]+\* ]]; then
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
            # Command has security issues - intercept it
            if [ -n "$VECTRAGUARD_SESSION_ID" ]; then
                # Route through vectra-guard exec which will block if needed
                BASH_COMMAND="vectra-guard exec --session '$VECTRAGUARD_SESSION_ID' -- $(printf '%q' "$cmd")"
            else
                # Check if it's critical
                if echo "$validation_output" | grep -qi "critical\|DANGEROUS_DELETE_ROOT\|DANGEROUS_DELETE_HOME"; then
                    echo "❌ BLOCKED: Critical command detected: $cmd" >&2
                    echo "$validation_output" | grep -i "critical\|DANGEROUS_DELETE" | head -1 >&2
                    echo "   Use 'vectra-guard exec -- <command>' to execute with protection" >&2
                    # Return 1 to prevent command execution (with extdebug, this skips execution)
                    VECTRA_LAST_CMD="$cmd"
                    return 1
                else
                    # Non-critical risky command - allow but will be logged
                    VECTRA_LAST_CMD="$cmd"
                fi
            fi
        else
            # Command is safe or validation timed out (allow to avoid breaking things)
            VECTRA_LAST_CMD="$cmd"
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

if command -v vectra-guard &> /dev/null; then
    # Initialize session
    _vectra_guard_init() {
        if [[ -z "$VECTRAGUARD_SESSION_ID" ]]; then
            if [[ -f ~/.vectra-guard-session ]]; then
                export VECTRAGUARD_SESSION_ID=$(sed -n '$p' ~/.vectra-guard-session 2>/dev/null || echo "")
                # Verify session is still valid
                if [[ -n "$VECTRAGUARD_SESSION_ID" ]] && ! vectra-guard session show "$VECTRAGUARD_SESSION_ID" &>/dev/null; then
                    unset VECTRAGUARD_SESSION_ID
                fi
            fi
            
            if [[ -z "$VECTRAGUARD_SESSION_ID" ]]; then
                SESSION=$(vectra-guard session start --agent "${USER}-zsh" --workspace "$HOME" 2>/dev/null | sed -n '$p' || echo "")
                if [[ -n "$SESSION" ]]; then
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
            local cmd_lower="${cmd:l}"  # zsh lowercase conversion
            
            # Pattern 1: Root deletion patterns
            if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/[[:space:]]*$ ]] || \
               [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/\* ]] || \
               [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/[[:space:]]+\* ]]; then
                return 0  # Protected
            fi
            
            # Pattern 2: Unix/Linux system directories (FHS standard)
            if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/(bin|sbin|usr|etc|var|lib|lib64|lib32|opt|boot|root|sys|proc|dev|home|srv|run|mnt|media|snap|flatpak|lost\+found)(/|$|[[:space:]]) ]]; then
                return 0  # Protected
            fi
            
            # Pattern 3: macOS specific directories
            if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/(applications|library|system|private|cores|users|volumes|network)(/|$|[[:space:]]) ]]; then
                return 0  # Protected
            fi
            
            # Pattern 4: Windows paths (WSL and native)
            if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/mnt/[a-z]/(windows|program[[:space:]]+files|programdata|users)(/|$|[[:space:]]) ]]; then
                return 0  # Protected
            fi
            if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+[a-z]:\\(windows|program[[:space:]]+files|programdata|users)(\\|$|[[:space:]]) ]]; then
                return 0  # Protected
            fi
            
            # Pattern 5: Common nested system paths
            if [[ "$cmd_lower" =~ rm[[:space:]]+-[rf]*[[:space:]]+/(usr/(bin|sbin|lib|lib64|local)|var/(log|lib|cache)|system/(library|applications)|private/(etc|var|tmp))(/|$|[[:space:]]) ]]; then
                return 0  # Protected
            fi
            
            return 1  # Not protected
        }
        
        # Quick check for obviously dangerous patterns (fast path)
        # Use comprehensive system directory detection
        if _vectra_guard_is_protected_path "$cmd"; then
            # Definitely dangerous - intercept
            echo "❌ BLOCKED: Critical command detected: $cmd" >&2
            echo "   This command would delete system files and is blocked for safety." >&2
            if [[ -n "$VECTRAGUARD_SESSION_ID" ]]; then
                echo "   Session: $VECTRAGUARD_SESSION_ID" >&2
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "BLOCKED: $cmd" &>/dev/null || true
            fi
            echo "   Use 'vectra-guard exec -- <command>' if you really need to run this." >&2
            # In zsh, we can't easily prevent execution, but vectra-guard exec will block it
            # Try to prevent by executing a safe command instead
            if [[ -n "$VECTRAGUARD_SESSION_ID" ]]; then
                # Route through vectra-guard exec which will block
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "BLOCKED: $cmd" &>/dev/null || true
            fi
            VECTRA_LAST_CMD="$cmd"
            return
        fi
        
        # Validate command through vectra-guard
        local validation_output
        validation_output=$(echo "$cmd" | vectra-guard validate /dev/stdin 2>&1)
        local validation_exit=$?
        
        if [ $validation_exit -ne 0 ]; then
            # Command has security issues
            if [[ -n "$VECTRAGUARD_SESSION_ID" ]]; then
                # Route through vectra-guard exec
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- $=cmd
            else
                # Check if critical
                if echo "$validation_output" | grep -qi "critical\|DANGEROUS_DELETE_ROOT\|DANGEROUS_DELETE_HOME"; then
                    echo "❌ BLOCKED: Critical command detected: $cmd" >&2
                    echo "$validation_output" | grep -i "critical\|DANGEROUS_DELETE" | head -1 >&2
                fi
            fi
        fi
        VECTRA_LAST_CMD="$cmd"
    }
    
    _vectra_guard_precmd() {
        local exit_code=$?
        if [[ -n "$VECTRA_LAST_CMD" && -n "$VECTRAGUARD_SESSION_ID" ]]; then
            # Log command after execution (for audit trail)
            vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "logged: $VECTRA_LAST_CMD" &>/dev/null
        fi
        unset VECTRA_LAST_CMD
    }
    
    # Register hooks
    autoload -Uz add-zsh-hook
    add-zsh-hook preexec _vectra_guard_preexec
    add-zsh-hook precmd _vectra_guard_precmd
    
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
for shell in "${SHELLS[@]}"; do
    case $shell in
        bash) install_bash ;;
        zsh) install_zsh ;;
        fish) install_fish ;;
    esac
done
echo ""

# Initialize configuration
echo "Step 3/4: Initializing vectra-guard..."
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

# Optional: Install command aliases
echo "Step 4/4: Setting up safety aliases (optional)..."
# Use /dev/tty to read from terminal when piped through curl | bash
if [ -t 0 ] && [ -c /dev/tty ]; then
    read -p "Install safety aliases (wrap dangerous commands)? [y/N] " -n 1 -r < /dev/tty
    echo
else
    # Non-interactive mode - skip aliases
    REPLY="n"
    echo "  ℹ️  Skipping in non-interactive mode"
fi
if [[ $REPLY =~ ^[Yy]$ ]]; then
    for shell in "${SHELLS[@]}"; do
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
echo "Protected shells: ${SHELLS[*]}"
echo "Convenience alias: vg (shorthand for vectra-guard)"
echo ""
echo "Next steps:"
if [[ " ${SHELLS[*]} " =~ " bash " ]]; then
    echo "1. Restart your terminal (or run: source ~/.bashrc)"
elif [[ " ${SHELLS[*]} " =~ " zsh " ]]; then
    echo "1. Restart your terminal (or run: source ~/.zshrc)"
elif [[ " ${SHELLS[*]} " =~ " fish " ]]; then
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
if [[ " ${SHELLS[*]} " =~ " bash " ]]; then
    echo "  mv ~/.bashrc.vectra-backup ~/.bashrc"
fi
if [[ " ${SHELLS[*]} " =~ " zsh " ]]; then
    echo "  mv ~/.zshrc.vectra-backup ~/.zshrc"
fi
if [[ " ${SHELLS[*]} " =~ " fish " ]]; then
    echo "  mv ~/.config/fish/config.fish.vectra-backup ~/.config/fish/config.fish"
fi
echo ""

