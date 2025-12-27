// +build !linux

package namespace

import (
	"fmt"
)

// SeccompProfile defines a seccomp filter profile
type SeccompProfile string

const (
	SeccompProfileStrict   SeccompProfile = "strict"
	SeccompProfileModerate SeccompProfile = "moderate"
	SeccompProfileMinimal  SeccompProfile = "minimal"
	SeccompProfileNone     SeccompProfile = "none"
)

// ApplySeccompFilter is a no-op on non-Linux platforms
func ApplySeccompFilter(profile SeccompProfile) error {
	// Seccomp is Linux-only
	return nil
}

// GetSeccompInfo returns information about seccomp support
func GetSeccompInfo() (string, error) {
	return "seccomp is only available on Linux", fmt.Errorf("unsupported platform")
}

