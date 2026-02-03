package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setConfigDir sets CLICKUP_CONFIG_DIR to a temp directory and returns a cleanup function.
func setConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLICKUP_CONFIG_DIR", dir)
	return dir
}

func TestConfigDir_EnvOverride(t *testing.T) {
	want := "/tmp/custom-clickup-config"
	t.Setenv("CLICKUP_CONFIG_DIR", want)

	got := ConfigDir()
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_Default(t *testing.T) {
	t.Setenv("CLICKUP_CONFIG_DIR", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	want := filepath.Join(home, ".config", "clickup")
	got := ConfigDir()
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigFile(t *testing.T) {
	dir := "/tmp/test-config"
	t.Setenv("CLICKUP_CONFIG_DIR", dir)

	want := filepath.Join(dir, "config.yml")
	got := ConfigFile()
	if got != want {
		t.Errorf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestLoad_NoFile(t *testing.T) {
	setConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Check defaults.
	if cfg.Prompt != "enabled" {
		t.Errorf("default Prompt = %q, want %q", cfg.Prompt, "enabled")
	}
	if cfg.Aliases == nil {
		t.Error("default Aliases should not be nil")
	}
	if cfg.Workspace != "" {
		t.Errorf("default Workspace = %q, want empty", cfg.Workspace)
	}
}

func TestSaveAndLoad(t *testing.T) {
	setConfigDir(t)

	original := &Config{
		Workspace:    "team123",
		Space:        "space456",
		SprintFolder: "folder789",
		Editor:       "vim",
		Prompt:       "disabled",
		Aliases: map[string]string{
			"ls": "task list",
			"v":  "task view",
		},
	}

	if err := original.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify the file was created.
	if _, err := os.Stat(ConfigFile()); err != nil {
		t.Fatalf("config file not found after Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Workspace != original.Workspace {
		t.Errorf("Workspace = %q, want %q", loaded.Workspace, original.Workspace)
	}
	if loaded.Space != original.Space {
		t.Errorf("Space = %q, want %q", loaded.Space, original.Space)
	}
	if loaded.SprintFolder != original.SprintFolder {
		t.Errorf("SprintFolder = %q, want %q", loaded.SprintFolder, original.SprintFolder)
	}
	if loaded.Editor != original.Editor {
		t.Errorf("Editor = %q, want %q", loaded.Editor, original.Editor)
	}
	if loaded.Prompt != original.Prompt {
		t.Errorf("Prompt = %q, want %q", loaded.Prompt, original.Prompt)
	}
	if len(loaded.Aliases) != len(original.Aliases) {
		t.Errorf("Aliases length = %d, want %d", len(loaded.Aliases), len(original.Aliases))
	}
	for k, v := range original.Aliases {
		if loaded.Aliases[k] != v {
			t.Errorf("Aliases[%q] = %q, want %q", k, loaded.Aliases[k], v)
		}
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := setConfigDir(t)

	// Write YAML that cannot unmarshal into Config (tab character in wrong position
	// plus structure that causes a parse error).
	invalidYAML := "workspace:\n\t- bad indent\n  :\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(invalidYAML), 0o644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() with invalid YAML should return an error")
	}
}

func TestSpaceForDir(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		dir      string
		want     string
	}{
		{
			name: "returns directory-specific space",
			cfg: Config{
				Space: "global-space",
				DirectoryDefaults: map[string]DirectoryConfig{
					"/home/user/project-a": {Space: "space-a"},
					"/home/user/project-b": {Space: "space-b"},
				},
			},
			dir:  "/home/user/project-a",
			want: "space-a",
		},
		{
			name: "falls back to global space when dir not in map",
			cfg: Config{
				Space: "global-space",
				DirectoryDefaults: map[string]DirectoryConfig{
					"/home/user/project-a": {Space: "space-a"},
				},
			},
			dir:  "/home/user/project-c",
			want: "global-space",
		},
		{
			name: "falls back to global space when DirectoryDefaults is nil",
			cfg: Config{
				Space:             "global-space",
				DirectoryDefaults: nil,
			},
			dir:  "/home/user/project-a",
			want: "global-space",
		},
		{
			name: "falls back to global space when dir entry has empty space",
			cfg: Config{
				Space: "global-space",
				DirectoryDefaults: map[string]DirectoryConfig{
					"/home/user/project-a": {Space: ""},
				},
			},
			dir:  "/home/user/project-a",
			want: "global-space",
		},
		{
			name: "returns empty when no global and no directory match",
			cfg: Config{
				Space:             "",
				DirectoryDefaults: map[string]DirectoryConfig{},
			},
			dir:  "/home/user/project-x",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.SpaceForDir(tt.dir)
			if got != tt.want {
				t.Errorf("SpaceForDir(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestSetDirectoryDefault(t *testing.T) {
	t.Run("initializes map when nil", func(t *testing.T) {
		cfg := &Config{}
		if cfg.DirectoryDefaults != nil {
			t.Fatal("DirectoryDefaults should start as nil")
		}

		cfg.SetDirectoryDefault("/home/user/project", DirectoryConfig{Space: "my-space"})

		if cfg.DirectoryDefaults == nil {
			t.Fatal("DirectoryDefaults should not be nil after SetDirectoryDefault")
		}
		dc, ok := cfg.DirectoryDefaults["/home/user/project"]
		if !ok {
			t.Fatal("expected directory entry to exist")
		}
		if dc.Space != "my-space" {
			t.Errorf("Space = %q, want %q", dc.Space, "my-space")
		}
	})

	t.Run("overwrites existing entry", func(t *testing.T) {
		cfg := &Config{
			DirectoryDefaults: map[string]DirectoryConfig{
				"/home/user/project": {Space: "old-space"},
			},
		}

		cfg.SetDirectoryDefault("/home/user/project", DirectoryConfig{Space: "new-space"})

		dc := cfg.DirectoryDefaults["/home/user/project"]
		if dc.Space != "new-space" {
			t.Errorf("Space = %q, want %q", dc.Space, "new-space")
		}
	})

	t.Run("adds new entry to existing map", func(t *testing.T) {
		cfg := &Config{
			DirectoryDefaults: map[string]DirectoryConfig{
				"/home/user/project-a": {Space: "space-a"},
			},
		}

		cfg.SetDirectoryDefault("/home/user/project-b", DirectoryConfig{Space: "space-b"})

		if len(cfg.DirectoryDefaults) != 2 {
			t.Errorf("DirectoryDefaults length = %d, want 2", len(cfg.DirectoryDefaults))
		}
		if cfg.DirectoryDefaults["/home/user/project-b"].Space != "space-b" {
			t.Errorf("Space = %q, want %q", cfg.DirectoryDefaults["/home/user/project-b"].Space, "space-b")
		}
	})
}

func TestSaveAndLoadWithDirectoryDefaults(t *testing.T) {
	setConfigDir(t)

	original := &Config{
		Workspace: "team1",
		Space:     "global-space",
		Aliases:   map[string]string{},
	}
	original.SetDirectoryDefault("/projects/app", DirectoryConfig{Space: "app-space"})
	original.SetDirectoryDefault("/projects/lib", DirectoryConfig{Space: "lib-space"})

	if err := original.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify SpaceForDir works after round-trip.
	if got := loaded.SpaceForDir("/projects/app"); got != "app-space" {
		t.Errorf("SpaceForDir(/projects/app) = %q, want %q", got, "app-space")
	}
	if got := loaded.SpaceForDir("/projects/lib"); got != "lib-space" {
		t.Errorf("SpaceForDir(/projects/lib) = %q, want %q", got, "lib-space")
	}
	if got := loaded.SpaceForDir("/projects/other"); got != "global-space" {
		t.Errorf("SpaceForDir(/projects/other) = %q, want %q", got, "global-space")
	}
}
