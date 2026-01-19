package namespace

import (
	"os"
	"testing"
)

func TestDetectEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Environment
	}{
		{
			name: "explicit dev environment",
			envVars: map[string]string{
				"VECTRAGUARD_ENV": "dev",
			},
			expected: EnvironmentDev,
		},
		{
			name: "explicit CI environment",
			envVars: map[string]string{
				"VECTRAGUARD_ENV": "ci",
			},
			expected: EnvironmentCI,
		},
		{
			name: "GitHub Actions",
			envVars: map[string]string{
				"GITHUB_ACTIONS": "true",
			},
			expected: EnvironmentCI,
		},
		{
			name: "GitLab CI",
			envVars: map[string]string{
				"GITLAB_CI": "true",
			},
			expected: EnvironmentCI,
		},
		{
			name: "generic CI",
			envVars: map[string]string{
				"CI": "true",
			},
			expected: EnvironmentCI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			for k, v := range tt.envVars {
				oldVal := os.Getenv(k)
				os.Setenv(k, v)
				defer func(key, val string) {
					if val == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, val)
					}
				}(k, oldVal)
			}

			result := DetectEnvironment()
			if result != tt.expected {
				t.Errorf("DetectEnvironment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectCapabilities(t *testing.T) {
	caps := DetectCapabilities()

	// Basic sanity checks
	t.Logf("Detected capabilities:")
	t.Logf("  Bubblewrap:        %v", caps.Bubblewrap)
	t.Logf("  Namespaces:        %v", caps.Namespaces)
	t.Logf("  Docker:            %v", caps.Docker)
	t.Logf("  Seccomp:           %v", caps.Seccomp)
	t.Logf("  OverlayFS:         %v", caps.OverlayFS)
	t.Logf("  UserNamespaces:    %v", caps.UserNamespaces)
	t.Logf("  MountNamespaces:   %v", caps.MountNamespaces)
	t.Logf("  NetworkNamespaces: %v", caps.NetworkNamespaces)

	// If we have user and mount namespaces, we should have general namespace support
	if caps.UserNamespaces && caps.MountNamespaces {
		if !caps.Namespaces {
			t.Errorf("Expected Namespaces=true when UserNamespaces and MountNamespaces are both true")
		}
	}
}

func TestSelectBestRuntime(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		caps     Capabilities
		expected RuntimeType
	}{
		{
			name: "dev with bubblewrap",
			env:  EnvironmentDev,
			caps: Capabilities{
				Bubblewrap: true,
				Namespaces: true,
				Docker:     true,
			},
			expected: RuntimeBubblewrap,
		},
		{
			name: "dev with namespaces only",
			env:  EnvironmentDev,
			caps: Capabilities{
				Bubblewrap: false,
				Namespaces: true,
				Docker:     true,
			},
			expected: RuntimeNamespace,
		},
		{
			name: "dev with docker only",
			env:  EnvironmentDev,
			caps: Capabilities{
				Bubblewrap: false,
				Namespaces: false,
				Docker:     true,
			},
			expected: RuntimeDocker,
		},
		{
			name: "CI with docker",
			env:  EnvironmentCI,
			caps: Capabilities{
				Bubblewrap: true,
				Namespaces: true,
				Docker:     true,
			},
			expected: RuntimeDocker,
		},
		{
			name: "CI without docker",
			env:  EnvironmentCI,
			caps: Capabilities{
				Bubblewrap: true,
				Namespaces: true,
				Docker:     false,
			},
			expected: RuntimeBubblewrap,
		},
		{
			name: "no capabilities",
			env:  EnvironmentDev,
			caps: Capabilities{
				Bubblewrap: false,
				Namespaces: false,
				Docker:     false,
			},
			expected: RuntimeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectBestRuntime(tt.env, tt.caps)
			if result != tt.expected {
				t.Errorf("SelectBestRuntime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRuntimeInfo(t *testing.T) {
	tests := []struct {
		runtime RuntimeType
		env     Environment
	}{
		{RuntimeBubblewrap, EnvironmentDev},
		{RuntimeNamespace, EnvironmentDev},
		{RuntimeDocker, EnvironmentDev},
		{RuntimeDocker, EnvironmentCI},
		{RuntimeNone, EnvironmentDev},
	}

	for _, tt := range tests {
		t.Run(string(tt.runtime)+"_"+string(tt.env), func(t *testing.T) {
			info := GetRuntimeInfo(tt.runtime, tt.env)
			if info == "" {
				t.Errorf("GetRuntimeInfo() returned empty string")
			}
			t.Logf("Runtime info: %s", info)
		})
	}
}

func TestIsInContainer(t *testing.T) {
	// Just test that it doesn't crash
	result := isInContainer()
	t.Logf("isInContainer() = %v", result)
}
