package cli

import (
	"flag"
	"os"

	"github.com/singl3focus/pmp/internal/config"
	"github.com/singl3focus/pmp/internal/interactive"
)

func runUI(args []string) error {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	return interactive.Run(active, os.Stdin, os.Stdout)
}
