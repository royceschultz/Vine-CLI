package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type StorageMode string

const (
	StorageLocal  StorageMode = "local"
	StorageGlobal StorageMode = "global"
)

type Config struct {
	Name     string      `json:"name"`
	Storage  StorageMode `json:"storage"`
	Database string      `json:"database,omitempty"`
}

// ProjectName returns the project name from config.
// Falls back to the database name, then the directory basename.
func ProjectName(projectRoot string, cfg *Config) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.Database != "" {
		return cfg.Database
	}
	return filepath.Base(projectRoot)
}

const (
	DotVineDir = ".vine"
	ConfigFile = "config.json"
)

// FindProjectRoot walks up from startDir looking for a .vine/ directory.
func FindProjectRoot(startDir string) (string, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, DotVineDir)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a vine project (no .vine directory found in %s or any parent)", startDir)
		}
		dir = parent
	}
}

// Load reads .vine/config.json from the given project root.
func Load(projectRoot string) (*Config, error) {
	path := filepath.Join(projectRoot, DotVineDir, ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes .vine/config.json to the given project root.
func Save(projectRoot string, cfg *Config) error {
	dirPath := filepath.Join(projectRoot, DotVineDir)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("creating .vine directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := filepath.Join(dirPath, ConfigFile)
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// DatabasePath resolves the full path to the SQLite database file.
func DatabasePath(projectRoot string, cfg *Config) (string, error) {
	switch cfg.Storage {
	case StorageLocal:
		return filepath.Join(projectRoot, DotVineDir, "vine.db"), nil
	case StorageGlobal:
		if cfg.Database == "" {
			return "", errors.New("global storage requires a database name")
		}
		dir, err := GlobalDatabasesDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, cfg.Database+".db"), nil
	default:
		return "", fmt.Errorf("unknown storage mode: %q", cfg.Storage)
	}
}

// GlobalDatabasesDir returns ~/.vine/databases/.
func GlobalDatabasesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".vine", "databases"), nil
}
