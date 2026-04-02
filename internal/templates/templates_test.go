package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldCreatesExpectedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Scaffold(dir); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	expected := []string{
		"config.yaml",
		filepath.Join("base", "global.md"),
		filepath.Join("blocks", "intro", "senior-dev.md"),
		filepath.Join("blocks", "communication", "concise.md"),
		filepath.Join("blocks", "communication", "detailed.md"),
		filepath.Join("blocks", "tools", "dev-tools.md"),
		filepath.Join("blocks", "tasks", "feature.md"),
		filepath.Join("blocks", "tasks", "review.md"),
		filepath.Join("blocks", "tasks", "bugfix.md"),
	}

	for _, rel := range expected {
		path := filepath.Join(dir, rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("missing expected file %s: %v", rel, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected non-empty file %s", rel)
		}
	}
}

func TestScaffoldIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Scaffold(dir); err != nil {
		t.Fatalf("first scaffold: %v", err)
	}

	// Overwrite one file with custom content.
	custom := filepath.Join(dir, "config.yaml")
	customContent := []byte("# my custom config\n")
	if err := os.WriteFile(custom, customContent, 0o644); err != nil {
		t.Fatalf("write custom: %v", err)
	}

	// Second scaffold should NOT overwrite the existing file.
	if err := Scaffold(dir); err != nil {
		t.Fatalf("second scaffold: %v", err)
	}

	data, err := os.ReadFile(custom)
	if err != nil {
		t.Fatalf("read custom: %v", err)
	}
	if string(data) != string(customContent) {
		t.Fatalf("scaffold overwrote existing file; got %q", string(data))
	}
}
