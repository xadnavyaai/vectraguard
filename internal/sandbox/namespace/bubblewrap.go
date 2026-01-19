package namespace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BubblewrapConfig holds configuration for bubblewrap sandbox
type BubblewrapConfig struct {
	Workspace     string
	CacheDir      string
	AllowNetwork  bool
	ReadOnlyPaths []string
	BindMounts    []BindMount
	Environment   map[string]string
}

// BindMount represents a bind mount configuration
type BindMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// BubblewrapExecutor executes commands in bubblewrap sandbox
type BubblewrapExecutor struct {
	config BubblewrapConfig
}

// NewBubblewrapExecutor creates a new bubblewrap executor
func NewBubblewrapExecutor(config BubblewrapConfig) *BubblewrapExecutor {
	return &BubblewrapExecutor{config: config}
}

// Execute runs a command in bubblewrap sandbox
func (e *BubblewrapExecutor) Execute(cmdArgs []string) error {
	// Build bubblewrap command
	bwrapArgs := e.buildBubblewrapArgs(cmdArgs)

	cmd := exec.Command("bwrap", bwrapArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range e.config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	return cmd.Run()
}

// buildBubblewrapArgs constructs the bubblewrap command arguments
func (e *BubblewrapExecutor) buildBubblewrapArgs(cmdArgs []string) []string {
	args := []string{}

	// 1. Bind root filesystem as read-only
	args = append(args, "--ro-bind", "/", "/")

	// 2. Create new /dev with essential devices
	args = append(args, "--dev", "/dev")

	// 3. Create new /proc
	args = append(args, "--proc", "/proc")

	// 4. Create writable /tmp (isolated)
	args = append(args, "--tmpfs", "/tmp")

	// 5. Bind workspace as writable
	if e.config.Workspace != "" {
		absWorkspace, err := filepath.Abs(e.config.Workspace)
		if err == nil {
			args = append(args, "--bind", absWorkspace, absWorkspace)
			// Also bind to /workspace for convenience
			args = append(args, "--bind", absWorkspace, "/workspace")
		}
	}

	// 6. Bind cache directories (preserve between runs)
	cacheBinds := e.getDefaultCacheBinds()
	for _, bind := range append(cacheBinds, e.config.BindMounts...) {
		if _, err := os.Stat(bind.Source); err == nil {
			if bind.ReadOnly {
				args = append(args, "--ro-bind", bind.Source, bind.Target)
			} else {
				args = append(args, "--bind", bind.Source, bind.Target)
			}
		}
	}

	// 7. Unshare all namespaces
	args = append(args, "--unshare-all")

	// 8. Network sharing (optional)
	if e.config.AllowNetwork {
		args = append(args, "--share-net")
	}

	// 9. Die with parent (cleanup on exit)
	args = append(args, "--die-with-parent")

	// 10. New session
	args = append(args, "--new-session")

	// 11. Security hardening
	args = append(args, "--cap-drop", "ALL") // Drop all capabilities

	// 12. Set working directory
	if e.config.Workspace != "" {
		args = append(args, "--chdir", e.config.Workspace)
	}

	// 13. Add the command to execute
	args = append(args, "--")
	args = append(args, cmdArgs...)

	return args
}

// getDefaultCacheBinds returns default cache directories to bind mount
func (e *BubblewrapExecutor) getDefaultCacheBinds() []BindMount {
	home := os.Getenv("HOME")
	if home == "" {
		return []BindMount{}
	}

	binds := []BindMount{}

	// Common cache directories
	cacheDirs := []struct {
		path   string
		target string
	}{
		{filepath.Join(home, ".cache"), filepath.Join(home, ".cache")},
		{filepath.Join(home, ".npm"), filepath.Join(home, ".npm")},
		{filepath.Join(home, ".cargo"), filepath.Join(home, ".cargo")},
		{filepath.Join(home, ".rustup"), filepath.Join(home, ".rustup")},
		{filepath.Join(home, "go"), filepath.Join(home, "go")},
		{filepath.Join(home, ".m2"), filepath.Join(home, ".m2")}, // Maven
		{filepath.Join(home, ".gradle"), filepath.Join(home, ".gradle")},
		{filepath.Join(home, ".pip"), filepath.Join(home, ".pip")},
		{filepath.Join(home, ".local", "share", "virtualenvs"), filepath.Join(home, ".local", "share", "virtualenvs")},
	}

	for _, dir := range cacheDirs {
		if _, err := os.Stat(dir.path); err == nil {
			binds = append(binds, BindMount{
				Source:   dir.path,
				Target:   dir.target,
				ReadOnly: false,
			})
		}
	}

	return binds
}

// IsBubblewrapAvailable checks if bubblewrap is available on the system
func IsBubblewrapAvailable() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}

// GetVersion returns the bubblewrap version
func GetVersion() (string, error) {
	cmd := exec.Command("bwrap", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse version from output (e.g., "bubblewrap 0.8.0")
	version := strings.TrimSpace(string(output))
	return version, nil
}
