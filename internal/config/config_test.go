package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadActiveMergesLocalOverGlobal(t *testing.T) {
	home := t.TempDir()
	setHomeDir(t, home)

	globalRoot := filepath.Join(home, ".pmp")
	if err := os.MkdirAll(globalRoot, 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	globalConfig := "separator: \"\\n\\n\"\npresets:\n  feature:\n    description: global\n    blocks:\n      - tasks/feature.md\n"
	if err := os.WriteFile(filepath.Join(globalRoot, "config.yaml"), []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	cwd := t.TempDir()
	projectRoot := filepath.Join(cwd, ".pmp")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projectConfig := "presets:\n  feature:\n    description: local\n    blocks:\n      - tasks/local.md\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte(projectConfig), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	active, err := LoadActive(cwd)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	if active.Config.Presets["feature"].Description != "local" {
		t.Fatalf("expected local preset override, got %q", active.Config.Presets["feature"].Description)
	}
}

func TestLoadActiveProjectBaseAlwaysIncludeReplacesGlobal(t *testing.T) {
	home := t.TempDir()
	setHomeDir(t, home)

	globalRoot := filepath.Join(home, ".pmp")
	if err := os.MkdirAll(globalRoot, 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	globalConfig := "base:\n  always_include:\n    - global.md\n"
	if err := os.WriteFile(filepath.Join(globalRoot, "config.yaml"), []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projectConfig := "base:\n  always_include:\n    - local.md\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte(projectConfig), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	active, err := LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	want := []string{"local.md"}
	if !reflect.DeepEqual(active.Config.Base.AlwaysInclude, want) {
		t.Fatalf("expected base always_include %v, got %v", want, active.Config.Base.AlwaysInclude)
	}
}

func TestLoadActiveProjectCanClearGlobalBaseAlwaysInclude(t *testing.T) {
	home := t.TempDir()
	setHomeDir(t, home)

	globalRoot := filepath.Join(home, ".pmp")
	if err := os.MkdirAll(globalRoot, 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	globalConfig := "base:\n  always_include:\n    - global.md\n"
	if err := os.WriteFile(filepath.Join(globalRoot, "config.yaml"), []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projectConfig := "base:\n  always_include: []\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte(projectConfig), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	active, err := LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	if len(active.Config.Base.AlwaysInclude) != 0 {
		t.Fatalf("expected empty base always_include, got %v", active.Config.Base.AlwaysInclude)
	}
}

func TestSavePresetWritesToActiveConfig(t *testing.T) {
	root := t.TempDir()
	setHomeDir(t, t.TempDir())
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	configPath := filepath.Join(projectRoot, "config.yaml")
	raw := "version: 1\npresets:\n  feature:\n    description: feature\n    blocks:\n      - tasks/feature.md\n"
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	active, err := LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	err = SavePreset(active, "review", Preset{
		Description: "review work",
		Blocks:      []string{"tasks/review.md", "tools/dev-tools.md"},
	})
	if err != nil {
		t.Fatalf("save preset: %v", err)
	}

	reloaded, err := loadFile(configPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.Presets["review"].Description != "review work" {
		t.Fatalf("unexpected preset description %q", reloaded.Presets["review"].Description)
	}
	if len(reloaded.Presets["review"].Blocks) != 2 {
		t.Fatalf("unexpected block count %d", len(reloaded.Presets["review"].Blocks))
	}
}

func TestLoadActiveFindsProjectConfigInParentDirectory(t *testing.T) {
	home := t.TempDir()
	setHomeDir(t, home)

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	configPath := filepath.Join(projectRoot, "config.yaml")
	raw := "version: 1\npresets:\n  feature:\n    description: feature\n    blocks:\n      - tasks/feature.md\n"
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	nested := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	active, err := LoadActive(nested)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	if active.ProjectRoot != projectRoot {
		t.Fatalf("expected project root %q, got %q", projectRoot, active.ProjectRoot)
	}
	if active.ActiveConfigPath != configPath {
		t.Fatalf("expected active config %q, got %q", configPath, active.ActiveConfigPath)
	}
}

func TestLoadActiveUsesProjectConfigWhenGlobalHomeLookupFails(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	configPath := filepath.Join(projectRoot, "config.yaml")
	raw := "presets:\n  feature:\n    description: feature\n    blocks:\n      - tasks/feature.md\n"
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	prevUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) {
		return "", errors.New("home unavailable")
	}
	t.Cleanup(func() {
		userHomeDir = prevUserHomeDir
	})

	active, err := LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	if active.ProjectRoot != projectRoot {
		t.Fatalf("expected project root %q, got %q", projectRoot, active.ProjectRoot)
	}
	if active.GlobalRoot != "" {
		t.Fatalf("expected empty global root, got %q", active.GlobalRoot)
	}
}

func setHomeDir(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}
