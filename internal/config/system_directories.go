package config

// SystemDirectories contains protected system directories across different operating systems
var SystemDirectories = struct {
	// Unix/Linux/Ubuntu system directories (FHS - Filesystem Hierarchy Standard)
	UnixLinux []string

	// macOS specific directories
	MacOS []string

	// Windows system directories
	Windows []string

	// Common across all Unix-like systems
	CommonUnix []string
}{
	// Common Unix/Linux directories (FHS standard)
	UnixLinux: []string{
		"/",           // Root directory
		"/bin",        // Essential system binaries
		"/sbin",       // Essential system admin binaries
		"/usr",        // User programs and data
		"/usr/bin",    // User binaries
		"/usr/sbin",   // User admin binaries
		"/usr/lib",    // User libraries
		"/usr/lib64",  // 64-bit libraries (some distros)
		"/usr/local",  // Local programs
		"/usr/local/bin",
		"/usr/local/sbin",
		"/etc",        // System configuration
		"/var",        // Variable data
		"/var/log",    // Log files
		"/var/lib",    // Variable libraries
		"/var/cache",  // Cache files
		"/lib",        // Essential libraries
		"/lib64",      // 64-bit essential libraries
		"/lib32",      // 32-bit libraries (some distros)
		"/opt",        // Optional software
		"/boot",       // Boot files
		"/root",       // Root user home
		"/sys",        // System files (sysfs)
		"/proc",       // Process files (procfs)
		"/dev",        // Device files
		"/home",       // User home directories
		"/srv",        // Service data
		"/run",        // Runtime data (systemd)
		"/tmp",        // Temporary files (though usually safe to clean)
		"/mnt",        // Mount points
		"/media",      // Removable media mount points
		"/lost+found", // Filesystem recovery
		"/snap",       // Snap packages (Ubuntu)
		"/flatpak",    // Flatpak packages
	},

	// macOS specific directories
	MacOS: []string{
		"/",                    // Root
		"/Applications",        // Applications
		"/Library",             // System-wide libraries
		"/System",              // System files
		"/System/Library",      // System libraries
		"/System/Applications", // System applications
		"/usr",                 // Unix system resources
		"/usr/bin",
		"/usr/sbin",
		"/usr/lib",
		"/usr/libexec",
		"/usr/local",
		"/bin",
		"/sbin",
		"/etc",
		"/var",
		"/private",             // Private system files
		"/private/etc",         // System configuration
		"/private/var",         // Variable data
		"/private/tmp",         // Temporary files
		"/cores",               // Core dumps
		"/dev",
		"/home",
		"/Users",               // User directories (macOS)
		"/Volumes",             // Mounted volumes
		"/Network",             // Network resources
		"/opt",                 // Optional software
		"/tmp",
		"/var/folders",         // macOS user data
	},

	// Windows system directories (when using Unix-style paths in WSL/Git Bash)
	Windows: []string{
		"/",                    // Root (WSL)
		"/mnt/c",               // Windows C: drive (WSL)
		"/mnt/c/Windows",       // Windows system
		"/mnt/c/Windows/System32",
		"/mnt/c/Program Files",
		"/mnt/c/Program Files (x86)",
		"/mnt/c/ProgramData",
		"/mnt/c/Users",
		"/mnt/c/Windows/System",
		"/mnt/c/Windows/SysWOW64",
		// Also protect Windows-style paths that might appear in commands
		"C:\\Windows",
		"C:\\Program Files",
		"C:\\Program Files (x86)",
		"C:\\ProgramData",
		"C:\\Users",
		"C:\\Windows\\System32",
		"C:\\Windows\\System",
		"C:\\Windows\\SysWOW64",
	},

	// Common across all Unix-like systems (Linux, macOS, BSD, etc.)
	CommonUnix: []string{
		"/",      // Root
		"/bin",   // Binaries
		"/sbin",  // System binaries
		"/usr",   // User programs
		"/etc",   // Configuration
		"/var",   // Variable data
		"/lib",   // Libraries
		"/opt",   // Optional
		"/boot",  // Boot
		"/root",  // Root home
		"/sys",   // System
		"/proc",  // Process
		"/dev",   // Devices
		"/home",  // User homes
		"/srv",   // Service data
	},
}

// GetAllProtectedDirectories returns a comprehensive list of protected directories
// for the specified operating system. If os is empty, returns Unix/Linux defaults.
func GetAllProtectedDirectories(os string) []string {
	allDirs := make(map[string]bool)

	// Always include common Unix directories
	for _, dir := range SystemDirectories.CommonUnix {
		allDirs[dir] = true
	}

	// Add OS-specific directories
	switch os {
	case "darwin", "macos", "mac":
		for _, dir := range SystemDirectories.MacOS {
			allDirs[dir] = true
		}
	case "windows", "win":
		for _, dir := range SystemDirectories.Windows {
			allDirs[dir] = true
		}
	default:
		// Default to Unix/Linux
		for _, dir := range SystemDirectories.UnixLinux {
			allDirs[dir] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(allDirs))
	for dir := range allDirs {
		result = append(result, dir)
	}

	return result
}

// GetSystemDirectoryNames returns just the directory names (without leading /)
// for use in regex patterns. This is useful for shell script pattern matching.
func GetSystemDirectoryNames() []string {
	names := []string{
		"bin", "sbin", "usr", "etc", "var", "lib", "lib64", "lib32",
		"opt", "boot", "root", "sys", "proc", "dev", "home", "srv",
		"run", "mnt", "media", "snap", "flatpak", "lost+found",
		// macOS
		"Applications", "Library", "System", "private", "cores",
		"Users", "Volumes", "Network",
		// Windows (WSL)
		"Windows", "Program Files", "ProgramData",
	}
	return names
}
