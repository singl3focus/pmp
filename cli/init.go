package cli

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/singl3focus/pmp/internal/templates"
)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	global := fs.Bool("global", false, "write to ~/.pmp instead of ./.pmp")
	if err := fs.Parse(args); err != nil {
		return err
	}

	targetRoot, err := resolveInitRoot(*global)
	if err != nil {
		return err
	}

	if err := templates.Scaffold(targetRoot); err != nil {
		return err
	}

	_, err = os.Stdout.WriteString("Initialized " + filepath.Clean(targetRoot) + "\n")
	return err
}

func resolveInitRoot(global bool) (string, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".pmp"), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	root, err := discoverInitProjectBase(cwd)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".pmp"), nil
}

func discoverInitProjectBase(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	var vcsRoot string
	for {
		pmpRoot := filepath.Join(dir, ".pmp")
		if _, err := os.Stat(filepath.Join(pmpRoot, "config.yaml")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		if vcsRoot == "" {
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				vcsRoot = dir
			} else if !os.IsNotExist(err) {
				return "", err
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if vcsRoot != "" {
		return vcsRoot, nil
	}
	return filepath.Clean(start), nil
}
