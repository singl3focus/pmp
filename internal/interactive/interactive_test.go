package interactive

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singl3focus/pmp/internal/config"
	"github.com/singl3focus/pmp/internal/output"
)

func TestFilterBlocksMatchesPathDescriptionAndTags(t *testing.T) {
	t.Parallel()

	entries := []blockEntry{
		{Path: "tasks/feature.md", Description: "Feature work", Tags: []string{"task", "feature"}},
		{Path: "tools/dev-tools.md", Description: "Tooling guidance", Tags: []string{"tools"}},
	}

	tests := []struct {
		name   string
		filter string
		want   []blockEntry
	}{
		{name: "path", filter: "feature", want: []blockEntry{entries[0]}},
		{name: "description", filter: "tooling", want: []blockEntry{entries[1]}},
		{name: "tag", filter: "task", want: []blockEntry{entries[0]}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := filterBlocks(entries, tt.filter)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected filter result: got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestDeleteLastRuneHandlesUnicode(t *testing.T) {
	t.Parallel()

	if got := deleteLastRune("\u041f\u0440\u0438\u0432\u0435\u0442"); got != "\u041f\u0440\u0438\u0432\u0435" {
		t.Fatalf("unexpected deleteLastRune result %q", got)
	}
}

func TestKeyTextAcceptsPastedRunes(t *testing.T) {
	t.Parallel()

	got, ok := keyText(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Add profiles"), Paste: true})
	if !ok {
		t.Fatalf("expected pasted text to be accepted")
	}
	if got != "Add profiles" {
		t.Fatalf("unexpected pasted text %q", got)
	}
}

func TestUpdateOutputDoesNotFinishWhenBuildHasError(t *testing.T) {
	t.Parallel()

	active := mustLoadActiveForInteractiveTest(t, "presets:\n  broken:\n    blocks:\n      - tasks/missing.md\n")
	m, err := newModel(active)
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.presetIndex = 1
	m.step = stepOutput
	m.rebuild()

	updated, _ := m.updateOutput(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(model)
	if next.finished {
		t.Fatal("expected model to stay open when buildErr is set")
	}
	if next.statusMessage == "" {
		t.Fatal("expected status message explaining why output cannot be confirmed")
	}
}

func TestOutputOptionsFollowSelectedMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		outputIndex int
		filePath    string
		want        output.Options
	}{
		{
			name:        "clipboard ignores default file path",
			outputIndex: 0,
			filePath:    "prompt.md",
			want:        output.Options{},
		},
		{
			name:        "stdout enables no copy only",
			outputIndex: 1,
			filePath:    "prompt.md",
			want:        output.Options{NoCopy: true},
		},
		{
			name:        "file uses trimmed file path",
			outputIndex: 2,
			filePath:    "  prompt.md  ",
			want:        output.Options{OutFile: "prompt.md"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := outputOptions(tt.outputIndex, tt.filePath)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected output options: got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestNewModelStartsWithEmptyFilePath(t *testing.T) {
	t.Parallel()

	m, err := newModel(mustLoadActiveForInteractiveTest(t, "version: 1\n"))
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	if m.filePath != "" {
		t.Fatalf("expected empty default file path, got %q", m.filePath)
	}
}

func TestUpdateOutputSeedsDefaultFilePathWhenEnteringFileMode(t *testing.T) {
	t.Parallel()

	m := model{outputIndex: 1}
	updated, _ := m.updateOutput(tea.KeyMsg{Type: tea.KeyDown})
	next := updated.(model)

	if next.outputIndex != 2 {
		t.Fatalf("expected file output to be selected, got %d", next.outputIndex)
	}
	if next.filePath != defaultFilePath() {
		t.Fatalf("expected default file path %q, got %q", defaultFilePath(), next.filePath)
	}
}

func TestSaveCurrentPresetPreservesDefaultVars(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(filepath.Join(projectRoot, "blocks", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "blocks", "tasks", "feature.md"), []byte("{{ index .Vars \"audience\" }}"), 0o644); err != nil {
		t.Fatalf("write block: %v", err)
	}

	configData := "presets:\n  feature:\n    description: feature\n    default_vars:\n      audience: team\n    blocks:\n      - tasks/feature.md\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte(configData), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	active, err := config.LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}

	m, err := newModel(active)
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.presetIndex = 1
	m.saveName = "feature"
	m.saveDescription = "updated description"

	updated, _ := m.saveCurrentPreset()
	next := updated.(*model)

	reloaded, err := config.LoadActive(root)
	if err != nil {
		t.Fatalf("reload active: %v", err)
	}

	if got := reloaded.Config.Presets["feature"].DefaultVars["audience"]; got != "team" {
		t.Fatalf("expected default_vars to be preserved, got %q", got)
	}
	if got := next.active.Config.Presets["feature"].DefaultVars["audience"]; got != "team" {
		t.Fatalf("expected in-memory preset vars to be preserved, got %q", got)
	}
}

func mustLoadActiveForInteractiveTest(t *testing.T, configData string) config.Active {
	t.Helper()

	root := t.TempDir()
	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(filepath.Join(projectRoot, "blocks", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte(configData), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	active, err := config.LoadActive(root)
	if err != nil {
		t.Fatalf("load active: %v", err)
	}
	return active
}
