package output

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/singl3focus/pmp/internal/engine"
)

type Mode string

const (
	ModeClipboard Mode = "clipboard"
	ModeStdout    Mode = "stdout"
	ModeFile      Mode = "file"
)

type Options struct {
	NoCopy  bool
	OutFile string
	JSON    bool
}

var clipboardCommandFunc = clipboardCommand

func Emit(result engine.BuildResult, opts Options) (Mode, error) {
	payload, err := marshal(result, opts.JSON)
	if err != nil {
		return "", err
	}

	if opts.OutFile != "" {
		if err := os.WriteFile(opts.OutFile, payload, 0o644); err != nil {
			return "", fmt.Errorf("write output file: %w", err)
		}
		return ModeFile, nil
	}

	if opts.JSON || opts.NoCopy || !clipboardAvailable() {
		if err := writeStdout(payload); err != nil {
			return "", err
		}
		return ModeStdout, nil
	}

	if err := writeClipboard(payload); err != nil {
		clipboardErr := err
		if err := writeStdout(payload); err != nil {
			return "", fmt.Errorf("clipboard failed (%w), stdout fallback also failed: %v", clipboardErr, err)
		}
		return ModeStdout, nil
	}
	return ModeClipboard, nil
}

func CheckClipboard() error {
	if !clipboardAvailable() {
		return fmt.Errorf("clipboard command is not available")
	}
	return nil
}

func marshal(result engine.BuildResult, asJSON bool) ([]byte, error) {
	if !asJSON {
		return []byte(result.Prompt), nil
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal json output: %w", err)
	}
	return data, nil
}

func writeClipboard(data []byte) error {
	cmd := clipboardCommandFunc()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open clipboard stdin: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start clipboard command: %w", err)
	}

	payload := encodeForClipboard(data)
	if _, err := stdin.Write(payload); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("write clipboard content: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("close clipboard stdin: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("clipboard command failed: %w", err)
	}
	return nil
}

func writeStdout(data []byte) error {
	if _, err := os.Stdout.Write(data); err != nil {
		return err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		if _, err := os.Stdout.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
}

func clipboardAvailable() bool {
	return clipboardCommandFunc().Path != ""
}

func clipboardCommand() *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		path, err := exec.LookPath("clip")
		if err != nil {
			return &exec.Cmd{}
		}
		return exec.Command(path)
	case "darwin":
		path, err := exec.LookPath("pbcopy")
		if err != nil {
			return &exec.Cmd{}
		}
		return exec.Command(path)
	default:
		if path, err := exec.LookPath("xclip"); err == nil {
			return exec.Command(path, "-selection", "clipboard")
		}
		if path, err := exec.LookPath("xsel"); err == nil {
			return exec.Command(path, "--clipboard", "--input")
		}
		return &exec.Cmd{}
	}
}
