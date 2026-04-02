package output

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/singl3focus/pmp/internal/engine"
)

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	payload, err := marshal(engine.BuildResult{
		PresetName:      "feature",
		Prompt:          "hello",
		BlocksUsed:      []string{"a.md"},
		EstimatedTokens: 1,
	}, true)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if decoded["preset_name"] != "feature" {
		t.Fatalf("unexpected preset_name %#v", decoded["preset_name"])
	}
}

func TestEmitFallsBackToStdoutWhenClipboardUnavailable(t *testing.T) {
	previousClipboardCommand := clipboardCommandFunc
	clipboardCommandFunc = func() *exec.Cmd {
		return &exec.Cmd{}
	}
	t.Cleanup(func() {
		clipboardCommandFunc = previousClipboardCommand
	})

	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = previousStdout
	})

	mode, err := Emit(engine.BuildResult{Prompt: "hello"}, Options{})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if mode != ModeStdout {
		t.Fatalf("expected stdout mode, got %q", mode)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected stdout payload %q", string(data))
	}
}

func TestEmitFallsBackToStdoutWhenClipboardWriteFails(t *testing.T) {
	previousClipboardCommand := clipboardCommandFunc
	clipboardCommandFunc = func() *exec.Cmd {
		return failingClipboardCommand(t)
	}
	t.Cleanup(func() {
		clipboardCommandFunc = previousClipboardCommand
	})

	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = previousStdout
	})

	mode, err := Emit(engine.BuildResult{Prompt: "hello"}, Options{})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if mode != ModeStdout {
		t.Fatalf("expected stdout mode, got %q", mode)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected stdout payload %q", string(data))
	}
}

func TestEmitReturnsCombinedErrorWhenClipboardAndStdoutFail(t *testing.T) {
	previousClipboardCommand := clipboardCommandFunc
	clipboardCommandFunc = func() *exec.Cmd {
		return failingClipboardCommand(t)
	}
	t.Cleanup(func() {
		clipboardCommandFunc = previousClipboardCommand
	})

	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	_ = reader.Close()
	_ = writer.Close()
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = previousStdout
	})

	_, err = Emit(engine.BuildResult{Prompt: "hello"}, Options{})
	if err == nil {
		t.Fatal("expected emit to fail when stdout fallback also fails")
	}
	if !strings.Contains(err.Error(), "stdout fallback also failed") {
		t.Fatalf("expected combined fallback error, got %v", err)
	}
}

func failingClipboardCommand(t *testing.T) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestFailingClipboardHelperProcess", "--")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func TestFailingClipboardHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(1)
}
