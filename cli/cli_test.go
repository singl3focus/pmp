package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteInitBuildListAndDoctor(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	build := VersionInfo{Version: "test", Commit: "abc123", Date: "2026-04-02T00:00:00Z"}

	if err := Execute([]string{"init"}, build); err != nil {
		t.Fatalf("init: %v", err)
	}

	outPath := filepath.Join(tmp, "prompt.md")
	if err := Execute([]string{"--preset", "feature", "-m", "Add profiles", "--out", outPath}, build); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected prompt output file: %v", err)
	}

	if err := Execute([]string{"list"}, build); err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := Execute([]string{"doctor"}, build); err != nil {
		t.Fatalf("doctor: %v", err)
	}
}

func TestExecuteWithoutArgsPrintsRootHelp(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	stdout, err := captureStdout(t, func() error {
		return Execute(nil, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("execute without args: %v", err)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("expected root help on stdout, got %q", stdout)
	}
	if strings.Contains(stdout, "error creating cancelreader") {
		t.Fatalf("expected no TUI startup failure, got %q", stdout)
	}
}

func TestExecuteVersionFlag(t *testing.T) {
	build := VersionInfo{Version: "1.2.3", Commit: "abc", Date: "2026-04-02"}
	for _, flag := range []string{"-v", "--version"} {
		stdout, err := captureStdout(t, func() error {
			return Execute([]string{flag}, build)
		})
		if err != nil {
			t.Fatalf("%s: %v", flag, err)
		}
		if !strings.Contains(stdout, "1.2.3") {
			t.Fatalf("%s: expected version in output, got %q", flag, stdout)
		}
	}
}

func TestExecuteUnknownFirstArgFallsBackToBuildFlow(t *testing.T) {
	err := Execute([]string{"feature"}, VersionInfo{})
	if err == nil {
		t.Fatal("expected build validation error")
	}
	if !strings.Contains(err.Error(), "missing required flag --preset") {
		t.Fatalf("expected build-style validation error, got %v", err)
	}
}

func TestExecuteCompletionCommandRemainsReachable(t *testing.T) {
	root := newRootCommand(VersionInfo{})
	root.SetArgs(normalizeArgs([]string{"completion", "powershell"}))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err != nil {
		t.Fatalf("completion: %v", err)
	}
	if !strings.Contains(stdout.String(), "Register-ArgumentCompleter") {
		t.Fatalf("expected PowerShell completion script, got %q", stdout.String())
	}
}

func TestNormalizeArgsPreservesCobraCompletionCommands(t *testing.T) {
	t.Parallel()

	tests := []string{"completion", "__complete", "__completeNoDesc", "preset"}
	for _, firstArg := range tests {
		firstArg := firstArg
		t.Run(firstArg, func(t *testing.T) {
			t.Parallel()

			args := normalizeArgs([]string{firstArg, "build"})
			if args[0] != firstArg {
				t.Fatalf("expected %q to stay unchanged, got %q", firstArg, args[0])
			}
		})
	}
}

func TestExecuteListHidesHiddenBlocksUnlessFlagIsSet(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	writeProjectConfig(t, tmp, "presets:\n  feature:\n    blocks:\n      - tasks/feature.md\n")
	hiddenPath := filepath.Join(tmp, ".pmp", "blocks", "tasks", "hidden.md")
	hidden := "---\ntitle: Hidden\ndescription: Hidden helper\nhidden: true\n---\nInternal helper\n"
	if err := os.WriteFile(hiddenPath, []byte(hidden), 0o644); err != nil {
		t.Fatalf("write hidden block: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return Execute([]string{"list"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if strings.Contains(stdout, "tasks/hidden.md") {
		t.Fatalf("expected hidden block to be omitted by default, got %q", stdout)
	}
	if !strings.Contains(stdout, "Hidden blocks: 1 not shown") {
		t.Fatalf("expected hidden summary, got %q", stdout)
	}

	stdout, err = captureStdout(t, func() error {
		return Execute([]string{"list", "--show-hidden"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("list show-hidden: %v", err)
	}
	if !strings.Contains(stdout, "tasks/hidden.md") || !strings.Contains(stdout, "hidden") {
		t.Fatalf("expected hidden block metadata, got %q", stdout)
	}
}

func TestExecutePresetAddShowListAndDelete(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	writeProjectConfig(t, tmp, "presets:\n  feature:\n    blocks:\n      - tasks/feature.md\n")
	if err := os.MkdirAll(filepath.Join(tmp, ".pmp", "blocks", "tools"), 0o755); err != nil {
		t.Fatalf("mkdir tools: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".pmp", "blocks", "tools", "dev-tools.md"), []byte("Use dev tools"), 0o644); err != nil {
		t.Fatalf("write dev tools block: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return Execute([]string{"preset", "add", "review", "--description", "Review flow", "--block", "tasks/feature.md", "--block", "tools/dev-tools.md"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("preset add: %v", err)
	}
	if !strings.Contains(stdout, `Saved preset "review" with 2 blocks`) {
		t.Fatalf("unexpected add output %q", stdout)
	}

	stdout, err = captureStdout(t, func() error {
		return Execute([]string{"preset", "show", "review"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("preset show: %v", err)
	}
	if !strings.Contains(stdout, "Name: review") || !strings.Contains(stdout, "tools/dev-tools.md") {
		t.Fatalf("unexpected show output %q", stdout)
	}

	stdout, err = captureStdout(t, func() error {
		return Execute([]string{"preset", "list"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("preset list: %v", err)
	}
	if !strings.Contains(stdout, "review: Review flow") {
		t.Fatalf("unexpected preset list output %q", stdout)
	}

	stdout, err = captureStdout(t, func() error {
		return Execute([]string{"preset", "delete", "review"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("preset delete: %v", err)
	}
	if !strings.Contains(stdout, `Deleted preset "review"`) {
		t.Fatalf("unexpected delete output %q", stdout)
	}

	stdout, err = captureStdout(t, func() error {
		return Execute([]string{"preset", "list"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("preset list after delete: %v", err)
	}
	if strings.Contains(stdout, "review") {
		t.Fatalf("expected deleted preset to disappear, got %q", stdout)
	}
}

func TestExecutePresetCommandsUseWritableConfigOnly(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	writeGlobalConfig(t, home, "presets:\n  review:\n    description: Global review\n    blocks:\n      - tasks/global-review.md\n")
	writeProjectConfig(t, tmp, "presets:\n  feature:\n    description: Local feature\n    blocks:\n      - tasks/feature.md\n")

	stdout, err := captureStdout(t, func() error {
		return Execute([]string{"preset", "list"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("preset list: %v", err)
	}
	if !strings.Contains(stdout, "feature: Local feature") {
		t.Fatalf("expected local preset in output, got %q", stdout)
	}
	if strings.Contains(stdout, "Global review") || strings.Contains(stdout, "review") {
		t.Fatalf("expected global-only preset to be hidden, got %q", stdout)
	}

	_, err = captureStdout(t, func() error {
		return Execute([]string{"preset", "show", "review"}, VersionInfo{})
	})
	if err == nil {
		t.Fatal("expected preset show to reject global-only preset")
	}
	if !strings.Contains(err.Error(), `preset "review" not found`) {
		t.Fatalf("unexpected preset show error: %v", err)
	}
}

func TestExecutePresetAddRejectsUnknownBlocks(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	writeProjectConfig(t, tmp, "presets:\n  feature:\n    blocks:\n      - tasks/feature.md\n")

	_, err = captureStdout(t, func() error {
		return Execute([]string{"preset", "add", "review", "--block", "tasks/missing.md"}, VersionInfo{})
	})
	if err == nil {
		t.Fatal("expected preset add to fail for unknown block")
	}
	if !strings.Contains(err.Error(), `resolve block "tasks/missing.md"`) {
		t.Fatalf("unexpected preset add error: %v", err)
	}

	configData, readErr := os.ReadFile(filepath.Join(tmp, ".pmp", "config.yaml"))
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if strings.Contains(string(configData), "review:") {
		t.Fatalf("expected invalid preset not to be written, got %q", string(configData))
	}
}

func TestExecuteBuildUsesStdoutWhenCopyDisabledByConfig(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	writeProjectConfig(t, tmp, "copy_by_default: false\nbase:\n  always_include:\n    - global.md\npresets:\n  feature:\n    blocks:\n      - tasks/feature.md\n")

	build := VersionInfo{Version: "test", Commit: "abc123", Date: "2026-04-02T00:00:00Z"}
	stdout, err := captureStdout(t, func() error {
		return Execute([]string{"--preset", "feature", "-m", "Add profiles"}, build)
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if !strings.Contains(stdout, "Add profiles") {
		t.Fatalf("expected prompt on stdout, got %q", stdout)
	}
	if strings.Contains(stdout, "Output: clipboard") {
		t.Fatalf("expected stdout output, got %q", stdout)
	}
}

func TestExecuteBuildDryRunJSONWritesJSON(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	writeProjectConfig(t, tmp, "presets:\n  feature:\n    blocks:\n      - tasks/feature.md\n")

	stdout, err := captureStdout(t, func() error {
		return Execute([]string{"build", "--preset", "feature", "-m", "Add profiles", "--dry-run", "--json"}, VersionInfo{})
	})
	if err != nil {
		t.Fatalf("build dry-run json: %v", err)
	}
	if !strings.Contains(stdout, "\"preset_name\": \"feature\"") {
		t.Fatalf("expected json output, got %q", stdout)
	}
	if strings.Contains(stdout, "Build plan") {
		t.Fatalf("expected structured json instead of dry-run text, got %q", stdout)
	}
}

func TestExecuteDoctorReportsMissingBaseBlocks(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	projectRoot := filepath.Join(tmp, ".pmp")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	configData := "base:\n  always_include:\n    - missing.md\npresets:\n  feature:\n    blocks: []\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte(configData), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	build := VersionInfo{Version: "test", Commit: "abc123", Date: "2026-04-02T00:00:00Z"}
	stdout, err := captureStdout(t, func() error {
		return Execute([]string{"doctor"}, build)
	})
	if err == nil {
		t.Fatalf("expected doctor to fail for missing blocks")
	}

	if !strings.Contains(stdout, "missing: missing.md") {
		t.Fatalf("expected missing base block in doctor output, got %q", stdout)
	}
	if !strings.Contains(err.Error(), "doctor found configuration issues") {
		t.Fatalf("unexpected doctor error %v", err)
	}
}

func TestExecuteDoctorReportsMalformedUnreferencedBlocks(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	if err := Execute([]string{"init"}, VersionInfo{}); err != nil {
		t.Fatalf("init: %v", err)
	}

	brokenPath := filepath.Join(tmp, ".pmp", "blocks", "tasks", "broken.md")
	broken := "---\ntitle: broken\ntags: [unterminated\n---\nbody\n"
	if err := os.WriteFile(brokenPath, []byte(broken), 0o644); err != nil {
		t.Fatalf("write broken block: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		return Execute([]string{"doctor"}, VersionInfo{})
	})
	if err == nil {
		t.Fatal("expected doctor to fail for malformed unreferenced block")
	}
	if !strings.Contains(stdout, "Block library") || !strings.Contains(stdout, "invalid front matter") {
		t.Fatalf("expected malformed block to be reported, got %q", stdout)
	}
}

func TestExecuteInitUsesRepositoryRootFromNestedDirectory(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git root: %v", err)
	}

	nested := filepath.Join(tmp, "pkg", "feature")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	if err := Execute([]string{"init"}, VersionInfo{}); err != nil {
		t.Fatalf("init: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, ".pmp", "config.yaml")); err != nil {
		t.Fatalf("expected config at repository root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nested, ".pmp", "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected no nested config, got %v", err)
	}
}

func writeProjectConfig(t *testing.T, root, configData string) {
	t.Helper()

	projectRoot := filepath.Join(root, ".pmp")
	if err := os.MkdirAll(filepath.Join(projectRoot, "base"), 0o755); err != nil {
		t.Fatalf("mkdir base: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "blocks", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte(configData), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "base", "global.md"), []byte("Global context"), 0o644); err != nil {
		t.Fatalf("write base block: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "blocks", "tasks", "feature.md"), []byte("Implement feature"), 0o644); err != nil {
		t.Fatalf("write feature block: %v", err)
	}
}

func writeGlobalConfig(t *testing.T, home, configData string) {
	t.Helper()

	globalRoot := filepath.Join(home, ".pmp")
	if err := os.MkdirAll(filepath.Join(globalRoot, "blocks", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir global tasks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(globalRoot, "config.yaml"), []byte(configData), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(globalRoot, "blocks", "tasks", "global-review.md"), []byte("Global review"), 0o644); err != nil {
		t.Fatalf("write global review block: %v", err)
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer

	defer func() {
		os.Stdout = previousStdout
	}()

	runErr := fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}

	return string(data), runErr
}
