// +build !linux

package namespace

import (
	"fmt"
)

// MountConfig holds configuration for mount namespace sandbox (stub for non-Linux)
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

// MountNamespaceExecutor executes commands in a custom mount namespace (stub)
type MountNamespaceExecutor struct {
	config MountConfig
}

// NewMountNamespaceExecutor creates a new mount namespace executor (stub)
func NewMountNamespaceExecutor(config MountConfig) *MountNamespaceExecutor {
	return &MountNamespaceExecutor{config: config}
}

// Execute runs a command (returns error on non-Linux)
func (e *MountNamespaceExecutor) Execute(cmdArgs []string) error {
	return fmt.Errorf("mount namespaces are only supported on Linux")
}

// IsMountNamespaceAvailable checks if mount namespace sandboxing is available
func IsMountNamespaceAvailable() bool {
	return false
}

