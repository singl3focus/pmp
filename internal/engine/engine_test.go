package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/singl3focus/pmp/internal/config"
)

func TestBuildPlacesMessageBottom(t *testing.T) {
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

	// Default message_position is "bottom".
	result, err := Build(BuildRequest{
		PresetName: "feature",
		Message:    "Add profiles",
	}, active)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if !strings.HasSuffix(result.Prompt, "Add profiles") {
		t.Fatalf("expected message last (bottom), got %q", result.Prompt)
	}
	if len(result.BlocksUsed) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(result.BlocksUsed))
	}
}

func TestBuildPlacesMessageTop(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte("message_position: top\nbase:\n  always_include:\n    - global.md\npresets:\n  feature:\n    blocks:\n      - tasks/feature.md\n"), 0o644); err != nil {
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
		t.Fatalf("expected message first (top), got %q", result.Prompt)
	}
}

func TestBuildSkipsTemplateRenderingForPlainBlocks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(filepath.Join(projectRoot, "blocks", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}

	// Block contains {{ }} that would break text/template, but has no "{{ ."
	// so it should be treated as plain text.
	content := "Use {{ range $i }}{{ $i }}{{ end }} in your Helm chart."
	if err := os.WriteFile(filepath.Join(projectRoot, "blocks", "tasks", "helm.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write block: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte("presets:\n  helm:\n    blocks:\n      - tasks/helm.md\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	active, err := config.LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	result, err := Build(BuildRequest{PresetName: "helm"}, active)
	if err != nil {
		t.Fatalf("build should not fail on plain block with curly braces: %v", err)
	}
	if !strings.Contains(result.Prompt, "{{ range $i }}") {
		t.Fatalf("expected literal template syntax preserved, got %q", result.Prompt)
	}
}

func TestCountTokensUsesRealTokenizer(t *testing.T) {
	t.Parallel()

	// "hello world" is 2 tokens in cl100k_base.
	got := countTokens("hello world")
	if got != 2 {
		t.Fatalf("expected 2 tokens for 'hello world', got %d", got)
	}

	if countTokens("") != 0 {
		t.Fatal("expected 0 tokens for empty string")
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
