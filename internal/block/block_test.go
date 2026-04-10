package block

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileWithFrontMatter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "senior-dev.md")
	content := "---\ntitle: Senior\ndescription: Strong defaults\ntags: [go, backend]\nweight: 10\nhidden: false\n---\nUse clean architecture.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	item, err := LoadFile(path, "intro/senior-dev.md", "project")
	if err != nil {
		t.Fatalf("load file: %v", err)
	}

	if item.Title != "Senior" {
		t.Fatalf("unexpected title %q", item.Title)
	}
	if item.Description != "Strong defaults" {
		t.Fatalf("unexpected description %q", item.Description)
	}
	if item.Category != "intro" {
		t.Fatalf("unexpected category %q", item.Category)
	}
	if item.Content != "Use clean architecture." {
		t.Fatalf("unexpected content %q", item.Content)
	}
}

func TestLoadMergedPrefersProjectRoot(t *testing.T) {
	t.Parallel()

	globalDir := filepath.Join(t.TempDir(), "blocks")
	projectDir := filepath.Join(t.TempDir(), "blocks")
	if err := os.MkdirAll(filepath.Join(globalDir, "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	if err := os.WriteFile(filepath.Join(globalDir, "tasks", "feature.md"), []byte("global"), 0o644); err != nil {
		t.Fatalf("write global: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "tasks", "feature.md"), []byte("project"), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}

	blocks, err := LoadMerged([]Root{
		{Dir: globalDir, Source: "global"},
		{Dir: projectDir, Source: "project"},
	})
	if err != nil {
		t.Fatalf("load merged: %v", err)
	}

	item := blocks["tasks/feature.md"]
	if item.Content != "project" {
		t.Fatalf("expected project override, got %q", item.Content)
	}
}

func TestResolveRejectsPathsOutsideRoot(t *testing.T) {
	t.Parallel()

	rootDir := filepath.Join(t.TempDir(), "blocks")
	if err := os.MkdirAll(filepath.Join(rootDir, "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	_, err := Resolve("../../../README.md", []Root{{Dir: rootDir, Source: "project"}})
	if err == nil {
		t.Fatalf("expected path traversal to be rejected")
	}
	if !strings.Contains(err.Error(), "escapes the block root") {
		t.Fatalf("unexpected error %q", err)
	}
}

func TestNeedsRenderUsesExplicitFrontMatter(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		template *bool
		content  string
		want     bool
	}{
		{"explicit true", &trueVal, "plain text", true},
		{"explicit false", &falseVal, "{{ .Vars.name }}", false},
		{"nil with template syntax", nil, "Hello {{ .Date }}", true},
		{"nil without template syntax", nil, "Hello {{ range $i }}{{ end }}", false},
		{"nil plain text", nil, "Just plain text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := Block{Template: tt.template, Content: tt.content}
			if got := b.NeedsRender(); got != tt.want {
				t.Fatalf("NeedsRender() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadFileStripsUTF8BOMBeforeParsingFrontMatter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "feature.md")
	content := string([]byte{0xEF, 0xBB, 0xBF}) + "---\ntitle: Feature\ndescription: Parsed metadata\n---\nBody text\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	item, err := LoadFile(path, "tasks/feature.md", "project")
	if err != nil {
		t.Fatalf("load file: %v", err)
	}

	if item.Title != "Feature" {
		t.Fatalf("unexpected title %q", item.Title)
	}
	if item.Description != "Parsed metadata" {
		t.Fatalf("unexpected description %q", item.Description)
	}
	if item.Content != "Body text" {
		t.Fatalf("unexpected content %q", item.Content)
	}
}

func TestSortedBlocksUsesVisibilityWeightAndLabel(t *testing.T) {
	t.Parallel()

	blocks := map[string]Block{
		"tasks/zeta.md":   {Path: "tasks/zeta.md", Title: "Zeta", Weight: 20},
		"tasks/alpha.md":  {Path: "tasks/alpha.md", Title: "Alpha", Weight: 10},
		"tools/hidden.md": {Path: "tools/hidden.md", Title: "Hidden", Weight: 1, Hidden: true},
		"tasks/plain.md":  {Path: "tasks/plain.md", Weight: 10},
	}

	got := SortedBlocks(blocks, false)
	if len(got) != 3 {
		t.Fatalf("expected hidden blocks to be excluded, got %d entries", len(got))
	}

	wantPaths := []string{"tasks/alpha.md", "tasks/plain.md", "tasks/zeta.md"}
	for idx, want := range wantPaths {
		if got[idx].Path != want {
			t.Fatalf("entry %d path = %q, want %q", idx, got[idx].Path, want)
		}
	}
}

func TestSortedBlocksCanIncludeHiddenEntries(t *testing.T) {
	t.Parallel()

	blocks := map[string]Block{
		"tasks/visible.md": {Path: "tasks/visible.md", Title: "Visible", Weight: 10},
		"tasks/hidden.md":  {Path: "tasks/hidden.md", Title: "Hidden", Weight: 0, Hidden: true},
	}

	got := SortedBlocks(blocks, true)
	if len(got) != 2 {
		t.Fatalf("expected hidden blocks to be included, got %d entries", len(got))
	}
	if got[0].Path != "tasks/visible.md" || got[1].Path != "tasks/hidden.md" {
		t.Fatalf("unexpected order: %#v", got)
	}
}
