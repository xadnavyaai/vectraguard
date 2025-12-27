// +build linux

package namespace

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// MountConfig holds configuration for mount namespace sandbox
type MountConfig struct {
	Workspace      string
	CacheDir       string
	AllowNetwork   bool
	ReadOnlyPaths  []string
	BindMounts     []BindMount
	UseOverlayFS   bool
	SeccompProfile SeccompProfile
	CapabilitySet  CapabilitySet
}

// MountNamespaceExecutor executes commands in a custom mount namespace
type MountNamespaceExecutor struct {
	config MountConfig
}

// NewMountNamespaceExecutor creates a new mount namespace executor
func NewMountNamespaceExecutor(config MountConfig) *MountNamespaceExecutor {
	return &MountNamespaceExecutor{config: config}
}

// Execute runs a command in a mount namespace sandbox
func (e *MountNamespaceExecutor) Execute(cmdArgs []string) error {
	// Fork and setup sandbox in child process
	// The parent will wait for the child to complete
	return e.executeInNamespace(cmdArgs)
}

// executeInNamespace sets up the sandbox and executes the command
func (e *MountNamespaceExecutor) executeInNamespace(cmdArgs []string) error {
	// Create new mount namespace
	if err := unix.Unshare(unix.CLONE_NEWNS); err != nil {
		return fmt.Errorf("failed to create mount namespace: %w", err)
	}

	// Make all mounts private (don't propagate changes)
	if err := unix.Mount("", "/", "", unix.MS_PRIVATE|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to make mounts private: %w", err)
	}

	// Setup filesystem isolation
	if err := e.setupFilesystemIsolation(); err != nil {
		return fmt.Errorf("failed to setup filesystem isolation: %w", err)
	}

	// Drop capabilities
	if err := DropCapabilities(e.config.CapabilitySet); err != nil {
		return fmt.Errorf("failed to drop capabilities: %w", err)
	}

	// Apply seccomp filter
	if err := ApplySeccompFilter(e.config.SeccompProfile); err != nil {
		return fmt.Errorf("failed to apply seccomp filter: %w", err)
	}

	// Ensure NO_NEW_PRIVS
	if err := EnsureNoNewPrivs(); err != nil {
		return fmt.Errorf("failed to set NO_NEW_PRIVS: %w", err)
	}

	// Execute the command
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Find the executable
	execPath, err := findExecutable(cmdArgs[0])
	if err != nil {
		return fmt.Errorf("executable not found: %w", err)
	}

	// Execute the command (replaces current process)
	return syscall.Exec(execPath, cmdArgs, os.Environ())
}

// setupFilesystemIsolation configures the filesystem for sandboxing
func (e *MountNamespaceExecutor) setupFilesystemIsolation() error {
	// Strategy 1: Remount root as read-only
	if err := e.remountRootReadOnly(); err != nil {
		return fmt.Errorf("failed to remount root as read-only: %w", err)
	}

	// Strategy 2: Setup writable /tmp with OverlayFS or tmpfs
	if err := e.setupWritableTmp(); err != nil {
		return fmt.Errorf("failed to setup writable /tmp: %w", err)
	}

	// Strategy 3: Bind mount workspace as writable
	if err := e.bindMountWorkspace(); err != nil {
		return fmt.Errorf("failed to bind mount workspace: %w", err)
	}

	// Strategy 4: Bind mount cache directories
	if err := e.bindMountCaches(); err != nil {
		return fmt.Errorf("failed to bind mount caches: %w", err)
	}

	return nil
}

// remountRootReadOnly remounts the root filesystem as read-only
func (e *MountNamespaceExecutor) remountRootReadOnly() error {
	// Remount root as read-only
	flags := unix.MS_BIND | unix.MS_REMOUNT | unix.MS_RDONLY
	if err := unix.Mount("/", "/", "", uintptr(flags), ""); err != nil {
		return fmt.Errorf("failed to remount / as read-only: %w", err)
	}

	// Also remount critical system directories as read-only
	criticalPaths := []string{
		"/bin", "/sbin", "/usr", "/lib", "/lib64", "/etc",
		"/var", "/opt", "/sys", "/proc", "/boot",
	}

	for _, path := range criticalPaths {
		if _, err := os.Stat(path); err == nil {
			// Bind mount to itself first, then remount as read-only
			unix.Mount(path, path, "", unix.MS_BIND, "")
			unix.Mount(path, path, "", unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY, "")
		}
	}

	return nil
}

// setupWritableTmp creates a writable /tmp using tmpfs or overlayfs
func (e *MountNamespaceExecutor) setupWritableTmp() error {
	if e.config.UseOverlayFS {
		// Use OverlayFS for /tmp
		return e.setupOverlayFS("/tmp")
	}

	// Fallback: use tmpfs
	if err := unix.Mount("tmpfs", "/tmp", "tmpfs", 0, "size=1G,mode=1777"); err != nil {
		return fmt.Errorf("failed to mount tmpfs on /tmp: %w", err)
	}

	return nil
}

// setupOverlayFS sets up OverlayFS for a path
func (e *MountNamespaceExecutor) setupOverlayFS(path string) error {
	// Create overlay directories in cache
	overlayDir := filepath.Join(e.config.CacheDir, "overlay", filepath.Base(path))
	upperDir := filepath.Join(overlayDir, "upper")
	workDir := filepath.Join(overlayDir, "work")

	if err := os.MkdirAll(upperDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return err
	}

	// Mount OverlayFS
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", path, upperDir, workDir)
	if err := unix.Mount("overlay", path, "overlay", 0, opts); err != nil {
		// Fallback to tmpfs if overlay fails
		return unix.Mount("tmpfs", path, "tmpfs", 0, "size=1G")
	}

	return nil
}

// bindMountWorkspace bind mounts the workspace as writable
func (e *MountNamespaceExecutor) bindMountWorkspace() error {
	if e.config.Workspace == "" {
		return nil
	}

	absWorkspace, err := filepath.Abs(e.config.Workspace)
	if err != nil {
		return err
	}

	// Ensure workspace directory exists
	if err := os.MkdirAll(absWorkspace, 0755); err != nil {
		return err
	}

	// Bind mount workspace to itself as writable
	if err := unix.Mount(absWorkspace, absWorkspace, "", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to bind mount workspace: %w", err)
	}

	// Also create a /workspace symlink for convenience
	workspaceLink := "/workspace"
	os.Remove(workspaceLink) // Remove if exists
	if err := os.Symlink(absWorkspace, workspaceLink); err != nil {
		// Non-fatal - just log and continue
	}

	return nil
}

// bindMountCaches bind mounts cache directories for persistence
func (e *MountNamespaceExecutor) bindMountCaches() error {
	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}

	// Default cache directories
	caches := []BindMount{
		{Source: filepath.Join(home, ".cache"), Target: filepath.Join(home, ".cache"), ReadOnly: false},
		{Source: filepath.Join(home, ".npm"), Target: filepath.Join(home, ".npm"), ReadOnly: false},
		{Source: filepath.Join(home, ".cargo"), Target: filepath.Join(home, ".cargo"), ReadOnly: false},
		{Source: filepath.Join(home, ".rustup"), Target: filepath.Join(home, ".rustup"), ReadOnly: false},
		{Source: filepath.Join(home, "go"), Target: filepath.Join(home, "go"), ReadOnly: false},
		{Source: filepath.Join(home, ".m2"), Target: filepath.Join(home, ".m2"), ReadOnly: false},
		{Source: filepath.Join(home, ".gradle"), Target: filepath.Join(home, ".gradle"), ReadOnly: false},
		{Source: filepath.Join(home, ".pip"), Target: filepath.Join(home, ".pip"), ReadOnly: false},
	}

	// Add custom bind mounts
	caches = append(caches, e.config.BindMounts...)

	for _, cache := range caches {
		if _, err := os.Stat(cache.Source); err != nil {
			continue // Skip if doesn't exist
		}

		// Ensure target directory exists
		if err := os.MkdirAll(cache.Target, 0755); err != nil {
			continue // Skip on error
		}

		// Bind mount
		flags := unix.MS_BIND
		if cache.ReadOnly {
			flags |= unix.MS_RDONLY
		}

		if err := unix.Mount(cache.Source, cache.Target, "", uintptr(flags), ""); err != nil {
			// Non-fatal - just skip this cache
			continue
		}
	}

	return nil
}

// findExecutable finds the full path to an executable
func findExecutable(name string) (string, error) {
	// If it's already an absolute path, use it
	if filepath.IsAbs(name) {
		return name, nil
	}

	// Search in PATH
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		execPath := filepath.Join(dir, name)
		if info, err := os.Stat(execPath); err == nil && !info.IsDir() {
			// Check if executable
			if info.Mode()&0111 != 0 {
				return execPath, nil
			}
		}
	}

	return "", fmt.Errorf("executable %q not found in PATH", name)
}

// IsMountNamespaceAvailable checks if mount namespace sandboxing is available
func IsMountNamespaceAvailable() bool {
	// Check for namespace support
	_, err := os.Stat("/proc/self/ns/mnt")
	return err == nil
}

