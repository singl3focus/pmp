package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/singl3focus/pmp/internal/block"
	"github.com/singl3focus/pmp/internal/config"
	"github.com/singl3focus/pmp/internal/output"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "doctor",
		Short:         "Validate configuration and block resolution",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	active, err := config.LoadActive(".")
	if err != nil {
		return err
	}

	hasIssues := false

	fmt.Println("Configuration")
	fmt.Printf("  Active config: %s\n", active.ActiveConfigPath)
	if active.ProjectRoot != "" {
		fmt.Printf("  Project root: %s\n", active.ProjectRoot)
	}
	if active.GlobalRoot != "" {
		fmt.Printf("  Global root: %s\n", active.GlobalRoot)
	}

	fmt.Println("Base")
	if len(active.Config.Base.AlwaysInclude) == 0 {
		fmt.Println("  none")
	}
	for _, relPath := range active.Config.Base.AlwaysInclude {
		if _, err := block.Resolve(relPath, active.BaseRoots()); err != nil {
			hasIssues = true
			fmt.Printf("  missing: %s (%v)\n", relPath, err)
			continue
		}
		fmt.Printf("  - %s\n", relPath)
	}

	fmt.Println("Presets")
	names := make([]string, 0, len(active.Config.Presets))
	for name := range active.Config.Presets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		preset := active.Config.Presets[name]
		fmt.Printf("  - %s (%d blocks)\n", name, len(preset.Blocks))
		for _, relPath := range preset.Blocks {
			if _, err := block.Resolve(relPath, active.BlockRoots()); err != nil {
				hasIssues = true
				fmt.Printf("    missing: %s (%v)\n", relPath, err)
			}
		}
	}

	fmt.Println("Block library")
	if _, err := block.LoadMerged(active.BlockRoots()); err != nil {
		hasIssues = true
		fmt.Printf("  invalid: %v\n", err)
	} else {
		fmt.Println("  valid")
	}

	fmt.Println("Clipboard")
	if err := output.CheckClipboard(); err != nil {
		fmt.Printf("  unavailable: %v\n", err)
	} else {
		fmt.Println("  available")
	}

	if hasIssues {
		return fmt.Errorf("doctor found configuration issues")
	}

	return nil
}
