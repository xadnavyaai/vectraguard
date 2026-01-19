package analyzer

import (
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

	// Tokenize the command so we can reason about individual arguments/paths.
	// This avoids brittle substring matching and ensures we can distinguish
	// between /usr and /usr/bin, etc.
	fields := strings.Fields(commandLower)

	// First, try to find the *most specific* protected directory that matches
	// any of the command arguments. This ensures that for nested directories
	// like /usr and /usr/bin we return /usr/bin when appropriate.
	bestMatch := ""

	for _, field := range fields {
		// Strip common quoting characters
		fieldTrimmed := strings.Trim(field, "\"'")
		if fieldTrimmed == "" {
			continue
		}

		// Check if it's a Unix-style absolute path (starts with /)
		// We handle Unix paths specially to work cross-platform
		if !strings.HasPrefix(fieldTrimmed, "/") {
			// Non-absolute paths aren't considered here – they may be relative
			// paths that are handled by other logic.
			continue
		}

		// For Unix-style absolute paths, normalize manually to preserve the leading /
		// This works cross-platform (even on Windows, we want to detect Unix paths in commands)
		argPath := strings.TrimRight(fieldTrimmed, "/")
		if argPath == "" {
			argPath = "/"
		}
		// Remove any double slashes but keep the leading one
		argPath = "/" + strings.TrimLeft(strings.ReplaceAll(argPath, "//", "/"), "/")

		argPathLower := strings.ToLower(argPath)

		for _, protectedDir := range protectedDirs {
			if protectedDir == "" {
				continue
			}

			protectedDirLower := strings.ToLower(strings.TrimSpace(protectedDir))
			// Normalize Unix-style paths manually (works cross-platform)
			var protectedDirNormalized string
			if strings.HasPrefix(protectedDirLower, "/") {
				// Unix-style path - normalize manually
				protectedDirNormalized = strings.TrimRight(protectedDirLower, "/")
				if protectedDirNormalized == "" {
					protectedDirNormalized = "/"
				} else {
					// Remove double slashes but keep leading /
					protectedDirNormalized = "/" + strings.TrimLeft(strings.ReplaceAll(protectedDirNormalized, "//", "/"), "/")
				}
			} else {
				// Relative path - make it absolute
				protectedDirNormalized = "/" + strings.TrimLeft(protectedDirLower, "/")
			}

			// Special handling for root directory. We treat arguments that are
			// exactly "/" or "/*" as targeting the root directory.
			if protectedDirNormalized == "/" {
				if argPathLower == "/" || argPathLower == "/*" {
					return true, "/"
				}
				continue
			}

			// Exact match: e.g. "rm -rf /etc"
			if argPathLower == protectedDirNormalized {
				if len(protectedDirNormalized) > len(bestMatch) {
					bestMatch = protectedDirNormalized
				}
				continue
			}

			// Prefix match: e.g. "rm -rf /etc/passwd" should match /etc
			if strings.HasPrefix(argPathLower, protectedDirNormalized+"/") {
				if len(protectedDirNormalized) > len(bestMatch) {
					bestMatch = protectedDirNormalized
				}
				continue
			}

			// Wildcard-style patterns such as "/etc/*" or "/etc/**"
			if strings.HasPrefix(argPathLower, protectedDirNormalized+"/*") {
				if len(protectedDirNormalized) > len(bestMatch) {
					bestMatch = protectedDirNormalized
				}
				continue
			}
		}
	}

	// If we found a specific protected directory match, return it.
	if bestMatch != "" {
		return true, bestMatch
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

	// Normalize path - handle Unix-style paths cross-platform
	pathLower := strings.ToLower(pathTrimmed)
	var pathNormalized string
	if strings.HasPrefix(pathLower, "/") {
		// Unix-style absolute path - normalize manually
		pathNormalized = strings.TrimRight(pathLower, "/")
		if pathNormalized == "" {
			pathNormalized = "/"
		} else {
			pathNormalized = "/" + strings.TrimLeft(strings.ReplaceAll(pathNormalized, "//", "/"), "/")
		}
	} else {
		// Not an absolute Unix path - not protected
		return false
	}

	for _, protectedDir := range protectedDirs {
		if protectedDir == "" {
			continue
		}

		protectedDirLower := strings.ToLower(strings.TrimSpace(protectedDir))
		var protectedDirNormalized string
		if strings.HasPrefix(protectedDirLower, "/") {
			// Unix-style path - normalize manually
			protectedDirNormalized = strings.TrimRight(protectedDirLower, "/")
			if protectedDirNormalized == "" {
				protectedDirNormalized = "/"
			} else {
				protectedDirNormalized = "/" + strings.TrimLeft(strings.ReplaceAll(protectedDirNormalized, "//", "/"), "/")
			}
		} else {
			// Relative path - make it absolute
			protectedDirNormalized = "/" + strings.TrimLeft(protectedDirLower, "/")
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
