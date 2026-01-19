package cve

import (
	"fmt"
	"strings"
	"time"
)

// PackageRef identifies a package in an ecosystem.
type PackageRef struct {
	Ecosystem string
	Name      string
	Version   string
}

func (p PackageRef) Key() string {
	return fmt.Sprintf("%s|%s|%s", strings.ToLower(p.Ecosystem), p.Name, p.Version)
}

// Vulnerability captures a normalized vulnerability record.
type Vulnerability struct {
	ID         string
	Summary    string
	Details    string
	Severity   string
	CVSS       float64
	Aliases    []string
	References []string
	Published  time.Time
	Modified   time.Time
}

// PackageVuln is a cached vulnerability result for a package.
type PackageVuln struct {
	Package         PackageRef
	Vulnerabilities []Vulnerability
	RetrievedAt     time.Time
}

// Cache stores cached CVE lookups.
type Cache struct {
	UpdatedAt time.Time
	Entries   map[string]PackageVuln
}
