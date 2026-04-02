package main

import (
	"fmt"
	"os"

	"github.com/singl3focus/pmp/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	initConsole()

	if err := cli.Execute(os.Args[1:], cli.VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
