// +build !linux

package namespace

// CapabilitySet defines which capabilities to keep/drop
type CapabilitySet string

const (
	CapSetNone    CapabilitySet = "none"    // Drop all capabilities
	CapSetMinimal CapabilitySet = "minimal" // Keep only essential caps
	CapSetNormal  CapabilitySet = "normal"  // Keep normal user caps
)

// DropCapabilities is a no-op on non-Linux platforms
func DropCapabilities(set CapabilitySet) error {
	// Capabilities are Linux-specific
	return nil
}

// GetCapabilityInfo returns info about capability support
func GetCapabilityInfo() (string, error) {
	return "capabilities not supported on this platform", nil
}

// EnsureNoNewPrivs is a no-op on non-Linux platforms
func EnsureNoNewPrivs() error {
	return nil
}

