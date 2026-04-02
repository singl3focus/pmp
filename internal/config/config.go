package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/singl3focus/pmp/internal/block"
	"gopkg.in/yaml.v3"
)

var ErrConfigNotFound = errors.New("no pmp config found; run `pmp init` or `pmp init --global`")

var userHomeDir = os.UserHomeDir

type Active struct {
	ProjectRoot      string
	GlobalRoot       string
	ActiveConfigPath string
	Config           Config
}

type Config struct {
	Version               int
	Separator             string
	CopyByDefault         bool
	TokenWarningThreshold int
	Base                  BaseConfig
	Presets               map[string]Preset
}

type BaseConfig struct {
	AlwaysInclude []string `yaml:"always_include"`
}

type Preset struct {
	Description string            `yaml:"description"`
	Blocks      []string          `yaml:"blocks"`
	DefaultVars map[string]string `yaml:"default_vars"`
}

type fileConfig struct {
	Version               int               `yaml:"version,omitempty"`
	Separator             string            `yaml:"separator,omitempty"`
	CopyByDefault         *bool             `yaml:"copy_by_default,omitempty"`
	TokenWarningThreshold int               `yaml:"token_warning_threshold,omitempty"`
	Base                  *baseFileConfig   `yaml:"base,omitempty"`
	Presets               map[string]Preset `yaml:"presets,omitempty"`
}

type baseFileConfig struct {
	AlwaysInclude *[]string `yaml:"always_include,omitempty"`
}

func Default() Config {
	return Config{
		Version:               1,
		Separator:             "\n\n",
		CopyByDefault:         true,
		TokenWarningThreshold: 24000,
		Base:                  BaseConfig{},
		Presets:               map[string]Preset{},
	}
}

func LoadActive(cwd string) (Active, error) {
	projectRoot, globalRoot, err := discoverRoots(cwd)
	if err != nil {
		return Active{}, err
	}

	projectPath := ""
	globalPath := ""
	if projectRoot != "" {
		projectPath = filepath.Join(projectRoot, "config.yaml")
	}
	if globalRoot != "" {
		globalPath = filepath.Join(globalRoot, "config.yaml")
	}
	if projectPath == "" && globalPath == "" {
		return Active{}, ErrConfigNotFound
	}

	cfg := Default()
	activePath := globalPath
	if globalPath != "" {
		globalCfg, err := loadFile(globalPath)
		if err != nil {
			return Active{}, err
		}
		cfg = merge(cfg, globalCfg)
	}
	if projectPath != "" {
		projectCfg, err := loadFile(projectPath)
		if err != nil {
			return Active{}, err
		}
		cfg = merge(cfg, projectCfg)
		activePath = projectPath
	}

	return Active{
		ProjectRoot:      projectRoot,
		GlobalRoot:       globalRoot,
		ActiveConfigPath: activePath,
		Config:           cfg,
	}, nil
}

func (a Active) WritableConfigPath() string {
	return a.ActiveConfigPath
}

func (a Active) BaseRoots() []block.Root {
	var roots []block.Root
	if a.GlobalRoot != "" {
		roots = append(roots, block.Root{Dir: filepath.Join(a.GlobalRoot, "base"), Source: "global"})
	}
	if a.ProjectRoot != "" {
		roots = append(roots, block.Root{Dir: filepath.Join(a.ProjectRoot, "base"), Source: "project"})
	}
	return roots
}

func (a Active) BlockRoots() []block.Root {
	var roots []block.Root
	if a.GlobalRoot != "" {
		roots = append(roots, block.Root{Dir: filepath.Join(a.GlobalRoot, "blocks"), Source: "global"})
	}
	if a.ProjectRoot != "" {
		roots = append(roots, block.Root{Dir: filepath.Join(a.ProjectRoot, "blocks"), Source: "project"})
	}
	return roots
}

func discoverRoots(cwd string) (string, string, error) {
	absCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", err
	}

	projectRoot, err := discoverProjectRoot(absCWD)
	if err != nil {
		return "", "", err
	}

	home, err := userHomeDir()
	if err != nil {
		if projectRoot != "" {
			return projectRoot, "", nil
		}
		return "", "", err
	}
	globalRoot := filepath.Join(home, ".pmp")
	if _, err := os.Stat(filepath.Join(globalRoot, "config.yaml")); err != nil {
		if !os.IsNotExist(err) {
			return "", "", err
		}
		globalRoot = ""
	}

	return projectRoot, globalRoot, nil
}

func discoverProjectRoot(start string) (string, error) {
	dir := start
	for {
		projectRoot := filepath.Join(dir, ".pmp")
		if _, err := os.Stat(filepath.Join(projectRoot, "config.yaml")); err == nil {
			return projectRoot, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func loadFile(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func SavePreset(active Active, name string, preset Preset) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("preset name is required")
	}

	target := active.WritableConfigPath()
	if target == "" {
		return ErrConfigNotFound
	}

	cfg, err := loadFile(target)
	if err != nil {
		return err
	}
	if cfg.Presets == nil {
		cfg.Presets = map[string]Preset{}
	}
	cfg.Presets[name] = preset

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config %s: %w", target, err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", target, err)
	}
	return nil
}

func merge(base Config, override fileConfig) Config {
	result := base
	if override.Version != 0 {
		result.Version = override.Version
	}
	if override.Separator != "" {
		result.Separator = override.Separator
	}
	if override.CopyByDefault != nil {
		result.CopyByDefault = *override.CopyByDefault
	}
	if override.TokenWarningThreshold != 0 {
		result.TokenWarningThreshold = override.TokenWarningThreshold
	}
	if override.Base != nil && override.Base.AlwaysInclude != nil {
		result.Base.AlwaysInclude = append([]string(nil), (*override.Base.AlwaysInclude)...)
	}
	if len(override.Presets) > 0 {
		for name, preset := range override.Presets {
			result.Presets[name] = preset
		}
	}
	return result
}
