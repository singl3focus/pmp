package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/singl3focus/pmp/internal/block"
	"github.com/singl3focus/pmp/internal/config"
)

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

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

	paths := block.SortedPaths(blocks)
	for _, path := range paths {
		item := blocks[path]
		fmt.Printf("  - %s", path)
		if item.Description != "" {
			fmt.Printf(": %s", item.Description)
		}
		fmt.Printf(" [%s]\n", item.Source)
	}

	return nil
}
