package softdelete

import (
	"path/filepath"
	"strings"
)

// isGitFile checks if a path is a git-related file.
func (m *Manager) isGitFile(path string) bool {
	if !m.config.ProtectGit {
		return false
	}
	
	// Normalize path
	normalized := filepath.Clean(path)
	base := filepath.Base(normalized)
	dir := filepath.Dir(normalized)
	
	// Check for .git directory
	if base == ".git" {
		return true
	}
	
	// Check if path is inside .git directory
	// Check for both forward and backward slashes
	// Also check if path starts with .git/ (relative path)
	if strings.Contains(normalized, "/.git/") || strings.Contains(normalized, "\\.git\\") || 
	   strings.HasPrefix(normalized, ".git/") || strings.HasPrefix(normalized, ".git\\") {
		return true
	}
	
	// Check for git config files in root
	gitConfigFiles := []string{
		".gitignore",
		".gitattributes",
		".gitconfig",
		".gitmodules",
		".gitkeep",
	}
	
	for _, gitFile := range gitConfigFiles {
		if base == gitFile {
			// Check if it's in a project root (not in a subdirectory)
			// This is a heuristic: if parent is not .git, it's likely a root git file
			if !strings.Contains(dir, ".git") {
				return true
			}
		}
	}
	
	return false
}

// IsGitProtected checks if a path should have enhanced git protection.
func (m *Manager) IsGitProtected(path string) bool {
	return m.isGitFile(path)
}

// GetGitProtectedPaths returns a list of git-related paths that should be protected.
func GetGitProtectedPaths() []string {
	return []string{
		".git/",
		".gitignore",
		".gitattributes",
		".gitconfig",
		".gitmodules",
		".gitkeep",
		".git/HEAD",
		".git/config",
		".git/hooks/",
		".git/refs/",
		".git/objects/",
		".git/index",
		".git/logs/",
	}
}

// ShouldBlockGitDeletion checks if a git deletion should be blocked entirely.
func (m *Manager) ShouldBlockGitDeletion(paths []string) (bool, []string) {
	if !m.config.ProtectGit {
		return false, nil
	}
	
	var gitPaths []string
	for _, path := range paths {
		if m.isGitFile(path) {
			gitPaths = append(gitPaths, path)
		}
	}
	
	if len(gitPaths) == 0 {
		return false, nil
	}
	
	// For critical git files (.git directory), we might want to require explicit confirmation
	// For now, we allow soft delete but with extra warnings
	return false, gitPaths
}
