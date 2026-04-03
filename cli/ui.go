package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/singl3focus/pmp/internal/config"
	"github.com/singl3focus/pmp/internal/interactive"
)

func newUICommand() *cobra.Command {
	return &cobra.Command{
		Use:           "ui",
		Short:         "Launch the interactive builder",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUI()
		},
	}
}

func runUI() error {
	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	return interactive.Run(active, os.Stdin, os.Stdout)
}
