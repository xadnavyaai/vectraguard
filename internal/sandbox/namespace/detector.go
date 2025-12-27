package namespace

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// RuntimeType represents the type of sandbox runtime available
type RuntimeType string

const (
	RuntimeBubblewrap RuntimeType = "bubblewrap"
	RuntimeNamespace  RuntimeType = "namespace"
	RuntimeDocker     RuntimeType = "docker"
	RuntimeNone       RuntimeType = "none"
)

// Environment represents the execution environment
type Environment string

const (
	EnvironmentDev  Environment = "dev"
	EnvironmentCI   Environment = "ci"
	EnvironmentProd Environment = "prod"
)

// Capabilities represents the available sandboxing capabilities
type Capabilities struct {
	Bubblewrap       bool
	Namespaces       bool
	Docker           bool
	Seccomp          bool
	OverlayFS        bool
	UserNamespaces   bool
	MountNamespaces  bool
	NetworkNamespaces bool
}

// DetectEnvironment determines if we're running in dev, CI, or production
func DetectEnvironment() Environment {
	// Check for explicit environment variable
	if env := os.Getenv("VECTRAGUARD_ENV"); env != "" {
		switch strings.ToLower(env) {
		case "dev", "development":
			return EnvironmentDev
		case "ci", "continuous-integration":
			return EnvironmentCI
		case "prod", "production":
			return EnvironmentProd
		}
	}

	// Check for CI environment variables
	ciVars := []string{
		"CI",                    // Generic CI
		"CONTINUOUS_INTEGRATION",// Generic CI
		"GITHUB_ACTIONS",        // GitHub Actions
		"GITLAB_CI",             // GitLab CI
		"CIRCLECI",              // CircleCI
		"TRAVIS",                // Travis CI
		"JENKINS_URL",           // Jenkins
		"BUILDKITE",             // Buildkite
		"DRONE",                 // Drone CI
		"BITBUCKET_PIPELINE",    // Bitbucket Pipelines
		"TF_BUILD",              // Azure Pipelines
	}

	for _, varName := range ciVars {
		if os.Getenv(varName) != "" {
			return EnvironmentCI
		}
	}

	// Check if we're in a container (likely CI/prod)
	if isInContainer() {
		return EnvironmentCI
	}

	// Check for .git directory (likely local dev)
	if _, err := os.Stat(".git"); err == nil {
		return EnvironmentDev
	}

	// Default to dev for safety (namespace is safer than Docker for unknown envs)
	return EnvironmentDev
}

// DetectCapabilities checks which sandboxing technologies are available
func DetectCapabilities() Capabilities {
	caps := Capabilities{}

	// Check for bubblewrap
	if _, err := exec.LookPath("bwrap"); err == nil {
		caps.Bubblewrap = true
	}

	// Check for Docker
	if _, err := exec.LookPath("docker"); err == nil {
		// Verify Docker daemon is running
		if cmd := exec.Command("docker", "info"); cmd.Run() == nil {
			caps.Docker = true
		}
	}

	// Check for namespace support (Linux only)
	if runtime.GOOS == "linux" {
		// Check for user namespaces
		if _, err := os.Stat("/proc/self/ns/user"); err == nil {
			caps.UserNamespaces = true
		}

		// Check for mount namespaces
		if _, err := os.Stat("/proc/self/ns/mnt"); err == nil {
			caps.MountNamespaces = true
		}

		// Check for network namespaces
		if _, err := os.Stat("/proc/self/ns/net"); err == nil {
			caps.NetworkNamespaces = true
		}

		// If we have user and mount namespaces, we can use custom namespace sandbox
		if caps.UserNamespaces && caps.MountNamespaces {
			caps.Namespaces = true
		}

		// Check for seccomp support
		if _, err := os.Stat("/proc/sys/kernel/seccomp"); err == nil {
			caps.Seccomp = true
		}

		// Check for OverlayFS support
		if _, err := os.Stat("/sys/module/overlay"); err == nil {
			caps.OverlayFS = true
		}
	}

	return caps
}

// SelectBestRuntime selects the best available runtime for the environment
func SelectBestRuntime(env Environment, caps Capabilities) RuntimeType {
	// For CI/Production, prefer Docker for maximum isolation
	if env == EnvironmentCI || env == EnvironmentProd {
		if caps.Docker {
			return RuntimeDocker
		}
		// Fallback to namespace if Docker not available
		if caps.Bubblewrap {
			return RuntimeBubblewrap
		}
		if caps.Namespaces {
			return RuntimeNamespace
		}
		return RuntimeNone
	}

	// For development, prefer lightweight solutions
	// Priority: bubblewrap > custom namespace > docker
	if caps.Bubblewrap {
		return RuntimeBubblewrap
	}

	if caps.Namespaces {
		return RuntimeNamespace
	}

	if caps.Docker {
		return RuntimeDocker
	}

	return RuntimeNone
}

// isInContainer checks if we're running inside a container
func isInContainer() bool {
	// Check for Docker/.dockerenv
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for container environment variable
	if os.Getenv("container") != "" {
		return true
	}

	// Check cgroup (Docker/Kubernetes)
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") ||
			strings.Contains(content, "kubepods") ||
			strings.Contains(content, "containerd") {
			return true
		}
	}

	return false
}

// GetRuntimeInfo returns human-readable information about the selected runtime
func GetRuntimeInfo(runtime RuntimeType, env Environment) string {
	switch runtime {
	case RuntimeBubblewrap:
		return "Using bubblewrap for fast, secure sandboxing (<1ms overhead)"
	case RuntimeNamespace:
		return "Using Linux namespaces for lightweight sandboxing"
	case RuntimeDocker:
		if env == EnvironmentDev {
			return "Using Docker (consider installing bubblewrap for faster dev experience)"
		}
		return "Using Docker for maximum isolation"
	case RuntimeNone:
		return "WARNING: No sandbox runtime available - commands will run on host"
	default:
		return "Unknown runtime"
	}
}

