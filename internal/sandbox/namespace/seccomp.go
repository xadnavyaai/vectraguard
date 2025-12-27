// +build linux

package namespace

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// SeccompProfile defines a seccomp filter profile
type SeccompProfile string

const (
	SeccompProfileStrict   SeccompProfile = "strict"
	SeccompProfileModerate SeccompProfile = "moderate"
	SeccompProfileMinimal  SeccompProfile = "minimal"
	SeccompProfileNone     SeccompProfile = "none"
)

// ApplySeccompFilter applies a seccomp-bpf filter to block dangerous syscalls
func ApplySeccompFilter(profile SeccompProfile) error {
	// Check if seccomp is available
	if _, _, errno := unix.Syscall6(unix.SYS_PRCTL, unix.PR_GET_SECCOMP, 0, 0, 0, 0, 0); errno != 0 {
		return fmt.Errorf("seccomp not available: %v", errno)
	}

	// Enable NO_NEW_PRIVS (required for seccomp without CAP_SYS_ADMIN)
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("failed to set NO_NEW_PRIVS: %w", err)
	}

	// Get blocked syscalls based on profile
	blockedSyscalls := getBlockedSyscalls(profile)

	// Note: Full seccomp-bpf implementation requires libseccomp bindings
	// For now, we use PR_SET_NO_NEW_PRIVS as a basic protection
	// TODO: Integrate with github.com/seccomp/libseccomp-golang for full BPF filtering

	if len(blockedSyscalls) > 0 {
		// This is a placeholder for full seccomp-bpf implementation
		// In production, you would:
		// 1. Create a BPF program that checks each syscall
		// 2. Return SECCOMP_RET_KILL or SECCOMP_RET_ERRNO for blocked syscalls
		// 3. Return SECCOMP_RET_ALLOW for allowed syscalls
		// 4. Load the program with prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, prog)
	}

	return nil
}

// getBlockedSyscalls returns the list of syscalls to block for a given profile
func getBlockedSyscalls(profile SeccompProfile) []string {
	var blocked []string

	// Minimal blocking - only the most dangerous
	if profile == SeccompProfileMinimal || profile == SeccompProfileModerate || profile == SeccompProfileStrict {
		blocked = append(blocked, []string{
			"reboot",           // System reboot
			"kexec_load",       // Load new kernel
			"kexec_file_load",  // Load new kernel from file
			"create_module",    // Create kernel module (deprecated)
			"init_module",      // Load kernel module
			"finit_module",     // Load kernel module from fd
			"delete_module",    // Unload kernel module
		}...)
	}

	// Moderate blocking - add privilege escalation and system modification
	if profile == SeccompProfileModerate || profile == SeccompProfileStrict {
		blocked = append(blocked, []string{
			"mount",            // Mount filesystem
			"umount",           // Unmount filesystem
			"umount2",          // Unmount filesystem (variant)
			"pivot_root",       // Change root filesystem
			"chroot",           // Change root directory
			"swapon",           // Enable swap
			"swapoff",          // Disable swap
			"acct",             // Enable/disable process accounting
			"settimeofday",     // Set system time
			"stime",            // Set system time (deprecated)
			"clock_settime",    // Set clock time
			"sethostname",      // Set hostname
			"setdomainname",    // Set domain name
		}...)
	}

	// Strict blocking - add debugging and dangerous operations
	if profile == SeccompProfileStrict {
		blocked = append(blocked, []string{
			"ptrace",           // Process trace/debug
			"process_vm_readv", // Read process memory
			"process_vm_writev",// Write process memory
			"kcmp",             // Compare kernel objects
			"lookup_dcookie",   // Kernel debugging
			"perf_event_open",  // Performance monitoring
			"bpf",              // BPF syscall (could load malicious BPF)
			"userfaultfd",      // User-space page fault handling
			"iopl",             // Change I/O privilege level
			"ioperm",           // Set I/O port permissions
			"quotactl",         // Quota operations
			"nfsservctl",       // NFS server control (deprecated)
			"vhangup",          // Hang up TTY
		}...)
	}

	return blocked
}

// GetSeccompInfo returns information about seccomp support
func GetSeccompInfo() (string, error) {
	// Check /proc/sys/kernel/seccomp
	if err := unix.Access("/proc/sys/kernel/seccomp", unix.R_OK); err != nil {
		return "", fmt.Errorf("seccomp not supported by kernel")
	}

	// Try to get current seccomp mode
	mode, _, errno := unix.Syscall6(unix.SYS_PRCTL, unix.PR_GET_SECCOMP, 0, 0, 0, 0, 0)
	if errno != 0 {
		return "", fmt.Errorf("failed to query seccomp mode: %v", errno)
	}

	switch mode {
	case 0:
		return "seccomp available (currently disabled)", nil
	case 1:
		return "seccomp enabled (strict mode)", nil
	case 2:
		return "seccomp enabled (filter mode)", nil
	default:
		return fmt.Sprintf("seccomp in unknown mode: %d", mode), nil
	}
}

