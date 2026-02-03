package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the user's CLI configuration.
type Config struct {
	Workspace         string                       `yaml:"workspace,omitempty"`
	Space             string                       `yaml:"space,omitempty"`
	SprintFolder      string                       `yaml:"sprint_folder,omitempty"`
	LinkField         string                       `yaml:"link_field,omitempty"`
	Editor            string                       `yaml:"editor,omitempty"`
	Prompt            string                       `yaml:"prompt,omitempty"`
	Aliases           map[string]string            `yaml:"aliases,omitempty"`
	DirectoryDefaults map[string]DirectoryConfig   `yaml:"directory_defaults,omitempty"`
}

// DirectoryConfig holds per-directory overrides.
type DirectoryConfig struct {
	Space     string `yaml:"space,omitempty"`
	LinkField string `yaml:"link_field,omitempty"`
}

// ConfigDir returns the path to the config directory (~/.config/clickup).
func ConfigDir() string {
	if dir := os.Getenv("CLICKUP_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "clickup")
}

// ConfigFile returns the path to the main config file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yml")
}

// Load reads the config from disk, returning defaults if the file doesn't exist.
func Load() (*Config, error) {
	cfg := &Config{
		Prompt:  "enabled",
		Aliases: map[string]string{},
	}

	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile(), data, 0o644)
}

// SpaceForDir returns the space override for a specific directory, falling back to the global default.
func (c *Config) SpaceForDir(dir string) string {
	if c.DirectoryDefaults != nil {
		if dc, ok := c.DirectoryDefaults[dir]; ok && dc.Space != "" {
			return dc.Space
		}
	}
	return c.Space
}

// LinkFieldForDir returns the link field name override for a specific directory,
// falling back to the global default.
func (c *Config) LinkFieldForDir(dir string) string {
	if c.DirectoryDefaults != nil {
		if dc, ok := c.DirectoryDefaults[dir]; ok && dc.LinkField != "" {
			return dc.LinkField
		}
	}
	return c.LinkField
}

// SetDirectoryDefault sets a per-directory config override.
func (c *Config) SetDirectoryDefault(dir string, dc DirectoryConfig) {
	if c.DirectoryDefaults == nil {
		c.DirectoryDefaults = make(map[string]DirectoryConfig)
	}
	c.DirectoryDefaults[dir] = dc
}
