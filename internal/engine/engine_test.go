package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/singl3focus/pmp/internal/config"
)

func TestBuildPlacesMessageFirst(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(filepath.Join(projectRoot, "base"), 0o755); err != nil {
		t.Fatalf("mkdir base: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "blocks", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}

	if err := os.WriteFile(filepath.Join(projectRoot, "base", "global.md"), []byte("Global context"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "blocks", "tasks", "feature.md"), []byte("Implement feature"), 0o644); err != nil {
		t.Fatalf("write block: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte("base:\n  always_include:\n    - global.md\npresets:\n  feature:\n    blocks:\n      - tasks/feature.md\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	active, err := config.LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	result, err := Build(BuildRequest{
		PresetName: "feature",
		Message:    "Add profiles",
	}, active)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if !strings.HasPrefix(result.Prompt, "Add profiles") {
		t.Fatalf("expected message first, got %q", result.Prompt)
	}
	if len(result.BlocksUsed) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(result.BlocksUsed))
	}
}

func TestBuildWarnsOnTokenLimit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(filepath.Join(projectRoot, "blocks", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "blocks", "tasks", "feature.md"), []byte(strings.Repeat("word ", 100)), 0o644); err != nil {
		t.Fatalf("write block: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte("presets:\n  feature:\n    blocks:\n      - tasks/feature.md\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	active, err := config.LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	result, err := Build(BuildRequest{
		PresetName: "feature",
		TokenLimit: 10,
	}, active)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected token warning")
	}
}
