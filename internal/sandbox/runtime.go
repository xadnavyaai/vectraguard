package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox/namespace"
)

// RuntimeExecutor is the interface for different sandbox runtimes
type RuntimeExecutor interface {
	Execute(cmdArgs []string) error
	Name() string
	IsAvailable() bool
}

// RuntimeSelector selects the best available runtime based on configuration
type RuntimeSelector struct {
	config config.Config
	logger *logging.Logger
}

// NewRuntimeSelector creates a new runtime selector
func NewRuntimeSelector(cfg config.Config, logger *logging.Logger) *RuntimeSelector {
	return &RuntimeSelector{
		config: cfg,
		logger: logger,
	}
}

// SelectRuntime selects the best available runtime
func (rs *RuntimeSelector) SelectRuntime(ctx context.Context) (RuntimeExecutor, error) {
	sandboxCfg := rs.config.Sandbox
	runtimeName := sandboxCfg.Runtime

	// Auto-detect environment if enabled
	var env namespace.Environment
	if sandboxCfg.AutoDetectEnv {
		env = namespace.DetectEnvironment()
		rs.logger.Debug("detected environment", map[string]any{"environment": env})
	} else {
		env = namespace.EnvironmentDev // Default to dev if not auto-detecting
	}

	// Detect available capabilities
	caps := namespace.DetectCapabilities()
	rs.logger.Debug("detected capabilities", map[string]any{
		"bubblewrap":        caps.Bubblewrap,
		"namespaces":        caps.Namespaces,
		"docker":            caps.Docker,
		"seccomp":           caps.Seccomp,
		"overlayfs":         caps.OverlayFS,
		"user_namespaces":   caps.UserNamespaces,
		"mount_namespaces":  caps.MountNamespaces,
		"network_namespaces": caps.NetworkNamespaces,
	})

	// Select runtime based on configuration
	var selectedRuntime namespace.RuntimeType
	if runtimeName == "auto" || runtimeName == "" {
		// Auto-select based on environment and capabilities
		if sandboxCfg.PreferFast && env == namespace.EnvironmentDev {
			// Prefer fast runtimes in dev
			selectedRuntime = namespace.SelectBestRuntime(env, caps)
		} else if env == namespace.EnvironmentCI || env == namespace.EnvironmentProd {
			// Prefer Docker in CI/prod for maximum compatibility
			if caps.Docker {
				selectedRuntime = namespace.RuntimeDocker
			} else {
				selectedRuntime = namespace.SelectBestRuntime(env, caps)
			}
		} else {
			selectedRuntime = namespace.SelectBestRuntime(env, caps)
		}
	} else {
		// Use explicitly configured runtime
		selectedRuntime = namespace.RuntimeType(runtimeName)
	}

	// Show runtime info if enabled
	if sandboxCfg.ShowRuntimeInfo {
		rs.logger.Info(namespace.GetRuntimeInfo(selectedRuntime, env), nil)
	}

	// Create runtime executor
	return rs.createExecutor(selectedRuntime, caps)
}

// createExecutor creates the appropriate executor for the runtime
func (rs *RuntimeSelector) createExecutor(runtime namespace.RuntimeType, caps namespace.Capabilities) (RuntimeExecutor, error) {
	sandboxCfg := rs.config.Sandbox
	workspaceDir := sandboxCfg.WorkspaceDir
	if workspaceDir == "" {
		var err error
		workspaceDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
	}

	cacheDir := sandboxCfg.CacheDir
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			cacheDir = filepath.Join(homeDir, ".cache", "vectra-guard")
		} else {
			cacheDir = "/tmp/vectra-guard-cache"
		}
	}

	// Ensure cache directory exists
	os.MkdirAll(cacheDir, 0755)

	// Convert config bind mounts to namespace bind mounts
	bindMounts := []namespace.BindMount{}
	for _, mount := range sandboxCfg.BindMounts {
		bindMounts = append(bindMounts, namespace.BindMount{
			Source:   mount.HostPath,
			Target:   mount.ContainerPath,
			ReadOnly: mount.ReadOnly,
		})
	}

	switch runtime {
	case namespace.RuntimeBubblewrap:
		if !caps.Bubblewrap {
			return nil, fmt.Errorf("bubblewrap not available")
		}

		bwrapConfig := namespace.BubblewrapConfig{
			Workspace:     workspaceDir,
			CacheDir:      cacheDir,
			AllowNetwork:  sandboxCfg.AllowNetwork || sandboxCfg.NetworkMode == "full",
			ReadOnlyPaths: sandboxCfg.ReadOnlyPaths,
			BindMounts:    bindMounts,
			Environment:   make(map[string]string),
		}

		executor := namespace.NewBubblewrapExecutor(bwrapConfig)
		return &bubblewrapRuntimeExecutor{
			executor: executor,
			logger:   rs.logger,
		}, nil

	case namespace.RuntimeNamespace:
		if !caps.Namespaces {
			return nil, fmt.Errorf("namespaces not available")
		}

		// Parse seccomp profile
		seccompProfile := namespace.SeccompProfileModerate
		switch strings.ToLower(sandboxCfg.SeccompProfile) {
		case "strict":
			seccompProfile = namespace.SeccompProfileStrict
		case "moderate":
			seccompProfile = namespace.SeccompProfileModerate
		case "minimal":
			seccompProfile = namespace.SeccompProfileMinimal
		case "none":
			seccompProfile = namespace.SeccompProfileNone
		}

		// Parse capability set
		capabilitySet := namespace.CapSetMinimal
		switch strings.ToLower(sandboxCfg.CapabilitySet) {
		case "none":
			capabilitySet = namespace.CapSetNone
		case "minimal":
			capabilitySet = namespace.CapSetMinimal
		case "normal":
			capabilitySet = namespace.CapSetNormal
		}

		mountConfig := namespace.MountConfig{
			Workspace:      workspaceDir,
			CacheDir:       cacheDir,
			AllowNetwork:   sandboxCfg.AllowNetwork || sandboxCfg.NetworkMode == "full",
			ReadOnlyPaths:  sandboxCfg.ReadOnlyPaths,
			BindMounts:     bindMounts,
			UseOverlayFS:   sandboxCfg.UseOverlayFS && caps.OverlayFS,
			SeccompProfile: seccompProfile,
			CapabilitySet:  capabilitySet,
		}

		executor := namespace.NewMountNamespaceExecutor(mountConfig)
		return &mountNamespaceRuntimeExecutor{
			executor: executor,
			logger:   rs.logger,
		}, nil

	case namespace.RuntimeDocker:
		if !caps.Docker {
			return nil, fmt.Errorf("docker not available")
		}

		// Use existing Docker executor (fallback to legacy implementation)
		return &dockerRuntimeExecutor{
			config: sandboxCfg,
			logger: rs.logger,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

// bubblewrapRuntimeExecutor wraps bubblewrap executor
type bubblewrapRuntimeExecutor struct {
	executor *namespace.BubblewrapExecutor
	logger   *logging.Logger
}

func (e *bubblewrapRuntimeExecutor) Execute(cmdArgs []string) error {
	return e.executor.Execute(cmdArgs)
}

func (e *bubblewrapRuntimeExecutor) Name() string {
	return "bubblewrap"
}

func (e *bubblewrapRuntimeExecutor) IsAvailable() bool {
	return namespace.IsBubblewrapAvailable()
}

// mountNamespaceRuntimeExecutor wraps mount namespace executor
type mountNamespaceRuntimeExecutor struct {
	executor *namespace.MountNamespaceExecutor
	logger   *logging.Logger
}

func (e *mountNamespaceRuntimeExecutor) Execute(cmdArgs []string) error {
	return e.executor.Execute(cmdArgs)
}

func (e *mountNamespaceRuntimeExecutor) Name() string {
	return "namespace"
}

func (e *mountNamespaceRuntimeExecutor) IsAvailable() bool {
	return namespace.IsMountNamespaceAvailable()
}

// dockerRuntimeExecutor wraps Docker executor (legacy)
type dockerRuntimeExecutor struct {
	config config.SandboxConfig
	logger *logging.Logger
}

func (e *dockerRuntimeExecutor) Execute(cmdArgs []string) error {
	// Use existing Docker execution logic
	// This is a placeholder - the actual implementation would use the
	// existing runInContainer function from sandbox.go
	return fmt.Errorf("docker execution not yet migrated - use legacy runInContainer")
}

func (e *dockerRuntimeExecutor) Name() string {
	return "docker"
}

func (e *dockerRuntimeExecutor) IsAvailable() bool {
	caps := namespace.DetectCapabilities()
	return caps.Docker
}

