package sandbox

import (
	"context"
	"os"
	goruntime "runtime"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox/namespace"
)

func TestRuntimeSelection(t *testing.T) {
	tests := []struct {
		name           string
		config         config.Config
		expectedType   string // "bubblewrap", "namespace", "docker", or "error"
		shouldHaveError bool
	}{
		{
			name: "auto runtime selection",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:       "auto",
					AutoDetectEnv: true,
					PreferFast:    true,
				},
			},
			expectedType:   "", // Depends on system capabilities
			shouldHaveError: false,
		},
		{
			name: "explicit bubblewrap",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:       "bubblewrap",
					AutoDetectEnv: false,
				},
			},
			expectedType:   "bubblewrap",
			shouldHaveError: !namespace.IsBubblewrapAvailable(), // Error if not available
		},
		{
			name: "explicit namespace",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:       "namespace",
					AutoDetectEnv: false,
				},
			},
			expectedType:   "namespace",
			shouldHaveError: !namespace.IsMountNamespaceAvailable(), // Error if not available
		},
		{
			name: "explicit docker",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:       "docker",
					AutoDetectEnv: false,
				},
			},
			expectedType:   "docker",
			shouldHaveError: false, // Docker executor always created (may fail at execution)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger("test", os.Stderr)
			selector := NewRuntimeSelector(tt.config, logger)

			runtime, err := selector.SelectRuntime(context.Background())

			if tt.shouldHaveError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			// On Windows/macOS, if no runtime is available, we may get an error or "none" runtime
			// This is expected behavior
			if err != nil {
				// On Windows/macOS, if Docker isn't available and no other runtime works, this is expected
				if (goruntime.GOOS == "windows" || goruntime.GOOS == "darwin") && (tt.name == "auto runtime selection" || tt.name == "explicit docker") {
					t.Logf("Expected error on %s when no runtime available: %v", goruntime.GOOS, err)
					return
				}
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if runtime == nil {
				t.Fatal("SelectRuntime() returned nil runtime")
			}

			runtimeName := runtime.Name()
			t.Logf("Selected runtime: %s", runtimeName)
			t.Logf("Runtime available: %v", runtime.IsAvailable())

			// On Windows, if no runtime is available, we might get "none" which is acceptable
			if runtimeName == "none" && goruntime.GOOS == "windows" {
				t.Logf("No runtime available on Windows - this is expected")
				return
			}

			// If we specified a specific runtime, verify it matches
			if tt.expectedType != "" && runtimeName != tt.expectedType {
				t.Errorf("Expected runtime %s, got %s", tt.expectedType, runtimeName)
			}
		})
	}
}

func TestRuntimeWithConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
	}{
		{
			name: "with workspace and cache",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:       "auto",
					WorkspaceDir:  "/tmp/test-workspace",
					CacheDir:      "/tmp/test-cache",
					AutoDetectEnv: true,
					PreferFast:    true,
					EnableCache:   true,
				},
			},
		},
		{
			name: "with bind mounts",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:       "auto",
					AutoDetectEnv: true,
					BindMounts: []config.BindMountConfig{
						{
							HostPath:      "/tmp/host",
							ContainerPath: "/tmp/container",
							ReadOnly:      true,
						},
					},
				},
			},
		},
		{
			name: "with network allowed",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:       "auto",
					AutoDetectEnv: true,
					AllowNetwork:  true,
				},
			},
		},
		{
			name: "with strict security",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Runtime:        "auto",
					AutoDetectEnv:  true,
					SeccompProfile: "strict",
					CapabilitySet:  "none",
					UseOverlayFS:   true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger("test", os.Stderr)
			selector := NewRuntimeSelector(tt.config, logger)

			runtime, err := selector.SelectRuntime(context.Background())
			if err != nil {
				t.Logf("SelectRuntime() error (may be expected): %v", err)
				return
			}

			if runtime == nil {
				t.Fatal("SelectRuntime() returned nil runtime")
			}

			t.Logf("Selected runtime: %s", runtime.Name())
			t.Logf("Configuration applied successfully")
		})
	}
}

func TestRuntimeDetection(t *testing.T) {
	// Test that runtime detection doesn't crash
	caps := namespace.DetectCapabilities()
	env := namespace.DetectEnvironment()

	t.Logf("Detected environment: %s", env)
	t.Logf("Detected capabilities:")
	t.Logf("  Bubblewrap:  %v", caps.Bubblewrap)
	t.Logf("  Namespaces:  %v", caps.Namespaces)
	t.Logf("  Docker:      %v", caps.Docker)
	t.Logf("  Seccomp:     %v", caps.Seccomp)
	t.Logf("  OverlayFS:   %v", caps.OverlayFS)

	selectedRuntime := namespace.SelectBestRuntime(env, caps)
	t.Logf("Best runtime for %s: %s", env, selectedRuntime)

	runtimeInfo := namespace.GetRuntimeInfo(selectedRuntime, env)
	t.Logf("Runtime info: %s", runtimeInfo)
}

