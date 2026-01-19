package cve

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
)

var exactVersionPattern = regexp.MustCompile(`^[vV]?[0-9][0-9A-Za-z.\-+]*$`)

// DiscoverPackages scans the workspace for supported manifests/lockfiles.
func DiscoverPackages(root string) ([]PackageRef, []string, error) {
	manifests := []string{}
	warnings := []string{}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			base := filepath.Base(path)
			switch base {
			case ".git", "node_modules", "vendor", ".vectra-guard", "dist", "build":
				return fs.SkipDir
			default:
				return nil
			}
		}
		switch filepath.Base(path) {
		case "package.json", "package-lock.json", "go.mod":
			manifests = append(manifests, path)
		}
		return nil
	})
	if err != nil {
		return nil, warnings, err
	}

	packages := make(map[string]PackageRef)
	for _, manifest := range manifests {
		name := filepath.Base(manifest)
		switch name {
		case "package-lock.json":
			refs, warn, err := parseNPMLock(manifest)
			warnings = append(warnings, warn...)
			if err != nil {
				return nil, warnings, err
			}
			for _, ref := range refs {
				packages[ref.Key()] = ref
			}
		case "package.json":
			refs, warn, err := parseNPMManifest(manifest)
			warnings = append(warnings, warn...)
			if err != nil {
				return nil, warnings, err
			}
			for _, ref := range refs {
				packages[ref.Key()] = ref
			}
		case "go.mod":
			refs, warn, err := parseGoMod(manifest)
			warnings = append(warnings, warn...)
			if err != nil {
				return nil, warnings, err
			}
			for _, ref := range refs {
				packages[ref.Key()] = ref
			}
		}
	}

	out := make([]PackageRef, 0, len(packages))
	for _, ref := range packages {
		out = append(out, ref)
	}
	return out, warnings, nil
}

type packageJSON struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

func parseNPMManifest(path string) ([]PackageRef, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return collectNPMDeps(path, pkg)
}

func collectNPMDeps(path string, pkg packageJSON) ([]PackageRef, []string, error) {
	out := []PackageRef{}
	warnings := []string{}
	add := func(name, version string) {
		version = strings.TrimSpace(version)
		if version == "" {
			return
		}
		if !isExactVersion(version) {
			warnings = append(warnings, fmt.Sprintf("skip %s (%s): non-exact version %q", path, name, version))
			return
		}
		out = append(out, PackageRef{
			Ecosystem: "npm",
			Name:      name,
			Version:   trimLeadingV(version),
		})
	}
	for name, version := range pkg.Dependencies {
		add(name, version)
	}
	for name, version := range pkg.DevDependencies {
		add(name, version)
	}
	for name, version := range pkg.OptionalDependencies {
		add(name, version)
	}
	return out, warnings, nil
}

type npmLockV2 struct {
	Packages     map[string]npmLockPkg `json:"packages"`
	Dependencies map[string]npmLockDep `json:"dependencies"`
}

type npmLockPkg struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type npmLockDep struct {
	Version      string                `json:"version"`
	Dependencies map[string]npmLockDep `json:"dependencies"`
}

func parseNPMLock(path string) ([]PackageRef, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	var lock npmLockV2
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}

	refs := make(map[string]PackageRef)
	warnings := []string{}

	for pkgPath, pkg := range lock.Packages {
		name := strings.TrimSpace(pkg.Name)
		if name == "" {
			if strings.HasPrefix(pkgPath, "node_modules/") {
				name = strings.TrimPrefix(pkgPath, "node_modules/")
			}
		}
		if name == "" || pkg.Version == "" {
			continue
		}
		ref := PackageRef{
			Ecosystem: "npm",
			Name:      name,
			Version:   trimLeadingV(pkg.Version),
		}
		refs[ref.Key()] = ref
	}

	var walkDeps func(map[string]npmLockDep)
	walkDeps = func(deps map[string]npmLockDep) {
		for name, dep := range deps {
			if dep.Version != "" {
				ref := PackageRef{
					Ecosystem: "npm",
					Name:      name,
					Version:   trimLeadingV(dep.Version),
				}
				refs[ref.Key()] = ref
			}
			if len(dep.Dependencies) > 0 {
				walkDeps(dep.Dependencies)
			}
		}
	}
	if len(lock.Dependencies) > 0 {
		walkDeps(lock.Dependencies)
	}

	out := make([]PackageRef, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref)
	}
	return out, warnings, nil
}

func parseGoMod(path string) ([]PackageRef, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	mod, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := []PackageRef{}
	for _, req := range mod.Require {
		if req.Mod.Path == "" || req.Mod.Version == "" {
			continue
		}
		out = append(out, PackageRef{
			Ecosystem: "Go",
			Name:      req.Mod.Path,
			Version:   req.Mod.Version,
		})
	}
	return out, nil, nil
}

func isExactVersion(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	if strings.Contains(version, "||") || strings.ContainsAny(version, " <>^~*") {
		return false
	}
	if strings.Contains(version, "x") || strings.Contains(version, "X") {
		return false
	}
	return exactVersionPattern.MatchString(version)
}

func trimLeadingV(version string) string {
	if strings.HasPrefix(version, "v") || strings.HasPrefix(version, "V") {
		return version[1:]
	}
	return version
}
