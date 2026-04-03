package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/singl3focus/pmp/internal/config"
	"github.com/singl3focus/pmp/internal/engine"
	"github.com/singl3focus/pmp/internal/output"
)

type buildFlags struct {
	preset     string
	message    string
	blocks     csvFlag
	vars       kvFlag
	tokenLimit int
	dryRun     bool
	noCopy     bool
	out        string
	json       bool
}

func newBuildCommand() *cobra.Command {
	flags := buildFlags{}

	cmd := &cobra.Command{
		Use:           "build",
		Short:         "Assemble a prompt explicitly",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(flags)
		},
	}

	bindBuildFlags(cmd.Flags(), &flags)
	return cmd
}

func bindBuildFlags(fs *pflag.FlagSet, flags *buildFlags) {
	fs.StringVarP(&flags.preset, "preset", "p", "", "preset name")
	fs.StringVarP(&flags.message, "message", "m", "", "task message")
	fs.Var(&flags.blocks, "block", "comma separated extra block paths")
	fs.Var(&flags.vars, "var", "template variable key=value")
	fs.IntVar(&flags.tokenLimit, "token-limit", 0, "warn when estimated tokens exceed this limit")
	fs.BoolVar(&flags.dryRun, "dry-run", false, "show resolved build plan without output")
	fs.BoolVar(&flags.noCopy, "no-copy", false, "print prompt to stdout instead of copying")
	fs.StringVar(&flags.out, "out", "", "write prompt or json result to file")
	fs.BoolVar(&flags.json, "json", false, "emit json result")
}

func runBuild(flags buildFlags) error {
	if flags.preset == "" {
		return fmt.Errorf("missing required flag --preset")
	}

	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	result, err := engine.Build(engine.BuildRequest{
		PresetName:  flags.preset,
		Message:     flags.message,
		ExtraBlocks: flags.blocks.Values(),
		Vars:        flags.vars.Values(),
		TokenLimit:  flags.tokenLimit,
		DryRun:      flags.dryRun,
	}, active)
	if err != nil {
		return err
	}

	if flags.dryRun {
		if flags.json {
			_, err := output.Emit(result, output.Options{
				NoCopy:  true,
				OutFile: flags.out,
				JSON:    true,
			})
			return err
		}
		return printDryRun(result)
	}

	noCopy := flags.noCopy || !active.Config.CopyByDefault
	mode, err := output.Emit(result, output.Options{
		NoCopy:  noCopy,
		OutFile: flags.out,
		JSON:    flags.json,
	})
	if err != nil {
		return err
	}

	if mode == output.ModeStdout {
		return nil
	}

	return printBuildSummary(result, mode)
}

func (f buildFlags) hasAnyFlags() bool {
	return f.preset != "" ||
		f.message != "" ||
		len(f.blocks.values) > 0 ||
		len(f.vars.values) > 0 ||
		f.tokenLimit != 0 ||
		f.dryRun ||
		f.noCopy ||
		f.out != "" ||
		f.json
}

func printDryRun(result engine.BuildResult) error {
	fmt.Println("Build plan")
	fmt.Printf("Preset: %s\n", result.PresetName)
	if result.Message != "" {
		fmt.Printf("Message: %s\n", result.Message)
	}
	fmt.Printf("Blocks (%d):\n", len(result.BlocksUsed))
	for _, path := range result.BlocksUsed {
		fmt.Printf("  - %s\n", path)
	}
	fmt.Printf("Estimated tokens: %d\n", result.EstimatedTokens)
	for _, warning := range result.Warnings {
		fmt.Printf("Warning: %s\n", warning)
	}
	return nil
}

func printBuildSummary(result engine.BuildResult, mode output.Mode) error {
	fmt.Printf("Preset: %s\n", result.PresetName)
	fmt.Printf("Blocks: %d\n", len(result.BlocksUsed))
	fmt.Printf("Estimated tokens: %d\n", result.EstimatedTokens)
	fmt.Printf("Output: %s\n", mode)
	for _, warning := range result.Warnings {
		fmt.Printf("Warning: %s\n", warning)
	}
	return nil
}

type csvFlag struct {
	values []string
}

func (f *csvFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *csvFlag) Type() string {
	return "csv"
}

func (f *csvFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		f.values = append(f.values, part)
	}
	return nil
}

func (f *csvFlag) Values() []string {
	return append([]string(nil), f.values...)
}

type kvFlag struct {
	values map[string]string
}

func (f *kvFlag) String() string {
	if len(f.values) == 0 {
		return ""
	}

	parts := make([]string, 0, len(f.values))
	for key, value := range f.values {
		parts = append(parts, key+"="+value)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func (f *kvFlag) Type() string {
	return "key=value"
}

func (f *kvFlag) Set(value string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}

	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid --var value %q, expected key=value", value)
	}

	key := strings.TrimSpace(parts[0])
	if key == "" {
		return fmt.Errorf("invalid --var value %q, key is empty", value)
	}

	f.values[key] = strings.TrimSpace(parts[1])
	return nil
}

func (f *kvFlag) Values() map[string]string {
	result := map[string]string{}
	for key, value := range f.values {
		result[key] = value
	}
	return result
}
