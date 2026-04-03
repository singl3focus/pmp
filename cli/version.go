package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(build VersionInfo) *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print version",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printVersion(build)
		},
	}
}

func printVersion(build VersionInfo) error {
	fmt.Printf("version: %s\n", build.Version)
	fmt.Printf("commit: %s\n", build.Commit)
	fmt.Printf("date: %s\n", build.Date)
	return nil
}
