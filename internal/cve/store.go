package cve

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store manages local CVE cache persistence.
type Store struct {
	Path  string
	Cache Cache
}

func LoadStore(path string) (*Store, error) {
	cache := Cache{
		Entries: make(map[string]PackageVuln),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Store{Path: path, Cache: cache}, nil
		}
		return nil, fmt.Errorf("read cve cache: %w", err)
	}
	if len(data) == 0 {
		return &Store{Path: path, Cache: cache}, nil
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parse cve cache: %w", err)
	}
	if cache.Entries == nil {
		cache.Entries = make(map[string]PackageVuln)
	}
	return &Store{Path: path, Cache: cache}, nil
}

func (s *Store) Save() error {
	if s.Cache.Entries == nil {
		s.Cache.Entries = make(map[string]PackageVuln)
	}
	s.Cache.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("create cve cache dir: %w", err)
	}
	data, err := json.MarshalIndent(s.Cache, "", "  ")
	if err != nil {
		return fmt.Errorf("encode cve cache: %w", err)
	}
	if err := os.WriteFile(s.Path, data, 0o644); err != nil {
		return fmt.Errorf("write cve cache: %w", err)
	}
	return nil
}

func (s *Store) Get(ref PackageRef) (PackageVuln, bool) {
	entry, ok := s.Cache.Entries[ref.Key()]
	return entry, ok
}

func (s *Store) Set(entry PackageVuln) {
	if s.Cache.Entries == nil {
		s.Cache.Entries = make(map[string]PackageVuln)
	}
	s.Cache.Entries[entry.Package.Key()] = entry
}

func (s *Store) IsFresh(entry PackageVuln, maxAge time.Duration) bool {
	if entry.RetrievedAt.IsZero() {
		return false
	}
	if maxAge <= 0 {
		return true
	}
	return time.Since(entry.RetrievedAt) <= maxAge
}
