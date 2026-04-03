package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type VersionInfo struct {
	Version string
	Commit  string
	Date    string
}

func Execute(args []string, build VersionInfo) error {
	root := newRootCommand(build)
	root.SetArgs(normalizeArgs(args))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	return root.Execute()
}

type rootOptions struct {
	build   buildFlags
	version bool
}

func newRootCommand(build VersionInfo) *cobra.Command {
	opts := rootOptions{}

	cmd := &cobra.Command{
		Use:           "pmp",
		Short:         "Assemble prompts from reusable markdown blocks.",
		Long:          "pmp assembles prompts from reusable markdown blocks.",
		Example:       "  pmp --preset feature -m \"Add CSV export\"\n  pmp -p review -m \"Review auth flow\"\n  pmp build --preset bugfix -m \"Fix race condition\"",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.version {
				return printVersion(build)
			}
			if len(args) == 0 && !opts.build.hasAnyFlags() {
				return cmd.Help()
			}
			return runBuild(opts.build)
		},
	}

	bindBuildFlags(cmd.Flags(), &opts.build)
	cmd.Flags().BoolVarP(&opts.version, "version", "v", false, "print version")

	cmd.AddCommand(
		newBuildCommand(),
		newInitCommand(),
		newListCommand(),
		newDoctorCommand(),
		newVersionCommand(build),
		newUICommand(),
	)

	return cmd
}

func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	if shouldUseImplicitBuild(args[0]) {
		return append([]string{"build"}, args...)
	}
	return args
}

func shouldUseImplicitBuild(arg string) bool {
	if strings.HasPrefix(arg, "-") {
		return false
	}

	switch arg {
	case "build", "init", "list", "doctor", "version", "ui", "help", "completion", "__complete", "__completeNoDesc":
		return false
	default:
		return true
	}
}
