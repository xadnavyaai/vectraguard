package config

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the merged configuration for vectra-guard.
type Config struct {
	Logging       LoggingConfig       `yaml:"logging" toml:"logging" json:"logging"`
	Policies      PolicyConfig        `yaml:"policies" toml:"policies" json:"policies"`
	EnvProtection EnvProtectionConfig `yaml:"env_protection" toml:"env_protection" json:"env_protection"`
	GuardLevel    GuardLevelConfig    `yaml:"guard_level" toml:"guard_level" json:"guard_level"`
}

// LoggingConfig controls output formatting.
type LoggingConfig struct {
	Format string `yaml:"format" toml:"format" json:"format"`
}

// GuardLevel determines the aggressiveness of protection
type GuardLevel string

const (
	GuardLevelOff      GuardLevel = "off"      // No protection
	GuardLevelLow      GuardLevel = "low"      // Only critical issues
	GuardLevelMedium   GuardLevel = "medium"   // Critical + high issues (default)
	GuardLevelHigh     GuardLevel = "high"     // Critical + high + medium issues
	GuardLevelParanoid GuardLevel = "paranoid" // Everything requires approval
)

// GuardLevelConfig controls the overall security posture
type GuardLevelConfig struct {
	Level                GuardLevel `yaml:"level" toml:"level" json:"level"`
	AllowUserBypass      bool       `yaml:"allow_user_bypass" toml:"allow_user_bypass" json:"allow_user_bypass"`
	BypassEnvVar         string     `yaml:"bypass_env_var" toml:"bypass_env_var" json:"bypass_env_var"`
	RequireApprovalAbove string     `yaml:"require_approval_above" toml:"require_approval_above" json:"require_approval_above"`
}

// PolicyConfig captures simple allow/deny rules applied during analysis.
type PolicyConfig struct {
	Allowlist           []string `yaml:"allowlist" toml:"allowlist" json:"allowlist"`
	Denylist            []string `yaml:"denylist" toml:"denylist" json:"denylist"`
	MonitorGitOps       bool     `yaml:"monitor_git_ops" toml:"monitor_git_ops" json:"monitor_git_ops"`
	BlockForceGit       bool     `yaml:"block_force_git" toml:"block_force_git" json:"block_force_git"`
	DetectProdEnv       bool     `yaml:"detect_prod_env" toml:"detect_prod_env" json:"detect_prod_env"`
	ProdEnvPatterns     []string `yaml:"prod_env_patterns" toml:"prod_env_patterns" json:"prod_env_patterns"`
	OnlyDestructiveSQL  bool     `yaml:"only_destructive_sql" toml:"only_destructive_sql" json:"only_destructive_sql"`
}

// EnvProtectionConfig controls environment variable protection and masking.
type EnvProtectionConfig struct {
	Enabled         bool              `yaml:"enabled" toml:"enabled" json:"enabled"`
	MaskingMode     string            `yaml:"masking_mode" toml:"masking_mode" json:"masking_mode"` // full, partial, hash, fake
	ProtectedVars   []string          `yaml:"protected_vars" toml:"protected_vars" json:"protected_vars"`
	AllowReadVars   []string          `yaml:"allow_read_vars" toml:"allow_read_vars" json:"allow_read_vars"`
	FakeValues      map[string]string `yaml:"fake_values" toml:"fake_values" json:"fake_values"`
	BlockEnvAccess  bool              `yaml:"block_env_access" toml:"block_env_access" json:"block_env_access"`
	BlockDotenvRead bool              `yaml:"block_dotenv_read" toml:"block_dotenv_read" json:"block_dotenv_read"`
}

// DefaultConfig returns the in-process defaults.
func DefaultConfig() Config {
	return Config{
		Logging: LoggingConfig{Format: "text"},
		Policies: PolicyConfig{
			Allowlist:          []string{},
			Denylist:           []string{"rm -rf /", "sudo ", ":(){ :|:& };:", "mkfs", "dd if="},
			MonitorGitOps:      true,
			BlockForceGit:      true,
			DetectProdEnv:      true,
			ProdEnvPatterns:    []string{"prod", "production", "prd", "live", "staging", "stg"},
			OnlyDestructiveSQL: true, // Only flag destructive SQL operations
		},
		EnvProtection: EnvProtectionConfig{
			Enabled:         true,
			MaskingMode:     "partial", // partial, full, hash, fake
			ProtectedVars:   []string{},
			AllowReadVars:   []string{"HOME", "USER", "PATH", "SHELL", "TERM", "LANG", "PWD"},
			FakeValues:      make(map[string]string),
			BlockEnvAccess:  false, // Warn but don't block by default
			BlockDotenvRead: true,  // Block .env file reads by default
		},
		GuardLevel: GuardLevelConfig{
			Level:                GuardLevelMedium,
			AllowUserBypass:      true,
			BypassEnvVar:         "VECTRAGUARD_BYPASS",
			RequireApprovalAbove: "medium",
		},
	}
}

// Load discovers and merges configuration from the provided path, project-local config, and user config.
// Precedence (highest wins): explicit flag > project > user > defaults.
func Load(configPath string, workdir string) (Config, []string, error) {
	cfg := DefaultConfig()
	var loaded []string

	paths, err := resolvePaths(configPath, workdir)
	if err != nil {
		return cfg, loaded, err
	}

	for _, path := range paths {
		if err := applyFile(&cfg, path); err != nil {
			return cfg, loaded, err
		}
		loaded = append(loaded, path)
	}

	return cfg, loaded, nil
}

func resolvePaths(explicit, workdir string) ([]string, error) {
	var paths []string
	// Lowest precedence first.
	if home, err := os.UserHomeDir(); err == nil {
		candidateYAML := filepath.Join(home, ".config", "vectra-guard", "config.yaml")
		candidateTOML := filepath.Join(home, ".config", "vectra-guard", "config.toml")
		if exists(candidateYAML) {
			paths = append(paths, candidateYAML)
		}
		if exists(candidateTOML) {
			paths = append(paths, candidateTOML)
		}
	}

	if workdir != "" {
		projectYAML := filepath.Join(workdir, "vectra-guard.yaml")
		projectTOML := filepath.Join(workdir, "vectra-guard.toml")
		if exists(projectYAML) {
			paths = append(paths, projectYAML)
		}
		if exists(projectTOML) {
			paths = append(paths, projectTOML)
		}
	}

	if explicit != "" {
		if !exists(explicit) {
			return nil, fmt.Errorf("config file not found: %s", explicit)
		}
		paths = append(paths, explicit)
	}

	return paths, nil
}

func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}

	var next Config
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		next, err = decodeYAML(data)
	case ".toml":
		next, err = decodeTOML(data)
	default:
		err = errors.New("unsupported config extension; use .yaml or .toml")
	}
	if err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}

	merge(cfg, next)
	return nil
}

func merge(dst *Config, src Config) {
	if src.Logging.Format != "" {
		dst.Logging.Format = src.Logging.Format
	}

	if len(src.Policies.Allowlist) > 0 {
		dst.Policies.Allowlist = src.Policies.Allowlist
	}
	if len(src.Policies.Denylist) > 0 {
		dst.Policies.Denylist = src.Policies.Denylist
	}
	
	// Merge policy booleans (false is a valid value, so we don't check for zero)
	dst.Policies.MonitorGitOps = src.Policies.MonitorGitOps
	dst.Policies.BlockForceGit = src.Policies.BlockForceGit
	dst.Policies.DetectProdEnv = src.Policies.DetectProdEnv
	dst.Policies.OnlyDestructiveSQL = src.Policies.OnlyDestructiveSQL
	
	if len(src.Policies.ProdEnvPatterns) > 0 {
		dst.Policies.ProdEnvPatterns = src.Policies.ProdEnvPatterns
	}
	
	// Merge guard level
	if src.GuardLevel.Level != "" {
		dst.GuardLevel.Level = src.GuardLevel.Level
	}
	if src.GuardLevel.BypassEnvVar != "" {
		dst.GuardLevel.BypassEnvVar = src.GuardLevel.BypassEnvVar
	}
	if src.GuardLevel.RequireApprovalAbove != "" {
		dst.GuardLevel.RequireApprovalAbove = src.GuardLevel.RequireApprovalAbove
	}
	dst.GuardLevel.AllowUserBypass = src.GuardLevel.AllowUserBypass
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func decodeYAML(data []byte) (Config, error) {
	var cfg Config
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	mode := ""
	var listTarget *[]string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			switch strings.TrimSuffix(line, ":") {
			case "logging":
				mode = "logging"
				listTarget = nil
			case "policies":
				mode = "policies"
				listTarget = nil
			case "guard_level":
				mode = "guard_level"
				listTarget = nil
			case "allowlist":
				if mode == "policies" {
					listTarget = &cfg.Policies.Allowlist
				}
			case "denylist":
				if mode == "policies" {
					listTarget = &cfg.Policies.Denylist
				}
			case "prod_env_patterns":
				if mode == "policies" {
					listTarget = &cfg.Policies.ProdEnvPatterns
				}
			}
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if listTarget != nil {
				*listTarget = append(*listTarget, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			}
			continue
		}

		// Parse key: value lines
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			
			switch mode {
			case "logging":
				if key == "format" {
					cfg.Logging.Format = value
				}
			case "guard_level":
				switch key {
				case "level":
					cfg.GuardLevel.Level = GuardLevel(value)
				case "allow_user_bypass":
					cfg.GuardLevel.AllowUserBypass = value == "true"
				case "bypass_env_var":
					cfg.GuardLevel.BypassEnvVar = value
				case "require_approval_above":
					cfg.GuardLevel.RequireApprovalAbove = value
				}
			case "policies":
				switch key {
				case "monitor_git_ops":
					cfg.Policies.MonitorGitOps = value == "true"
				case "block_force_git":
					cfg.Policies.BlockForceGit = value == "true"
				case "detect_prod_env":
					cfg.Policies.DetectProdEnv = value == "true"
				case "only_destructive_sql":
					cfg.Policies.OnlyDestructiveSQL = value == "true"
				}
			}
		}
	}

	return cfg, scanner.Err()
}

func decodeTOML(data []byte) (Config, error) {
	var cfg Config
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	section := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch section {
		case "logging":
			if key == "format" {
				cfg.Logging.Format = trimQuotes(value)
			}
		case "policies":
			switch key {
			case "allowlist":
				cfg.Policies.Allowlist = parseStringArray(value)
			case "denylist":
				cfg.Policies.Denylist = parseStringArray(value)
			}
		}
	}

	return cfg, scanner.Err()
}

func parseStringArray(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" {
		return nil
	}
	raw := strings.Split(value, ",")
	var cleaned []string
	for _, item := range raw {
		item = trimQuotes(strings.TrimSpace(item))
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}

func trimQuotes(s string) string {
	return strings.Trim(s, `"`)
}

type ctxKey struct{}

// WithConfig stores the config on the context.
func WithConfig(ctx context.Context, cfg Config) context.Context {
	return context.WithValue(ctx, ctxKey{}, cfg)
}

// FromContext extracts the config or returns defaults.
func FromContext(ctx context.Context) Config {
	if ctx == nil {
		return DefaultConfig()
	}
	if cfg, ok := ctx.Value(ctxKey{}).(Config); ok {
		return cfg
	}
	return DefaultConfig()
}
