package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/singl3focus/pmp/internal/block"
	"github.com/singl3focus/pmp/internal/config"
)

func newPresetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "preset",
		Short:         "Manage presets in the active config",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newPresetListCommand(),
		newPresetShowCommand(),
		newPresetAddCommand(),
		newPresetDeleteCommand(),
	)

	return cmd
}

func newPresetListCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List presets from the active config",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPresetList()
		},
	}
}

func newPresetShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "show <name>",
		Short:         "Show a preset definition",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPresetShow(args[0])
		},
	}
}

func newPresetAddCommand() *cobra.Command {
	var description string
	var blocks csvFlag

	cmd := &cobra.Command{
		Use:           "add <name>",
		Short:         "Create or update a preset",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPresetAdd(args[0], description, blocks.Values())
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "preset description")
	cmd.Flags().Var(&blocks, "block", "comma separated block paths")
	return cmd
}

func newPresetDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "delete <name>",
		Short:         "Delete a preset from the active config",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPresetDelete(args[0])
		},
	}
}

func runPresetList() error {
	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	writable, err := config.LoadWritableConfig(active)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(writable.Presets))
	for name := range writable.Presets {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println("Presets")
	for _, name := range names {
		preset := writable.Presets[name]
		fmt.Printf("  - %s", name)
		if preset.Description != "" {
			fmt.Printf(": %s", preset.Description)
		}
		fmt.Printf(" (%d blocks)\n", len(preset.Blocks))
	}
	return nil
}

func runPresetShow(name string) error {
	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	writable, err := config.LoadWritableConfig(active)
	if err != nil {
		return err
	}

	preset, ok := writable.Presets[name]
	if !ok {
		return fmt.Errorf("preset %q not found", name)
	}

	fmt.Printf("Name: %s\n", name)
	if preset.Description != "" {
		fmt.Printf("Description: %s\n", preset.Description)
	}
	fmt.Printf("Blocks: %d\n", len(preset.Blocks))
	for _, path := range preset.Blocks {
		fmt.Printf("  - %s\n", path)
	}
	if len(preset.DefaultVars) > 0 {
		fmt.Println("Default vars:")
		keys := make([]string, 0, len(preset.DefaultVars))
		for key := range preset.DefaultVars {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("  - %s=%s\n", key, preset.DefaultVars[key])
		}
	}
	return nil
}

func runPresetAdd(name, description string, blocks []string) error {
	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	writable, err := config.LoadWritableConfig(active)
	if err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("preset name is required")
	}

	existing, ok := writable.Presets[name]
	preset := existing
	if !ok {
		preset = config.Preset{}
	}
	if description != "" || !ok {
		preset.Description = strings.TrimSpace(description)
	}

	cleanBlocks := dedupeStrings(blocks)
	if len(cleanBlocks) > 0 {
		resolved, err := resolveBlocks(active, cleanBlocks)
		if err != nil {
			return err
		}
		preset.Blocks = resolved
	} else if !ok {
		return fmt.Errorf("at least one --block is required for a new preset")
	}

	if err := config.SavePreset(active, name, preset); err != nil {
		return err
	}

	fmt.Printf("Saved preset %q with %d blocks\n", name, len(preset.Blocks))
	return nil
}

func runPresetDelete(name string) error {
	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	if err := config.DeletePreset(active, name); err != nil {
		return err
	}

	fmt.Printf("Deleted preset %q\n", name)
	return nil
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func resolveBlocks(active config.Active, paths []string) ([]string, error) {
	resolved := make([]string, 0, len(paths))
	for _, path := range paths {
		item, err := block.Resolve(path, active.BlockRoots())
		if err != nil {
			return nil, fmt.Errorf("resolve block %q: %w", path, err)
		}
		resolved = append(resolved, item.Path)
	}
	return resolved, nil
}
