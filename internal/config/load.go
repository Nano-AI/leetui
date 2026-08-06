package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Load reads config.toml, applying defaults for anything absent.
//
// A missing file is not an error: it returns defaults with the path set, so a first run
// works with no setup and Save writes a complete file.
func Load() (Config, error) {
	dir, err := Dir()
	if err != nil {
		return Default(), err
	}
	path := filepath.Join(dir, "config.toml")

	cfg := Default()
	cfg.path = path

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		// A malformed config must not wipe the user's file. Report and run on defaults.
		return applyDefaults(Default(), path), fmt.Errorf("parse %s: %w", path, err)
	}
	return applyDefaults(cfg, path), nil
}

// applyDefaults fills zero values that have a meaningful non-zero default. Booleans are
// left alone: false is a legitimate choice a user may have written on purpose.
func applyDefaults(c Config, path string) Config {
	d := Default()
	if c.Workspace == "" {
		c.Workspace = d.Workspace
	}
	if c.DefaultLang == "" {
		c.DefaultLang = d.DefaultLang
	}
	if c.Sync.RequestsPerSecond <= 0 {
		c.Sync.RequestsPerSecond = d.Sync.RequestsPerSecond
	}
	if c.Sync.PageSize <= 0 {
		c.Sync.PageSize = d.Sync.PageSize
	}
	if c.Sync.CompanyRefreshDays <= 0 {
		c.Sync.CompanyRefreshDays = d.Sync.CompanyRefreshDays
	}
	if c.Keys == nil {
		c.Keys = map[string]string{}
	}
	c.path = path
	return c
}

// Save writes the config back to disk.
func (c Config) Save() error {
	if c.path == "" {
		dir, err := Dir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, "config.toml")
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	// Write via a temp file so an interrupted save cannot truncate a good config.
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// Path returns where this config was loaded from.
func (c Config) Path() string { return c.path }
