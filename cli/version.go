package cli

import (
	"flag"
	"fmt"
	"os"
)

func runVersion(build VersionInfo, args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("version: %s\n", build.Version)
	fmt.Printf("commit: %s\n", build.Commit)
	fmt.Printf("date: %s\n", build.Date)
	return nil
}
