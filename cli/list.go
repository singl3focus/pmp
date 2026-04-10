package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/singl3focus/pmp/internal/block"
	"github.com/singl3focus/pmp/internal/config"
)

func newListCommand() *cobra.Command {
	var showHidden bool
	var verbose bool

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "Show presets and available blocks",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(showHidden, verbose)
		},
	}

	cmd.Flags().BoolVar(&showHidden, "show-hidden", false, "include hidden blocks in output")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show extra block metadata")
	return cmd
}

func runList(showHidden, verbose bool) error {
	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	fmt.Println("Presets")
	presetNames := make([]string, 0, len(active.Config.Presets))
	for name := range active.Config.Presets {
		presetNames = append(presetNames, name)
	}
	sort.Strings(presetNames)
	for _, name := range presetNames {
		preset := active.Config.Presets[name]
		fmt.Printf("  - %s", name)
		if preset.Description != "" {
			fmt.Printf(": %s", preset.Description)
		}
		fmt.Printf(" (%d blocks)\n", len(preset.Blocks))
	}

	fmt.Println()
	fmt.Println("Blocks")
	blocks, err := block.LoadMerged(active.BlockRoots())
	if err != nil {
		return err
	}

	hiddenCount := 0
	for _, item := range blocks {
		if item.Hidden {
			hiddenCount++
		}
	}

	for _, item := range block.SortedBlocks(blocks, showHidden) {
		fmt.Printf("  - %s", item.Path)
		if item.Description != "" {
			fmt.Printf(": %s", item.Description)
		}

		var meta []string
		meta = append(meta, item.Source)
		if item.Weight != 0 {
			meta = append(meta, fmt.Sprintf("weight=%d", item.Weight))
		}
		if item.Hidden {
			meta = append(meta, "hidden")
		}
		if verbose && item.Title != "" {
			meta = append(meta, fmt.Sprintf("title=%q", item.Title))
		}
		if verbose && len(item.Tags) > 0 {
			meta = append(meta, "tags="+strings.Join(item.Tags, ","))
		}

		fmt.Printf(" [%s]\n", strings.Join(meta, ", "))
	}

	if hiddenCount > 0 && !showHidden {
		fmt.Printf("\nHidden blocks: %d not shown (use --show-hidden)\n", hiddenCount)
	}

	return nil
}
