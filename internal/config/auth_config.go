package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AuthConfig stores authentication state on disk as a fallback when the OS keyring is unavailable.
type AuthConfig struct {
	Token      string `yaml:"token,omitempty"`
	AuthMethod string `yaml:"auth_method,omitempty"` // "oauth" or "token"
}

// AuthFile returns the path to the auth config file.
func AuthFile() string {
	return filepath.Join(ConfigDir(), "auth.yml")
}

// LoadAuth reads the auth config from disk.
func LoadAuth() (*AuthConfig, error) {
	ac := &AuthConfig{}

	data, err := os.ReadFile(AuthFile())
	if err != nil {
		if os.IsNotExist(err) {
			return ac, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, ac); err != nil {
		return nil, err
	}
	return ac, nil
}

// Save writes the auth config to disk.
func (a *AuthConfig) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	return os.WriteFile(AuthFile(), data, 0o600)
}

// Clear removes the stored auth config.
func (a *AuthConfig) Clear() error {
	a.Token = ""
	a.AuthMethod = ""
	return os.Remove(AuthFile())
}
