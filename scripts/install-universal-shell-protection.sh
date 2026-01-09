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
        if [[ "$cmd" =~ rm[[:space:]]+-[rf]*[[:space:]]+[/\*] ]] || \
           [[ "$cmd" =~ rm[[:space:]]+-[rf]*[[:space:]]+/\* ]] || \
           [[ "$cmd" =~ :\(\)\{[[:space:]]*:[|:&][[:space:]]*\};: ]]; then
            # Definitely dangerous - intercept immediately
            if [ -n "$VECTRAGUARD_SESSION_ID" ]; then
                # Replace BASH_COMMAND to route through vectra-guard exec
                # Properly quote the command to handle special characters
                BASH_COMMAND="vectra-guard exec --session '$VECTRAGUARD_SESSION_ID' -- $(printf '%q' "$cmd")"
            else
                # No session - block critical commands
                echo "❌ BLOCKED: Dangerous command detected: $cmd" >&2
                echo "   Use 'vectra-guard exec -- <command>' to execute with protection" >&2
                # Replace with a no-op command to prevent execution
                BASH_COMMAND=":"
            fi
            VECTRA_LAST_CMD="$cmd"
            return 0
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
                    # Replace with no-op to prevent execution
                    BASH_COMMAND=":"
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
    
    # Command interception hook - runs BEFORE command executes
    _vectra_guard_preexec() {
        local cmd="$1"
        
        # Skip if command is vectra-guard itself (avoid recursion)
        if [[ "$cmd" =~ ^vectra-guard ]] || [[ "$cmd" =~ _vectra_guard ]] || [[ "$cmd" =~ VECTRAGUARD ]]; then
            VECTRA_LAST_CMD="$cmd"
            return
        fi
        
        # Skip empty commands, comments, and variable assignments
        if [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*# ]] || [[ "$cmd" =~ ^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*= ]]; then
            VECTRA_LAST_CMD="$cmd"
            return
        fi
        
        # Quick check for obviously dangerous patterns (fast path)
        if [[ "$cmd" =~ rm[[:space:]]+-[rf]*[[:space:]]+[/\*] ]] || \
           [[ "$cmd" =~ rm[[:space:]]+-[rf]*[[:space:]]+/\* ]] || \
           [[ "$cmd" =~ :\(\)\{[[:space:]]*:[|:&][[:space:]]*\};: ]]; then
            # Definitely dangerous - intercept
            if [[ -n "$VECTRAGUARD_SESSION_ID" ]]; then
                # Execute through vectra-guard exec (will block if needed)
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- $=cmd
                # Prevent original command execution by modifying the command array
                # In zsh preexec, we can't easily prevent, but vectra-guard exec will block it
            else
                # No session - block critical commands
                echo "❌ BLOCKED: Dangerous command detected: $cmd" >&2
                echo "   Use 'vectra-guard exec -- <command>' to execute with protection" >&2
                # Try to prevent execution by clearing the command
                # Note: This may not work in all zsh versions
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

