package analyzer

import (
	"path/filepath"
	"strings"
)

// ValidateProtectedDirectory checks if a command targets a protected directory.
// Returns true if the directory is protected and should be blocked.
func ValidateProtectedDirectory(command string, protectedDirs []string) (bool, string) {
	if len(protectedDirs) == 0 {
		return false, ""
	}

	commandLower := strings.ToLower(command)
	
	// Extract potential paths from the command
	// Look for common destructive operations: rm, mv, cp, chmod, chown, etc.
	destructiveOps := []string{"rm ", "mv ", "cp ", "chmod ", "chown ", "chgrp ", "find ", "tar ", "dd "}
	hasDestructiveOp := false
	for _, op := range destructiveOps {
		// Check for operation followed by space or flag (to avoid false positives)
		if strings.Contains(commandLower, op) {
			// For tar and dd, they need specific patterns to be destructive
			if op == "tar " {
				// tar is only destructive if it's extracting/creating to protected dir
				// or if it has -C flag pointing to protected dir
				if strings.Contains(commandLower, "tar ") {
					hasDestructiveOp = true
				}
			} else if op == "dd " {
				// dd is only destructive if it has of= pointing to protected dir
				// For now, we'll check it, but it's less common
				if strings.Contains(commandLower, "dd ") {
					hasDestructiveOp = true
				}
			} else {
				hasDestructiveOp = true
			}
			if hasDestructiveOp {
				break
			}
		}
	}
	
	// If no destructive operation, it's safe
	if !hasDestructiveOp {
		return false, ""
	}

	// Check each protected directory
	for _, protectedDir := range protectedDirs {
		if protectedDir == "" {
			continue
		}
		
		protectedDirLower := strings.ToLower(strings.TrimSpace(protectedDir))
		
		// Normalize the protected directory path
		protectedDirNormalized := filepath.Clean(protectedDirLower)
		if !strings.HasPrefix(protectedDirNormalized, "/") {
			protectedDirNormalized = "/" + protectedDirNormalized
		}
		
		// Check for exact match or prefix match
		// Pattern: rm -rf /etc or rm -rf /etc/...
		patterns := []string{
			" " + protectedDirNormalized + " ",      // Space before and after
			" " + protectedDirNormalized + "/",      // Space before, slash after
			" " + protectedDirNormalized + "\"",     // Space before, quote after
			" " + protectedDirNormalized + "'",      // Space before, single quote after
			" " + protectedDirNormalized + "$",      // Space before, end of line
			"\"" + protectedDirNormalized + "\"",    // Quoted
			"'" + protectedDirNormalized + "'",      // Single quoted
			" " + protectedDirNormalized + "/*",    // With wildcard
			" " + protectedDirNormalized + "/* ",   // With wildcard and space
			" " + protectedDirNormalized + "/ *",   // With space and wildcard
		}
		
		// Also check for patterns like /etc/passwd, /etc/shadow, etc.
		if strings.Contains(commandLower, protectedDirNormalized+"/") {
			return true, protectedDirNormalized
		}
		
		// Check for patterns like rm -rf /etc
		for _, pattern := range patterns {
			if strings.Contains(commandLower, pattern) {
				return true, protectedDirNormalized
			}
		}
		
		// Check for root directory patterns (special case)
		// Root protection should only apply to direct operations on /, not all absolute paths
		if protectedDirNormalized == "/" {
			rootPatterns := []string{
				" rm -rf /",      // Space before rm
				" rm -r /",       // Without force
				" rm -rf / ",     // With trailing space
				" rm -r / ",      // Without force, with space
				" rm -rf /*",     // With wildcard
				" rm -r /*",      // Without force, with wildcard
				" rm -rf /* ",    // With wildcard and space
				" rm -r /* ",     // Without force, wildcard, space
				" rm -rf / *",    // Space between / and *
				" rm -r / *",     // Without force, space between
				"find / ",        // Find from root
				"find / -",       // Find from root with dash
				"find / -delete", // Find from root with delete
			}
			for _, pattern := range rootPatterns {
				if strings.Contains(commandLower, pattern) {
					return true, "/"
				}
			}
			// Also check if command starts with root operations
			if strings.HasPrefix(commandLower, "rm -rf /") ||
				strings.HasPrefix(commandLower, "rm -r /") ||
				strings.HasPrefix(commandLower, "find /") {
				// But exclude if it's followed by a specific path (like /tmp)
				// Only match if it's /, /*, or / with space/wildcard
				if strings.HasPrefix(commandLower, "rm -rf / ") ||
					strings.HasPrefix(commandLower, "rm -r / ") ||
					strings.HasPrefix(commandLower, "rm -rf /*") ||
					strings.HasPrefix(commandLower, "rm -r /*") ||
					strings.HasPrefix(commandLower, "rm -rf /\"") ||
					strings.HasPrefix(commandLower, "rm -r /\"") ||
					strings.HasPrefix(commandLower, "find / ") ||
					strings.HasPrefix(commandLower, "find / -") {
					return true, "/"
				}
			}
			// Don't match root for other absolute paths - they should be checked against specific protected dirs
			return false, ""
		}
	}
	
	return false, ""
}

// IsProtectedDirectory checks if a given path matches any protected directory.
// This is a helper function for more precise path matching.
func IsProtectedDirectory(path string, protectedDirs []string) bool {
	if len(protectedDirs) == 0 || path == "" {
		return false
	}
	
	pathTrimmed := strings.TrimSpace(path)
	
	// Skip relative paths and home paths - they're not protected
	if strings.HasPrefix(pathTrimmed, "./") || strings.HasPrefix(pathTrimmed, "../") || 
		strings.HasPrefix(pathTrimmed, "~/") || pathTrimmed == "~" {
		return false
	}
	
	pathNormalized := filepath.Clean(strings.ToLower(pathTrimmed))
	if !strings.HasPrefix(pathNormalized, "/") {
		// Only normalize to absolute if it's not a relative path
		if !strings.HasPrefix(pathNormalized, ".") && !strings.HasPrefix(pathNormalized, "~") {
			pathNormalized = "/" + pathNormalized
		} else {
			return false // Relative paths are not protected
		}
	}
	
	for _, protectedDir := range protectedDirs {
		if protectedDir == "" {
			continue
		}
		
		protectedDirNormalized := filepath.Clean(strings.ToLower(strings.TrimSpace(protectedDir)))
		if !strings.HasPrefix(protectedDirNormalized, "/") {
			protectedDirNormalized = "/" + protectedDirNormalized
		}
		
		// Exact match
		if pathNormalized == protectedDirNormalized {
			return true
		}
		
		// Prefix match (path is inside protected directory)
		if strings.HasPrefix(pathNormalized, protectedDirNormalized+"/") {
			return true
		}
		
		// Special case: root directory
		if protectedDirNormalized == "/" {
			// Any absolute path is under root (except root itself which is exact match)
			if pathNormalized != "/" && strings.HasPrefix(pathNormalized, "/") {
				return true
			}
		}
	}
	
	return false
}

